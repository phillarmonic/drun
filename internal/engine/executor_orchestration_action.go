package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/docker"
	"github.com/phillarmonic/drun/v2/internal/domain/orchestration"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/repository"
)

// executeOrchestration executes orchestration action statements from task bodies
func (e *Engine) executeOrchestration(orchestrStmt *statement.Orchestration, ctx *ExecutionContext) error {
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute orchestration: %s %s\n", orchestrStmt.GroupName, orchestrStmt.Action)
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "[VERBOSE] Orchestration: %s %s\n", orchestrStmt.GroupName, orchestrStmt.Action)
	}

	// Find the orchestration group
	var orchestration *ast.OrchestrateStatement
	if ctx.Program != nil {
		for _, orch := range ctx.Program.Orchestrations {
			if orch.Name == orchestrStmt.GroupName {
				orchestration = orch
				break
			}
		}
	}

	if orchestration == nil {
		return fmt.Errorf("orchestration group '%s' not found", orchestrStmt.GroupName)
	}

	// Find all service definitions
	services := make(map[string]*ast.ServiceStatement)
	if ctx.Program != nil {
		baseDir := ""
		if ctx != nil && ctx.CurrentFile != "" {
			baseDir = filepath.Dir(ctx.CurrentFile)
			if filepath.Base(baseDir) == ".drun" {
				baseDir = filepath.Dir(baseDir)
			}
		} else if cwd, err := os.Getwd(); err == nil {
			baseDir = cwd
		}

		for _, svc := range ctx.Program.Services {
			if baseDir != "" && svc.Path != "" && !filepath.IsAbs(svc.Path) {
				resolved := filepath.Join(baseDir, svc.Path)
				svc.Path = filepath.Clean(resolved)
			}
			services[svc.Name] = svc
		}
	}

	// Get ordered list of services based on dependencies
	orderedServices, err := e.resolveServiceOrder(orchestration.Services, services)
	if err != nil {
		return fmt.Errorf("failed to resolve service order: %w", err)
	}

	// Apply service filters if provided
	if len(orchestrStmt.ServiceFilters) > 0 {
		filteredServices := make([]string, 0, len(orchestrStmt.ServiceFilters))
		for _, rawFilter := range orchestrStmt.ServiceFilters {
			resolved := e.interpolateVariables(rawFilter, ctx)
			if resolved == "" {
				continue
			}
			if _, exists := services[resolved]; !exists {
				return fmt.Errorf("service '%s' not found in orchestration '%s'", resolved, orchestration.Name)
			}
			filteredServices = append(filteredServices, resolved)
		}
		if len(filteredServices) == 0 {
			return fmt.Errorf("no services matched filters: %v", orchestrStmt.ServiceFilters)
		}
		orderedServices = filteredServices
	}

	// Handle "starting from" option for resuming orchestration
	if startingFrom, ok := orchestrStmt.Options["starting_from"]; ok {
		resolved := e.interpolateVariables(startingFrom, ctx)
		if resolved == "" {
			return fmt.Errorf("starting_from service name is empty after interpolation")
		}

		// Find the index of the starting service
		startIdx := -1
		for i, svc := range orderedServices {
			if svc == resolved {
				startIdx = i
				break
			}
		}

		if startIdx == -1 {
			return fmt.Errorf("starting_from service '%s' not found in orchestration '%s'", resolved, orchestration.Name)
		}

		// Check that all dependencies before the starting service are running and healthy
		_, _ = fmt.Fprintf(e.output, "🔍  Checking dependencies before '%s'...\n", resolved)
		for i := 0; i < startIdx; i++ {
			serviceName := orderedServices[i]
			service := services[serviceName]

			healthy, err := e.serviceIsRunningAndHealthy(service)
			if err != nil || !healthy {
				return fmt.Errorf("cannot start from '%s': dependency '%s' is not running or healthy (run full 'up' first)", resolved, serviceName)
			}
			_, _ = fmt.Fprintf(e.output, "  ✓  %s is running and healthy\n", serviceName)
		}

		// Filter to start from the specified service onwards
		_, _ = fmt.Fprintf(e.output, "✅  All dependencies satisfied. Starting from '%s'...\n\n", resolved)
		orderedServices = orderedServices[startIdx:]
	}

	// Execute the action
	switch orchestrStmt.Action {
	case "start":
		if err := e.runOrchestrationHook(ctx, orchestration.PreTask, orchestration.Name, "pre"); err != nil {
			return err
		}
		return e.orchestrateStartWithProgress(ctx, orchestration, orderedServices, services, false, false)
	case "up":
		if err := e.runOrchestrationHook(ctx, orchestration.PreTask, orchestration.Name, "pre"); err != nil {
			return err
		}
		return e.orchestrateUpWithProgress(ctx, orchestration, orderedServices, services)
	case "stop":
		errStop := e.orchestrateStopWithProgress(ctx, orchestration, orderedServices, services)
		errHook := e.runOrchestrationHook(ctx, orchestration.PostTask, orchestration.Name, "post")
		return errors.Join(errStop, errHook)
	case "restart":
		errStop := e.orchestrateStop(ctx, orchestration, orderedServices, services)
		errPost := e.runOrchestrationHook(ctx, orchestration.PostTask, orchestration.Name, "post")
		if err := errors.Join(errStop, errPost); err != nil {
			return err
		}
		if err := e.runOrchestrationHook(ctx, orchestration.PreTask, orchestration.Name, "pre"); err != nil {
			return err
		}
		return e.orchestrateStart(ctx, orchestration, orderedServices, services)
	case "recreate":
		useCache := resolveCacheOption(orchestrStmt.Options, true)
		return e.orchestrateRecreate(ctx, orchestration, orderedServices, services, useCache)
	case "status":
		return e.orchestrateStatus(orchestration, orderedServices, services)
	case "show endpoints", "endpoints":
		return e.orchestrateShowEndpoints(orchestration, orderedServices, services)
	case "health", "health_check":
		return e.orchestrateHealth(orchestration, orderedServices, services)
	case "logs":
		return e.orchestrateLogs(ctx, orchestration, orderedServices, services)
	case "build":
		useCache := resolveCacheOption(orchestrStmt.Options, true)
		return e.orchestrateBuild(ctx, orchestration, orderedServices, services, useCache)
	case "pull":
		return e.orchestratePull(orchestration, orderedServices, services)
	case "down":
		errDown := e.orchestrateDown(ctx, orchestration, orderedServices, services)
		errHook := e.runOrchestrationHook(ctx, orchestration.PostTask, orchestration.Name, "post")
		return errors.Join(errDown, errHook)
	case "clone_repositories", "clone repositories":
		return e.orchestrateCloneRepositories(orchestration, orderedServices, services)
	case "update repositories":
		branchFilter := ""
		if branch, ok := orchestrStmt.Options["branch"]; ok {
			branchFilter = branch
		}
		return e.orchestrateUpdateRepositories(context.Background(), orchestration, orderedServices, services, branchFilter)
	case "list branches":
		branchFilter := ""
		if branch, ok := orchestrStmt.Options["branch"]; ok {
			branchFilter = e.interpolateVariables(branch, ctx)
		}
		return e.orchestrateListBranches(context.Background(), orchestration, orderedServices, services, branchFilter)
	case "switch branch to default":
		// Check if a service filter was specified (e.g., "orchestrate group switch branch to default service name")
		serviceFilter := ""
		if len(orchestrStmt.ServiceFilters) > 0 {
			serviceFilter = e.interpolateVariables(orchestrStmt.ServiceFilters[0], ctx)
		}
		return e.orchestrateSwitchToDefault(context.Background(), orchestration, orderedServices, services, serviceFilter)
	case "set all branches to default":
		return e.orchestrateSetAllDefault(context.Background(), orchestration, orderedServices, services)
	default:
		return fmt.Errorf("unknown orchestration action: %s", orchestrStmt.Action)
	}
}

