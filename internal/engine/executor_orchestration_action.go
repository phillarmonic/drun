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
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/docker"
	"github.com/phillarmonic/drun/internal/domain/statement"
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
		for _, svc := range ctx.Program.Services {
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

	// Execute the action
	switch orchestrStmt.Action {
	case "start":
		if err := e.runOrchestrationHook(ctx, orchestration.PreTask, orchestration.Name, "pre"); err != nil {
			return err
		}
		return e.orchestrateStartWithProgress(ctx, orchestration, orderedServices, services)
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
	case "status":
		return e.orchestrateStatus(orchestration, orderedServices, services)
	case "health", "health_check":
		return e.orchestrateHealth(orchestration, orderedServices, services)
	case "logs":
		return e.orchestrateLogs(ctx, orchestration, orderedServices, services)
	case "build":
		return e.orchestrateBuild(ctx, orchestration, orderedServices, services)
	case "pull":
		return e.orchestratePull(orchestration, orderedServices, services)
	case "down":
		errDown := e.orchestrateDown(ctx, orchestration, orderedServices, services)
		errHook := e.runOrchestrationHook(ctx, orchestration.PostTask, orchestration.Name, "post")
		return errors.Join(errDown, errHook)
	case "clone_repositories":
		return e.orchestrateCloneRepositories(orchestration, orderedServices, services)
	default:
		return fmt.Errorf("unknown orchestration action: %s", orchestrStmt.Action)
	}
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

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Starting %s...\n", serviceName)

		alreadyHealthy, stateErr := e.serviceIsRunningAndHealthy(service)
		if stateErr != nil && e.verbose {
			_, _ = fmt.Fprintf(e.output, "    [VERBOSE] Unable to confirm current state for %s: %v\n", serviceName, stateErr)
		}
		if alreadyHealthy && stateErr == nil {
			_, _ = fmt.Fprintf(e.output, "    ✓ %s already running and healthy (skipping)\n", serviceName)
			continue
		}

		if err := e.runServiceHook(ctx, service.PreTask, serviceName, "pre"); err != nil {
			return err
		}

		if service.Build != nil && service.Build.Required {
			_, _ = fmt.Fprintf(e.output, "    🔨 Building %s...\n", serviceName)
			if err := e.performServiceBuild(ctx, service, false); err != nil {
				return fmt.Errorf("failed to build service '%s': %w", serviceName, err)
			}
		}

		if err := e.startService(service); err != nil {
			return fmt.Errorf("failed to start service '%s': %w", serviceName, err)
		}

		// Wait for health check if configured
		if service.HealthCheck != nil {
			_, _ = fmt.Fprintf(e.output, "    ⏳ Waiting for %s to become healthy...\n", serviceName)
			if err := e.waitForHealth(service); err != nil {
				_, _ = fmt.Fprintf(e.output, "    ⚠ Health check failed for %s: %v\n", serviceName, err)
				// Continue anyway unless circuit breaker is enabled
			} else {
				_, _ = fmt.Fprintf(e.output, "    ✓ %s is healthy\n", serviceName)
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓ %s started\n", serviceName)
		}
	}

	_, _ = fmt.Fprintf(e.output, "✅ All services started successfully\n")
	return nil
}

// orchestrateStop stops services in reverse order
func (e *Engine) orchestrateStop(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🛑 Stopping orchestration: %s\n", orch.Name)

	// Reverse order for shutdown
	for i := len(orderedServices) - 1; i >= 0; i-- {
		serviceName := orderedServices[i]
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Stopping %s...\n", serviceName)

		if err := e.stopService(service); err != nil {
			_, _ = fmt.Fprintf(e.output, "    ⚠ Failed to stop %s: %v\n", serviceName, err)
			// Continue stopping other services
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓ %s stopped\n", serviceName)
			if err := e.runServiceHook(ctx, service.PostTask, serviceName, "post"); err != nil {
				return err
			}
		}
	}

	_, _ = fmt.Fprintf(e.output, "✅ All services stopped\n")
	return nil
}

// orchestrateStatus shows status of all services
func (e *Engine) orchestrateStatus(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "📊 Status of orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		status := e.getServiceStatus(service)
		_, _ = fmt.Fprintf(e.output, "  %s: %s\n", serviceName, status)
	}

	return nil
}

// orchestrateHealth checks health for all services
func (e *Engine) orchestrateHealth(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🏥 Health check for orchestration: %s\n", orch.Name)

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
			_, _ = fmt.Fprintf(e.output, "    ✓ %s is healthy\n", serviceName)
		}
	}

	if len(unhealthy) > 0 {
		return fmt.Errorf("services unhealthy: %s", strings.Join(unhealthy, ", "))
	}

	_, _ = fmt.Fprintf(e.output, "✅ All services healthy\n")
	return nil
}

// orchestrateLogs displays logs for the selected services
func (e *Engine) orchestrateLogs(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "📝 Logs for orchestration: %s\n", orch.Name)

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
	_, _ = fmt.Fprintf(e.output, "📦 Repository cloning plan for orchestration: %s\n", orch.Name)

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

