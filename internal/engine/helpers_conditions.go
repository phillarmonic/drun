package engine

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/scm"
	"github.com/phillarmonic/drun/v2/internal/types"
)

// Domain: Condition Evaluation Helpers
// This file contains helper methods for evaluating conditions

var semanticVersionConditionPattern = regexp.MustCompile(`^(.+?)\s+is\s+(older|newer)\s+than\s+version\s+(.+)$`)
var fileComparisonConditionPattern = regexp.MustCompile(`^file\s+(.+?)\s+(not\s+)?matches\s+file\s+(.+?)\s*$`)

// evaluateFileComparisonCondition handles exact file-content comparisons:
//
//	file ./generated.json matches file ./vendored.json
//	file ./generated.json not matches file ./vendored.json
//
// Paths support interpolation and resolve with the same workdir rules as other
// filesystem conditions. Matching is byte-for-byte; no text normalization is
// performed.
func (e *Engine) evaluateFileComparisonCondition(condition string, ctx *ExecutionContext) (bool, bool, error) {
	match := fileComparisonConditionPattern.FindStringSubmatch(strings.TrimSpace(condition))
	if match == nil {
		return false, false, nil
	}

	left, err := e.resolveFileComparisonOperand(match[1], ctx)
	if err != nil {
		return false, true, fmt.Errorf("resolving left file path: %w", err)
	}
	right, err := e.resolveFileComparisonOperand(match[3], ctx)
	if err != nil {
		return false, true, fmt.Errorf("resolving right file path: %w", err)
	}

	// #nosec G304 -- file comparison explicitly reads the user-declared path.
	leftData, err := os.ReadFile(e.resolveFilesystemPath(left, ctx))
	if err != nil {
		return false, true, fmt.Errorf("reading left file %q: %w", left, err)
	}
	// #nosec G304 -- file comparison explicitly reads the user-declared path.
	rightData, err := os.ReadFile(e.resolveFilesystemPath(right, ctx))
	if err != nil {
		return false, true, fmt.Errorf("reading right file %q: %w", right, err)
	}

	equal := bytes.Equal(leftData, rightData)
	if strings.TrimSpace(match[2]) == "not" {
		return !equal, true, nil
	}
	return equal, true, nil
}

func (e *Engine) resolveFileComparisonOperand(operand string, ctx *ExecutionContext) (string, error) {
	value, err := e.interpolateVariablesWithError(strings.TrimSpace(operand), ctx)
	if err != nil {
		return "", err
	}
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if value == "" {
		return "", fmt.Errorf("path is empty")
	}
	return value, nil
}

// evaluateSemanticVersionCondition handles fluent stable-version comparisons:
//
//	$candidate is older than version "{$current}"
//	$candidate is newer than version "{$current}"
func (e *Engine) evaluateSemanticVersionCondition(condition string, ctx *ExecutionContext) (bool, bool, error) {
	match := semanticVersionConditionPattern.FindStringSubmatch(strings.TrimSpace(condition))
	if match == nil {
		return false, false, nil
	}

	left, err := e.resolveVersionConditionOperand(match[1], ctx)
	if err != nil {
		return false, true, fmt.Errorf("resolving left version: %w", err)
	}
	right, err := e.resolveVersionConditionOperand(match[3], ctx)
	if err != nil {
		return false, true, fmt.Errorf("resolving right version: %w", err)
	}
	// Git queries intentionally capture descriptive placeholders during dry-run.
	// They cannot participate in a real comparison, so keep the condition false
	// without weakening validation for actual execution values.
	if e.dryRun && (strings.HasPrefix(left, "[DRY RUN]") || strings.HasPrefix(right, "[DRY RUN]")) {
		return false, true, nil
	}
	leftVersion, err := scm.ParseVersion(left)
	if err != nil {
		return false, true, fmt.Errorf("left side of semantic version comparison: %w", err)
	}
	rightVersion, err := scm.ParseVersion(right)
	if err != nil {
		return false, true, fmt.Errorf("right side of semantic version comparison: %w", err)
	}

	comparison := leftVersion.Compare(rightVersion)
	if match[2] == "older" {
		return comparison < 0, true, nil
	}
	return comparison > 0, true, nil
}

