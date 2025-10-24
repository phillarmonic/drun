package engine

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
)

// ServiceProgress tracks the progress of a service operation
type ServiceProgress struct {
	Name      string
	Status    string // "pending", "starting", "healthy", "failed", "stopped"
	Message   string
	StartTime time.Time
	EndTime   time.Time
	Error     error
	mu        sync.RWMutex
}

// ProgressDisplay manages the visual display of orchestration progress
type ProgressDisplay struct {
	services map[string]*ServiceProgress
	output   io.Writer
	mu       sync.RWMutex
}

// NewProgressDisplay creates a new progress display
func NewProgressDisplay(output io.Writer) *ProgressDisplay {
	return &ProgressDisplay{
		services: make(map[string]*ServiceProgress),
		output:   output,
	}
}

// StartService marks a service as starting
func (pd *ProgressDisplay) StartService(name string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.services[name] = &ServiceProgress{
		Name:      name,
		Status:    "starting",
		Message:   "Starting service...",
		StartTime: time.Now(),
	}
}

// UpdateService updates a service's status
func (pd *ProgressDisplay) UpdateService(name, status, message string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if svc, ok := pd.services[name]; ok {
		svc.mu.Lock()
		svc.Status = status
		svc.Message = message
		svc.mu.Unlock()
	}
}

// FailService marks a service as failed
func (pd *ProgressDisplay) FailService(name string, err error) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if svc, ok := pd.services[name]; ok {
		svc.mu.Lock()
		svc.Status = "failed"
		svc.Error = err
		svc.EndTime = time.Now()
		svc.mu.Unlock()
	}
}

// CompleteService marks a service as completed successfully
func (pd *ProgressDisplay) CompleteService(name, message string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if svc, ok := pd.services[name]; ok {
		svc.mu.Lock()
		svc.Status = "healthy"
		svc.Message = message
		svc.EndTime = time.Now()
		svc.mu.Unlock()
	}
}

// Render displays the current progress state
func (pd *ProgressDisplay) Render() {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	for name, svc := range pd.services {
		svc.mu.RLock()
		icon := pd.getStatusIcon(svc.Status)
		elapsed := ""
		if !svc.StartTime.IsZero() {
			if svc.EndTime.IsZero() {
				elapsed = fmt.Sprintf(" [%s]", time.Since(svc.StartTime).Round(time.Second))
			} else {
				elapsed = fmt.Sprintf(" [%s]", svc.EndTime.Sub(svc.StartTime).Round(time.Second))
			}
		}

		msg := svc.Message
		if svc.Error != nil {
			msg = fmt.Sprintf("%s: %v", msg, svc.Error)
		}

		_, _ = fmt.Fprintf(pd.output, "  %s %-12s %s%s\n", icon, name, msg, elapsed)
		svc.mu.RUnlock()
	}
}