func parseBoolOption(options map[string]string, key string) (bool, bool) {
	raw, ok := options[key]
	if !ok {
		return false, false
	}

	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "true", "yes", "y", "1", "on", "enabled":
		return true, true
	case "false", "no", "n", "0", "off", "disabled":
		return false, true
	default:
		return false, false
	}
}

func resolveCacheOption(options map[string]string, defaultValue bool) bool {
	if value, ok := parseBoolOption(options, "cache"); ok {
		return value
	}

	if value, ok := parseBoolOption(options, "no_cache"); ok {
		return !value
	}

	return defaultValue
}

// resolveServiceOrder resolves service dependencies and returns services in startup order
func (e *Engine) resolveServiceOrder(serviceNames []string, services map[string]*ast.ServiceStatement) ([]string, error) {
	// Build dependency graph
	deps := make(map[string][]string)
	for _, name := range serviceNames {
		svc, ok := services[name]
		if !ok {
			return nil, fmt.Errorf("service '%s' not found", name)
		}
		deps[name] = svc.Dependencies
	}

	// Topological sort
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	var result []string

	var visit func(string) error
	visit = func(name string) error {
		if temp[name] {
			return fmt.Errorf("circular dependency detected involving service '%s'", name)
		}
		if visited[name] {
			return nil
		}

		temp[name] = true
		for _, dep := range deps[name] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		temp[name] = false
		visited[name] = true
		result = append(result, name)
		return nil
	}

	for _, name := range serviceNames {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// orchestrateStart starts services in dependency order
func (e *Engine) orchestrateStart(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🚀  Starting orchestration: %s\n", orch.Name)

	// Check and provision Docker networks before starting services
	if err := e.checkAndProvisionNetworks(services); err != nil {
		return fmt.Errorf("network provisioning failed: %w", err)
	}

	// Check DNS resolution for specified domains
	if err := e.checkDNSResolution(orch); err != nil {
		// DNS check failures are warnings, not errors
		_, _ = fmt.Fprintf(e.output, "⚠️  %v\n\n", err)
	}

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Starting %s...\n", serviceName)

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
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			repoManager := repository.NewManager(workDir)

			// Use orchestration's SSH key as fallback if service doesn't have one
			sshKey := service.Repository.SSHKey
			if sshKey == "" && orch.GitSSHKey != "" {
				sshKey = orch.GitSSHKey
			}

			repoConfig := &orchestration.Repository{
				URL:           service.Repository.URL,
				Branch:        service.Repository.Branch,
				Tag:           service.Repository.Tag,
				SSHKey:        sshKey,
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
				} else {
					// Check for updates
					_, _ = fmt.Fprintf(e.output, "    🔍  Checking for repository updates for %s...\n", serviceName)

					hasUpdates, err := repoManager.HasRemoteUpdates(context.Background(), repoConfig, service.Path)
					if err != nil {
						if e.verbose {
							_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Unable to check for updates for %s: %v\n", serviceName, err)
						}
						// If we can't check for updates, proceed with existing logic
					} else {
						hasRepoUpdates = hasUpdates
						if hasUpdates {
							_, _ = fmt.Fprintf(e.output, "    📥  Repository updates available for %s\n", serviceName)
						}
					}
				}
			}
		}

		// If service is already healthy and no repository updates, skip it
		if alreadyHealthy && stateErr == nil && !hasRepoUpdates && !needsClone {
			_, _ = fmt.Fprintf(e.output, "    ✓  %s already running and healthy (no updates)\n", serviceName)
			continue
		}

		// Clone/update repository FIRST if configured
		if service.Repository != nil {
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			repoManager := repository.NewManager(workDir)

			// Use orchestration's SSH key as fallback if service doesn't have one
			sshKey := service.Repository.SSHKey
			if sshKey == "" && orch.GitSSHKey != "" {
				sshKey = orch.GitSSHKey
			}

			repoConfig := &orchestration.Repository{
				URL:           service.Repository.URL,
				Branch:        service.Repository.Branch,
				Tag:           service.Repository.Tag,
				SSHKey:        sshKey,
				Clone:         service.Repository.Clone,
				UpdateOnStart: service.Repository.UpdateOnStart,
			}

			if needsClone {
				// Repository doesn't exist, clone it
				_, _ = fmt.Fprintf(e.output, "    📦  Cloning repository for %s...\n", serviceName)
				_, _ = fmt.Fprintf(e.output, "    📂  Target directory: %s\n", service.Path)
				if err := repoManager.Clone(context.Background(), repoConfig, service.Path); err != nil {
					return fmt.Errorf("failed to clone repository for service '%s': %w", serviceName, err)
				}
			} else if hasRepoUpdates {
				// Repository exists and has updates, pull them
				_, _ = fmt.Fprintf(e.output, "    📥  Pulling repository updates for %s...\n", serviceName)
				_, _ = fmt.Fprintf(e.output, "    📂  Repository directory: %s\n", service.Path)
				if err := repoManager.Update(context.Background(), repoConfig, service.Path); err != nil {
					return fmt.Errorf("failed to update repository for service '%s': %w", serviceName, err)
				}
			}

			_, _ = fmt.Fprintf(e.output, "    ✓  Repository ready for %s\n", serviceName)
		}

		// Run pre-task after repository is ready
		if err := e.runServiceHook(ctx, service.PreTask, serviceName, "pre"); err != nil {
			return err
		}

		if service.Build != nil && service.Build.Required {
			_, _ = fmt.Fprintf(e.output, "    🔨  Building %s...\n", serviceName)
			if err := e.performServiceBuild(ctx, service, false, true); err != nil {
				return fmt.Errorf("failed to build service '%s': %w", serviceName, err)
			}
		}

		if err := e.startService(service); err != nil {
			return fmt.Errorf("failed to start service '%s': %w", serviceName, err)
		}

		// Wait for health check if configured
		if service.HealthCheck != nil {
			_, _ = fmt.Fprintf(e.output, "    ⏳  Waiting for %s to become healthy...\n", serviceName)
			if err := e.waitForHealth(service); err != nil {
				_, _ = fmt.Fprintf(e.output, "    ⚠️  Health check failed for %s: %v\n", serviceName, err)
				// Continue anyway unless circuit breaker is enabled
			} else {
				_, _ = fmt.Fprintf(e.output, "    ✓  %s is healthy\n", serviceName)
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓  %s started\n", serviceName)
		}
	}

	_, _ = fmt.Fprintf(e.output, "✅  All services started successfully\n")
	return nil
}