func (e *Engine) resolveVersionConditionOperand(operand string, ctx *ExecutionContext) (string, error) {
	operand = strings.TrimSpace(operand)
	if strings.HasPrefix(operand, "$ ") {
		operand = strings.Replace(operand, "$ ", "$", 1)
	}
	if strings.HasPrefix(operand, "$") {
		operand = "{" + operand + "}"
	}
	value, err := e.interpolateVariablesWithError(operand, ctx)
	if err != nil {
		return "", err
	}
	return strings.Trim(strings.TrimSpace(value), `"'`), nil
}

// checkConditionForUndefinedVars checks if a condition contains undefined variables
func (e *Engine) checkConditionForUndefinedVars(condition string, ctx *ExecutionContext) error {
	condition = stripBraceInterpolations(condition)

	// For conditions, we only need to check simple variable references like "$var is value"
	// Complex expressions in conditions are handled by the condition evaluation itself
	// Match variables with alphanumeric, underscore, hyphen, and dot characters
	re := regexp.MustCompile(`\$[\w.-]+`)
	matches := re.FindAllString(condition, -1)

	var undefinedVars []string
	for _, match := range matches {
		varName := match[1:] // Remove $ prefix

		// Check if variable exists in parameters or variables
		if _, exists := ctx.Parameters[varName]; exists {
			continue
		}
		if _, exists := ctx.Variables[varName]; exists {
			continue
		}
		if _, exists := ctx.Variables[match]; exists { // Check with $ prefix
			continue
		}

		// Variable not found
		undefinedVars = append(undefinedVars, match)
	}

	if len(undefinedVars) > 0 {
		if len(undefinedVars) == 1 {
			return fmt.Errorf("undefined variable: {%s}", undefinedVars[0])
		}
		return fmt.Errorf("undefined variables: {%s}", strings.Join(undefinedVars, "}, {"))
	}

	return nil
}

func stripBraceInterpolations(s string) string {
	var b strings.Builder
	depth := 0

	for _, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			} else {
				b.WriteRune(r)
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}

// evaluateEnvCondition evaluates environment variable conditionals
// Handles: "VARNAME exists", "VARNAME is "value"", "VARNAME is not empty",
// "VARNAME exists and is not empty", "VARNAME exists and is "value""
func (e *Engine) evaluateEnvCondition(condition string, ctx *ExecutionContext) bool {
	condition = strings.TrimSpace(condition)

	// Extract variable name first
	var varName string
	var rest string

	// Find the first space to separate var name from the condition
	spaceIdx := strings.Index(condition, " ")
	if spaceIdx == -1 {
		// No space, just the variable name (shouldn't happen in valid syntax)
		varName = condition
		rest = ""
	} else {
		varName = condition[:spaceIdx]
		rest = strings.TrimSpace(condition[spaceIdx+1:])
	}

	// Handle compound conditions with "and" - must come after we have the varName
	if strings.Contains(rest, " and ") {
		parts := strings.SplitN(rest, " and ", 2)
		if len(parts) == 2 {
			// Evaluate first condition with varName
			left := e.evaluateEnvConditionWithVar(varName, strings.TrimSpace(parts[0]), ctx)
			if !left {
				return false // Short-circuit if first condition fails
			}
			// Evaluate second condition with same varName
			right := e.evaluateEnvConditionWithVar(varName, strings.TrimSpace(parts[1]), ctx)
			return right
		}
	}

	// Single condition evaluation
	return e.evaluateEnvConditionWithVar(varName, rest, ctx)
}

// evaluateEnvConditionWithVar evaluates a single env condition for a given variable
func (e *Engine) evaluateEnvConditionWithVar(varName string, condition string, ctx *ExecutionContext) bool {
	condition = strings.TrimSpace(condition)

	// Get the environment variable value
	envValue, envExists := os.LookupEnv(varName)

	// Handle "exists" check
	if condition == "exists" || condition == "" {
		return envExists
	}

	// Handle "is not empty" check
	if condition == "is not empty" {
		return envExists && strings.TrimSpace(envValue) != ""
	}

	// Handle "is empty" check
	if condition == "is empty" {
		return !envExists || strings.TrimSpace(envValue) == ""
	}

	// Handle "is "value"" check
	if strings.HasPrefix(condition, "is ") && !strings.HasPrefix(condition, "is not ") {
		expectedValue := strings.TrimSpace(condition[3:])
		// Remove quotes if present
		expectedValue = strings.Trim(expectedValue, "\"'")
		return envExists && envValue == expectedValue
	}

	// Handle "is not "value"" check
	if strings.HasPrefix(condition, "is not ") && !strings.HasPrefix(condition, "is not empty") {
		expectedValue := strings.TrimSpace(condition[7:])
		// Remove quotes if present
		expectedValue = strings.Trim(expectedValue, "\"'")
		return !envExists || envValue != expectedValue
	}

	// Default: check if environment variable exists
	return envExists
}

