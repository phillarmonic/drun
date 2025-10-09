package engine

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/detection"
	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Domain: Detection Helpers
// This file contains helper methods for tool and environment detection

// executeDetectOperation executes detect operations
func (e *Engine) executeDetectOperation(detector *detection.Detector, stmt *statement.Detection, ctx *ExecutionContext) error {
	switch stmt.Target {
	case "project":
		if stmt.Condition == "type" {
			types := detector.DetectProjectType()
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would detect project types: %v\n", types)
			} else {
				_, _ = fmt.Fprintf(e.output, "🔍 Detected project types: %v\n", types)
			}
		}
	default:
		// Detect tool
		if stmt.Condition == "version" {
			version := detector.GetToolVersion(stmt.Target)
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would detect %s version: %s\n", stmt.Target, version)
			} else {
				_, _ = fmt.Fprintf(e.output, "🔍 Detected %s version: %s\n", stmt.Target, version)
			}
			// Set the detected version in variables (e.g., docker_version)
			ctx.Variables[stmt.Target+"_version"] = version
		} else {
			available := detector.IsToolAvailable(stmt.Target)
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s is available: %t\n", stmt.Target, available)
			} else {
				_, _ = fmt.Fprintf(e.output, "🔍 %s available: %t\n", stmt.Target, available)
			}
		}
	}

	return nil
}

// executeIfAvailable executes "if tool is available" and "if tool is not available" conditions
func (e *Engine) executeIfAvailable(detector *detection.Detector, stmt *statement.Detection, ctx *ExecutionContext) error {
	// Build list of all tools to check (primary + alternatives)
	toolsToCheck := []string{stmt.Target}
	toolsToCheck = append(toolsToCheck, stmt.Alternatives...)

	// Check availability for all tools
	var conditionMet bool
	var conditionText string

	if stmt.Condition == "not_available" {
		// For "not available": condition is true if ANY tool is not available (OR logic)
		conditionMet = false
		for _, tool := range toolsToCheck {
			if !detector.IsToolAvailable(tool) {
				conditionMet = true
				break
			}
		}

		if len(toolsToCheck) == 1 {
			conditionText = fmt.Sprintf("%s is not available", stmt.Target)
		} else {
			toolNames := strings.Join(toolsToCheck, ", ")
			conditionText = fmt.Sprintf("any of [%s] is not available", toolNames)
		}
	} else {
		// For "is available": condition is true if ALL tools are available (AND logic)
		conditionMet = true
		for _, tool := range toolsToCheck {
			if !detector.IsToolAvailable(tool) {
				conditionMet = false
				break
			}
		}

		if len(toolsToCheck) == 1 {
			conditionText = fmt.Sprintf("%s is available", stmt.Target)
		} else {
			toolNames := strings.Join(toolsToCheck, ", ")
			conditionText = fmt.Sprintf("all of [%s] are available", toolNames)
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s: %t\n", conditionText, conditionMet)
		if conditionMet {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute if body\n")
			for _, bodyStmt := range stmt.Body {
				if err := e.executeStatement(bodyStmt, ctx); err != nil {
					return err
				}
			}
		} else if len(stmt.ElseBody) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute else body\n")
			for _, elseStmt := range stmt.ElseBody {
				if err := e.executeStatement(elseStmt, ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Checking if %s: %t\n", conditionText, conditionMet)
	}

	if conditionMet {
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	} else if len(stmt.ElseBody) > 0 {
		for _, elseStmt := range stmt.ElseBody {
			if err := e.executeStatement(elseStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeIfVersion executes "if tool version comparison" conditions
func (e *Engine) executeIfVersion(detector *detection.Detector, stmt *statement.Detection, ctx *ExecutionContext) error {
	version := detector.GetToolVersion(stmt.Target)
	targetVersion := e.interpolateVariables(stmt.Value, ctx)

	matches := detector.CompareVersion(version, stmt.Condition, targetVersion)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s version %s %s %s: %t (current: %s)\n",
			stmt.Target, version, stmt.Condition, targetVersion, matches, version)
		if matches {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute if-version body for %s\n", stmt.Target)
			for _, bodyStmt := range stmt.Body {
				if err := e.executeStatement(bodyStmt, ctx); err != nil {
					return err
				}
			}
		} else if len(stmt.ElseBody) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute else body for %s\n", stmt.Target)
			for _, elseStmt := range stmt.ElseBody {
				if err := e.executeStatement(elseStmt, ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Checking %s version %s %s %s: %t (current: %s)\n",
			stmt.Target, version, stmt.Condition, targetVersion, matches, version)
	}

	if matches {
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	} else if len(stmt.ElseBody) > 0 {
		for _, elseStmt := range stmt.ElseBody {
			if err := e.executeStatement(elseStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeWhenEnvironment executes "when in environment" conditions
func (e *Engine) executeWhenEnvironment(detector *detection.Detector, stmt *statement.Detection, ctx *ExecutionContext) error {
	currentEnv := detector.DetectEnvironment()
	matches := currentEnv == stmt.Target

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if in %s environment: %t (current: %s)\n",
			stmt.Target, matches, currentEnv)
		if matches {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute when-environment body\n")
			for _, bodyStmt := range stmt.Body {
				if err := e.executeStatement(bodyStmt, ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Checking if in %s environment: %t (current: %s)\n",
			stmt.Target, matches, currentEnv)
	}

	if matches {
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeDetectAvailable executes "detect available" operations with tool alternatives
func (e *Engine) executeDetectAvailable(detector *detection.Detector, stmt *statement.Detection, ctx *ExecutionContext) error {
	// Build list of tools to try (primary + alternatives)
	toolsToTry := []string{stmt.Target}
	toolsToTry = append(toolsToTry, stmt.Alternatives...)

	var workingTool string
	var found bool

	// Try each tool variant until we find one that works
	for _, tool := range toolsToTry {
		if detector.IsToolAvailable(tool) {
			workingTool = tool
			found = true
			break
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would detect available tool from: %v\n", toolsToTry)
		if found {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would find: %s\n", workingTool)
			if stmt.CaptureVar != "" {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture as %s: %s\n", stmt.CaptureVar, workingTool)
				// Set the variable in dry-run mode too
				ctx.Variables[stmt.CaptureVar] = workingTool
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would find: none available\n")
			if stmt.CaptureVar != "" {
				// Set a placeholder in dry-run mode when no tool is found
				ctx.Variables[stmt.CaptureVar] = "[DRY RUN] no tool available"
			}
		}
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Detecting available tool from: %v\n", toolsToTry)
	}

	if found {
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅ Found: %s\n", workingTool)
		}

		// Capture the working tool variant in a variable if specified
		if stmt.CaptureVar != "" {
			ctx.Variables[stmt.CaptureVar] = workingTool
			if e.verbose {
				_, _ = fmt.Fprintf(e.output, "📝 Captured as %s: %s\n", stmt.CaptureVar, workingTool)
			}
		}
	} else {
		_, _ = fmt.Fprintf(e.output, "❌ None of the tools are available: %v\n", toolsToTry)
	}

	return nil
}