// orchestrateStop stops services in reverse order
func (e *Engine) orchestrateStop(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🛑  Stopping orchestration: %s\n", orch.Name)

	// Reverse order for shutdown
	for i := len(orderedServices) - 1; i >= 0; i-- {
		serviceName := orderedServices[i]
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Stopping %s...\n", serviceName)

		if err := e.stopService(service); err != nil {
			_, _ = fmt.Fprintf(e.output, "    ⚠️  Failed to stop %s: %v\n", serviceName, err)
			// Continue stopping other services
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓  %s stopped\n", serviceName)
			if err := e.runServiceHook(ctx, service.PostTask, serviceName, "post"); err != nil {
				return err
			}
		}
	}

	_, _ = fmt.Fprintf(e.output, "✅  All services stopped\n")
	return nil
}

// orchestrateStatus shows status of all services
func (e *Engine) orchestrateStatus(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "📊  Status of orchestration: %s\n", orch.Name)

	// Check DNS resolution for specified domains
	if err := e.checkDNSResolution(orch); err != nil {
		// DNS check failures are warnings, not errors
		_, _ = fmt.Fprintf(e.output, "⚠️  %v\n\n", err)
	}

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		status := e.getServiceStatus(service)
		_, _ = fmt.Fprintf(e.output, "  %s: %s\n", serviceName, status)
	}

	return nil
}