// orchestrateBuild builds all services
func (e *Engine) orchestrateBuild(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🔨 Building orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Building %s...\n", serviceName)

		if err := e.performServiceBuild(ctx, service, true); err != nil {
			return fmt.Errorf("failed to build service '%s': %w", serviceName, err)
		}

		_, _ = fmt.Fprintf(e.output, "    ✓ %s built\n", serviceName)
	}

	return nil
}

// orchestratePull pulls images for all services
func (e *Engine) orchestratePull(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "📥 Pulling images for orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Pulling %s...\n", serviceName)

		if err := e.pullService(service); err != nil {
			return fmt.Errorf("failed to pull service '%s': %w", serviceName, err)
		}

		_, _ = fmt.Fprintf(e.output, "    ✓ %s pulled\n", serviceName)
	}

	return nil
}

// orchestrateDown stops and removes containers
func (e *Engine) orchestrateDown(ctx *ExecutionContext, orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🗑️  Taking down orchestration: %s\n", orch.Name)

	for i := len(orderedServices) - 1; i >= 0; i-- {
		serviceName := orderedServices[i]
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Taking down %s...\n", serviceName)

		if err := e.downService(service); err != nil {
			_, _ = fmt.Fprintf(e.output, "    ⚠ Failed to take down %s: %v\n", serviceName, err)
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓ %s taken down\n", serviceName)
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

func (e *Engine) buildServiceWithOutput(service *ast.ServiceStatement) error {
	cmd := e.buildDockerComposeCmd(service, "build")

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

	// Get absolute path to service directory
	servicePath, _ := filepath.Abs(service.Path)

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

	if strings.Contains(taskName, ".") && ctx.Project != nil {
		namespace = strings.SplitN(taskName, ".", 2)[0]

		if template, exists := ctx.Project.IncludedTemplates[taskName]; exists {
			targetTask = &ast.TaskStatement{
				Token:       template.Token,
				Name:        template.Name,
				Description: template.Description,
				Parameters:  template.Parameters,
				Body:        template.Body,
			}
		} else if task, exists := ctx.Project.IncludedTasks[taskName]; exists {
			targetTask = task
		}
	}

	if targetTask == nil {
		for _, task := range ctx.Program.Tasks {
			if task.Name == taskName {
				targetTask = task
				break
			}
		}
	}

	if targetTask == nil {
		for _, template := range ctx.Program.Templates {
			if template.Name == taskName {
				targetTask = &ast.TaskStatement{
					Token:       template.Token,
					Name:        template.Name,
					Description: template.Description,
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

func (e *Engine) performServiceBuild(_ *ExecutionContext, service *ast.ServiceStatement, allowFallback bool) error {
	buildCfg := service.Build
	if buildCfg == nil {
		if allowFallback {
			return e.buildServiceWithOutput(service)
		}
		return nil
	}

	if buildCfg.Makefile != "" {
		return e.executeMakefileBuild(service)
	}

	if buildCfg.Command != "" {
		return e.executeBuildCommand(service)
	}

	if allowFallback || buildCfg.Required {
		return e.buildServiceWithOutput(service)
	}

	return nil
}

func (e *Engine) executeBuildCommand(service *ast.ServiceStatement) error {
	if service.Build == nil || service.Build.Command == "" {
		return nil
	}

	workDir, err := e.resolveBuildWorkingDir(service)
	if err != nil {
		return err
	}

	return e.runShellCommandInDir(service.Build.Command, workDir, true)
}

func (e *Engine) executeMakefileBuild(service *ast.ServiceStatement) error {
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

	for _, cmdStr := range service.Build.PreMakeCommands {
		if err := e.runShellCommandInDir(cmdStr, workDir, service.Build.Verbose); err != nil {
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
			if err := e.runShellCommandInDir(service.Build.FallbackCommand, workDir, true); err != nil {
				return fmt.Errorf("make command failed and fallback command also failed: %w", err)
			}
		} else if service.Build.RetryOnFailure && service.Build.MaxRetries > 0 {
			return fmt.Errorf("make command failed after %d attempts: %w", attempts, lastErr)
		} else {
			return lastErr
		}
	}

	for _, cmdStr := range service.Build.PostMakeCommands {
		if err := e.runShellCommandInDir(cmdStr, workDir, service.Build.Verbose); err != nil {
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

func (e *Engine) runShellCommandInDir(cmdStr, workDir string, verbose bool) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(context.Background(), parts[0], parts[1:]...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	if verbose {
		cmd.Stdout = e.output
		cmd.Stderr = e.output
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
					_, _ = fmt.Fprintf(e.output, "✓ Created network: %s\n", networkName)
				} else {
					return fmt.Errorf("required network %s does not exist and autoprovision is disabled", networkName)
				}
			} else {
				_, _ = fmt.Fprintf(e.output, "⚠️  Network %s does not exist (not required)\n", networkName)
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "✓ Network %s exists\n", networkName)
		}
	}

	return nil
}
