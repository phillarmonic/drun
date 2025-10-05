package interpolation

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/internal/types"
)

// Interpolator handles string interpolation for variables and expressions
type Interpolator struct {
	allowUndefined bool

	// Cached regex patterns for performance
	interpolationRegex *regexp.Regexp
	envVarRegex        *regexp.Regexp
	quotedArgRegex     *regexp.Regexp
	paramArgRegex      *regexp.Regexp

	// Callback functions for complex resolution (provided by engine)
	resolveVariableOps func(expr string, ctx interface{}) string
	resolveBuiltinOps  func(funcName string, operations string, ctx interface{}) (string, error)
}

// NewInterpolator creates a new interpolator
func NewInterpolator() *Interpolator {
	return &Interpolator{
		interpolationRegex: regexp.MustCompile(`\{([^}]+)\}`),
		envVarRegex:        regexp.MustCompile(`\$\{([^}]+)\}`),
		quotedArgRegex:     regexp.MustCompile(`^([^(]+)\((.+)\)$`),
		paramArgRegex:      regexp.MustCompile(`^([^(]+)\(([^)]+)\)$`),
	}
}

// SetAllowUndefined sets whether undefined variables are allowed
func (i *Interpolator) SetAllowUndefined(allow bool) {
	i.allowUndefined = allow
}

// IsStrictMode returns whether strict variable checking is enabled (undefined variables cause errors)
func (i *Interpolator) IsStrictMode() bool {
	return !i.allowUndefined
}

// SetResolveVariableOpsCallback sets the callback for resolving variable operations
func (i *Interpolator) SetResolveVariableOpsCallback(fn func(expr string, ctx interface{}) string) {
	i.resolveVariableOps = fn
}

// SetResolveBuiltinOpsCallback sets the callback for resolving builtin operations
func (i *Interpolator) SetResolveBuiltinOpsCallback(fn func(funcName string, operations string, ctx interface{}) (string, error)) {
	i.resolveBuiltinOps = fn
}

// Context represents the interpolation context (generic interface to avoid circular dependency)
type Context interface {
	GetParameters() map[string]*types.Value
	GetVariables() map[string]string
	GetProject() ProjectContext
	GetCurrentFile() string
	GetCurrentTask() string
}

// ProjectContext provides project-level settings
type ProjectContext interface {
	GetName() string
	GetVersion() string
	GetSettings() map[string]string
}

// Interpolate performs variable and environment variable interpolation
// This is the main entry point - delegates to InterpolateWithError and swallows errors
func (i *Interpolator) Interpolate(s string, ctx Context) string {
	result, _ := i.InterpolateWithError(s, ctx)
	return result
}

// InterpolateWithError performs variable and environment variable interpolation with error reporting
func (i *Interpolator) InterpolateWithError(message string, ctx Context) (string, error) {
	// First pass: resolve ${VAR} environment variables (shell-style)
	// Quick check: if there are no ${...} patterns, skip this phase
	if strings.Contains(message, "${") {
		var envUndefinedVars []string
		message = i.envVarRegex.ReplaceAllStringFunc(message, func(match string) string {
			// Extract content (remove ${ and })
			content := match[2 : len(match)-1]

			// Check if it has a default value (:-syntax)
			if strings.Contains(content, ":-") {
				parts := strings.SplitN(content, ":-", 2)
				varName := strings.TrimSpace(parts[0])
				defaultValue := strings.TrimSpace(parts[1])

				// Try to get the environment variable
				if value, exists := os.LookupEnv(varName); exists {
					return value
				}
				// Return default if not found
				return defaultValue
			}

			// No default value - must exist or fail
			if value, exists := os.LookupEnv(content); exists {
				return value
			}

			// Variable doesn't exist and no default provided
			if !i.allowUndefined {
				envUndefinedVars = append(envUndefinedVars, content)
			}
			return match // Keep original if not found
		})

		// If we found undefined env vars, return error now
		if len(envUndefinedVars) > 0 {
			if len(envUndefinedVars) == 1 {
				return message, fmt.Errorf("undefined environment variable: ${%s}", envUndefinedVars[0])
			}
			return message, fmt.Errorf("undefined environment variables: ${%s}", strings.Join(envUndefinedVars, "}, ${"))
		}
	}

	// Second pass: resolve {$var} Drun variables
	var undefinedVars []string

	// Use cached regex for better performance
	result := i.interpolationRegex.ReplaceAllStringFunc(message, func(match string) string {
		// Extract content (remove { and })
		content := match[1 : len(match)-1]

		// Try to resolve simple variables first (most common case)
		if resolved, found := i.resolveSimpleVariableDirectly(content, ctx); found {
			return resolved
		}

		// Check for conditional expressions first (they can return empty strings)
		// Ternary: "$var ? 'true_val' : 'false_val'"
		if strings.Contains(content, "?") && strings.Contains(content, ":") {
			if result, matched := i.resolveTernaryExpression(content, ctx); matched {
				return result // Accept even if empty
			}
		}

		// If-then-else: "if $var then 'val1' else 'val2'"
		if strings.HasPrefix(strings.TrimSpace(content), "if ") && strings.Contains(content, " then ") && strings.Contains(content, " else ") {
			if result, matched := i.resolveIfThenElse(content, ctx); matched {
				return result // Accept even if empty
			}
		}

		// Fall back to complex expression resolution
		if resolved := i.resolveExpression(content, ctx); resolved != "" {
			return resolved
		}

		// If nothing worked, check if we should be strict about undefined variables
		if !i.allowUndefined {
			// For complex expressions, check if the base variable exists
			if i.isComplexExpression(content) {
				baseVar := i.extractBaseVariable(content)
				if baseVar != "" && !i.variableExists(baseVar, ctx) {
					undefinedVars = append(undefinedVars, baseVar)
					return match
				}
				// If base variable exists but expression failed, allow it (might be a function call or other valid expression)
			} else {
				// For simple variables, report as undefined
				undefinedVars = append(undefinedVars, content)
			}
			return match // Return original placeholder for now
		}

		// If allowing undefined variables, return the original placeholder
		return match
	})

	// If we found undefined variables in strict mode, return an error
	if len(undefinedVars) > 0 {
		if len(undefinedVars) == 1 {
			return result, fmt.Errorf("undefined variable: {%s}", undefinedVars[0])
		}
		return result, fmt.Errorf("undefined variables: {%s}", strings.Join(undefinedVars, "}, {"))
	}

	return result, nil
}