// orchestrateShowEndpoints displays all service endpoints
func (e *Engine) orchestrateShowEndpoints(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🌐  Service endpoints for orchestration: %s\n", orch.Name)
	_, _ = fmt.Fprintf(e.output, "\n")

	var runningWithEndpoints []struct {
		name     string
		endpoint string
		status   string
	}
	var noEndpoint []string
	var stopped []string

	// Collect all service information
	for _, serviceName := range orderedServices {
		service := services[serviceName]
		status := e.getServiceStatus(service)

		if status != "running" {
			stopped = append(stopped, serviceName)
			continue
		}

		if service.HealthCheck == nil || service.HealthCheck.Endpoint == "" {
			noEndpoint = append(noEndpoint, serviceName)
			continue
		}

		runningWithEndpoints = append(runningWithEndpoints, struct {
			name     string
			endpoint string
			status   string
		}{
			name:     serviceName,
			endpoint: service.HealthCheck.Endpoint,
			status:   status,
		})
	}

	// Display running services with endpoints
	if len(runningWithEndpoints) > 0 {
		_, _ = fmt.Fprintf(e.output, "✅  Running services:\n")
		for _, svc := range runningWithEndpoints {
			_, _ = fmt.Fprintf(e.output, "   • %-20s %s\n", svc.name+":", svc.endpoint)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	// Display running services without endpoints
	if len(noEndpoint) > 0 {
		_, _ = fmt.Fprintf(e.output, "ℹ️  Running (no endpoint configured):\n")
		for _, name := range noEndpoint {
			_, _ = fmt.Fprintf(e.output, "   • %s\n", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	// Display stopped services
	if len(stopped) > 0 {
		_, _ = fmt.Fprintf(e.output, "⏹️  Stopped services:\n")
		for _, name := range stopped {
			_, _ = fmt.Fprintf(e.output, "   • %s\n", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	if len(runningWithEndpoints) == 0 {
		_, _ = fmt.Fprintf(e.output, "⚠️  No running services with endpoints found\n")
	}

	return nil
}

// orchestrateHealth checks health for all services
func (e *Engine) orchestrateHealth(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🏥  Health check for orchestration: %s\n", orch.Name)

	var unhealthy []string

	for _, serviceName := range orderedServices {
		service := services[serviceName]

		if service.HealthCheck == nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  no health check configured\n", serviceName)
			continue
		}

		_, _ = fmt.Fprintf(e.output, "  %s: checking health...\n", serviceName)
		if err := e.waitForHealth(service); err != nil {
			_, _ = fmt.Fprintf(e.output, "    ⚠️  %s is unhealthy: %v\n", serviceName, err)
			unhealthy = append(unhealthy, fmt.Sprintf("%s (%v)", serviceName, err))
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓  %s is healthy\n", serviceName)
		}
	}

	if len(unhealthy) > 0 {
		return fmt.Errorf("services unhealthy: %s", strings.Join(unhealthy, ", "))
	}

	_, _ = fmt.Fprintf(e.output, "✅  All services healthy\n")
	return nil
}

// orchestrateLogs displays logs for the selected services
func (e *Engine) orchestrateLogs(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "📝  Logs for orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]

		if e.dryRun {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would show logs for %s\n", serviceName)
			continue
		}

		_, _ = fmt.Fprintf(e.output, "  ▸ Showing logs for %s...\n", serviceName)
		if err := e.runDockerCompose(service, "logs"); err != nil {
			return fmt.Errorf("failed to retrieve logs for '%s': %w", serviceName, err)
		}
	}

	return nil
}

// orchestrateCloneRepositories reports repository cloning order
func (e *Engine) orchestrateCloneRepositories(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "📦  Repository cloning plan for orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		if service.Repository == nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ℹ️  no repository configured, skipping\n", serviceName)
			continue
		}

		_, _ = fmt.Fprintf(e.output, "  %s: %s", serviceName, service.Repository.URL)
		if service.Repository.Branch != "" {
			_, _ = fmt.Fprintf(e.output, " (branch %s)", service.Repository.Branch)
		}
		if service.Repository.Tag != "" {
			_, _ = fmt.Fprintf(e.output, " (tag %s)", service.Repository.Tag)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	if !e.dryRun {
		return fmt.Errorf("clone_repositories action currently supported only in dry-run mode")
	}

	return nil
}

// orchestrateUpdateRepositories updates repositories for services
func (e *Engine) orchestrateUpdateRepositories(ctx context.Context, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement, branchFilter string) error {
	_, _ = fmt.Fprintf(e.output, "🔄  Updating repositories for orchestration: %s\n", orch.Name)
	if branchFilter != "" {
		_, _ = fmt.Fprintf(e.output, "  Filter: only updating services on branch '%s'\n", branchFilter)
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create repository manager
	repoManager := repository.NewManager(workDir)

	updatedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		if service.Repository == nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ℹ️  no repository configured, skipping\n", serviceName)
			skippedCount++
			continue
		}

		// Convert AST repository config to domain model
		// Use orchestration's SSH key as fallback if service doesn't have one
		sshKey := service.Repository.SSHKey
		if sshKey == "" && orch.GitSSHKey != "" {
			sshKey = orch.GitSSHKey
		}

		repoConfig := &orchestration.Repository{
			URL:           service.Repository.URL,
			Branch:        service.Repository.Branch,
			Tag:           service.Repository.Tag,
			SSHKey:        sshKey,
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
			_, _ = fmt.Fprintf(e.output, "  [VERBOSE] Checking repository for %s at: %s\n", serviceName, fullPath)
		}

		gitPath := filepath.Join(fullPath, ".git")
		if _, err := os.Stat(gitPath); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  repository not cloned locally at %s, skipping update\n", serviceName, fullPath)
			skippedCount++
			continue
		}

		// If branch filter is specified, check current branch
		if branchFilter != "" {
			currentBranch, err := repoManager.GetCurrentBranch(ctx, service.Path)
			if err != nil {
				_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to get current branch: %v\n", serviceName, err)
				errorCount++
				continue
			}

			// Normalize branch names for comparison (handle master/main aliases)
			normalizedCurrent := normalizeBranchName(currentBranch)
			normalizedFilter := normalizeBranchName(branchFilter)

			if normalizedCurrent != normalizedFilter {
				_, _ = fmt.Fprintf(e.output, "  %s: ⏭️  on branch '%s' (not '%s'), skipping\n", serviceName, currentBranch, branchFilter)
				skippedCount++
				continue
			}
		}

		// Update the repository
		_, _ = fmt.Fprintf(e.output, "  %s: 🔄  updating...", serviceName)
		if err := repoManager.Update(ctx, repoConfig, service.Path); err != nil {
			_, _ = fmt.Fprintf(e.output, " ❌  failed: %v\n", err)
			errorCount++
			continue
		}

		currentBranch, _ := repoManager.GetCurrentBranch(ctx, service.Path)
		_, _ = fmt.Fprintf(e.output, " ✅  updated (branch: %s)\n", currentBranch)
		updatedCount++
	}

	_, _ = fmt.Fprintf(e.output, "\n📊  Summary: %d updated, %d skipped, %d errors\n", updatedCount, skippedCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("repository update completed with %d error(s)", errorCount)
	}

	return nil
}

// normalizeBranchName normalizes branch names for comparison (handles master/main aliases)
func normalizeBranchName(branch string) string {
	branch = strings.ToLower(strings.TrimSpace(branch))
	// Treat master and main as equivalent
	if branch == "master" || branch == "main" {
		return "main"
	}
	return branch
}

// orchestrateListBranches lists repositories and their current branches
// If branchFilter is provided, only shows repositories on that branch
func (e *Engine) orchestrateListBranches(ctx context.Context, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement, branchFilter string) error {
	if branchFilter != "" {
		_, _ = fmt.Fprintf(e.output, "🌿  Repositories on branch '%s' for orchestration: %s\n", branchFilter, orch.Name)
	} else {
		_, _ = fmt.Fprintf(e.output, "🌿  Branch status for orchestration: %s\n", orch.Name)
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create repository manager
	repoManager := repository.NewManager(workDir)

	var matchingRepos []struct {
		serviceName   string
		currentBranch string
	}
	var noRepo []string
	var errors []string
	var skipped []string

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		if service.Repository == nil {
			noRepo = append(noRepo, serviceName)
			continue
		}

		// Check if repository exists
		fullPath := service.Path
		if !filepath.IsAbs(fullPath) {
			fullPath = filepath.Join(workDir, service.Path)
		}

		gitPath := filepath.Join(fullPath, ".git")
		if _, err := os.Stat(gitPath); os.IsNotExist(err) {
			if branchFilter == "" {
				_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  repository not cloned locally\n", serviceName)
			}
			errors = append(errors, fmt.Sprintf("%s (not cloned)", serviceName))
			continue
		}

		// Get current branch
		currentBranch, err := repoManager.GetCurrentBranch(ctx, service.Path)
		if err != nil {
			if branchFilter == "" {
				_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to get current branch: %v\n", serviceName, err)
			}
			errors = append(errors, fmt.Sprintf("%s (%v)", serviceName, err))
			continue
		}

		// If branch filter is provided, only show repos on that branch
		if branchFilter != "" {
			normalizedCurrent := normalizeBranchName(currentBranch)
			normalizedFilter := normalizeBranchName(branchFilter)

			if normalizedCurrent == normalizedFilter {
				matchingRepos = append(matchingRepos, struct {
					serviceName   string
					currentBranch string
				}{
					serviceName:   serviceName,
					currentBranch: currentBranch,
				})
			} else {
				skipped = append(skipped, serviceName)
			}
		} else {
			// No filter: show all repos with their current branch
			matchingRepos = append(matchingRepos, struct {
				serviceName   string
				currentBranch string
			}{
				serviceName:   serviceName,
				currentBranch: currentBranch,
			})
		}
	}

	// Display results
	if len(matchingRepos) > 0 {
		if branchFilter != "" {
			_, _ = fmt.Fprintf(e.output, "\n✅  Repositories on branch '%s':\n", branchFilter)
		} else {
			_, _ = fmt.Fprintf(e.output, "\n📋 Repository branches:\n")
		}
		for _, item := range matchingRepos {
			_, _ = fmt.Fprintf(e.output, "  • %-20s  branch: %s\n", item.serviceName+":", item.currentBranch)
		}
	} else if branchFilter != "" {
		_, _ = fmt.Fprintf(e.output, "\n⚠️  No repositories found on branch '%s'\n", branchFilter)
	}

	if len(noRepo) > 0 && branchFilter == "" {
		_, _ = fmt.Fprintf(e.output, "\nℹ️  Services without repository:\n")
		for _, name := range noRepo {
			_, _ = fmt.Fprintf(e.output, "  • %s\n", name)
		}
	}

	if len(errors) > 0 && branchFilter == "" {
		_, _ = fmt.Fprintf(e.output, "\n❌  Errors:\n")
		for _, errMsg := range errors {
			_, _ = fmt.Fprintf(e.output, "  • %s\n", errMsg)
		}
	}

	if branchFilter != "" {
		_, _ = fmt.Fprintf(e.output, "\n📊  Summary: %d on branch '%s', %d skipped, %d without repo, %d errors\n",
			len(matchingRepos), branchFilter, len(skipped), len(noRepo), len(errors))
	} else {
		_, _ = fmt.Fprintf(e.output, "\n📊  Summary: %d repositories, %d without repo, %d errors\n",
			len(matchingRepos), len(noRepo), len(errors))
	}

	return nil
}

// orchestrateSwitchToDefault switches a specific service (or all if no service specified) to the default branch
func (e *Engine) orchestrateSwitchToDefault(ctx context.Context, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement, serviceFilter string) error {
	_, _ = fmt.Fprintf(e.output, "🔄  Switching to default branch for orchestration: %s\n", orch.Name)
	if serviceFilter != "" {
		_, _ = fmt.Fprintf(e.output, "  Filter: only switching service '%s'\n", serviceFilter)
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create repository manager
	repoManager := repository.NewManager(workDir)

	switchedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, serviceName := range orderedServices {
		// Apply service filter if provided
		if serviceFilter != "" && serviceName != serviceFilter {
			continue
		}

		service := services[serviceName]
		if service.Repository == nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ℹ️  no repository configured, skipping\n", serviceName)
			skippedCount++
			continue
		}

		// Convert AST repository config to domain model
		sshKey := service.Repository.SSHKey
		if sshKey == "" && orch.GitSSHKey != "" {
			sshKey = orch.GitSSHKey
		}

		repoConfig := &orchestration.Repository{
			URL:           service.Repository.URL,
			Branch:        service.Repository.Branch,
			Tag:           service.Repository.Tag,
			SSHKey:        sshKey,
			Clone:         service.Repository.Clone,
			UpdateOnStart: service.Repository.UpdateOnStart,
		}

		// Check if repository exists
		fullPath := service.Path
		if !filepath.IsAbs(fullPath) {
			fullPath = filepath.Join(workDir, service.Path)
		}

		gitPath := filepath.Join(fullPath, ".git")
		if _, err := os.Stat(gitPath); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  repository not cloned locally, skipping\n", serviceName)
			skippedCount++
			continue
		}

		// Get current branch
		currentBranch, err := repoManager.GetCurrentBranch(ctx, service.Path)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to get current branch: %v\n", serviceName, err)
			errorCount++
			continue
		}

		// Get default branch
		defaultBranch, err := repoManager.GetDefaultBranch(ctx, repoConfig, service.Path)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to get default branch: %v\n", serviceName, err)
			errorCount++
			continue
		}

		// Check if already on default branch
		normalizedCurrent := normalizeBranchName(currentBranch)
		normalizedDefault := normalizeBranchName(defaultBranch)

		if normalizedCurrent == normalizedDefault {
			_, _ = fmt.Fprintf(e.output, "  %s: ✓  already on default branch (%s), skipping\n", serviceName, currentBranch)
			skippedCount++
			continue
		}

		// Check if repository has uncommitted changes
		isClean, err := repoManager.IsClean(ctx, service.Path)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to check repository status: %v\n", serviceName, err)
			errorCount++
			continue
		}

		if !isClean {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  has uncommitted changes, skipping (use 'git stash' or commit changes first)\n", serviceName)
			skippedCount++
			continue
		}

		// Switch to default branch
		_, _ = fmt.Fprintf(e.output, "  %s: 🔄  switching from %s to %s...", serviceName, currentBranch, defaultBranch)
		if err := repoManager.Checkout(ctx, service.Path, defaultBranch); err != nil {
			_, _ = fmt.Fprintf(e.output, " ❌  failed: %v\n", err)
			errorCount++
			continue
		}

		// Pull latest changes
		if err := repoManager.Update(ctx, repoConfig, service.Path); err != nil {
			_, _ = fmt.Fprintf(e.output, " ⚠️  switched but failed to pull: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(e.output, " ✅  switched and updated\n")
		}
		switchedCount++
	}

	_, _ = fmt.Fprintf(e.output, "\n📊  Summary: %d switched, %d skipped, %d errors\n", switchedCount, skippedCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("branch switch completed with %d error(s)", errorCount)
	}

	if serviceFilter != "" && switchedCount == 0 {
		return fmt.Errorf("service '%s' not found or could not be switched", serviceFilter)
	}

	return nil
}

