package engine

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/shell"
)

// Domain: Shell Command Execution
// This file contains executors for:
// - Single-line shell commands
// - Multi-line shell scripts
// - Platform-specific shell configuration

// executeShell executes a shell command statement
func (e *Engine) executeShell(shellStmt *ast.ShellStatement, ctx *ExecutionContext) error {
	if shellStmt.IsMultiline {
		return e.executeMultilineShell(shellStmt, ctx)
	}
	return e.executeSingleLineShell(shellStmt, ctx)
}

// executeMultilineShell executes multiline shell commands as a single shell session
func (e *Engine) executeMultilineShell(shellStmt *ast.ShellStatement, ctx *ExecutionContext) error {
	// Interpolate variables in all commands
	var interpolatedCommands []string
	for _, cmd := range shellStmt.Commands {
		interpolatedCmd := e.interpolateVariables(cmd, ctx)
		interpolatedCommands = append(interpolatedCommands, interpolatedCmd)
	}

	// Join commands with newlines to create a single script
	script := strings.Join(interpolatedCommands, "\n")

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute multiline shell commands:\n")
		for i, cmd := range interpolatedCommands {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN]   %d: %s\n", i+1, cmd)
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

	// Show what we're about to execute (verbose mode only)
	if e.verbose {
		switch shellStmt.Action {
		case "run":
			_, _ = fmt.Fprintf(e.output, "🏃 Running multiline commands (%d lines):\n", len(interpolatedCommands))
		case "exec":
			_, _ = fmt.Fprintf(e.output, "⚡ Executing multiline commands (%d lines):\n", len(interpolatedCommands))
		case "shell":
			_, _ = fmt.Fprintf(e.output, "🐚 Shell multiline commands (%d lines):\n", len(interpolatedCommands))
		case "capture":
			_, _ = fmt.Fprintf(e.output, "📥 Capturing multiline commands (%d lines):\n", len(interpolatedCommands))
		}

		// Show each command with line numbers
		for i, cmd := range interpolatedCommands {
			_, _ = fmt.Fprintf(e.output, "  %d: %s\n", i+1, cmd)
		}
	}

	// Execute the script as a single shell session
	result, err := shell.Execute(script, opts)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌ Multiline command failed: %v\n", err)
		return err
	}

	// Handle capture
	if shellStmt.CaptureVar != "" && shellStmt.Action == "capture" {
		ctx.Variables[shellStmt.CaptureVar] = result.Stdout
		_, _ = fmt.Fprintf(e.output, "📦 Captured output in variable '%s'\n", shellStmt.CaptureVar)
	}

	// Show execution summary
	if result.Success {
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅ Multiline commands completed successfully (exit code: %d, duration: %v)\n",
				result.ExitCode, result.Duration)
		}
	} else {
		_, _ = fmt.Fprintf(e.output, "⚠️  Multiline commands completed with exit code: %d (duration: %v)\n",
			result.ExitCode, result.Duration)
	}

	return nil
}

// getPlatformShellConfig returns the shell configuration for the current platform
func (e *Engine) getPlatformShellConfig(ctx *ExecutionContext) *shell.Options {
	opts := shell.DefaultOptions()

	if ctx.Project == nil || len(ctx.Project.ShellConfigs) == 0 {
		return opts
	}

	// Determine current platform
	platform := runtime.GOOS

	// Get platform-specific configuration
	config, exists := ctx.Project.ShellConfigs[platform]
	if !exists {
		return opts
	}

	// Apply platform configuration
	if config.Executable != "" {
		opts.Shell = config.Executable
	}

	// Add startup arguments to environment or handle them appropriately
	// Note: The shell package currently doesn't support startup args directly,
	// so we'll store them in environment for now
	if len(config.Args) > 0 {
		if opts.Environment == nil {
			opts.Environment = make(map[string]string)
		}
		// Store args as a space-separated string for now
		opts.Environment["DRUN_SHELL_ARGS"] = strings.Join(config.Args, " ")
	}

	// Apply environment variables
	if len(config.Environment) > 0 {
		if opts.Environment == nil {
			opts.Environment = make(map[string]string)
		}
		for key, value := range config.Environment {
			opts.Environment[key] = value
		}
	}

	return opts
}
