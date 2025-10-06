package engine

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
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

// executeSingleLineShell executes a single-line shell command
func (e *Engine) executeSingleLineShell(shellStmt *ast.ShellStatement, ctx *ExecutionContext) error {
	// Interpolate variables in the command
	interpolatedCommand, err := e.interpolateVariablesWithError(shellStmt.Command, ctx)
	if err != nil {
		return fmt.Errorf("in shell command: %w", err)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute shell command: %s\n", interpolatedCommand)
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
			_, _ = fmt.Fprintf(e.output, "🏃 Running: %s\n", interpolatedCommand)
		case "exec":
			_, _ = fmt.Fprintf(e.output, "⚡ Executing: %s\n", interpolatedCommand)
		case "shell":
			_, _ = fmt.Fprintf(e.output, "🐚 Shell: %s\n", interpolatedCommand)
		case "capture":
			_, _ = fmt.Fprintf(e.output, "📥 Capturing: %s\n", interpolatedCommand)
		}
	}

	// Execute the command
	result, err := shell.Execute(interpolatedCommand, opts)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌ Command failed: %v\n", err)
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
			_, _ = fmt.Fprintf(e.output, "✅ Command completed successfully (exit code: %d, duration: %v)\n",
				result.ExitCode, result.Duration)
		}
	} else {
		_, _ = fmt.Fprintf(e.output, "⚠️  Command completed with exit code: %d (duration: %v)\n",
			result.ExitCode, result.Duration)
	}

	return nil
}