// orchestrateSetAllDefault sets all services to their default branch
func (e *Engine) orchestrateSetAllDefault(ctx context.Context, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🔄  Setting all repositories to default branch for orchestration: %s\n", orch.Name)

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create repository manager
	repoManager := repository.NewManager(workDir)

	switchedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		if service.Repository == nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ℹ️  no repository configured, skipping\n", serviceName)
			skippedCount++
			continue
		}

		// Convert AST repository config to domain model
		sshKey := service.Repository.SSHKey
		if sshKey == "" && orch.GitSSHKey != "" {
			sshKey = orch.GitSSHKey
		}

		repoConfig := &orchestration.Repository{
			URL:           service.Repository.URL,
			Branch:        service.Repository.Branch,
			Tag:           service.Repository.Tag,
			SSHKey:        sshKey,
			Clone:         service.Repository.Clone,
			UpdateOnStart: service.Repository.UpdateOnStart,
		}

		// Check if repository exists
		fullPath := service.Path
		if !filepath.IsAbs(fullPath) {
			fullPath = filepath.Join(workDir, service.Path)
		}

		gitPath := filepath.Join(fullPath, ".git")
		if _, err := os.Stat(gitPath); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  repository not cloned locally, skipping\n", serviceName)
			skippedCount++
			continue
		}

		// Get current branch
		currentBranch, err := repoManager.GetCurrentBranch(ctx, service.Path)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to get current branch: %v\n", serviceName, err)
			errorCount++
			continue
		}

		// Get default branch
		defaultBranch, err := repoManager.GetDefaultBranch(ctx, repoConfig, service.Path)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to get default branch: %v\n", serviceName, err)
			errorCount++
			continue
		}

		// Check if already on default branch
		normalizedCurrent := normalizeBranchName(currentBranch)
		normalizedDefault := normalizeBranchName(defaultBranch)

		if normalizedCurrent == normalizedDefault {
			_, _ = fmt.Fprintf(e.output, "  %s: ✓  already on default branch (%s), skipping\n", serviceName, currentBranch)
			skippedCount++
			continue
		}

		// Check if repository has uncommitted changes
		isClean, err := repoManager.IsClean(ctx, service.Path)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  failed to check repository status: %v\n", serviceName, err)
			errorCount++
			continue
		}

		if !isClean {
			_, _ = fmt.Fprintf(e.output, "  %s: ⚠️  has uncommitted changes, skipping (use 'git stash' or commit changes first)\n", serviceName)
			skippedCount++
			continue
		}

		// Switch to default branch
		_, _ = fmt.Fprintf(e.output, "  %s: 🔄  switching from %s to %s...", serviceName, currentBranch, defaultBranch)
		if err := repoManager.Checkout(ctx, service.Path, defaultBranch); err != nil {
			_, _ = fmt.Fprintf(e.output, " ❌  failed: %v\n", err)
			errorCount++
			continue
		}

		// Pull latest changes
		if err := repoManager.Update(ctx, repoConfig, service.Path); err != nil {
			_, _ = fmt.Fprintf(e.output, " ⚠️  switched but failed to pull: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(e.output, " ✅  switched and updated\n")
		}
		switchedCount++
	}

	_, _ = fmt.Fprintf(e.output, "\n📊  Summary: %d switched, %d skipped, %d errors\n", switchedCount, skippedCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("branch switch completed with %d error(s)", errorCount)
	}

	return nil
}

// orchestrateBuild builds all services
func (e *Engine) orchestrateBuild(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement, useCache bool) error {
	_, _ = fmt.Fprintf(e.output, "🔨  Building orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Building %s...\n", serviceName)

		if err := e.performServiceBuild(ctx, service, true, useCache); err != nil {
			return fmt.Errorf("failed to build service '%s': %w", serviceName, err)
		}

		_, _ = fmt.Fprintf(e.output, "    ✓  %s built\n", serviceName)
	}

	return nil
}

// orchestratePull pulls images for all services
func (e *Engine) orchestratePull(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "📥  Pulling images for orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Pulling %s...\n", serviceName)

		if err := e.pullService(service); err != nil {
			return fmt.Errorf("failed to pull service '%s': %w", serviceName, err)
		}

		_, _ = fmt.Fprintf(e.output, "    ✓  %s pulled\n", serviceName)
	}

	return nil
}

// orchestrateRecreate forces recreation of services by taking them down, rebuilding, and starting again
func (e *Engine) orchestrateRecreate(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement, useCache bool) error {
	_, _ = fmt.Fprintf(e.output, "🔁  Force recreating orchestration: %s\n", orch.Name)

	errDown := e.orchestrateDown(ctx, orch, orderedServices, services)
	errPost := e.runOrchestrationHook(ctx, orch.PostTask, orch.Name, "post")
	if err := errors.Join(errDown, errPost); err != nil {
		return err
	}

	if err := e.runOrchestrationHook(ctx, orch.PreTask, orch.Name, "pre"); err != nil {
		return err
	}

	if err := e.orchestrateBuild(ctx, orch, orderedServices, services, useCache); err != nil {
		return err
	}

	return e.orchestrateStartWithProgress(ctx, orch, orderedServices, services, false, false)
}

