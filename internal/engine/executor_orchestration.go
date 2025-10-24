package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/orchestration"
	"github.com/phillarmonic/drun/internal/envfile"
	"github.com/phillarmonic/drun/internal/healthcheck"
	"github.com/phillarmonic/drun/internal/makeexec"
	"github.com/phillarmonic/drun/internal/repository"
)

// OrchestrationExecutor handles orchestration execution
type OrchestrationExecutor struct {
	workDir          string
	serviceRegistry  *orchestration.ServiceRegistry
	orchestrRegistry *orchestration.OrchestrationRegistry
	healthChecker    *healthcheck.Checker
	repoManager      *repository.Manager
	makeExecutor     *makeexec.Executor
	envFileManager   *envfile.Manager
	engine           *Engine
}

// NewOrchestrationExecutor creates a new orchestration executor
func NewOrchestrationExecutor(engine *Engine, workDir string) *OrchestrationExecutor {
	return &OrchestrationExecutor{
		workDir:          workDir,
		serviceRegistry:  orchestration.NewServiceRegistry(),
		orchestrRegistry: orchestration.NewOrchestrationRegistry(),
		healthChecker:    healthcheck.NewChecker(),
		repoManager:      repository.NewManager(workDir),
		makeExecutor:     makeexec.NewExecutor(workDir),
		envFileManager:   envfile.NewManager(workDir),
		engine:           engine,
	}
}

// RegisterService registers a service from AST
func (oe *OrchestrationExecutor) RegisterService(serviceStmt *ast.ServiceStatement) error {
	service := &orchestration.Service{
		Name:         serviceStmt.Name,
		Path:         serviceStmt.Path,
		Description:  serviceStmt.Description,
		Dependencies: serviceStmt.Dependencies,
		Environment:  serviceStmt.Environment,
		PreTask:      serviceStmt.PreTask,
		PostTask:     serviceStmt.PostTask,
		Status:       orchestration.ServiceStatusUnknown,
	}

	// Convert repository config
	if serviceStmt.Repository != nil {
		service.Repository = &orchestration.Repository{
			URL:            serviceStmt.Repository.URL,
			Branch:         serviceStmt.Repository.Branch,
			Tag:            serviceStmt.Repository.Tag,
			SSHKey:         serviceStmt.Repository.SSHKey,
			CloneIfMissing: serviceStmt.Repository.CloneIfMissing,
			UpdateOnStart:  serviceStmt.Repository.UpdateOnStart,
		}
	}

	// Convert health check config
	if serviceStmt.HealthCheck != nil {
		timeout, _ := time.ParseDuration(serviceStmt.HealthCheck.Timeout)
		interval, _ := time.ParseDuration(serviceStmt.HealthCheck.Interval)
		startPeriod, _ := time.ParseDuration(serviceStmt.HealthCheck.StartPeriod)

		service.HealthCheck = &orchestration.HealthCheck{
			Type:        serviceStmt.HealthCheck.Type,
			Endpoint:    serviceStmt.HealthCheck.Endpoint,
			Domain:      serviceStmt.HealthCheck.Domain,
			Container:   serviceStmt.HealthCheck.Container,
			Command:     serviceStmt.HealthCheck.Command,
			Timeout:     timeout,
			Interval:    interval,
			Retries:     serviceStmt.HealthCheck.Retries,
			Condition:   serviceStmt.HealthCheck.Condition,
			RecordType:  serviceStmt.HealthCheck.RecordType,
			ExpectedIP:  serviceStmt.HealthCheck.ExpectedIP,
			ExpectedIPs: serviceStmt.HealthCheck.ExpectedIPs,
			Headers:     serviceStmt.HealthCheck.Headers,
			WorkingDir:  serviceStmt.HealthCheck.WorkingDir,
			StartPeriod: startPeriod,
		}
	}

	// Convert build config
	if serviceStmt.Build != nil {
		makefileTimeout, _ := time.ParseDuration(serviceStmt.Build.MakefileTimeout)
		retryDelay, _ := time.ParseDuration(serviceStmt.Build.RetryDelay)

		service.Build = &orchestration.BuildConfig{
			Required:         serviceStmt.Build.Required,
			Command:          serviceStmt.Build.Command,
			Makefile:         serviceStmt.Build.Makefile,
			MakeTarget:       serviceStmt.Build.MakeTarget,
			MakeArgs:         serviceStmt.Build.MakeArgs,
			PreMakeCommands:  serviceStmt.Build.PreMakeCommands,
			PostMakeCommands: serviceStmt.Build.PostMakeCommands,
			WorkingDirectory: serviceStmt.Build.WorkingDirectory,
			MakefileTimeout:  makefileTimeout,
			ParallelJobs:     serviceStmt.Build.ParallelJobs,
			Verbose:          serviceStmt.Build.Verbose,
			RetryOnFailure:   serviceStmt.Build.RetryOnFailure,
			MaxRetries:       serviceStmt.Build.MaxRetries,
			RetryDelay:       retryDelay,
			FallbackCommand:  serviceStmt.Build.FallbackCommand,
		}
	}

	// Convert compose config
	if serviceStmt.Compose != nil {
		service.Compose = &orchestration.ComposeConfig{
			File:    serviceStmt.Compose.File,
			Project: serviceStmt.Compose.Project,
		}

		if serviceStmt.Compose.Options != nil {
			timeout, _ := time.ParseDuration(serviceStmt.Compose.Options.Timeout)
			waitTimeout, _ := time.ParseDuration(serviceStmt.Compose.Options.WaitTimeout)

			service.Compose.Options = &orchestration.ComposeOptions{
				ForceRecreate: serviceStmt.Compose.Options.ForceRecreate,
				NoDeps:        serviceStmt.Compose.Options.NoDeps,
				Build:         serviceStmt.Compose.Options.Build,
				Pull:          serviceStmt.Compose.Options.Pull,
				Timeout:       timeout,
				Scale:         serviceStmt.Compose.Options.Scale,
				Wait:          serviceStmt.Compose.Options.Wait,
				WaitTimeout:   waitTimeout,
				Detach:        serviceStmt.Compose.Options.Detach,
				RemoveOrphans: serviceStmt.Compose.Options.RemoveOrphans,
				RestartPolicy: serviceStmt.Compose.Options.RestartPolicy,
				MemoryLimit:   serviceStmt.Compose.Options.MemoryLimit,
				CPULimit:      serviceStmt.Compose.Options.CPULimit,
			}
		}
	}

	// Convert env file config
	if serviceStmt.EnvFile != nil {
		service.EnvFile = &orchestration.EnvFileConfig{
			Required: serviceStmt.EnvFile.Required,
			Task:     serviceStmt.EnvFile.Task,
		}
	}

	return oe.serviceRegistry.Register(service)
}

