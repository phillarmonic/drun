package engine

import (
	"fmt"
	"time"

	"github.com/phillarmonic/drun/internal/domain/statement"
	"github.com/phillarmonic/drun/internal/fileops"
)

// Domain: File Operations Execution
// This file contains executors for:
// - File operations (create, delete, copy, move)

// executeFile executes a file operation statement
func (e *Engine) executeFile(fileStmt *statement.File, ctx *ExecutionContext) error {
	// Interpolate variables in paths and content
	target := e.interpolateVariables(fileStmt.Target, ctx)
	source := e.interpolateVariables(fileStmt.Source, ctx)
	content := e.interpolateVariables(fileStmt.Content, ctx)

	// Create file operation
	op := &fileops.FileOperation{
		Type:    fileStmt.Action,
		Target:  target,
		Source:  source,
		Content: content,
		IsDir:   fileStmt.IsDir,
	}

	if e.dryRun {
		result, err := op.Execute(true) // dry run
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "❌ File operation failed: %v\n", err)
			return err
		}
		_, _ = fmt.Fprintf(e.output, "📁 %s\n", result.Message)
		if fileStmt.CaptureVar != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture file content in variable '%s'\n", fileStmt.CaptureVar)
			// Set a placeholder value for the captured variable in dry-run mode
			ctx.Variables[fileStmt.CaptureVar] = "[DRY RUN] file content"
		}
		return nil
	}

	// Handle special actions that need preprocessing
	switch fileStmt.Action {
	case "backup":
		if target == "" {
			// Generate default backup name with timestamp
			timestamp := time.Now().Format("2006-01-02-15-04-05")
			target = source + ".backup-" + timestamp
		}
		op.Target = target
		op.Type = "copy" // Backup is essentially a copy operation
	case "check_exists":
		// Check if file exists
		if e.fileExists(target) {
			_, _ = fmt.Fprintf(e.output, "✅ File exists: %s\n", target)
		} else {
			_, _ = fmt.Fprintf(e.output, "❌ File does not exist: %s\n", target)
		}
		return nil
	case "get_size":
		// Get file size
		size, err := e.getFileSize(target)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "❌ Failed to get file size: %v\n", err)
			return err
		}
		_, _ = fmt.Fprintf(e.output, "📏 File size: %s (%d bytes)\n", target, size)
		return nil
	}

	// Show what we're about to do
	switch fileStmt.Action {
	case "create":
		if fileStmt.IsDir {
			_, _ = fmt.Fprintf(e.output, "📁 Creating directory: %s\n", target)
		} else {
			_, _ = fmt.Fprintf(e.output, "📄 Creating file: %s\n", target)
		}
	case "copy":
		_, _ = fmt.Fprintf(e.output, "📋 Copying: %s → %s\n", source, target)
	case "move":
		_, _ = fmt.Fprintf(e.output, "🚚 Moving: %s → %s\n", source, target)
	case "delete":
		if fileStmt.IsDir {
			_, _ = fmt.Fprintf(e.output, "🗑️  Deleting directory: %s\n", target)
		} else {
			_, _ = fmt.Fprintf(e.output, "🗑️  Deleting file: %s\n", target)
		}
	case "read":
		_, _ = fmt.Fprintf(e.output, "📖 Reading file: %s\n", target)
	case "write":
		_, _ = fmt.Fprintf(e.output, "✏️  Writing to file: %s\n", target)
	case "append":
		_, _ = fmt.Fprintf(e.output, "➕ Appending to file: %s\n", target)
	case "backup":
		_, _ = fmt.Fprintf(e.output, "💾 Backing up: %s → %s\n", source, target)
	}

	// Execute the file operation
	result, err := op.Execute(false)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌ File operation failed: %v\n", err)
		return err
	}

	// Handle capture for read operations
	if fileStmt.CaptureVar != "" && fileStmt.Action == "read" {
		ctx.Variables[fileStmt.CaptureVar] = result.Content
		_, _ = fmt.Fprintf(e.output, "📦 Captured file content in variable '%s' (%d bytes)\n",
			fileStmt.CaptureVar, len(result.Content))
	}

	// Show success message
	if result.Success {
		_, _ = fmt.Fprintf(e.output, "✅ %s\n", result.Message)
	} else {
		_, _ = fmt.Fprintf(e.output, "⚠️  %s\n", result.Message)
	}

	return nil
}