// orchestrateDown stops and removes containers
func (e *Engine) orchestrateDown(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🗑️  Taking down orchestration: %s\n", orch.Name)

	// Check DNS resolution for specified domains (helpful before any orchestration action)
	if err := e.checkDNSResolution(orch); err != nil {
		// DNS check failures are warnings, not errors
		_, _ = fmt.Fprintf(e.output, "⚠️  %v\n\n", err)
	}

	for i := len(orderedServices) - 1; i >= 0; i-- {
		serviceName := orderedServices[i]
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Taking down %s...\n", serviceName)

		if err := e.downService(service); err != nil {
			_, _ = fmt.Fprintf(e.output, "    ⚠️  Failed to take down %s: %v\n", serviceName, err)
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓  %s taken down\n", serviceName)
			if err := e.runServiceHook(ctx, service.PostTask, serviceName, "post"); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper functions for Docker Compose operations

func (e *Engine) startService(service *ast.ServiceStatement) error {
	alreadyHealthy, err := e.serviceIsRunningAndHealthy(service)
	if err == nil && alreadyHealthy {
		return nil
	}

	return e.runDockerCompose(service, "up", "-d")
}

func (e *Engine) stopService(service *ast.ServiceStatement) error {
	return e.runDockerCompose(service, "stop")
}

func (e *Engine) buildServiceWithOutput(service *ast.ServiceStatement, useCache bool) error {
	args := []string{"build"}
	if !useCache {
		args = append(args, "--no-cache")
	}

	cmd := e.buildDockerComposeCmd(service, args...)

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Running: %s\n", strings.Join(cmd.Args, " "))
	}

	// Stream the output in real-time
	cmd.Stdout = e.output
	cmd.Stderr = e.output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose build failed: %w", err)
	}

	return nil
}

func (e *Engine) pullService(service *ast.ServiceStatement) error {
	return e.runDockerCompose(service, "pull")
}

func (e *Engine) downService(service *ast.ServiceStatement) error {
	return e.runDockerCompose(service, "down")
}

func (e *Engine) getServiceStatus(service *ast.ServiceStatement) string {
	// Run docker compose ps to check status
	cmd := e.buildDockerComposeCmd(service, "ps", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	if len(output) == 0 || strings.TrimSpace(string(output)) == "[]" {
		return "stopped"
	}

	if strings.Contains(string(output), `"State":"running"`) {
		return "running"
	}

	return "stopped"
}

func (e *Engine) serviceIsRunningAndHealthy(service *ast.ServiceStatement) (bool, error) {
	status := e.getServiceStatus(service)
	if status != "running" {
		return false, nil
	}

	if service.HealthCheck == nil {
		return true, nil
	}

	healthy, err := e.performHealthCheck(service)
	if err != nil {
		return false, err
	}

	return healthy, nil
}

func (e *Engine) runDockerCompose(service *ast.ServiceStatement, args ...string) error {
	cmd := e.buildDockerComposeCmd(service, args...)

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Running: %s\n", strings.Join(cmd.Args, " "))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (e *Engine) buildDockerComposeCmd(service *ast.ServiceStatement, args ...string) *exec.Cmd {
	// Build docker compose command
	composeFile := "docker-compose.yml"
	if service.Compose != nil && service.Compose.File != "" {
		composeFile = service.Compose.File
	}

	// Service path should already be absolute at this point (resolved during orchestration setup)
	// We must NOT call filepath.Abs here as it would resolve relative paths from CWD instead of spec file location
	servicePath := service.Path
	if !filepath.IsAbs(servicePath) {
		// This should never happen - paths should be resolved earlier
		// If we hit this, use Abs as a fallback but log a warning
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "    [WARNING] Service path is relative, resolving from CWD: %s\n", servicePath)
		}
		absPath, err := filepath.Abs(servicePath)
		if err != nil {
			// Fall back to using the relative path and let docker fail with a better error
			if e.verbose {
				_, _ = fmt.Fprintf(e.output, "    [WARNING] Failed to resolve absolute path: %v\n", err)
			}
		} else {
			servicePath = absPath
		}
	}

	cmdArgs := []string{"compose", "-f", composeFile}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = servicePath

	// Set up environment with correct PWD for the service directory
	env := os.Environ()
	// Update PWD to point to the service directory
	for i, envVar := range env {
		if strings.HasPrefix(envVar, "PWD=") {
			env[i] = "PWD=" + servicePath
			break
		}
	}
	// If PWD wasn't found, add it
	if !strings.Contains(strings.Join(env, " "), "PWD=") {
		env = append(env, "PWD="+servicePath)
	}
	cmd.Env = env

	return cmd
}

func (e *Engine) runOrchestrationHook(ctx *ExecutionContext, taskName, orchestrationName, phase string) error {
	if taskName == "" {
		return nil
	}

	if ctx == nil {
		return fmt.Errorf("orchestration '%s' %s-task '%s' failed: execution context unavailable", orchestrationName, phase, taskName)
	}

	if err := e.runNamedTask(ctx, taskName); err != nil {
		return fmt.Errorf("orchestration '%s' %s-task '%s' failed: %w", orchestrationName, phase, taskName, err)
	}

	return nil
}

func (e *Engine) runServiceHook(ctx *ExecutionContext, taskName, serviceName, phase string) error {
	if taskName == "" {
		return nil
	}

	if ctx == nil {
		return fmt.Errorf("service '%s' %s-task '%s' failed: execution context unavailable", serviceName, phase, taskName)
	}

	if err := e.runNamedTask(ctx, taskName); err != nil {
		return fmt.Errorf("service '%s' %s-task '%s' failed: %w", serviceName, phase, taskName, err)
	}

	return nil
}

func (e *Engine) runNamedTask(ctx *ExecutionContext, taskName string) error {
	task, namespace, err := e.resolveTaskReference(ctx, taskName)
	if err != nil {
		return err
	}

	hookCtx := &ExecutionContext{
		Parameters:       ctx.Parameters,
		Variables:        make(map[string]string, len(ctx.Variables)),
		Project:          ctx.Project,
		CurrentFile:      ctx.CurrentFile,
		CurrentTask:      taskName,
		CurrentNamespace: namespace,
		Program:          ctx.Program,
	}

	for k, v := range ctx.Variables {
		hookCtx.Variables[k] = v
	}

	if err := e.setupTaskParameters(task, map[string]string{}, hookCtx); err != nil {
		return err
	}

	if err := e.executeTask(task, hookCtx); err != nil {
		return err
	}

	for k, v := range hookCtx.Variables {
		ctx.Variables[k] = v
	}

	return nil
}

