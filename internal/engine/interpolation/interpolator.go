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
	envVarRegex    *regexp.Regexp
	quotedArgRegex *regexp.Regexp
	paramArgRegex  *regexp.Regexp

	// Callback functions for complex resolution (provided by engine)
	resolveVariableOps func(expr string, ctx interface{}) string
	resolveBuiltinOps  func(funcName string, operations string, ctx interface{}) (string, error)
	resolveBuiltin     func(funcName string, args []string, ctx interface{}) (string, error)

	// Error collection during interpolation
	builtinErrors []string

	// Future: allowedFailures can be used to allow specific builtins to fail silently
	// Example: allowedFailures = map[string]bool{"optional_function": true}
	// Currently unused - all builtin failures cause task failures for predictability
	allowedFailures map[string]bool //nolint:unused
}

// NewInterpolator creates a new interpolator
func NewInterpolator() *Interpolator {
	return &Interpolator{
		envVarRegex:    regexp.MustCompile(`\$\{([^}]+)\}`),
		quotedArgRegex: regexp.MustCompile(`^([^(]+)\((.+)\)$`),
		paramArgRegex:  regexp.MustCompile(`^([^(]+)\(([^)]+)\)$`),
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

// SetResolveBuiltinCallback sets the callback for resolving builtin function calls
func (i *Interpolator) SetResolveBuiltinCallback(fn func(funcName string, args []string, ctx interface{}) (string, error)) {
	i.resolveBuiltin = fn
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
	GetParameters() interface{} // Returns map[string]*ast.ProjectParameterStatement
}

// Interpolate performs variable and environment variable interpolation
// This is the main entry point - delegates to InterpolateWithError and swallows errors
func (i *Interpolator) Interpolate(s string, ctx Context) string {
	result, _ := i.InterpolateWithError(s, ctx)
	return result
}

// InterpolateWithError performs variable and environment variable interpolation with error reporting
func (i *Interpolator) InterpolateWithError(message string, ctx Context) (string, error) {
	// Reset error collection
	i.builtinErrors = nil

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

	// Second pass: resolve {$var} Drun variables. Placeholders use balanced { } so
	// nested {$x} inside a ternary branch (e.g. {$a ? 'prefix-{$b}' : ''}) is one span.
	var undefinedVars []string
	result, err := i.expandDrunBraceInterpolations(message, ctx, &undefinedVars)
	if err != nil {
		return message, err
	}
	undefinedVars = dedupeStrings(undefinedVars)

	// If we found undefined variables in strict mode, return an error
	if len(undefinedVars) > 0 {
		if len(undefinedVars) == 1 {
			return result, fmt.Errorf("undefined variable: {%s}", undefinedVars[0])
		}
		return result, fmt.Errorf("undefined variables: {%s}", strings.Join(undefinedVars, "}, {"))
	}

	// Check for builtin errors (e.g., secret() calls that failed)
	if len(i.builtinErrors) > 0 {
		if len(i.builtinErrors) == 1 {
			return result, fmt.Errorf("%s", i.builtinErrors[0])
		}
		return result, fmt.Errorf("multiple errors: %s", strings.Join(i.builtinErrors, "; "))
	}

	return result, nil
}

const maxDrunInterpolationPasses = 64

func dedupeStrings(a []string) []string {
	if len(a) < 2 {
		return a
	}
	seen := make(map[string]struct{}, len(a))
	out := a[:0]
	for _, s := range a {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// findBalancedInterpolationSpan returns the first {...} span at or after start, using brace
// depth so nested {$var} inside the placeholder is part of the same span.
func findBalancedInterpolationSpan(s string, start int) (begin, end int, ok bool) {
	for i := start; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		begin = i
		depth := 0
		for j := i; j < len(s); j++ {
			switch s[j] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return begin, j + 1, true
				}
			}
		}
		return 0, 0, false
	}
	return 0, 0, false
}

// expandDrunBraceInterpolations repeatedly expands {$...} placeholders until none change
// (so a ternary can yield text that still contains {$x}).
func (i *Interpolator) expandDrunBraceInterpolations(message string, ctx Context, undefinedVars *[]string) (string, error) {
	for pass := 0; pass < maxDrunInterpolationPasses; pass++ {
		var b strings.Builder
		pos := 0
		changed := false
		for pos < len(message) {
			begin, end, ok := findBalancedInterpolationSpan(message, pos)
			if !ok {
				b.WriteString(message[pos:])
				break
			}
			b.WriteString(message[pos:begin])
			match := message[begin:end]
			content := message[begin+1 : end-1]
			// Legacy regex required [^}]+ — empty {} is literal text (e.g. in expanded param values).
			if len(content) == 0 {
				b.WriteString(match)
				pos = end
				continue
			}
			repl := i.resolveDrunBraceContent(content, match, ctx, undefinedVars)
			if repl != match {
				changed = true
			}
			b.WriteString(repl)
			pos = end
		}
		message = b.String()
		if !changed {
			break
		}
	}
	return message, nil
}

func (i *Interpolator) resolveDrunBraceContent(content, match string, ctx Context, undefinedVars *[]string) string {
	// Condition expressions join tokens with spaces (e.g. "if not {$node}:" → "not { $node }"), so brace
	// content can be " $node" instead of "$node". Trim so resolution matches unspaced {$var} forms.
	content = strings.TrimSpace(content)
	if content == "" {
		return match
	}

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
				*undefinedVars = append(*undefinedVars, baseVar)
				return match
			}
			// If base variable exists but expression failed, allow it (might be a function call or other valid expression)
		} else {
			// For simple variables, report as undefined
			*undefinedVars = append(*undefinedVars, content)
		}
		return match // Return original placeholder for now
	}

	// If allowing undefined variables, return the original placeholder
	return match
}
