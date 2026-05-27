package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/drun/internal/domain/statement"
	"github.com/phillarmonic/drun/internal/shell"
)

// Domain: Utility Helpers
// This file contains miscellaneous utility helper methods

// parseArrayLiteralString parses an array literal string like ["item1", "item2", "item3"] into a slice of strings
func (e *Engine) parseArrayLiteralString(arrayStr string) []string {
	// Remove brackets
	arrayStr = strings.TrimSpace(arrayStr)
	if len(arrayStr) < 2 || arrayStr[0] != '[' || arrayStr[len(arrayStr)-1] != ']' {
		return []string{}
	}

	content := arrayStr[1 : len(arrayStr)-1]
	content = strings.TrimSpace(content)

	// Handle empty array
	if content == "" {
		return []string{}
	}

	var items []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, char := range content {
		switch char {
		case '\\':
			if !escaped {
				escaped = true
				continue
			}
			current.WriteRune(char)
		case '"':
			if !escaped {
				inQuotes = !inQuotes
			} else {
				current.WriteRune(char)
			}
		case ',':
			if !inQuotes && !escaped {
				// End of current item
				item := strings.TrimSpace(current.String())
				if item != "" {
					items = append(items, item)
				}
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
		escaped = false
	}

	// Add the last item
	item := strings.TrimSpace(current.String())
	if item != "" {
		items = append(items, item)
	}

	return items
}

type serviceContextInfo struct {
	Name string
	Path string
}

func (e *Engine) resolveServiceContext(rawName string, isLiteral bool, ctx *ExecutionContext) (*serviceContextInfo, error) {
	if ctx == nil || ctx.Program == nil {
		return nil, fmt.Errorf("service-scoped commands require program context")
	}
	if len(ctx.Program.Services) == 0 {
		return nil, fmt.Errorf("service-scoped commands require at least one service definition")
	}

	serviceName, err := e.resolveServiceNameValue(rawName, isLiteral, ctx)
	if err != nil {
		return nil, err
	}

	var targetServicePath string
	for _, svc := range ctx.Program.Services {
		if svc.Name == serviceName {
			targetServicePath = svc.Path
			break
		}
	}

	if targetServicePath == "" {
		return nil, fmt.Errorf("service '%s' not found", serviceName)
	}

	servicePath := targetServicePath
	if !filepath.IsAbs(servicePath) {
		baseDir := ""
		if ctx.CurrentFile != "" {
			baseDir = filepath.Dir(ctx.CurrentFile)
			if filepath.Base(baseDir) == ".drun" {
				baseDir = filepath.Dir(baseDir)
			}
		}
		if baseDir == "" {
			if cwd, err := os.Getwd(); err == nil {
				baseDir = cwd
			}
		}
		if baseDir != "" {
			servicePath = filepath.Join(baseDir, servicePath)
		}
	}

	absPath, err := filepath.Abs(servicePath)
	if err != nil {
		return nil, fmt.Errorf("resolving path for service '%s': %w", serviceName, err)
	}

	return &serviceContextInfo{
		Name: serviceName,
		Path: absPath,
	}, nil
}

func (e *Engine) resolveServiceNameValue(rawName string, isLiteral bool, ctx *ExecutionContext) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("service-scoped commands require execution context")
	}

	candidate := strings.TrimSpace(rawName)
	if isLiteral {
		candidate = strings.TrimSpace(e.interpolateVariables(candidate, ctx))
	} else {
		candidate = e.resolveDynamicServiceName(candidate, ctx)
	}

	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", fmt.Errorf("service name resolved to empty")
	}

	return candidate, nil
}

func (e *Engine) resolveDynamicServiceName(identifier string, ctx *ExecutionContext) string {
	name := strings.TrimSpace(identifier)
	if name == "" {
		return ""
	}

	lookup := func(key string) (string, bool) {
		if ctx.Parameters != nil {
			if val, exists := ctx.Parameters[key]; exists && val != nil {
				return val.AsString(), true
			}
		}
		if ctx.Variables != nil {
			if val, exists := ctx.Variables[key]; exists {
				return val, true
			}
		}
		return "", false
	}

	if strings.HasPrefix(name, "$") {
		trimmed := name[1:]
		if value, ok := lookup(trimmed); ok {
			return value
		}
		name = trimmed
	}

	if value, ok := lookup(name); ok {
		return value
	}

	interpolated := e.interpolateVariables("{"+name+"}", ctx)
	if interpolated != "{"+name+"}" {
		return interpolated
	}

	return name
}

// executeSingleLineShell executes a single-line shell command
func (e *Engine) executeSingleLineShell(shellStmt *statement.Shell, ctx *ExecutionContext, svcCtx *serviceContextInfo) error {
	// Interpolate variables in the command
	interpolatedCommand, err := e.interpolateVariablesWithError(shellStmt.Command, ctx)
	if err != nil {
		return fmt.Errorf("in shell command: %w", err)
	}

	if e.dryRun {
		if svcCtx != nil {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute shell command in service '%s' (%s): %s\n", svcCtx.Name, svcCtx.Path, interpolatedCommand)
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute shell command: %s\n", interpolatedCommand)
		}
		if shellStmt.CaptureVar != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture output as: %s\n", shellStmt.CaptureVar)
			// Set a placeholder value for the captured variable in dry-run mode
			ctx.Variables[shellStmt.CaptureVar] = "[DRY RUN] command output"
		}
		return nil
	}

	// Configure shell options based on the action type and platform configuration
	opts := e.getPlatformShellConfig(ctx)
	opts.CaptureOutput = true
	opts.StreamOutput = shellStmt.StreamOutput
	opts.Output = e.output
	if svcCtx != nil {
		opts.WorkingDir = svcCtx.Path
	} else if ctx != nil && ctx.WorkingDir != "" {
		opts.WorkingDir = ctx.WorkingDir
	}

	// Show what we're about to execute (verbose mode only)
	if e.verbose {
		switch shellStmt.Action {
		case "run":
			if svcCtx != nil {
				_, _ = fmt.Fprintf(e.output, "🏃 Running in service '%s': %s\n", svcCtx.Name, interpolatedCommand)
			} else {
				_, _ = fmt.Fprintf(e.output, "🏃 Running: %s\n", interpolatedCommand)
			}
		case "exec":
			_, _ = fmt.Fprintf(e.output, "⚡ Executing: %s\n", interpolatedCommand)
		case "shell":
			_, _ = fmt.Fprintf(e.output, "🐚 Shell: %s\n", interpolatedCommand)
		case "capture":
			_, _ = fmt.Fprintf(e.output, "📥  Capturing: %s\n", interpolatedCommand)
		}
	}

	// Execute the command
	result, err := shell.Execute(interpolatedCommand, opts)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌  Command failed: %v\n", err)
		return err
	}

	// Handle capture
	if shellStmt.CaptureVar != "" && shellStmt.Action == "capture" {
		ctx.Variables[shellStmt.CaptureVar] = result.Stdout
		_, _ = fmt.Fprintf(e.output, "📦  Captured output in variable '%s'\n", shellStmt.CaptureVar)
	}

	// Show execution summary
	if result.Success {
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅  Command completed successfully (exit code: %d, duration: %v)\n",
				result.ExitCode, result.Duration)
		}
	} else {
		_, _ = fmt.Fprintf(e.output, "⚠️  Command completed with exit code: %d (duration: %v)\n",
			result.ExitCode, result.Duration)
	}

	return nil
}
