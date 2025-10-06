package engine

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/shell"
)

// Domain: Expression Evaluation Helpers
// This file contains helper methods for evaluating expressions and builtin operations

// evaluateExpression evaluates an AST expression and returns its string value
func (e *Engine) evaluateExpression(expr ast.Expression, ctx *ExecutionContext) (string, error) {
	if expr == nil {
		return "", nil
	}

	switch ex := expr.(type) {
	case *ast.LiteralExpression:
		return ex.Value, nil

	case *ast.IdentifierExpression:
		// Look up variable value
		varName := ex.Value
		if strings.HasPrefix(varName, "$") {
			// Direct $variable reference
			if value, exists := ctx.Variables[varName]; exists {
				return value, nil
			}
			return "", fmt.Errorf("undefined variable: %s", varName)
		} else {
			// {variable} reference - look up without braces
			if value, exists := ctx.Variables[varName]; exists {
				return value, nil
			}
			return "", fmt.Errorf("undefined variable: %s", varName)
		}

	case *ast.ArrayLiteral:
		// Convert array literal to bracket-enclosed comma-separated string
		// This preserves the array format so loops can properly split it
		var elements []string
		for _, elem := range ex.Elements {
			val, err := e.evaluateExpression(elem, ctx)
			if err != nil {
				return "", err
			}
			elements = append(elements, val)
		}
		return "[" + strings.Join(elements, ",") + "]", nil

	case *ast.BinaryExpression:
		return e.evaluateBinaryExpression(ex, ctx)

	case *ast.FunctionCallExpression:
		return e.evaluateFunctionCall(ex, ctx)

	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// evaluateBinaryExpression evaluates binary operations like {a} - {b}
func (e *Engine) evaluateBinaryExpression(expr *ast.BinaryExpression, ctx *ExecutionContext) (string, error) {
	leftVal, err := e.evaluateExpression(expr.Left, ctx)
	if err != nil {
		return "", err
	}

	rightVal, err := e.evaluateExpression(expr.Right, ctx)
	if err != nil {
		return "", err
	}

	switch expr.Operator {
	case "-":
		// Try to parse as numbers for arithmetic
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			result := leftNum - rightNum
			// Return as integer if it's a whole number
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		return "", fmt.Errorf("cannot subtract non-numeric values: %s - %s", leftVal, rightVal)

	case "+":
		// Try to parse as numbers for arithmetic
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			result := leftNum + rightNum
			// Return as integer if it's a whole number
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		// If not numbers, concatenate as strings
		return leftVal + rightVal, nil

	case "*":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			result := leftNum * rightNum
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		return "", fmt.Errorf("cannot multiply non-numeric values: %s * %s", leftVal, rightVal)

	case "/":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if rightNum == 0 {
				return "", fmt.Errorf("division by zero")
			}
			result := leftNum / rightNum
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		return "", fmt.Errorf("cannot divide non-numeric values: %s / %s", leftVal, rightVal)

	case "==":
		if leftVal == rightVal {
			return "true", nil
		}
		return "false", nil

	case "!=":
		if leftVal != rightVal {
			return "true", nil
		}
		return "false", nil

	case "<":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum < rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal < rightVal {
			return "true", nil
		}
		return "false", nil

	case ">":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum > rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal > rightVal {
			return "true", nil
		}
		return "false", nil

	case "<=":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum <= rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal <= rightVal {
			return "true", nil
		}
		return "false", nil

	case ">=":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum >= rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal >= rightVal {
			return "true", nil
		}
		return "false", nil

	default:
		return "", fmt.Errorf("unsupported binary operator: %s", expr.Operator)
	}
}

// evaluateFunctionCall evaluates function calls like now(), current git branch
func (e *Engine) evaluateFunctionCall(expr *ast.FunctionCallExpression, ctx *ExecutionContext) (string, error) {
	switch expr.Function {
	case "now":
		return fmt.Sprintf("%d", time.Now().Unix()), nil

	default:
		// For other functions, treat them as shell commands or interpolation
		functionStr := expr.Function
		if len(expr.Arguments) > 0 {
			var args []string
			for _, arg := range expr.Arguments {
				argVal, err := e.evaluateExpression(arg, ctx)
				if err != nil {
					return "", err
				}
				args = append(args, argVal)
			}
			functionStr += "(" + strings.Join(args, ", ") + ")"
		}

		// Try to execute as shell command
		shellOpts := e.getPlatformShellConfig(ctx)
		result, err := shell.Execute(functionStr, shellOpts)
		if err != nil {
			return "", fmt.Errorf("failed to execute function '%s': %v", functionStr, err)
		}
		return strings.TrimSpace(result.Stdout), nil
	}
}

// parseBuiltinOperations parses operations for builtin functions (e.g., "replace '/' by '-'")
func (e *Engine) parseBuiltinOperations(operations string) (*VariableOperationChain, error) {
	// Split by | to handle multiple operations
	parts := strings.Split(operations, "|")
	var ops []VariableOperation

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse individual operation
		tokens := strings.Fields(part)
		if len(tokens) == 0 {
			continue
		}

		op, err := e.parseBuiltinOperation(tokens)
		if err != nil {
			return nil, err
		}
		if op != nil {
			ops = append(ops, *op)
		}
	}

	if len(ops) == 0 {
		return nil, nil
	}

	return &VariableOperationChain{
		Variable:   "", // Not used for builtin operations
		Operations: ops,
	}, nil
}

// parseBuiltinOperation parses a single builtin operation
func (e *Engine) parseBuiltinOperation(tokens []string) (*VariableOperation, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	opType := tokens[0]
	args := []string{}

	switch opType {
	case "replace":
		// "replace '/' by '-'" or "replace '/' with '-'"
		if len(tokens) >= 4 && (tokens[2] == "by" || tokens[2] == "with") {
			// Remove quotes from arguments
			from := strings.Trim(tokens[1], `"'`)
			to := strings.Trim(tokens[3], `"'`)
			args = append(args, from, to)
		}
	case "without":
		// "without prefix 'v'" or "without suffix '.tmp'"
		if len(tokens) >= 3 {
			args = append(args, tokens[1]) // "prefix" or "suffix"
			argValue := strings.Join(tokens[2:], " ")
			argValue = strings.Trim(argValue, `"'`)
			args = append(args, argValue)
		}
	case "uppercase", "lowercase", "trim":
		// No arguments needed
	default:
		return nil, fmt.Errorf("unknown builtin operation: %s", opType)
	}

	return &VariableOperation{
		Type: opType,
		Args: args,
	}, nil
}

// applyBuiltinOperations applies operations to a builtin function result
func (e *Engine) applyBuiltinOperations(value string, chain *VariableOperationChain, ctx *ExecutionContext) (string, error) {
	currentValue := value

	for _, op := range chain.Operations {
		newValue, err := e.applyBuiltinOperation(currentValue, op, ctx)
		if err != nil {
			return "", fmt.Errorf("builtin operation '%s' failed: %v", op.Type, err)
		}
		currentValue = newValue
	}

	return currentValue, nil
}

// applyBuiltinOperation applies a single operation to a builtin function result
func (e *Engine) applyBuiltinOperation(value string, op VariableOperation, ctx *ExecutionContext) (string, error) {
	switch op.Type {
	case "replace":
		if len(op.Args) >= 2 {
			return strings.ReplaceAll(value, op.Args[0], op.Args[1]), nil
		}
		return "", fmt.Errorf("replace operation requires 2 arguments")
	case "without":
		return e.applyWithoutOperation(value, op.Args)
	case "uppercase":
		return strings.ToUpper(value), nil
	case "lowercase":
		return strings.ToLower(value), nil
	case "trim":
		return strings.TrimSpace(value), nil
	default:
		return "", fmt.Errorf("unknown builtin operation type: %s", op.Type)
	}
}
