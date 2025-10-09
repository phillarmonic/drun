package engine

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Domain: Git Operations Execution
// This file contains executors for:
// - Git clone, commit, push, pull operations
// - Branch management

// executeGit executes Git operations
func (e *Engine) executeGit(gitStmt *statement.Git, ctx *ExecutionContext) error {
	// Interpolate variables in Git statement
	operation := gitStmt.Operation
	resource := gitStmt.Resource
	name := e.interpolateVariables(gitStmt.Name, ctx)

	// Interpolate options
	options := make(map[string]string, len(gitStmt.Options))
	for key, value := range gitStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun {
		return e.buildGitCommand(operation, resource, name, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch operation {
	case "create":
		switch resource {
		case "branch":
			_, _ = fmt.Fprintf(e.output, "🌿 Creating Git branch")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, ": %s", name)
			}
		case "tag":
			_, _ = fmt.Fprintf(e.output, "🏷️  Creating Git tag")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, ": %s", name)
			}
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "checkout":
		_, _ = fmt.Fprintf(e.output, "🔀 Checking out Git branch")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "merge":
		_, _ = fmt.Fprintf(e.output, "🔀 Merging Git branch")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "commit":
		_, _ = fmt.Fprintf(e.output, "💾 Committing Git changes")
		if message, exists := options["message"]; exists {
			_, _ = fmt.Fprintf(e.output, ": %s", message)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "push":
		if resource == "tag" {
			_, _ = fmt.Fprintf(e.output, "📤 Pushing Git tag")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, ": %s", name)
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "📤 Pushing Git changes")
			if remote, exists := options["remote"]; exists {
				_, _ = fmt.Fprintf(e.output, " to %s", remote)
			}
			if branch, exists := options["branch"]; exists {
				_, _ = fmt.Fprintf(e.output, "/%s", branch)
			}
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "clone":
		_, _ = fmt.Fprintf(e.output, "📥 Cloning Git repository")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "init":
		_, _ = fmt.Fprintf(e.output, "🆕 Initializing Git repository\n")
	case "add":
		_, _ = fmt.Fprintf(e.output, "➕ Adding files to Git")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "status":
		_, _ = fmt.Fprintf(e.output, "📊 Checking Git status\n")
	case "show":
		if resource == "branch" {
			_, _ = fmt.Fprintf(e.output, "🌿 Showing current Git branch\n")
		} else {
			_, _ = fmt.Fprintf(e.output, "📖 Showing Git information\n")
		}
	default:
		_, _ = fmt.Fprintf(e.output, "🔗 Running Git %s", operation)
		if resource != "" {
			_, _ = fmt.Fprintf(e.output, " %s", resource)
		}
		if name != "" {
			_, _ = fmt.Fprintf(e.output, " %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	// Build and execute the actual command
	return e.buildGitCommand(operation, resource, name, options, false)
}