func (e *Engine) resolveTaskReference(ctx *ExecutionContext, taskName string) (*ast.TaskStatement, string, error) {
	if ctx == nil || ctx.Program == nil {
		return nil, "", fmt.Errorf("task '%s' not found: no program context", taskName)
	}

	var targetTask *ast.TaskStatement
	var namespace string
	var err error

	if strings.Contains(taskName, ".") && ctx.Project != nil {
		namespace = strings.SplitN(taskName, ".", 2)[0]

		if template, exists := ctx.Project.IncludedTemplates[taskName]; exists {
			targetTask = &ast.TaskStatement{
				Token:       template.Token,
				Name:        template.Name,
				Description: template.Description,
				Annotations: template.Annotations,
				Parameters:  template.Parameters,
				Body:        template.Body,
			}
		} else if tasks, exists := ctx.Project.IncludedTasks[taskName]; exists {
			targetTask, err = selectTaskVariant(taskName, tasks)
			if err != nil {
				return nil, "", err
			}
		}
	}

	if targetTask == nil {
		targetTask, err = resolveTaskVariantByName(taskName, ctx.Program.Tasks)
		if err == nil && targetTask != nil {
			return targetTask, namespace, nil
		}
	}

	if targetTask == nil {
		for _, template := range ctx.Program.Templates {
			if template.Name == taskName {
				targetTask = &ast.TaskStatement{
					Token:       template.Token,
					Name:        template.Name,
					Description: template.Description,
					Annotations: template.Annotations,
					Parameters:  template.Parameters,
					Body:        template.Body,
				}
				break
			}
		}
	}

	if targetTask == nil {
		return nil, "", fmt.Errorf("task '%s' not found", taskName)
	}

	return targetTask, namespace, nil
}

func (e *Engine) performServiceBuild(ctx *ExecutionContext, service *ast.ServiceStatement, allowFallback bool, useCache bool) error {
	buildCfg := service.Build
	if buildCfg == nil {
		if allowFallback {
			return e.buildServiceWithOutput(service, useCache)
		}
		return nil
	}

	if buildCfg.Makefile != "" {
		return e.executeMakefileBuild(ctx, service)
	}

	if buildCfg.Command != "" {
		return e.executeBuildCommand(ctx, service)
	}

	if allowFallback || buildCfg.Required {
		return e.buildServiceWithOutput(service, useCache)
	}

	return nil
}

func (e *Engine) executeBuildCommand(ctx *ExecutionContext, service *ast.ServiceStatement) error {
	if service.Build == nil || service.Build.Command == "" {
		return nil
	}

	workDir, err := e.resolveBuildWorkingDir(service)
	if err != nil {
		return err
	}

	// Interpolate variables and secrets in the build command
	interpolatedCommand, err := e.interpolateVariablesWithError(service.Build.Command, ctx)
	if err != nil {
		return fmt.Errorf("failed to interpolate build command: %w", err)
	}

	return e.runShellCommandInDir(interpolatedCommand, workDir, true, service.Build.AllocateTTY)
}

func (e *Engine) executeMakefileBuild(ctx *ExecutionContext, service *ast.ServiceStatement) error {
	if service.Build == nil || service.Build.Makefile == "" {
		return nil
	}

	workDir, err := e.resolveBuildWorkingDir(service)
	if err != nil {
		return err
	}

	makefilePath := filepath.Join(workDir, service.Build.Makefile)
	if _, err := os.Stat(makefilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("makefile not found at %s", makefilePath)
		}
		return fmt.Errorf("checking Makefile at %s: %w", makefilePath, err)
	}

	// Interpolate and execute pre-make commands
	for _, cmdStr := range service.Build.PreMakeCommands {
		interpolatedCmd, err := e.interpolateVariablesWithError(cmdStr, ctx)
		if err != nil {
			return fmt.Errorf("failed to interpolate pre-make command: %w", err)
		}
		if err := e.runShellCommandInDir(interpolatedCmd, workDir, service.Build.Verbose, false); err != nil {
			return fmt.Errorf("pre-make command failed: %w", err)
		}
	}

	attempts := 1
	if service.Build.RetryOnFailure && service.Build.MaxRetries > 0 {
		attempts += service.Build.MaxRetries
	}

	var delay time.Duration
	if service.Build.RetryDelay != "" {
		if parsed, err := time.ParseDuration(service.Build.RetryDelay); err == nil {
			delay = parsed
		}
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 && delay > 0 {
			time.Sleep(delay)
		}

		if err := e.runMakeCommand(service.Build, workDir); err != nil {
			lastErr = err
			continue
		}

		lastErr = nil
		break
	}

	if lastErr != nil {
		if service.Build.FallbackCommand != "" {
			// Interpolate the fallback command
			interpolatedFallback, err := e.interpolateVariablesWithError(service.Build.FallbackCommand, ctx)
			if err != nil {
				return fmt.Errorf("failed to interpolate fallback command: %w", err)
			}
			if err := e.runShellCommandInDir(interpolatedFallback, workDir, true, service.Build.AllocateTTY); err != nil {
				return fmt.Errorf("make command failed and fallback command also failed: %w", err)
			}
		} else if service.Build.RetryOnFailure && service.Build.MaxRetries > 0 {
			return fmt.Errorf("make command failed after %d attempts: %w", attempts, lastErr)
		} else {
			return lastErr
		}
	}

	// Interpolate and execute post-make commands
	for _, cmdStr := range service.Build.PostMakeCommands {
		interpolatedCmd, err := e.interpolateVariablesWithError(cmdStr, ctx)
		if err != nil {
			return fmt.Errorf("failed to interpolate post-make command: %w", err)
		}
		if err := e.runShellCommandInDir(interpolatedCmd, workDir, service.Build.Verbose, false); err != nil {
			return fmt.Errorf("post-make command failed: %w", err)
		}
	}

	return nil
}

