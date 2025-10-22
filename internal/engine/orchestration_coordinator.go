package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/phillarmonic/drun/internal/domain/orchestration"
)

// OrchestrationCoordinator coordinates orchestration of multiple services
type OrchestrationCoordinator struct {
	executor *OrchestrationExecutor
	mu       sync.RWMutex
}

// NewOrchestrationCoordinator creates a new orchestration coordinator
func NewOrchestrationCoordinator(executor *OrchestrationExecutor) *OrchestrationCoordinator {
	return &OrchestrationCoordinator{
		executor: executor,
	}
}

// StartOrchestration starts an orchestration group
func (oc *OrchestrationCoordinator) StartOrchestration(ctx context.Context, execCtx *ExecutionContext, orchestrationName string) error {
	orchestr, err := oc.executor.orchestrRegistry.Get(orchestrationName)
	if err != nil {
		return fmt.Errorf("orchestration not found: %w", err)
	}

	// Mark orchestration as starting
	orchestr.MarkStarting()

	// Run pre-task if specified
	if orchestr.PreTask != "" {
		if err := oc.executor.executeTask(ctx, execCtx, orchestr.PreTask); err != nil {
			orchestr.MarkFailed()
			return fmt.Errorf("orchestration pre-task failed: %w", err)
		}
	}

	// Create context with timeout
	if orchestr.StartupTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, orchestr.StartupTimeout)
		defer cancel()
	}

	// Start services based on strategy
	switch orchestr.Strategy {
	case orchestration.StrategySequential:
		err = oc.startServicesSequential(ctx, execCtx, orchestr)
	case orchestration.StrategyParallel:
		err = oc.startServicesParallel(ctx, execCtx, orchestr)
	case orchestration.StrategyDependencyBased:
		err = oc.startServicesDependencyBased(ctx, execCtx, orchestr)
	default:
		err = fmt.Errorf("unknown orchestration strategy: %s", orchestr.Strategy)
	}

	if err != nil {
		orchestr.MarkFailed()

		// Stop all services if stop_on_failure is enabled
		if orchestr.StopOnFailure {
			oc.StopOrchestration(context.Background(), execCtx, orchestrationName)
		}

		return err
	}

	// Start health monitoring if configured
	if orchestr.HealthCheckInterval > 0 {
		go oc.monitorOrchestrationHealth(ctx, execCtx, orchestr)
	}

	orchestr.MarkHealthy()
	return nil
}

// StopOrchestration stops an orchestration group
func (oc *OrchestrationCoordinator) StopOrchestration(ctx context.Context, execCtx *ExecutionContext, orchestrationName string) error {
	orchestr, err := oc.executor.orchestrRegistry.Get(orchestrationName)
	if err != nil {
		return fmt.Errorf("orchestration not found: %w", err)
	}

	// Mark orchestration as stopping
	orchestr.Status = "stopping"

	// Create context with timeout
	if orchestr.ShutdownTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, orchestr.ShutdownTimeout)
		defer cancel()
	}

	// Stop services in reverse order
	services := reverseStringSlice(orchestr.Services)

	for _, serviceName := range services {
		if err := oc.executor.StopService(ctx, execCtx, serviceName); err != nil {
			// Log error but continue stopping other services
			fmt.Printf("Error stopping service %s: %v\n", serviceName, err)
		}
	}

	// Run post-task if specified
	if orchestr.PostTask != "" {
		if err := oc.executor.executeTask(ctx, execCtx, orchestr.PostTask); err != nil {
			return fmt.Errorf("orchestration post-task failed: %w", err)
		}
	}

	orchestr.MarkStopped()
	return nil
}

// RestartOrchestration restarts an orchestration group
func (oc *OrchestrationCoordinator) RestartOrchestration(ctx context.Context, execCtx *ExecutionContext, orchestrationName string) error {
	if err := oc.StopOrchestration(ctx, execCtx, orchestrationName); err != nil {
		return fmt.Errorf("failed to stop orchestration: %w", err)
	}

	if err := oc.StartOrchestration(ctx, execCtx, orchestrationName); err != nil {
		return fmt.Errorf("failed to start orchestration: %w", err)
	}

	return nil
}