// RegisterOrchestration registers an orchestration from AST
func (oe *OrchestrationExecutor) RegisterOrchestration(orchestrStmt *ast.OrchestrateStatement) error {
	healthCheckInterval, _ := time.ParseDuration(orchestrStmt.HealthCheckInterval)
	startupTimeout, _ := time.ParseDuration(orchestrStmt.StartupTimeout)
	shutdownTimeout, _ := time.ParseDuration(orchestrStmt.ShutdownTimeout)
	recoveryTimeout, _ := time.ParseDuration(orchestrStmt.RecoveryTimeout)
	makefileTimeout, _ := time.ParseDuration(orchestrStmt.MakefileTimeout)
	cloneTimeout, _ := time.ParseDuration(orchestrStmt.CloneTimeout)

	orchestr := &orchestration.Orchestration{
		Name:                orchestrStmt.Name,
		Description:         orchestrStmt.Description,
		Services:            orchestrStmt.Services,
		Strategy:            orchestration.OrchestrationStrategy(orchestrStmt.Strategy),
		CircuitBreaker:      orchestrStmt.CircuitBreaker,
		StopOnFailure:       orchestrStmt.StopOnFailure,
		HealthCheckInterval: healthCheckInterval,
		StartupTimeout:      startupTimeout,
		ShutdownTimeout:     shutdownTimeout,
		PreTask:             orchestrStmt.PreTask,
		PostTask:            orchestrStmt.PostTask,
		FailureThreshold:    orchestrStmt.FailureThreshold,
		RecoveryTimeout:     recoveryTimeout,
		MakefileOrder:       orchestrStmt.MakefileOrder,
		MakefileTimeout:     makefileTimeout,
		CloneOrder:          orchestrStmt.CloneOrder,
		CloneTimeout:        cloneTimeout,
		Scale:               orchestrStmt.Scale,
		UpdateStrategy:      orchestration.UpdateStrategy(orchestrStmt.UpdateStrategy),
		MaxUnavailable:      orchestrStmt.MaxUnavailable,
		UpdateTimeout:       func() time.Duration { d, _ := time.ParseDuration(orchestrStmt.UpdateTimeout); return d }(),
		Status:              "unknown",
	}

	return oe.orchestrRegistry.Register(orchestr)
}