// evaluateCondition evaluates condition expressions
func (e *Engine) evaluateCondition(condition string, ctx *ExecutionContext) bool {
	// Simple condition evaluation
	// Handle various patterns like "variable is value", "variable is not empty", etc.

	// Handle environment variable conditionals
	if strings.HasPrefix(condition, "env ") {
		return e.evaluateEnvCondition(strings.TrimPrefix(condition, "env "), ctx)
	}

	if result, handled := e.evaluateFilesystemExistsCondition(condition, ctx); handled {
		return result
	}

	// Handle "folder/directory is not empty" pattern
	if strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is not empty", 2)
		if len(parts) >= 1 {
			left := strings.TrimSpace(parts[0])

			// Check if this is a folder/directory path check
			if strings.HasPrefix(left, "folder ") || strings.HasPrefix(left, "directory ") || strings.HasPrefix(left, "dir ") {
				var folderPath string
				if strings.HasPrefix(left, "folder ") {
					folderPath = strings.TrimSpace(left[7:]) // Remove "folder "
				} else if strings.HasPrefix(left, "directory ") {
					folderPath = strings.TrimSpace(left[10:]) // Remove "directory "
				} else if strings.HasPrefix(left, "dir ") {
					folderPath = strings.TrimSpace(left[4:]) // Remove "dir "
				}

				// Remove quotes if present
				folderPath = strings.Trim(folderPath, "\"'")

				// Interpolate variables in the path
				folderPath = e.interpolateVariables(folderPath, ctx)

				// Check if directory exists and is not empty
				if !e.dirExists(folderPath, ctx) {
					return false // Directory doesn't exist, treat as empty
				}

				isEmpty, err := e.isDirEmpty(folderPath, ctx)
				if err != nil {
					return false // Error checking, treat as empty
				}
				return !isEmpty // Return true if directory is NOT empty
			}
		}
	}

	// Handle "variable is not empty" pattern
	if strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is not empty", 2)
		if len(parts) >= 1 {
			left := strings.TrimSpace(parts[0])

			// Strip $ prefix if present
			paramName := left
			if strings.HasPrefix(left, "$") {
				paramName = left[1:]
			}

			// Try to get the value of the left side from parameters
			if value, exists := ctx.Parameters[paramName]; exists {
				valueStr := value.AsString()
				// For lists, check if the list is empty
				if value.Type == types.ListType {
					if list, err := value.AsList(); err == nil {
						return len(list) > 0
					}
				}
				// For other types, check if string representation is not empty
				return strings.TrimSpace(valueStr) != ""
			}

			// Try interpolating the variable
			interpolated := e.interpolateVariables("{"+left+"}", ctx)
			// If interpolation didn't change it, the variable doesn't exist (treat as empty)
			if interpolated == "{"+left+"}" {
				return false
			}
			return strings.TrimSpace(interpolated) != ""
		}
	}

	// Handle "variable is not value" pattern
	if strings.Contains(condition, " is not ") && !strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is not ", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// Handle "empty" keyword - treat as empty string
			if right == "empty" {
				right = ""
			}
			leftInterpolated := strings.Trim(e.interpolateVariables(left, ctx), "\"'")
			rightInterpolated := strings.Trim(e.interpolateVariables(right, ctx), "\"'")

			// Strip $ prefix if present
			paramName := left
			if strings.HasPrefix(left, "$") {
				paramName = left[1:]
			}

			// Try to get the value of the left side from parameters first
			if value, exists := ctx.Parameters[paramName]; exists {
				return value.AsString() != rightInterpolated
			}

			// Try to get the value from variables (let statements)
			if value, exists := ctx.Variables[paramName]; exists {
				return value != rightInterpolated
			}

			// Also try with the $ prefix (variables stored with $ prefix)
			if value, exists := ctx.Variables["$"+paramName]; exists {
				return value != rightInterpolated
			}

			// If not found in parameters or variables, compare as strings
			return leftInterpolated != rightInterpolated
		}
	}

	// Handle "folder/directory is empty" pattern
	if strings.Contains(condition, " is empty") && !strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is empty", 2)
		if len(parts) >= 1 {
			left := strings.TrimSpace(parts[0])

			// Check if this is a folder/directory path check
			if strings.HasPrefix(left, "folder ") || strings.HasPrefix(left, "directory ") || strings.HasPrefix(left, "dir ") {
				var folderPath string
				if strings.HasPrefix(left, "folder ") {
					folderPath = strings.TrimSpace(left[7:]) // Remove "folder "
				} else if strings.HasPrefix(left, "directory ") {
					folderPath = strings.TrimSpace(left[10:]) // Remove "directory "
				} else if strings.HasPrefix(left, "dir ") {
					folderPath = strings.TrimSpace(left[4:]) // Remove "dir "
				}

				// Remove quotes if present
				folderPath = strings.Trim(folderPath, "\"'")

				// Interpolate variables in the path
				folderPath = e.interpolateVariables(folderPath, ctx)

				// Check if directory exists and is empty
				if !e.dirExists(folderPath, ctx) {
					return true // Directory doesn't exist, treat as empty
				}

				isEmpty, err := e.isDirEmpty(folderPath, ctx)
				if err != nil {
					return true // Error checking, treat as empty
				}
				return isEmpty // Return true if directory IS empty
			}
		}
	}

	// Handle "variable is value" pattern
	if strings.Contains(condition, " is ") {
		parts := strings.SplitN(condition, " is ", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// Handle "empty" keyword - treat as empty string
			if right == "empty" {
				right = ""
			}
			leftInterpolated := strings.Trim(e.interpolateVariables(left, ctx), "\"'")
			rightInterpolated := strings.Trim(e.interpolateVariables(right, ctx), "\"'")

			// Strip $ prefix if present
			paramName := left
			if strings.HasPrefix(left, "$") {
				paramName = left[1:]
			}

			// Try to get the value of the left side from parameters first
			if value, exists := ctx.Parameters[paramName]; exists {
				return value.AsString() == rightInterpolated
			}

			// Try to get the value from variables (let statements)
			if value, exists := ctx.Variables[paramName]; exists {
				return value == rightInterpolated
			}

			// Also try with the $ prefix (variables stored with $ prefix)
			if value, exists := ctx.Variables["$"+paramName]; exists {
				return value == rightInterpolated
			}

			// If not found in parameters or variables, compare as strings
			return leftInterpolated == rightInterpolated
		}
	}

	// Interpolate variables in the condition for other cases
	interpolatedCondition := e.interpolateVariables(condition, ctx)
	trimmed := strings.TrimSpace(interpolatedCondition)

	// "if not <expr>:" parses as condition "not <expr>" (no leading "if"). After interpolation
	// this becomes e.g. "not false" / "not true" — negate the inner truthiness.
	if strings.HasPrefix(strings.ToLower(trimmed), "not ") {
		inner := strings.TrimSpace(trimmed[len("not "):])
		return !conditionStringTruthy(inner)
	}

	// Handle boolean values directly
	switch strings.ToLower(trimmed) {
	case "true":
		return true
	case "false":
		return false
	}

	// Default: treat non-empty strings as true
	return trimmed != ""
}

