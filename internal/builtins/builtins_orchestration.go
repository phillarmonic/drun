package builtins

import (
	"fmt"
	"strings"
)

// Note: Orchestration built-in functions are defined here but require
// integration with the engine's orchestration executor to function properly.
// They are documented here for future implementation.

// OrchestrationContext provides context for orchestration operations
type OrchestrationContext interface {
	// Service operations
	GetServiceStatus(name string) (string, error)
	GetServiceHealth(name string) (string, error)
	IsServiceHealthy(name string) (bool, error)
	IsServiceRunning(name string) (bool, error)

	// Orchestration operations
	GetOrchestrationStatus(name string) (string, error)
	GetOrchestrationHealthStatus(name string) (map[string]interface{}, error)
	IsOrchestrationHealthy(name string) (bool, error)
	IsOrchestrationRunning(name string) (bool, error)

	// DNS operations
	DNSResolve(domain string) (string, error)
	DNSCheck(domain, recordType string) (bool, error)
	DNSValidate(domain, expectedIP string) (bool, error)

	// Repository operations
	GitClone(url, path string) error
	GitStatus(path string) (string, error)
	GitBranch(path string) (string, error)
	GitTag(path string) (string, error)

	// File operations
	FileExists(path string) (bool, error)

	// Docker operations
	DockerPS(serviceName string) (string, error)
	DockerLogs(containerID string) (string, error)
	DockerStats(containerID string) (string, error)

	// Compose operations
	ComposeConfig(path string) (string, error)

	// Makefile operations
	Make(target, path string) error
	MakeList(path string) ([]string, error)
	MakeDryRun(target, path string) (string, error)

	// Environment file operations
	ReadFile(path string) (string, error)
	GeneratePassword(length int) (string, error)
	EnvVar(name string) (string, error)
	ReplaceInFile(path, old, new string) error
}

// OrchestrationBuiltins provides built-in functions for orchestration
// Note: These use a simpler interface (args []string, ctx interface{})
// for flexibility. They should be adapted to BuiltinFunction when integrated.
var OrchestrationBuiltins = map[string]func(args []string, ctx interface{}) (string, error){
	// Service status functions
	"service_status": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("service_status requires 1 argument (service_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("service_status requires OrchestrationContext")
		}

		return orchCtx.GetServiceStatus(args[0])
	},

	"service_health": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("service_health requires 1 argument (service_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("service_health requires OrchestrationContext")
		}

		return orchCtx.GetServiceHealth(args[0])
	},

	"service_healthy": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("service_healthy requires 1 argument (service_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("service_healthy requires OrchestrationContext")
		}

		healthy, err := orchCtx.IsServiceHealthy(args[0])
		if err != nil {
			return "", err
		}

		if healthy {
			return "true", nil
		}
		return "false", nil
	},

	"service_running": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("service_running requires 1 argument (service_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("service_running requires OrchestrationContext")
		}

		running, err := orchCtx.IsServiceRunning(args[0])
		if err != nil {
			return "", err
		}

		if running {
			return "true", nil
		}
		return "false", nil
	},

	// Orchestration status functions
	"orchestrate_status": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("orchestrate_status requires 1 argument (orchestration_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("orchestrate_status requires OrchestrationContext")
		}

		return orchCtx.GetOrchestrationStatus(args[0])
	},

	"orchestrate_health_status": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("orchestrate_health_status requires 1 argument (orchestration_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("orchestrate_health_status requires OrchestrationContext")
		}

		status, err := orchCtx.GetOrchestrationHealthStatus(args[0])
		if err != nil {
			return "", err
		}

		// Format status as string
		var parts []string
		for key, value := range status {
			parts = append(parts, fmt.Sprintf("%s: %v", key, value))
		}
		return strings.Join(parts, ", "), nil
	},

	"orchestrate_healthy": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("orchestrate_healthy requires 1 argument (orchestration_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("orchestrate_healthy requires OrchestrationContext")
		}

		healthy, err := orchCtx.IsOrchestrationHealthy(args[0])
		if err != nil {
			return "", err
		}

		if healthy {
			return "true", nil
		}
		return "false", nil
	},

	// DNS functions
	"dns_resolve": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("dns_resolve requires 1 argument (domain)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("dns_resolve requires OrchestrationContext")
		}

		return orchCtx.DNSResolve(args[0])
	},

	"dns_check": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 2 {
			return "", fmt.Errorf("dns_check requires 2 arguments (domain, record_type)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("dns_check requires OrchestrationContext")
		}

		valid, err := orchCtx.DNSCheck(args[0], args[1])
		if err != nil {
			return "", err
		}

		if valid {
			return "success", nil
		}
		return "failed", nil
	},

	"dns_validate": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 2 {
			return "", fmt.Errorf("dns_validate requires 2 arguments (domain, expected_ip)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("dns_validate requires OrchestrationContext")
		}

		valid, err := orchCtx.DNSValidate(args[0], args[1])
		if err != nil {
			return "", err
		}

		if valid {
			return "valid", nil
		}
		return "invalid", nil
	},

	// Git functions
	"git_status": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("git_status requires 1 argument (path)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("git_status requires OrchestrationContext")
		}

		return orchCtx.GitStatus(args[0])
	},

	"git_branch": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("git_branch requires 1 argument (path)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("git_branch requires OrchestrationContext")
		}

		return orchCtx.GitBranch(args[0])
	},

	"git_tag": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("git_tag requires 1 argument (path)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("git_tag requires OrchestrationContext")
		}

		return orchCtx.GitTag(args[0])
	},

	// Docker functions
	"docker_ps": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("docker_ps requires 1 argument (service_name)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("docker_ps requires OrchestrationContext")
		}

		return orchCtx.DockerPS(args[0])
	},

	"docker_logs": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("docker_logs requires 1 argument (container_id)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("docker_logs requires OrchestrationContext")
		}

		return orchCtx.DockerLogs(args[0])
	},

	"docker_stats": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("docker_stats requires 1 argument (container_id)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("docker_stats requires OrchestrationContext")
		}

		return orchCtx.DockerStats(args[0])
	},

	// Compose functions
	"compose_config": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("compose_config requires 1 argument (path)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("compose_config requires OrchestrationContext")
		}

		return orchCtx.ComposeConfig(args[0])
	},

	// Makefile functions
	"make_list": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("make_list requires 1 argument (path)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("make_list requires OrchestrationContext")
		}

		targets, err := orchCtx.MakeList(args[0])
		if err != nil {
			return "", err
		}

		return strings.Join(targets, ", "), nil
	},

	"make_dry_run": func(args []string, ctx interface{}) (string, error) {
		if len(args) != 2 {
			return "", fmt.Errorf("make_dry_run requires 2 arguments (target, path)")
		}

		orchCtx, ok := ctx.(OrchestrationContext)
		if !ok {
			return "", fmt.Errorf("make_dry_run requires OrchestrationContext")
		}

		return orchCtx.MakeDryRun(args[0], args[1])
	},
}

// RegisterOrchestrationBuiltins registers all orchestration built-in functions
// Note: This function is a placeholder for future integration.
// Orchestration builtins will need to be integrated with the engine's
// execution context to access orchestration state.
func RegisterOrchestrationBuiltins() {
	// TODO: Integrate with engine's builtin system
	// for name, fn := range OrchestrationBuiltins {
	// 	Registry[name] = adaptOrchestrationBuiltin(fn)
	// }
}
