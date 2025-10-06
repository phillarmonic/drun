package interpolation

import (
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/internal/builtins"
)

// resolveSimpleVariableDirectly handles simple variable resolution with proper empty string support
func (i *Interpolator) resolveSimpleVariableDirectly(variable string, ctx Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	params := ctx.GetParameters()
	vars := ctx.GetVariables()
	project := ctx.GetProject()

	// Handle variables with $ prefix (most common case for interpolation)
	if strings.HasPrefix(variable, "$") {
		// First try to find the variable with the $ prefix (shell captures)
		if value, exists := vars[variable]; exists {
			return value, true
		}

		// Then try without the $ prefix (legacy variables)
		varName := variable[1:] // Remove the $ prefix

		// Check parameters (stored without $ prefix)
		if value, exists := params[varName]; exists {
			return value.AsString(), true
		}

		// Check captured variables (stored without $ prefix)
		if value, exists := vars[varName]; exists {
			return value, true
		}

		// Check project-level variables for backward compatibility
		if project != nil {
			// Check built-in project variables
			if varName == "project" && project.GetName() != "" {
				return project.GetName(), true
			}
			if varName == "version" && project.GetVersion() != "" {
				return project.GetVersion(), true
			}
			// Check project settings
			if settings := project.GetSettings(); settings != nil {
				if value, exists := settings[varName]; exists {
					return value, true
				}
			}
		}
	} else {
		// Check parameters (bare identifiers)
		if value, exists := params[variable]; exists {
			return value.AsString(), true
		}
		// Check captured variables (bare identifiers)
		if value, exists := vars[variable]; exists {
			return value, true
		}

		// Check project-level variables for backward compatibility
		if project != nil {
			// Check built-in project variables
			if variable == "project" && project.GetName() != "" {
				return project.GetName(), true
			}
			if variable == "version" && project.GetVersion() != "" {
				return project.GetVersion(), true
			}
			// Check project settings
			if settings := project.GetSettings(); settings != nil {
				if value, exists := settings[variable]; exists {
					return value, true
				}
			}
		}
	}

	return "", false
}

