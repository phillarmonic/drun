package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Domain: Working Directory Management
// This file implements the "use workdir" statement executor.

// executeChangeWorkdir handles the `use workdir "path"` statement.
//
// Behavior:
//   - Interpolates variables in the path.
//   - If the path is relative, it is resolved against ctx.OriginalWorkingDir
//     (the cwd captured at task start), NOT against the current ctx.WorkingDir.
//     This means multiple `use workdir` calls in one task don't chain.
//   - Validates that the resolved directory exists.
//   - Sets ctx.WorkingDir for all subsequent shell commands in this task.
//   - In dry-run mode, logs the intended change without resolving the path.
func (e *Engine) executeChangeWorkdir(stmt *statement.ChangeWorkdir, ctx *ExecutionContext) error {
	// Interpolate variables in the path
	interpolatedPath, err := e.interpolateVariablesWithError(stmt.Path, ctx)
	if err != nil {
		return fmt.Errorf("use workdir: %w", err)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set working directory to: %s\n", interpolatedPath)
		return nil
	}

	// Resolve the directory:
	// - Absolute paths are used as-is.
	// - Relative paths are resolved against OriginalWorkingDir (the process cwd
	//   captured when this task started), not the current WorkingDir override.
	resolved := interpolatedPath
	if !filepath.IsAbs(resolved) {
		base := ctx.OriginalWorkingDir
		if base == "" {
			// Fallback: use the real process cwd
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				return fmt.Errorf("use workdir: could not determine current working directory: %w", cwdErr)
			}
			base = cwd
		}
		resolved = filepath.Join(base, resolved)
	}

	// Clean up the path
	resolved = filepath.Clean(resolved)

	// Validate the directory exists
	info, statErr := os.Stat(resolved)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return fmt.Errorf("use workdir: directory %q does not exist", resolved)
		}
		return fmt.Errorf("use workdir: cannot access directory %q: %w", resolved, statErr)
	}
	if !info.IsDir() {
		return fmt.Errorf("use workdir: path %q is not a directory", resolved)
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "📁 Working directory set to: %s\n", resolved)
	}

	ctx.WorkingDir = resolved
	return nil
}