// startServicesSequential starts services one after another
func (oc *OrchestrationCoordinator) startServicesSequential(ctx context.Context, execCtx *ExecutionContext, orchestr *orchestration.Orchestration) error {
	for _, serviceName := range orchestr.Services {
		if err := oc.executor.StartService(ctx, execCtx, serviceName); err != nil {
			return fmt.Errorf("failed to start service %s: %w", serviceName, err)
		}
	}
	return nil
}

// startServicesParallel starts all services in parallel
func (oc *OrchestrationCoordinator) startServicesParallel(ctx context.Context, execCtx *ExecutionContext, orchestr *orchestration.Orchestration) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(orchestr.Services))

	for _, serviceName := range orchestr.Services {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := oc.executor.StartService(ctx, execCtx, name); err != nil {
				errChan <- fmt.Errorf("failed to start service %s: %w", name, err)
			}
		}(serviceName)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	return nil
}

// startServicesDependencyBased starts services based on their dependencies
func (oc *OrchestrationCoordinator) startServicesDependencyBased(ctx context.Context, execCtx *ExecutionContext, orchestr *orchestration.Orchestration) error {
	// Build dependency graph
	serviceMap := make(map[string]*orchestration.Service)
	for _, serviceName := range orchestr.Services {
		service, err := oc.executor.serviceRegistry.Get(serviceName)
		if err != nil {
			return fmt.Errorf("service not found: %w", err)
		}
		serviceMap[serviceName] = service
	}

	// Topological sort based on dependencies
	sorted, err := oc.topologicalSort(orchestr.Services, serviceMap)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Start services in dependency order
	for _, serviceName := range sorted {
		// Check if service has dependencies
		service := serviceMap[serviceName]
		if len(service.Dependencies) > 0 {
			// Wait for dependencies to be healthy
			for _, depName := range service.Dependencies {
				if err := oc.waitForServiceHealthy(ctx, depName); err != nil {
					return fmt.Errorf("dependency %s not healthy: %w", depName, err)
				}
			}
		}

		// Start service
		if err := oc.executor.StartService(ctx, execCtx, serviceName); err != nil {
			return fmt.Errorf("failed to start service %s: %w", serviceName, err)
		}
	}

	return nil
}

// topologicalSort performs topological sort on services based on dependencies
func (oc *OrchestrationCoordinator) topologicalSort(services []string, serviceMap map[string]*orchestration.Service) ([]string, error) {
	// Build adjacency list
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize
	for _, serviceName := range services {
		adj[serviceName] = []string{}
		inDegree[serviceName] = 0
	}

	// Build graph
	for _, serviceName := range services {
		service := serviceMap[serviceName]
		for _, dep := range service.Dependencies {
			// dep -> serviceName (dep must come before serviceName)
			adj[dep] = append(adj[dep], serviceName)
			inDegree[serviceName]++
		}
	}

	// Kahn's algorithm
	queue := []string{}
	for serviceName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, serviceName)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, neighbor := range adj[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != len(services) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// waitForServiceHealthy waits for a service to become healthy
func (oc *OrchestrationCoordinator) waitForServiceHealthy(ctx context.Context, serviceName string) error {
	service, err := oc.executor.serviceRegistry.Get(serviceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	// Poll for health status
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for service %s to be healthy", serviceName)
		case <-ticker.C:
			if service.IsHealthy() {
				return nil
			}
		}
	}
}

// monitorOrchestrationHealth monitors the health of all services in an orchestration
func (oc *OrchestrationCoordinator) monitorOrchestrationHealth(ctx context.Context, execCtx *ExecutionContext, orchestr *orchestration.Orchestration) {
	ticker := time.NewTicker(orchestr.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			healthy, unhealthy := oc.checkAllServicesHealth(ctx, orchestr)

			if len(unhealthy) > 0 {
				orchestr.MarkDegraded()

				// Check circuit breaker
				if orchestr.CircuitBreaker {
					orchestr.MarkFailed()

					if orchestr.StopOnFailure {
						// Stop all services
						oc.StopOrchestration(context.Background(), execCtx, orchestr.Name)
					}

					// Attempt recovery if configured
					if orchestr.ShouldRecover() {
						go oc.attemptRecovery(context.Background(), execCtx, orchestr)
					}

					return
				}
			} else if len(healthy) == len(orchestr.Services) {
				orchestr.MarkHealthy()
			}
		}
	}
}

