package engine

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
)

// Domain: Docker Operations Execution
// This file contains executors for:
// - Docker build, run, push, pull operations
// - Docker Compose operations
// - Container management

// executeTry executes a try/catch/finally statement
// executeDocker executes Docker operations
func (e *Engine) executeDocker(dockerStmt *ast.DockerStatement, ctx *ExecutionContext) error {
	// Interpolate variables in Docker statement
	operation := dockerStmt.Operation
	resource := dockerStmt.Resource
	name := e.interpolateVariables(dockerStmt.Name, ctx)

	// Interpolate options
	options := make(map[string]string, len(dockerStmt.Options))
	for key, value := range dockerStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun {
		return e.buildDockerCommand(operation, resource, name, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch operation {
	case "build":
		_, _ = fmt.Fprintf(e.output, "🔨 Building Docker image")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "push":
		_, _ = fmt.Fprintf(e.output, "📤 Pushing Docker image")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		if registry, exists := options["to"]; exists {
			_, _ = fmt.Fprintf(e.output, " to %s", registry)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "pull":
		_, _ = fmt.Fprintf(e.output, "📥 Pulling Docker image")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "run":
		_, _ = fmt.Fprintf(e.output, "🚀 Running Docker container")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		if port, exists := options["port"]; exists {
			_, _ = fmt.Fprintf(e.output, " on port %s", port)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "stop":
		_, _ = fmt.Fprintf(e.output, "🛑 Stopping Docker container")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "remove":
		_, _ = fmt.Fprintf(e.output, "🗑️  Removing Docker %s", resource)
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "compose":
		command := options["command"]
		switch command {
		case "up":
			_, _ = fmt.Fprintf(e.output, "🚀 Starting Docker Compose services\n")
		case "down":
			_, _ = fmt.Fprintf(e.output, "🛑 Stopping Docker Compose services\n")
		case "build":
			_, _ = fmt.Fprintf(e.output, "🔨 Building Docker Compose services\n")
		default:
			_, _ = fmt.Fprintf(e.output, "🐳 Running Docker Compose: %s\n", command)
		}
	case "scale":
		if resource == "compose" {
			replicas := options["replicas"]
			_, _ = fmt.Fprintf(e.output, "📊 Scaling Docker Compose service")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, " %s", name)
			}
			if replicas != "" {
				_, _ = fmt.Fprintf(e.output, " to %s replicas", replicas)
			}
			_, _ = fmt.Fprintf(e.output, "\n")
		}
	default:
		_, _ = fmt.Fprintf(e.output, "🐳 Running Docker %s", operation)
		if resource != "" {
			_, _ = fmt.Fprintf(e.output, " %s", resource)
		}
		if name != "" {
			_, _ = fmt.Fprintf(e.output, " %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	// Build and execute the actual command
	return e.buildDockerCommand(operation, resource, name, options, false)
}
