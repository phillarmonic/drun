package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/orchestration"
	"github.com/phillarmonic/drun/internal/repository"
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

// orchestrateUpWithProgress brings up services with full rebuild and repository updates
func (e *Engine) orchestrateUpWithProgress(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	// "up" command: update repos on default branches and force rebuild
	return e.orchestrateStartWithProgress(ctx, orch, orderedServices, services, true, true)
}

// orchestrateStartWithProgress starts services with BuildKit-style progress display
func (e *Engine) orchestrateStartWithProgress(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement, updateRepos bool, forceBuild bool) error {
	actionVerb := "Starting"
	if updateRepos || forceBuild {
		actionVerb = "Bringing up"
	}
	_, _ = fmt.Fprintf(e.output, "🚀  %s orchestration: %s\n", actionVerb, orch.Name)
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

	// Check DNS resolution for specified domains
	if err := e.checkDNSResolution(orch); err != nil {
		// DNS check failures are warnings, not errors
		_, _ = fmt.Fprintf(e.output, "⚠️  %v\n\n", err)
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

		// Check for repository updates first (if repository is configured)
		hasRepoUpdates := false
		needsClone := false
		if service.Repository != nil {
			workDir, err := os.Getwd()
			if err != nil {
				progress.FailService(serviceName, fmt.Errorf("failed to get working directory: %w", err))
				progress.RenderInline(serviceName)
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			repoManager := repository.NewManager(workDir)
			repoConfig := &orchestration.Repository{
				URL:           service.Repository.URL,
				Branch:        service.Repository.Branch,
				Tag:           service.Repository.Tag,
				SSHKey:        service.Repository.SSHKey,
				Clone:         service.Repository.Clone,
				UpdateOnStart: service.Repository.UpdateOnStart,
			}

			// Check if repository exists
			// Service path is already absolute after resolution, so use it directly
			fullPath := service.Path
			if !filepath.IsAbs(fullPath) {
				// Fallback: join with workDir if somehow still relative
				fullPath = filepath.Join(workDir, service.Path)
			}

			if e.verbose {
				_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Checking repository at: %s\n", fullPath)
			}

			if _, err := os.Stat(filepath.Join(fullPath, ".git")); os.IsNotExist(err) {
				// Repository doesn't exist, needs to be cloned
				needsClone = true
				if e.verbose {
					_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Repository not found at %s, will clone\n", fullPath)
				}
			} else {
				// Repository exists - check if we should update it
				// Respect the "update on start" setting
				if !service.Repository.UpdateOnStart {
					// Skip update check if explicitly disabled
					if e.verbose {
						_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Repository update disabled for %s (update on start: false)\n", serviceName)
					}
				} else if updateRepos {
					// For "up" command: check if on default branch and force update
					currentBranch, err := repoManager.GetCurrentBranch(context.Background(), service.Path)
					if err == nil && (currentBranch == "main" || currentBranch == "master") {
						hasRepoUpdates = true
						_, _ = fmt.Fprintf(e.output, "\n  📥 Updating repository on default branch (%s) for %s\n", currentBranch, serviceName)
					}
				} else {
					// For "start" command: only check for updates, don't force
					progress.UpdateService(serviceName, "checking", "Checking for repository updates...")
					progress.RenderInline(serviceName)

					hasUpdates, err := repoManager.HasRemoteUpdates(context.Background(), repoConfig, service.Path)
					if err != nil {
						if e.verbose {
							_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Unable to check for updates for %s: %v\n", serviceName, err)
						}
						// If we can't check for updates, proceed with existing logic
					} else {
						hasRepoUpdates = hasUpdates
						if hasUpdates {
							_, _ = fmt.Fprintf(e.output, "\n  📥 Repository updates available for %s\n", serviceName)
						}
					}
				}
			}
		}

		// If service is already healthy and no repository updates and not forcing rebuild, skip it
		if alreadyHealthy && stateErr == nil && !hasRepoUpdates && !needsClone && !forceBuild {
			progress.CompleteService(serviceName, "Already running (no updates)")
			progress.RenderInline(serviceName)
			continue
		}

		// Clone/update repository FIRST if configured
		if service.Repository != nil {
			workDir, err := os.Getwd()
			if err != nil {
				progress.FailService(serviceName, fmt.Errorf("failed to get working directory: %w", err))
				progress.RenderInline(serviceName)
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			repoManager := repository.NewManager(workDir)
			repoConfig := &orchestration.Repository{
				URL:           service.Repository.URL,
				Branch:        service.Repository.Branch,
				Tag:           service.Repository.Tag,
				SSHKey:        service.Repository.SSHKey,
				Clone:         service.Repository.Clone,
				UpdateOnStart: service.Repository.UpdateOnStart,
			}

			if needsClone {
				// Repository doesn't exist, clone it
				progress.UpdateService(serviceName, "cloning", "Cloning repository...")
				progress.RenderInline(serviceName)

				_, _ = fmt.Fprintf(e.output, "\n  📂 Cloning to: %s\n", service.Path)

				if err := repoManager.Clone(context.Background(), repoConfig, service.Path); err != nil {
					progress.FailService(serviceName, fmt.Errorf("repository clone failed: %w", err))
					progress.RenderInline(serviceName)

					if orch.StopOnFailure || orch.CircuitBreaker {
						_, _ = fmt.Fprintf(e.output, "\n🔴 Circuit breaker triggered! Rolling back dependent services...\n\n")
						return fmt.Errorf("failed to clone repository for service '%s': %w", serviceName, err)
					}
					return fmt.Errorf("failed to clone repository for service '%s': %w", serviceName, err)
				}
			} else if hasRepoUpdates {
				// Repository exists and has updates, pull them
				progress.UpdateService(serviceName, "updating", "Pulling repository updates...")
				progress.RenderInline(serviceName)

				_, _ = fmt.Fprintf(e.output, "\n  📂 Updating repository at: %s\n", service.Path)

				if err := repoManager.Update(context.Background(), repoConfig, service.Path); err != nil {
					progress.FailService(serviceName, fmt.Errorf("repository update failed: %w", err))
					progress.RenderInline(serviceName)

					if orch.StopOnFailure || orch.CircuitBreaker {
						_, _ = fmt.Fprintf(e.output, "\n🔴 Circuit breaker triggered! Rolling back dependent services...\n\n")
						return fmt.Errorf("failed to update repository for service '%s': %w", serviceName, err)
					}
					return fmt.Errorf("failed to update repository for service '%s': %w", serviceName, err)
				}
				_, _ = fmt.Fprintf(e.output, "  ✓ Repository updated for %s\n\n", serviceName)
			}

			progress.UpdateService(serviceName, "starting", "Repository ready")
			progress.RenderInline(serviceName)
		}

		// Run pre-task if specified
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

		// Now display "Starting service..." before build and start
		progress.UpdateService(serviceName, "starting", "Starting service...")
		progress.RenderInline(serviceName)

		// Build the service if build is required OR if forceBuild is set
		shouldBuild := (service.Build != nil && service.Build.Required) || forceBuild
		if shouldBuild {
			progress.UpdateService(serviceName, "building", "Building container...")
			progress.RenderInline(serviceName)

			// Show build output header
			_, _ = fmt.Fprintf(e.output, "\n🔨 Building %s:\n", serviceName)

			if err := e.performServiceBuild(ctx, service, false, true); err != nil {
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

	// Display HTTP service URLs
	e.displayServiceURLs(orderedServices, services)

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

// displayServiceURLs displays HTTP health check URLs for all services
func (e *Engine) displayServiceURLs(orderedServices []string, services map[string]*ast.ServiceStatement) {
	httpServices := []struct {
		name string
		url  string
	}{}

	// Collect all services with HTTP health checks
	for _, serviceName := range orderedServices {
		service := services[serviceName]
		if service.HealthCheck != nil && service.HealthCheck.Type == "http" && service.HealthCheck.Endpoint != "" {
			httpServices = append(httpServices, struct {
				name string
				url  string
			}{
				name: serviceName,
				url:  service.HealthCheck.Endpoint,
			})
		}
	}

	// Display URLs if any found
	if len(httpServices) > 0 {
		_, _ = fmt.Fprintf(e.output, "\n🌐 Service URLs:\n")
		for _, svc := range httpServices {
			_, _ = fmt.Fprintf(e.output, "   • %s: %s\n", svc.name, svc.url)
		}
	}
}