// RenderInline renders a single service update inline
func (pd *ProgressDisplay) RenderInline(name string) {
	pd.mu.RLock()
	svc, ok := pd.services[name]
	pd.mu.RUnlock()

	if !ok {
		return
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	icon := pd.getStatusIcon(svc.Status)
	elapsed := ""
	if !svc.StartTime.IsZero() {
		if svc.EndTime.IsZero() {
			elapsed = fmt.Sprintf(" [%s]", time.Since(svc.StartTime).Round(time.Second))
		} else {
			elapsed = fmt.Sprintf(" [%s]", svc.EndTime.Sub(svc.StartTime).Round(time.Second))
		}
	}

	msg := svc.Message
	if svc.Error != nil {
		msg = fmt.Sprintf("%s: %v", msg, svc.Error)
	}

	_, _ = fmt.Fprintf(pd.output, "  %s %-12s %s%s\n", icon, name, msg, elapsed)
}

func (pd *ProgressDisplay) getStatusIcon(status string) string {
	switch status {
	case "pending":
		return "⏸️ "
	case "building":
		return "🔨 "
	case "starting":
		return "🔄 "
	case "healthy":
		return "✅ "
	case "failed":
		return "❌ "
	case "stopped":
		return "⏹️ "
	case "stopping":
		return "🛑 "
	default:
		return "  "
	}
}

// RenderSummary displays a final summary
func (pd *ProgressDisplay) RenderSummary() {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	var successful, failed, total int
	total = len(pd.services)

	for _, svc := range pd.services {
		svc.mu.RLock()
		switch svc.Status {
		case "healthy":
			successful++
		case "failed":
			failed++
		}
		svc.mu.RUnlock()
	}

	_, _ = fmt.Fprintf(pd.output, "\n")
	if failed > 0 {
		_, _ = fmt.Fprintf(pd.output, "❌ %d/%d services failed\n", failed, total)
	} else {
		_, _ = fmt.Fprintf(pd.output, "✅ %d/%d services completed successfully\n", successful, total)
	}
}

// orchestrateStartWithProgress starts services with BuildKit-style progress display
func (e *Engine) orchestrateStartWithProgress(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🚀  Starting orchestration: %s\n", orch.Name)
	_, _ = fmt.Fprintf(e.output, "   %d services in dependency order\n", len(orderedServices))
	if orch.CircuitBreaker || orch.StopOnFailure {
		_, _ = fmt.Fprintf(e.output, "   🔴  Circuit breaker: ENABLED - will stop all on failure\n")
	}
	_, _ = fmt.Fprintf(e.output, "\n")

	progress := NewProgressDisplay(e.output)
	pd := progress // Alias for use in nested scope

	// Check and provision Docker networks before starting services
	if err := e.checkAndProvisionNetworks(services); err != nil {
		return fmt.Errorf("network provisioning failed: %w", err)
	}

	// Initialize all services as pending
	for _, name := range orderedServices {
		progress.services[name] = &ServiceProgress{
			Name:   name,
			Status: "pending",
		}
	}

	// Show initial state
	progress.Render()
	_, _ = fmt.Fprintf(e.output, "\n")

	// Start services one by one
	for _, serviceName := range orderedServices {
		service := services[serviceName]

		// Update to starting
		progress.StartService(serviceName)
		progress.UpdateService(serviceName, "starting", "Checking current state...")
		progress.RenderInline(serviceName)

		alreadyHealthy, stateErr := e.serviceIsRunningAndHealthy(service)
		if stateErr != nil && e.verbose {
			_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Unable to confirm current state for %s: %v\n", serviceName, stateErr)
		}
		if alreadyHealthy && stateErr == nil {
			progress.CompleteService(serviceName, "Already running")
			progress.RenderInline(serviceName)
			continue
		}

		progress.UpdateService(serviceName, "starting", "Starting service...")
		progress.RenderInline(serviceName)

		if service.PreTask != "" {
			progress.UpdateService(serviceName, "pre", fmt.Sprintf("Running pre-task %s...", service.PreTask))
			progress.RenderInline(serviceName)

			if err := e.runServiceHook(ctx, service.PreTask, serviceName, "pre"); err != nil {
				progress.FailService(serviceName, err)
				progress.RenderInline(serviceName)
				return err
			}

			progress.UpdateService(serviceName, "starting", "Pre-task complete")
			progress.RenderInline(serviceName)
		}

		// Build the service if build is required
		if service.Build != nil && service.Build.Required {
			progress.UpdateService(serviceName, "building", "Building container...")
			progress.RenderInline(serviceName)

			// Show build output header
			_, _ = fmt.Fprintf(e.output, "\n🔨 Building %s:\n", serviceName)

			if err := e.performServiceBuild(ctx, service, false); err != nil {
				progress.FailService(serviceName, err)
				progress.RenderInline(serviceName)

				// Check if we should stop on failure
				if orch.StopOnFailure || orch.CircuitBreaker {
					_, _ = fmt.Fprintf(e.output, "\n🔴 Circuit breaker triggered! Rolling back dependent services...\n\n")

					// Only stop services that depend on the failed service
					startedServices := []string{}
					failedServiceIndex := -1

					// Find the index of the failed service
					for i, svcName := range orderedServices {
						if svcName == serviceName {
							failedServiceIndex = i
							break
						}
					}

					// Only stop services that were started after the failed service
					for i := failedServiceIndex + 1; i < len(orderedServices); i++ {
						svcName := orderedServices[i]
						if svc, ok := pd.services[svcName]; ok {
							svc.mu.RLock()
							isStarted := svc.Status == "healthy" || svc.Status == "starting"
							svc.mu.RUnlock()

							if isStarted {
								startedServices = append(startedServices, svcName)
							}
						}
					}

					// Stop dependent services in reverse order
					for i := len(startedServices) - 1; i >= 0; i-- {
						svcName := startedServices[i]
						pd.UpdateService(svcName, "stopping", "Rolling back...")
						pd.RenderInline(svcName)
						_ = e.stopService(services[svcName]) // Ignore error during rollback
						pd.UpdateService(svcName, "stopped", "Stopped (rollback)")
						pd.RenderInline(svcName)
					}

					_, _ = fmt.Fprintf(e.output, "\n")
					progress.RenderSummary()
					return fmt.Errorf("circuit breaker: failed to build '%s', dependent services stopped", serviceName)
				}

				progress.RenderSummary()
				return fmt.Errorf("failed to build service '%s': %w", serviceName, err)
			}

			// Show build completion and update progress
			_, _ = fmt.Fprintf(e.output, "✅ Build completed for %s\n\n", serviceName)
			progress.UpdateService(serviceName, "starting", "Build complete, starting...")
			progress.RenderInline(serviceName)
		}

		// Start the service
		if err := e.startService(service); err != nil {
			progress.FailService(serviceName, err)
			progress.RenderInline(serviceName)

			// Check if we should stop on failure
			if orch.StopOnFailure || orch.CircuitBreaker {
				_, _ = fmt.Fprintf(e.output, "\n🔴 Circuit breaker triggered! Rolling back dependent services...\n\n")

				// Only stop services that depend on the failed service
				// For now, we'll stop all services that were started after the failed one
				// In the future, we could implement proper dependency tracking
				startedServices := []string{}
				failedServiceIndex := -1

				// Find the index of the failed service
				for i, svcName := range orderedServices {
					if svcName == serviceName {
						failedServiceIndex = i
						break
					}
				}

				// Only stop services that were started after the failed service
				// This is a simple heuristic - in practice, you might want more sophisticated dependency tracking
				for i := failedServiceIndex + 1; i < len(orderedServices); i++ {
					svcName := orderedServices[i]
					if svc, ok := pd.services[svcName]; ok {
						svc.mu.RLock()
						isStarted := svc.Status == "healthy" || svc.Status == "starting"
						svc.mu.RUnlock()

						if isStarted {
							startedServices = append(startedServices, svcName)
						}
					}
				}

				// Stop dependent services in reverse order
				for i := len(startedServices) - 1; i >= 0; i-- {
					svcName := startedServices[i]
					pd.UpdateService(svcName, "stopping", "Rolling back...")
					pd.RenderInline(svcName)
					_ = e.stopService(services[svcName]) // Ignore error during rollback
					pd.UpdateService(svcName, "stopped", "Stopped (rollback)")
					pd.RenderInline(svcName)
				}

				_, _ = fmt.Fprintf(e.output, "\n")
				progress.RenderSummary()
				return fmt.Errorf("circuit breaker: failed to start '%s', dependent services stopped", serviceName)
			}

			progress.RenderSummary()
			return fmt.Errorf("failed to start service '%s': %w", serviceName, err)
		}

		// Wait for health check if configured
		if service.HealthCheck != nil {
			progress.UpdateService(serviceName, "starting", "Waiting for health check...")
			progress.RenderInline(serviceName)

			if err := e.waitForHealth(service); err != nil {
				progress.FailService(serviceName, err)
				progress.RenderInline(serviceName)

				// Check if we should stop on failure
				if orch.StopOnFailure || orch.CircuitBreaker {
					_, _ = fmt.Fprintf(e.output, "\n🔴 Circuit breaker triggered! Rolling back dependent services...\n\n")

					// Only stop services that depend on the failed service
					startedServices := []string{}
					failedServiceIndex := -1

					// Find the index of the failed service
					for i, svcName := range orderedServices {
						if svcName == serviceName {
							failedServiceIndex = i
							break
						}
					}

					// Only stop services that were started after the failed service
					for i := failedServiceIndex + 1; i < len(orderedServices); i++ {
						svcName := orderedServices[i]
						if svc, ok := pd.services[svcName]; ok {
							svc.mu.RLock()
							isStarted := svc.Status == "healthy" || svc.Status == "starting"
							svc.mu.RUnlock()

							if isStarted {
								startedServices = append(startedServices, svcName)
							}
						}
					}

					// Stop dependent services in reverse order
					for i := len(startedServices) - 1; i >= 0; i-- {
						svcName := startedServices[i]
						pd.UpdateService(svcName, "stopping", "Rolling back...")
						pd.RenderInline(svcName)
						_ = e.stopService(services[svcName])
						pd.UpdateService(svcName, "stopped", "Stopped (rollback)")
						pd.RenderInline(svcName)
					}

					_, _ = fmt.Fprintf(e.output, "\n")
					progress.RenderSummary()
					return fmt.Errorf("circuit breaker: health check failed for '%s', dependent services stopped", serviceName)
				}

				// Continue but mark as warning (degraded mode)
				progress.UpdateService(serviceName, "starting", fmt.Sprintf("⚠️  Unhealthy: %v", err))
			} else {
				progress.CompleteService(serviceName, "Healthy")
			}
		} else {
			progress.CompleteService(serviceName, "Started")
		}

		progress.RenderInline(serviceName)
	}

	progress.RenderSummary()
	return nil
}

// orchestrateStopWithProgress stops services with progress display
func (e *Engine) orchestrateStopWithProgress(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🛑 Stopping orchestration: %s\n", orch.Name)
	_, _ = fmt.Fprintf(e.output, "   %d services in reverse order\n\n", len(orderedServices))

	progress := NewProgressDisplay(e.output)

	// Reverse order for shutdown
	for i := len(orderedServices) - 1; i >= 0; i-- {
		serviceName := orderedServices[i]
		service := services[serviceName]

		progress.StartService(serviceName)
		progress.UpdateService(serviceName, "stopping", "Stopping service...")
		progress.RenderInline(serviceName)

		if err := e.stopService(service); err != nil {
			progress.FailService(serviceName, err)
			progress.RenderInline(serviceName)
			// Continue stopping other services
		} else {
			if service.PostTask != "" {
				progress.UpdateService(serviceName, "post", fmt.Sprintf("Running post-task %s...", service.PostTask))
				progress.RenderInline(serviceName)

				if err := e.runServiceHook(ctx, service.PostTask, serviceName, "post"); err != nil {
					progress.FailService(serviceName, err)
					progress.RenderInline(serviceName)
					return err
				}
			}

			progress.UpdateService(serviceName, "stopped", "Stopped")
			progress.RenderInline(serviceName)
		}
	}

	_, _ = fmt.Fprintf(e.output, "\n✅ All services stopped\n")
	return nil
}
