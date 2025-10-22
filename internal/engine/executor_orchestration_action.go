package engine

import (
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

	// Execute the action
	switch orchestrStmt.Action {
	case "start":
		return e.orchestrateStartWithProgress(orchestration, orderedServices, services)
	case "stop":
		return e.orchestrateStopWithProgress(orchestration, orderedServices, services)
	case "restart":
		if err := e.orchestrateStop(orchestration, orderedServices, services); err != nil {
			return err
		}
		return e.orchestrateStart(orchestration, orderedServices, services)
	case "status":
		return e.orchestrateStatus(orchestration, orderedServices, services)
	case "build":
		return e.orchestrateBuild(orchestration, orderedServices, services)
	case "pull":
		return e.orchestratePull(orchestration, orderedServices, services)
	case "down":
		return e.orchestrateDown(orchestration, orderedServices, services)
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
func (e *Engine) orchestrateStart(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🚀 Starting orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Starting %s...\n", serviceName)

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
func (e *Engine) orchestrateStop(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
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

// orchestrateBuild builds all services
func (e *Engine) orchestrateBuild(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🔨 Building orchestration: %s\n", orch.Name)

	for _, serviceName := range orderedServices {
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Building %s...\n", serviceName)

		if err := e.buildService(service); err != nil {
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
func (e *Engine) orchestrateDown(orch *ast.OrchestrateStatement, orderedServices []string, services map[string]*ast.ServiceStatement) error {
	_, _ = fmt.Fprintf(e.output, "🗑️  Taking down orchestration: %s\n", orch.Name)

	for i := len(orderedServices) - 1; i >= 0; i-- {
		serviceName := orderedServices[i]
		service := services[serviceName]
		_, _ = fmt.Fprintf(e.output, "  ▸ Taking down %s...\n", serviceName)

		if err := e.downService(service); err != nil {
			_, _ = fmt.Fprintf(e.output, "    ⚠ Failed to take down %s: %v\n", serviceName, err)
		} else {
			_, _ = fmt.Fprintf(e.output, "    ✓ %s taken down\n", serviceName)
		}
	}

	return nil
}

// Helper functions for Docker Compose operations

func (e *Engine) startService(service *ast.ServiceStatement) error {
	return e.runDockerCompose(service, "up", "-d")
}

func (e *Engine) stopService(service *ast.ServiceStatement) error {
	return e.runDockerCompose(service, "stop")
}

func (e *Engine) buildService(service *ast.ServiceStatement) error {
	return e.runDockerCompose(service, "build")
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
	cmd.Env = os.Environ()

	return cmd
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
	defer resp.Body.Close()

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
	conn.Close()
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