func (e *Engine) runMakeCommand(buildCfg *ast.BuildConfig, workDir string) error {
	args := []string{"-f", buildCfg.Makefile}

	if buildCfg.ParallelJobs > 0 {
		args = append(args, fmt.Sprintf("-j%d", buildCfg.ParallelJobs))
	}

	if buildCfg.MakeTarget != "" {
		args = append(args, buildCfg.MakeTarget)
	}

	args = append(args, buildCfg.MakeArgs...)

	runCtx := context.Background()
	var cancel context.CancelFunc

	if buildCfg.MakefileTimeout != "" {
		if timeout, err := time.ParseDuration(buildCfg.MakefileTimeout); err == nil && timeout > 0 {
			runCtx, cancel = context.WithTimeout(context.Background(), timeout)
			defer cancel()
		}
	}

	// #nosec G204 -- service builds intentionally invoke make with configured targets and flags.
	cmd := exec.CommandContext(runCtx, "make", args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	if buildCfg.Verbose {
		cmd.Stdout = e.output
		cmd.Stderr = e.output
		if err := cmd.Run(); err != nil {
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("make command timed out after %s", buildCfg.MakefileTimeout)
			}
			return fmt.Errorf("make command failed: %w", err)
		}
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("make command timed out after %s", buildCfg.MakefileTimeout)
		}
		return fmt.Errorf("make command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (e *Engine) runShellCommandInDir(cmdStr, workDir string, verbose bool, allocateTTY bool) error {
	if cmdStr == "" {
		return fmt.Errorf("empty command")
	}

	// Run command through shell to support operators like &&, ||, |, etc.
	var cmd *exec.Cmd
	if allocateTTY {
		cmd = e.createTTYCommand(cmdStr)
	} else {
		// #nosec G204 -- orchestration action hooks intentionally execute user-authored commands.
		cmd = exec.CommandContext(context.Background(), "sh", "-c", cmdStr)
	}

	cmd.Dir = workDir
	cmd.Env = os.Environ()

	if verbose {
		cmd.Stdout = e.output
		cmd.Stderr = e.output
		if allocateTTY {
			// For TTY allocation, also connect stdin
			cmd.Stdin = os.Stdin
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command failed: %w", err)
		}
		return nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// createTTYCommand creates a command with a pseudo-TTY allocated
// This is OS-specific because Linux and macOS have different `script` command syntax
func (e *Engine) createTTYCommand(cmdStr string) *exec.Cmd {
	// Detect OS using runtime package
	switch runtime.GOOS {
	case "darwin":
		// macOS: script -q /dev/null sh -c "command"
		// #nosec G204 -- orchestration action TTY hooks intentionally execute user-authored commands.
		return exec.CommandContext(context.Background(), "script", "-q", "/dev/null", "sh", "-c", cmdStr)
	default:
		// Linux and others: script -q -e -c "command" /dev/null
		// #nosec G204 -- orchestration action TTY hooks intentionally execute user-authored commands.
		return exec.CommandContext(context.Background(), "script", "-q", "-e", "-c", cmdStr, "/dev/null")
	}
}

func (e *Engine) resolveBuildWorkingDir(service *ast.ServiceStatement) (string, error) {
	workDir := service.Path
	if service.Build != nil && service.Build.WorkingDirectory != "" {
		if filepath.IsAbs(service.Build.WorkingDirectory) {
			workDir = service.Build.WorkingDirectory
		} else {
			workDir = filepath.Join(service.Path, service.Build.WorkingDirectory)
		}
	}

	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolving build working directory '%s': %w", workDir, err)
	}

	return absDir, nil
}

// waitForHealth waits for a service to become healthy based on its health check configuration
func (e *Engine) waitForHealth(service *ast.ServiceStatement) error {
	if service.HealthCheck == nil {
		return nil
	}

	hc := service.HealthCheck
	timeout := 30 // default 30 seconds (reserved for future use)
	interval := 2 // default 2 seconds
	retries := 15 // default 15 retries

	// Parse timeout if specified (reserved for future timeout implementation)
	if hc.Timeout != "" {
		if d, err := parseDurationToSeconds(hc.Timeout); err == nil {
			timeout = d
		}
	}
	_ = timeout // Reserved for future use in overall operation timeout

	// Parse interval if specified
	if hc.Interval != "" {
		if d, err := parseDurationToSeconds(hc.Interval); err == nil {
			interval = d
		}
	}

	// Use retries if specified
	if hc.Retries > 0 {
		retries = hc.Retries
	}

	// Perform health checks
	for attempt := 1; attempt <= retries; attempt++ {
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "      [VERBOSE] Health check attempt %d/%d...\n", attempt, retries)
		}

		healthy, err := e.performHealthCheck(service)
		if err == nil && healthy {
			return nil
		}

		if attempt < retries {
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}

	return fmt.Errorf("health check failed after %d attempts", retries)
}

// performHealthCheck performs a single health check based on the service configuration
func (e *Engine) performHealthCheck(service *ast.ServiceStatement) (bool, error) {
	hc := service.HealthCheck
	if hc == nil {
		return true, nil
	}

	switch strings.ToLower(hc.Type) {
	case "http", "https":
		return e.checkHTTPHealth(hc)
	case "tcp":
		return e.checkTCPHealth(hc)
	case "docker":
		return e.checkDockerHealth(service)
	default:
		return true, nil // Unknown type, assume healthy
	}
}

// checkHTTPHealth performs an HTTP health check
func (e *Engine) checkHTTPHealth(hc *ast.HealthCheckConfig) (bool, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(hc.Endpoint)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Check status code
	expectedStatus := 200
	if hc.Condition != "" {
		if status, err := strconv.Atoi(hc.Condition); err == nil {
			expectedStatus = status
		}
	}

	return resp.StatusCode == expectedStatus, nil
}

// checkTCPHealth performs a TCP port health check
func (e *Engine) checkTCPHealth(hc *ast.HealthCheckConfig) (bool, error) {
	conn, err := net.DialTimeout("tcp", hc.Endpoint, 5*time.Second)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = conn.Close()
	}()
	return true, nil
}

// checkDockerHealth checks if the Docker container reports healthy
func (e *Engine) checkDockerHealth(service *ast.ServiceStatement) (bool, error) {
	cmd := e.buildDockerComposeCmd(service, "ps", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	// Parse JSON output and check health status
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" || outputStr == "[]" {
		return false, fmt.Errorf("container not found")
	}

	// Check for "healthy" status in output
	return strings.Contains(outputStr, `"Health":"healthy"`) ||
		strings.Contains(outputStr, `"State":"running"`), nil
}

// parseDurationToSeconds converts a duration string like "30s", "1m" to seconds
func parseDurationToSeconds(durationStr string) (int, error) {
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0, err
	}
	return int(d.Seconds()), nil
}

// checkAndProvisionNetworks checks and provisions Docker networks for services
func (e *Engine) checkAndProvisionNetworks(services map[string]*ast.ServiceStatement) error {
	networkManager := docker.NewNetworkManager()
	ctx := context.Background()

	// Collect all required networks
	requiredNetworks := make(map[string]*ast.DockerNetworkConfig)
	for _, service := range services {
		if service.Networks != nil {
			for name, networkConfig := range service.Networks {
				requiredNetworks[name] = networkConfig
			}
		}
	}

	// Check and provision each network
	for networkName, networkConfig := range requiredNetworks {
		exists, err := networkManager.CheckNetworkExists(ctx, networkName)
		if err != nil {
			return fmt.Errorf("failed to check network %s: %w", networkName, err)
		}

		if !exists {
			if networkConfig.Required {
				if networkConfig.AutoProvision {
					// Create the network
					_, _ = fmt.Fprintf(e.output, "Creating Docker network: %s\n", networkName)
					err = networkManager.CreateNetwork(ctx, networkName, networkConfig.Driver, networkConfig.Options)
					if err != nil {
						return fmt.Errorf("failed to create network %s: %w", networkName, err)
					}
					_, _ = fmt.Fprintf(e.output, "✓  Created network: %s\n", networkName)
				} else {
					return fmt.Errorf("required network %s does not exist and autoprovision is disabled", networkName)
				}
			} else {
				_, _ = fmt.Fprintf(e.output, "⚠️  Network %s does not exist (not required)\n", networkName)
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "✓  Network %s exists\n", networkName)
		}
	}

	return nil
}

// checkDNSResolution checks if specified domains resolve correctly
func (e *Engine) checkDNSResolution(orch *ast.OrchestrateStatement) error {
	if len(orch.DNSChecks) == 0 {
		return nil
	}

	var failedDomains []string

	// Use a custom resolver with timeout
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 500 * time.Millisecond, // 500ms timeout per DNS query
			}
			return d.DialContext(ctx, network, address)
		},
	}

	for _, domain := range orch.DNSChecks {
		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		// Try to resolve the domain with timeout
		_, err := resolver.LookupHost(ctx, domain)
		cancel()

		if err != nil {
			failedDomains = append(failedDomains, domain)
		}
	}

	// Only show output if there are failures
	if len(failedDomains) > 0 {
		_, _ = fmt.Fprintf(e.output, "🔍  DNS resolution check:\n")
		for _, domain := range failedDomains {
			_, _ = fmt.Fprintf(e.output, "   ❌  %s - not resolvable\n", domain)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
		return fmt.Errorf("DNS resolution failed for: %s\nThese domains may need to be added to your /etc/hosts file", strings.Join(failedDomains, ", "))
	}

	// All domains resolved - no output needed
	return nil
}