// resolveExpression resolves various types of expressions
func (i *Interpolator) resolveExpression(expr string, ctx Context) string {
	// 0. Check for conditional expressions (ternary and if-then-else)
	// Ternary: "$var ? 'true_val' : 'false_val'"
	if strings.Contains(expr, "?") && strings.Contains(expr, ":") {
		result, matched := i.resolveTernaryExpression(expr, ctx)
		if matched {
			return result // Return even if empty string
		}
	}

	// If-then-else: "if $var then 'val1' else 'val2'" or "if $var is 'value' then 'val1' else 'val2'"
	if strings.HasPrefix(strings.TrimSpace(expr), "if ") && strings.Contains(expr, " then ") && strings.Contains(expr, " else ") {
		result, matched := i.resolveIfThenElse(expr, ctx)
		if matched {
			return result // Return even if empty string
		}
	}

	// 1. Check for variable operations (e.g., "$version without prefix 'v'")
	// Delegate to engine callback if available
	if i.resolveVariableOps != nil {
		if result := i.resolveVariableOps(expr, ctx); result != "" {
			return result
		}
	}

	// 2. Check for context-aware builtin functions first
	if expr == "current file" && ctx != nil {
		if ctx.GetCurrentFile() != "" {
			return ctx.GetCurrentFile()
		}
		return "<no file>"
	}

	// 3. Check for builtin function with piped operations (e.g., "current git branch | replace '/' by '-'")
	if strings.Contains(expr, "|") {
		parts := strings.SplitN(expr, "|", 2)
		if len(parts) == 2 {
			funcName := strings.TrimSpace(parts[0])
			operations := strings.TrimSpace(parts[1])

			// Check if the first part is a builtin function
			if builtins.IsBuiltin(funcName) {
				if result, err := builtins.CallBuiltin(funcName); err == nil {
					// Delegate to engine callback for operations
					if i.resolveBuiltinOps != nil {
						if finalResult, err := i.resolveBuiltinOps(funcName, operations, ctx); err == nil {
							return finalResult
						}
					}
					// If operations parsing fails, just return the builtin result
					return result
				}
			}
		}
	}

	// 4. Check if it's a simple builtin function call (no arguments)
	if builtins.IsBuiltin(expr) {
		if result, err := builtins.CallBuiltin(expr); err == nil {
			return result
		}
	}

	// 5. Check for function calls with quoted string arguments
	// Pattern: "function('arg')" or "function(\"arg\")" or "function('arg1', 'arg2')"
	if matches := i.quotedArgRegex.FindStringSubmatch(expr); len(matches) == 3 {
		funcName := strings.TrimSpace(matches[1])
		argsStr := matches[2]

		// Parse arguments - handle both single and multiple quoted arguments
		args := i.parseQuotedArguments(argsStr)

		if builtins.IsBuiltin(funcName) && len(args) > 0 {
			if result, err := builtins.CallBuiltin(funcName, args...); err == nil {
				return result
			}
		}
	}

	// 6. Check for function calls with parameter arguments
	// Pattern: "function(param)" where param is a parameter name
	if matches := i.paramArgRegex.FindStringSubmatch(expr); len(matches) == 3 {
		funcName := strings.TrimSpace(matches[1])
		paramName := strings.TrimSpace(matches[2])

		// Resolve the parameter first
		if ctx != nil {
			params := ctx.GetParameters()
			if paramValue, exists := params[paramName]; exists {
				if builtins.IsBuiltin(funcName) {
					if result, err := builtins.CallBuiltin(funcName, paramValue.AsString()); err == nil {
						return result
					}
				}
			}
		}
	}

	// 7. Check for $globals.key or $globals.namespace.key syntax for project settings
	if strings.HasPrefix(expr, "$globals.") {
		if ctx != nil {
			project := ctx.GetProject()
			if project != nil {
				key := expr[9:] // Remove "$globals." prefix

				// First check local project settings
				if settings := project.GetSettings(); settings != nil {
					if value, exists := settings[key]; exists {
						return value
					}
				}

				// Then check included/namespaced settings (e.g., $globals.docker.api_url)
				if includedSettings := getIncludedSettings(project); includedSettings != nil {
					if value, exists := includedSettings[key]; exists {
						return value
					}
				}

				// Check special project variables
				if key == "version" && project.GetVersion() != "" {
					return project.GetVersion()
				}
				if key == "project" && project.GetName() != "" {
					return project.GetName()
				}
				// Check current task
				if key == "current_task" && ctx.GetCurrentTask() != "" {
					return ctx.GetCurrentTask()
				}
			}
		}
		return ""
	}

	// 8. Check for $params.key or $params.namespace.key syntax for project parameters
	// Project parameters are loaded into ctx.Parameters by the engine,
	// but $params.key makes it explicit that we're accessing a project-level parameter
	if strings.HasPrefix(expr, "$params.") {
		if ctx != nil {
			key := expr[8:] // Remove "$params." prefix

			// First check ctx.Parameters (loaded from local project params)
			params := ctx.GetParameters()
			if params != nil {
				if value, exists := params[key]; exists {
					return value.AsString()
				}
			}

			// Namespaced parameters (e.g., $params.docker.registry) are loaded into
			// ctx.Parameters by setupTaskParameters with their full namespaced keys,
			// so they're already handled by the check above
		}
		return ""
	}

	// 9. Check for simple parameter lookup (fallback for complex expressions)
	if ctx != nil {
		params := ctx.GetParameters()
		vars := ctx.GetVariables()

		// Check for variables with $ prefix first (parameters and task-scoped variables)
		if strings.HasPrefix(expr, "$") {
			varName := expr[1:] // Remove the $ prefix

			// Check parameters (stored without $ prefix)
			if value, exists := params[varName]; exists {
				return value.AsString()
			}

			// Check captured variables (stored without $ prefix)
			if value, exists := vars[varName]; exists {
				return value
			}
		} else {
			// Check parameters (bare identifiers)
			if value, exists := params[expr]; exists {
				return value.AsString()
			}
			// Check captured variables (bare identifiers)
			if value, exists := vars[expr]; exists {
				return value
			}
		}
	}

	return ""
}

// Helper functions to safely get included settings/params from project context
// We need these because ProjectContext.GetIncludedSettings/Params might not be available
// in all implementations of the interface (to avoid circular dependencies)

func getIncludedSettings(project ProjectContext) map[string]string {
	// Use type assertion to access the concrete implementation
	type settingsProvider interface {
		GetIncludedSettings() map[string]string
	}
	if provider, ok := project.(settingsProvider); ok {
		return provider.GetIncludedSettings()
	}
	return nil
}

func getIncludedParams(project ProjectContext) interface{} {
	// Use type assertion to access the concrete implementation
	// This returns map[string]*ast.ProjectParameterStatement
	type paramsProvider interface {
		GetIncludedParams() interface{}
	}
	if provider, ok := project.(paramsProvider); ok {
		return provider.GetIncludedParams()
	}
	return nil
}

// Helper to extract default value from a parameter stored in interface{}
func getParamDefaultValue(paramInterface interface{}) (string, bool) {
	// Try to access as ProjectParameterStatement fields
	// We use reflection-style access since we can't import ast here
	if paramMap, ok := paramInterface.(interface {
		GetDefaultValue() string
		GetHasDefault() bool
	}); ok {
		if paramMap.GetHasDefault() {
			return paramMap.GetDefaultValue(), true
		}
	}

	// Fallback: try struct field access via reflection
	// The parameter is *ast.ProjectParameterStatement which has HasDefault and DefaultValue fields
	return "", false
}

// parseQuotedArguments parses comma-separated quoted arguments
func (i *Interpolator) parseQuotedArguments(argsStr string) []string {
	var args []string

	// Simple regex to match quoted strings
	quotedRe := regexp.MustCompile(`['"]([^'"]*?)['"]`)
	matches := quotedRe.FindAllStringSubmatch(argsStr, -1)

	for _, match := range matches {
		if len(match) > 1 {
			args = append(args, match[1])
		}
	}

	return args
}
