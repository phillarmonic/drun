package engine

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/detection"
	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Domain: Tool Requirements Execution
// This file contains the executor for "requires tools:" blocks.
// Tool requirements are checked eagerly — if a tool is missing or doesn't
// meet the version constraints, execution fails immediately.

// executeRequiresTools checks that all required tools are available and meet version constraints.
func (e *Engine) executeRequiresTools(stmt *statement.RequiresTools, ctx *ExecutionContext) error {
	detector := detection.NewDetector()
	return e.checkToolRequirements(detector, stmt.Tools)
}

// checkToolRequirements validates a list of tool requirements against the system.
// This is shared between task-level execution and project-level startup checks.
func (e *Engine) checkToolRequirements(detector *detection.Detector, tools []statement.ToolRequirement) error {
	for _, tool := range tools {
		// Step 1: Check if the tool is installed
		if !detector.IsToolAvailable(tool.Name) {
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] ❌ Required tool '%s' is not installed\n", tool.Name)
				continue
			}
			return fmt.Errorf("required tool '%s' is not installed", tool.Name)
		}

		// Step 2: If there are version constraints, check them
		if len(tool.Constraints) > 0 {
			currentVersion := detector.GetToolVersion(tool.Name)
			if currentVersion == "" {
				if e.dryRun {
					_, _ = fmt.Fprintf(e.output, "[DRY RUN] ⚠️  Could not determine version for '%s'\n", tool.Name)
					continue
				}
				return fmt.Errorf("required tool '%s' is installed but version could not be determined (needed: %s)",
					tool.Name, formatConstraints(tool.Constraints))
			}

			for _, constraint := range tool.Constraints {
				if !detector.CompareVersion(currentVersion, constraint.Operator, constraint.Version) {
					if e.dryRun {
						_, _ = fmt.Fprintf(e.output, "[DRY RUN] ❌ Tool '%s' version %s does not satisfy %s %s\n",
							tool.Name, currentVersion, constraint.Operator, constraint.Version)
						continue
					}
					return fmt.Errorf("required tool '%s' version %s does not satisfy constraint %s %s",
						tool.Name, currentVersion, constraint.Operator, constraint.Version)
				}
			}

			if e.verbose || e.dryRun {
				_, _ = fmt.Fprintf(e.output, "✅  %s %s (%s)\n",
					tool.Name, currentVersion, formatConstraints(tool.Constraints))
			}
		} else {
			// No version constraints — tool just needs to exist
			if e.verbose || e.dryRun {
				_, _ = fmt.Fprintf(e.output, "✅  %s is available\n", tool.Name)
			}
		}
	}

	return nil
}

// checkProjectToolRequirements checks project-level tool requirements at startup.
func (e *Engine) checkProjectToolRequirements(projectCtx *ProjectContext) error {
	if projectCtx == nil || len(projectCtx.RequiredTools) == 0 {
		return nil
	}

	detector := detection.NewDetector()
	return e.checkToolRequirements(detector, projectCtx.RequiredTools)
}

// formatConstraints formats a slice of version constraints for error messages.
func formatConstraints(constraints []statement.VersionConstraint) string {
	parts := make([]string, len(constraints))
	for i, c := range constraints {
		parts[i] = c.Operator + " " + c.Version
	}
	return strings.Join(parts, ", ")
}
