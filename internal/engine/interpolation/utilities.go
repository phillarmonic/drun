package interpolation

import (
	"strings"
)

// isComplexExpression checks if an expression contains operations or function calls
func (i *Interpolator) isComplexExpression(expr string) bool {
	// Check for variable operations like "without", "with", etc.
	if strings.Contains(expr, " without ") || strings.Contains(expr, " with ") {
		return true
	}
	// Check for function calls (contains parentheses or dots)
	if strings.Contains(expr, "(") || strings.Contains(expr, ".") {
		return true
	}
	// Check for pipe operations
	if strings.Contains(expr, " | ") {
		return true
	}
	return false
}

// extractBaseVariable extracts the base variable from a complex expression
func (i *Interpolator) extractBaseVariable(expr string) string {
	// For expressions like "$version without prefix 'v'", extract "$version"
	parts := strings.Fields(expr)
	if len(parts) > 0 && strings.HasPrefix(parts[0], "$") {
		return parts[0]
	}
	return ""
}

// variableExists checks if a variable exists in the context
func (i *Interpolator) variableExists(varName string, ctx Context) bool {
	if ctx == nil {
		return false
	}

	params := ctx.GetParameters()
	vars := ctx.GetVariables()
	project := ctx.GetProject()

	// Remove $ prefix for checking
	cleanName := varName
	if strings.HasPrefix(varName, "$") {
		cleanName = varName[1:]
	}

	// Check parameters
	if _, exists := params[cleanName]; exists {
		return true
	}
	// Check variables
	if _, exists := vars[cleanName]; exists {
		return true
	}
	// Check variables with $ prefix
	if _, exists := vars[varName]; exists {
		return true
	}

	// Check project-level variables for backward compatibility
	if project != nil {
		// Check built-in project variables
		if cleanName == "project" && project.GetName() != "" {
			return true
		}
		if cleanName == "version" && project.GetVersion() != "" {
			return true
		}
		// Check project settings
		if settings := project.GetSettings(); settings != nil {
			if _, exists := settings[cleanName]; exists {
				return true
			}
		}
	}

	return false
}