func (e *Engine) evaluateFilesystemExistsCondition(condition string, ctx *ExecutionContext) (bool, bool) {
	condition = strings.TrimSpace(condition)

	type subject struct {
		prefix string
		isDir  bool
	}

	subjects := []subject{
		{prefix: "file ", isDir: false},
		{prefix: "folder ", isDir: true},
		{prefix: "directory ", isDir: true},
		{prefix: "dir ", isDir: true},
	}

	for _, subject := range subjects {
		if !strings.HasPrefix(condition, subject.prefix) {
			continue
		}

		remainder := strings.TrimSpace(strings.TrimPrefix(condition, subject.prefix))
		switch {
		case strings.HasSuffix(remainder, " not exists"):
			path := strings.TrimSpace(strings.TrimSuffix(remainder, " not exists"))
			path = strings.Trim(e.interpolateVariables(path, ctx), "\"'")
			if subject.isDir {
				return !e.dirExists(path, ctx), true
			}
			return !e.fileExists(path, ctx), true
		case strings.HasSuffix(remainder, " exists"):
			path := strings.TrimSpace(strings.TrimSuffix(remainder, " exists"))
			path = strings.Trim(e.interpolateVariables(path, ctx), "\"'")
			if subject.isDir {
				return e.dirExists(path, ctx), true
			}
			return e.fileExists(path, ctx), true
		}
	}

	return false, false
}

// conditionStringTruthy matches the final evaluation rules for a fully interpolated fragment.
func conditionStringTruthy(s string) bool {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	default:
		return s != ""
	}
}