// checkAllServicesHealth checks the health of all services
func (oc *OrchestrationCoordinator) checkAllServicesHealth(ctx context.Context, orchestr *orchestration.Orchestration) (healthy, unhealthy []string) {
	for _, serviceName := range orchestr.Services {
		service, err := oc.executor.serviceRegistry.Get(serviceName)
		if err != nil {
			unhealthy = append(unhealthy, serviceName)
			continue
		}

		if service.HealthCheck != nil {
			if err := oc.executor.healthChecker.Check(ctx, service.HealthCheck); err != nil {
				service.MarkUnhealthy()
				unhealthy = append(unhealthy, serviceName)
			} else {
				service.MarkHealthy()
				healthy = append(healthy, serviceName)
			}
		} else {
			// No health check, assume healthy if running
			if service.IsRunning() {
				healthy = append(healthy, serviceName)
			} else {
				unhealthy = append(unhealthy, serviceName)
			}
		}
	}

	return healthy, unhealthy
}

// attemptRecovery attempts to recover from failure
func (oc *OrchestrationCoordinator) attemptRecovery(ctx context.Context, execCtx *ExecutionContext, orchestr *orchestration.Orchestration) {
	// Wait for recovery timeout
	time.Sleep(orchestr.RecoveryTimeout)

	// Reset circuit breaker
	orchestr.ResetCircuitBreaker()

	// Attempt to restart orchestration
	if err := oc.RestartOrchestration(ctx, execCtx, orchestr.Name); err != nil {
		orchestr.MarkFailed()
	}
}

// reverseStringSlice reverses a string slice
func reverseStringSlice(s []string) []string {
	result := make([]string, len(s))
	for i, v := range s {
		result[len(s)-1-i] = v
	}
	return result
}

// GetOrchestrationStatus returns the status of an orchestration
func (oc *OrchestrationCoordinator) GetOrchestrationStatus(orchestrationName string) (*OrchestrationStatus, error) {
	orchestr, err := oc.executor.orchestrRegistry.Get(orchestrationName)
	if err != nil {
		return nil, fmt.Errorf("orchestration not found: %w", err)
	}

	status := &OrchestrationStatus{
		Name:         orchestr.Name,
		Status:       string(orchestr.Status),
		FailureCount: orchestr.FailureCount,
		CircuitOpen:  orchestr.CircuitOpen,
		Services:     make(map[string]string),
	}

	for _, serviceName := range orchestr.Services {
		service, err := oc.executor.serviceRegistry.Get(serviceName)
		if err != nil {
			status.Services[serviceName] = "unknown"
			continue
		}
		status.Services[serviceName] = string(service.Status)
	}

	return status, nil
}

// OrchestrationStatus represents the status of an orchestration
type OrchestrationStatus struct {
	Name         string
	Status       string
	FailureCount int
	CircuitOpen  bool
	Services     map[string]string
}

// BuildDependencyGraph builds a dependency graph for all services
func (oc *OrchestrationCoordinator) BuildDependencyGraph(orchestrationName string) (map[string][]string, error) {
	orchestr, err := oc.executor.orchestrRegistry.Get(orchestrationName)
	if err != nil {
		return nil, fmt.Errorf("orchestration not found: %w", err)
	}

	graph := make(map[string][]string)

	for _, serviceName := range orchestr.Services {
		service, err := oc.executor.serviceRegistry.Get(serviceName)
		if err != nil {
			return nil, fmt.Errorf("service not found: %w", err)
		}

		graph[serviceName] = service.Dependencies
	}

	return graph, nil
}

// GetServiceStartOrder returns the order in which services should be started
func (oc *OrchestrationCoordinator) GetServiceStartOrder(orchestrationName string) ([]string, error) {
	orchestr, err := oc.executor.orchestrRegistry.Get(orchestrationName)
	if err != nil {
		return nil, fmt.Errorf("orchestration not found: %w", err)
	}

	// Build service map
	serviceMap := make(map[string]*orchestration.Service)
	for _, serviceName := range orchestr.Services {
		service, err := oc.executor.serviceRegistry.Get(serviceName)
		if err != nil {
			return nil, fmt.Errorf("service not found: %w", err)
		}
		serviceMap[serviceName] = service
	}

	// Perform topological sort
	return oc.topologicalSort(orchestr.Services, serviceMap)
}
