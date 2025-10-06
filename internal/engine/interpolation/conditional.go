package interpolation

import (
	"strings"
)

// resolveTernaryExpression resolves ternary conditional expressions: $var ? 'true_val' : 'false_val'
// Returns (result, matched) where matched indicates if this was a valid ternary expression
func (i *Interpolator) resolveTernaryExpression(expr string, ctx Context) (string, bool) {
	// Find the ? and : positions
	questionPos := strings.Index(expr, "?")
	colonPos := strings.LastIndex(expr, ":")

	if questionPos == -1 || colonPos == -1 || questionPos >= colonPos {
		return "", false // Invalid ternary expression
	}

	// Extract parts
	condition := strings.TrimSpace(expr[:questionPos])
	trueValue := strings.TrimSpace(expr[questionPos+1 : colonPos])
	falseValue := strings.TrimSpace(expr[colonPos+1:])

	// Resolve the condition variable
	conditionValue, found := i.resolveSimpleVariableDirectly(condition, ctx)
	if !found {
		// Try resolving as expression
		conditionValue = i.resolveExpression(condition, ctx)
	}

	// Evaluate condition as boolean
	isTrue := i.isTruthy(conditionValue)

	// Return the appropriate value (unquote if needed)
	if isTrue {
		return i.unquoteString(trueValue), true
	}
	return i.unquoteString(falseValue), true
}

// resolveIfThenElse resolves if-then-else conditional expressions
// Supports:
//   - if $var then 'val1' else 'val2'
//   - if $var is 'value' then 'val1' else 'val2'
//   - if $var is not 'value' then 'val1' else 'val2'
//
// Returns (result, matched) where matched indicates if this was a valid if-then-else expression
func (i *Interpolator) resolveIfThenElse(expr string, ctx Context) (string, bool) {
	expr = strings.TrimSpace(expr)

	// Must start with "if "
	if !strings.HasPrefix(expr, "if ") {
		return "", false
	}

	// Find " then " and " else "
	thenPos := strings.Index(expr, " then ")
	elsePos := strings.LastIndex(expr, " else ")

	if thenPos == -1 || elsePos == -1 || thenPos >= elsePos {
		return "", false // Invalid if-then-else expression
	}

	// Extract parts
	conditionPart := strings.TrimSpace(expr[3:thenPos])       // Skip "if "
	trueValue := strings.TrimSpace(expr[thenPos+6 : elsePos]) // Skip " then "
	falseValue := strings.TrimSpace(expr[elsePos+6:])         // Skip " else "

	// Evaluate the condition
	isTrue := i.evaluateIfCondition(conditionPart, ctx)

	// Return the appropriate value (unquote if needed)
	if isTrue {
		return i.unquoteString(trueValue), true
	}
	return i.unquoteString(falseValue), true
}

// evaluateIfCondition evaluates the condition part of an if-then-else expression
func (i *Interpolator) evaluateIfCondition(condition string, ctx Context) bool {
	condition = strings.TrimSpace(condition)

	// Check for "is not" comparison
	if strings.Contains(condition, " is not ") {
		parts := strings.SplitN(condition, " is not ", 2)
		if len(parts) == 2 {
			leftValue := i.resolveVariableOrValue(strings.TrimSpace(parts[0]), ctx)
			rightValue := i.unquoteString(strings.TrimSpace(parts[1]))
			return leftValue != rightValue
		}
	}

	// Check for "is" comparison
	if strings.Contains(condition, " is ") {
		parts := strings.SplitN(condition, " is ", 2)
		if len(parts) == 2 {
			leftValue := i.resolveVariableOrValue(strings.TrimSpace(parts[0]), ctx)
			rightValue := i.unquoteString(strings.TrimSpace(parts[1]))
			return leftValue == rightValue
		}
	}

	// Simple boolean check - resolve the variable and check if it's truthy
	value := i.resolveVariableOrValue(condition, ctx)
	return i.isTruthy(value)
}

// resolveVariableOrValue resolves a variable or returns the literal value
func (i *Interpolator) resolveVariableOrValue(expr string, ctx Context) string {
	expr = strings.TrimSpace(expr)

	// If it's a variable (starts with $), resolve it
	if strings.HasPrefix(expr, "$") {
		if resolved, found := i.resolveSimpleVariableDirectly(expr, ctx); found {
			return resolved
		}
		// Try resolving as expression
		return i.resolveExpression(expr, ctx)
	}

	// Otherwise, return as literal value (unquoted if needed)
	return i.unquoteString(expr)
}

// isTruthy checks if a value should be considered true
func (i *Interpolator) isTruthy(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "yes" || value == "1" || value == "on"
}

// unquoteString removes single or double quotes from a string if present
func (i *Interpolator) unquoteString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
