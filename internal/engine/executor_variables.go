package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/shell"
)

// Domain: Variable Operations Execution
// This file contains executors for:
// - Variable declarations (let, set)
// - Variable transformations (uppercase, lowercase, trim, concat, split, replace, etc.)
// - Variable capture (from expressions and shell commands)

// executeVariable executes variable operation statements
func (e *Engine) executeVariable(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	switch varStmt.Operation {
	case "let":
		return e.executeLetStatement(varStmt, ctx)
	case "set":
		return e.executeSetStatement(varStmt, ctx)
	case "transform":
		return e.executeTransformStatement(varStmt, ctx)
	case "capture":
		return e.executeCaptureStatement(varStmt, ctx)
	case "capture_shell":
		return e.executeCaptureShellStatement(varStmt, ctx)
	default:
		return fmt.Errorf("unknown variable operation: %s", varStmt.Operation)
	}
}

// executeLetStatement executes "let variable = value" statements
func (e *Engine) executeLetStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	value, err := e.evaluateExpression(varStmt.Value, ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %v", err)
	}

	// Interpolate the value if it contains braces (for builtin function calls)
	interpolatedValue := e.interpolateVariables(value, ctx)

	// Determine the variable name (namespace it if in an included snippet/task)
	varName := varStmt.Variable
	if ctx.CurrentNamespace != "" {
		// Namespace the variable (e.g., docker.image)
		varName = ctx.CurrentNamespace + "." + varStmt.Variable
	}

	// Store the variable in the context even in dry run for interpolation
	ctx.Variables[varName] = interpolatedValue

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set variable %s = %s\n", varName, interpolatedValue)
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "📝 Set variable %s = %s\n", varName, interpolatedValue)

	return nil
}

// executeSetStatement executes "set variable to value" statements
func (e *Engine) executeSetStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	value, err := e.evaluateExpression(varStmt.Value, ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %v", err)
	}

	// Interpolate the value if it contains braces (for builtin function calls)
	interpolatedValue := e.interpolateVariables(value, ctx)

	// Determine the variable name (namespace it if in an included snippet/task)
	varName := varStmt.Variable
	if ctx.CurrentNamespace != "" {
		// Namespace the variable (e.g., docker.image)
		varName = ctx.CurrentNamespace + "." + varStmt.Variable
	}

	// Store the variable in the context even in dry run for interpolation
	ctx.Variables[varName] = interpolatedValue

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set variable %s to %s\n", varName, interpolatedValue)
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "📝 Set variable %s to %s\n", varName, interpolatedValue)
	}

	return nil
}

// executeTransformStatement executes "transform variable with function args" statements
func (e *Engine) executeTransformStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	// Determine the variable name (namespace it if in an included snippet/task)
	varName := varStmt.Variable
	if ctx.CurrentNamespace != "" {
		varName = ctx.CurrentNamespace + "." + varStmt.Variable
	}

	// Get the current value of the variable
	currentValue, exists := ctx.Variables[varName]
	if !exists {
		return fmt.Errorf("variable '%s' not found", varName)
	}

	// Apply the transformation function
	newValue, err := e.applyTransformation(currentValue, varStmt.Function, varStmt.Arguments, ctx)
	if err != nil {
		return fmt.Errorf("transformation failed: %v", err)
	}

	// Update the variable with the transformed value even in dry run for interpolation
	ctx.Variables[varName] = newValue

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would transform variable %s with %s: %s -> %s\n",
			varName, varStmt.Function, currentValue, newValue)
		return nil
	}
	_, _ = fmt.Fprintf(e.output, "🔄 Transformed variable %s with %s: %s -> %s\n",
		varName, varStmt.Function, currentValue, newValue)

	return nil
}

// executeCaptureStatement executes "capture variable_name from expression" statements
func (e *Engine) executeCaptureStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	expression, err := e.evaluateExpression(varStmt.Value, ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %v", err)
	}

	// The expression is already evaluated, so we can use it directly as the value
	value := expression

	// Determine the variable name (namespace it if in an included snippet/task)
	varName := varStmt.Variable
	if ctx.CurrentNamespace != "" {
		varName = ctx.CurrentNamespace + "." + varStmt.Variable
	}

	// Store the captured value in the context
	ctx.Variables[varName] = value

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture %s: %s\n",
			varName, value)
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "📥 Captured %s: %s\n",
			varName, value)
	}

	return nil
}

// executeCaptureShellStatement executes "capture from shell command as $variable" statements
func (e *Engine) executeCaptureShellStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	// Extract the command from the literal expression
	literalExpr, ok := varStmt.Value.(*ast.LiteralExpression)
	if !ok {
		return fmt.Errorf("expected literal expression for shell capture command")
	}

	// Interpolate variables in the command
	command := e.interpolateVariables(literalExpr.Value, ctx)

	// Execute the shell command
	shellOpts := e.getPlatformShellConfig(ctx)
	result, err := shell.Execute(command, shellOpts)
	if err != nil {
		return fmt.Errorf("failed to capture from shell command '%s': %v", command, err)
	}

	// Determine the variable name (namespace it if in an included snippet/task)
	varName := varStmt.Variable
	if ctx.CurrentNamespace != "" {
		varName = ctx.CurrentNamespace + "." + varStmt.Variable
	}

	// Store the captured output (trimmed)
	value := strings.TrimSpace(result.Stdout)
	ctx.Variables[varName] = value

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture %s from shell: %s\n",
			varName, value)
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "📥 Captured %s from shell: %s\n",
			varName, value)
	}

	return nil
}

// applyTransformation applies a transformation function to a value
func (e *Engine) applyTransformation(value, function string, args []string, ctx *ExecutionContext) (string, error) {
	// Interpolate arguments
	interpolatedArgs := make([]string, len(args))
	for i, arg := range args {
		interpolatedArgs[i] = e.interpolateVariables(arg, ctx)
	}

	switch function {
	case "uppercase":
		return strings.ToUpper(value), nil
	case "lowercase":
		return strings.ToLower(value), nil
	case "trim":
		return strings.TrimSpace(value), nil
	case "concat":
		if len(interpolatedArgs) > 0 {
			return value + interpolatedArgs[0], nil
		}
		return value, nil
	case "split":
		if len(interpolatedArgs) > 0 {
			parts := strings.Split(value, interpolatedArgs[0])
			return strings.Join(parts, "\n"), nil // Return as newline-separated for display
		}
		return value, nil
	case "replace":
		if len(interpolatedArgs) >= 2 {
			return strings.ReplaceAll(value, interpolatedArgs[0], interpolatedArgs[1]), nil
		}
		return value, nil
	case "join":
		if len(interpolatedArgs) > 0 {
			// Assume value is a newline-separated list
			parts := strings.Split(value, "\n")
			return strings.Join(parts, interpolatedArgs[0]), nil
		}
		return value, nil
	case "length":
		return fmt.Sprintf("%d", len(value)), nil
	case "slice":
		if len(interpolatedArgs) >= 2 {
			start, err1 := strconv.Atoi(interpolatedArgs[0])
			end, err2 := strconv.Atoi(interpolatedArgs[1])
			if err1 == nil && err2 == nil && start >= 0 && end <= len(value) && start <= end {
				return value[start:end], nil
			}
		}
		return value, nil
	default:
		return "", fmt.Errorf("unknown transformation function: %s", function)
	}
}