// StartService starts a single service
func (oe *OrchestrationExecutor) StartService(ctx context.Context, execCtx *ExecutionContext, serviceName string) error {
	service, err := oe.serviceRegistry.Get(serviceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	alreadyHealthy, stateErr := oe.isServiceRunningAndHealthy(ctx, service)
	if stateErr != nil && oe.engine != nil && oe.engine.verbose {
		_, _ = fmt.Fprintf(oe.engine.output, "    [VERBOSE] Unable to confirm current state for %s: %v\n", serviceName, stateErr)
	}
	if alreadyHealthy && stateErr == nil {
		service.MarkHealthy()
		return nil
	}

	// Mark service as starting
	service.MarkStarting()

	// Run pre-task if specified
	if service.PreTask != "" {
		if err := oe.executeTask(ctx, execCtx, service.PreTask); err != nil {
			return fmt.Errorf("pre-task failed: %w", err)
		}
	}

	// Clone repository if configured
	if service.Repository != nil {
		if err := oe.repoManager.EnsureRepository(ctx, service.Repository, service.Path); err != nil {
			service.MarkFailed()
			return fmt.Errorf("repository setup failed: %w", err)
		}
	}

	// Setup environment file if configured
	if service.EnvFile != nil {
		// Run env file task if specified
		if service.EnvFile.Task != "" {
			if err := oe.executeTask(ctx, execCtx, service.EnvFile.Task); err != nil {
				service.MarkFailed()
				return fmt.Errorf("env file task failed: %w", err)
			}
		}

		// Ensure env file exists
		if err := oe.envFileManager.Ensure(ctx, service.EnvFile, service.Path); err != nil {
			service.MarkFailed()
			return fmt.Errorf("env file setup failed: %w", err)
		}
	}

	// Build service if required
	if service.Build != nil && service.Build.Required {
		if service.Build.Makefile != "" {
			if err := oe.makeExecutor.Execute(ctx, service.Build, service.Path); err != nil {
				service.MarkFailed()
				return fmt.Errorf("build failed: %w", err)
			}
		} else if service.Build.Command != "" {
			if err := oe.executeCommand(ctx, service.Build.Command, service.Path); err != nil {
				service.MarkFailed()
				return fmt.Errorf("build command failed: %w", err)
			}
		}
	}

	// Start service using docker compose
	if err := oe.startDockerCompose(ctx, service); err != nil {
		service.MarkFailed()
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait for service to be healthy
	if service.HealthCheck != nil {
		if err := oe.waitForHealthy(ctx, service); err != nil {
			service.MarkFailed()
			return fmt.Errorf("health check failed: %w", err)
		}
	}

	service.MarkHealthy()
	return nil
}

// StopService stops a single service
func (oe *OrchestrationExecutor) StopService(ctx context.Context, execCtx *ExecutionContext, serviceName string) error {
	service, err := oe.serviceRegistry.Get(serviceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}

	// Mark service as stopping
	service.Status = "stopping"

	// Stop service using docker compose
	if err := oe.stopDockerCompose(ctx, service); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Run post-task if specified
	if service.PostTask != "" {
		if err := oe.executeTask(ctx, execCtx, service.PostTask); err != nil {
			return fmt.Errorf("post-task failed: %w", err)
		}
	}

	service.MarkStopped()
	return nil
}

// startDockerCompose starts a service using docker compose
func (oe *OrchestrationExecutor) startDockerCompose(ctx context.Context, service *orchestration.Service) error {
	args := []string{"compose"}

	// Add project name if specified
	if service.Compose != nil && service.Compose.Project != "" {
		args = append(args, "-p", service.Compose.Project)
	}

	// Add compose file if specified
	composeFile := "docker-compose.yml"
	if service.Compose != nil && service.Compose.File != "" {
		composeFile = service.Compose.File
	}
	args = append(args, "-f", composeFile)

	// Add up command
	args = append(args, "up", "-d")

	// Add options if specified
	if service.Compose != nil && service.Compose.Options != nil {
		opts := service.Compose.Options

		if opts.ForceRecreate {
			args = append(args, "--force-recreate")
		}
		if opts.NoDeps {
			args = append(args, "--no-deps")
		}
		if opts.Build {
			args = append(args, "--build")
		}
		if opts.Pull != "" {
			args = append(args, "--pull", opts.Pull)
		}
		if opts.Wait {
			args = append(args, "--wait")
		}
		if opts.RemoveOrphans {
			args = append(args, "--remove-orphans")
		}
	}

	// Execute docker compose command
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = filepath.Join(oe.workDir, service.Path)

	// Set environment variables
	if len(service.Environment) > 0 {
		env := os.Environ()
		for key, value := range service.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// stopDockerCompose stops a service using docker compose
func (oe *OrchestrationExecutor) stopDockerCompose(ctx context.Context, service *orchestration.Service) error {
	args := []string{"compose"}

	// Add project name if specified
	if service.Compose != nil && service.Compose.Project != "" {
		args = append(args, "-p", service.Compose.Project)
	}

	// Add compose file if specified
	composeFile := "docker-compose.yml"
	if service.Compose != nil && service.Compose.File != "" {
		composeFile = service.Compose.File
	}
	args = append(args, "-f", composeFile)

	// Add down command
	args = append(args, "down")

	// Execute docker compose command
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = filepath.Join(oe.workDir, service.Path)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// waitForHealthy waits for a service to become healthy
func (oe *OrchestrationExecutor) waitForHealthy(ctx context.Context, service *orchestration.Service) error {
	return oe.healthChecker.CheckWithRetries(ctx, service.HealthCheck)
}

// executeTask executes a task by name
func (oe *OrchestrationExecutor) executeTask(ctx context.Context, execCtx *ExecutionContext, taskName string) error {
	// Find the task in the program
	var targetTask *ast.TaskStatement
	if execCtx != nil && execCtx.Program != nil {
		for _, task := range execCtx.Program.Tasks {
			if task.Name == taskName {
				targetTask = task
				break
			}
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task '%s' not found", taskName)
	}

	// Execute task
	return oe.engine.executeTask(targetTask, execCtx)
}

// executeCommand executes a shell command
func (oe *OrchestrationExecutor) executeCommand(ctx context.Context, cmdStr, workDir string) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = filepath.Join(oe.workDir, workDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (oe *OrchestrationExecutor) isServiceRunningAndHealthy(ctx context.Context, service *orchestration.Service) (bool, error) {
	running, err := oe.isServiceRunning(ctx, service)
	if err != nil {
		return false, err
	}

	if !running {
		return false, nil
	}

	if service.HealthCheck == nil {
		return true, nil
	}

	if err := oe.healthChecker.Check(ctx, service.HealthCheck); err != nil {
		return false, err
	}

	return true, nil
}

func (oe *OrchestrationExecutor) isServiceRunning(ctx context.Context, service *orchestration.Service) (bool, error) {
	args := []string{"compose"}

	if service.Compose != nil && service.Compose.Project != "" {
		args = append(args, "-p", service.Compose.Project)
	}

	composeFile := "docker-compose.yml"
	if service.Compose != nil && service.Compose.File != "" {
		composeFile = service.Compose.File
	}
	args = append(args, "-f", composeFile, "ps", "--format", "json")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = filepath.Join(oe.workDir, service.Path)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("docker compose ps failed: %w\nOutput: %s", err, strings.TrimSpace(string(output)))
	}

	data := strings.TrimSpace(string(output))
	if data == "" || data == "[]" {
		return false, nil
	}

	if strings.Contains(data, `"State":"running"`) ||
		strings.Contains(data, `"Running":true`) ||
		strings.Contains(data, `"Status":"running"`) {
		return true, nil
	}

	return false, nil
}
