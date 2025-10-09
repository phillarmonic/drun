package engine

import (
	"fmt"
	"strings"
)

// Domain: Builtin Operations Helpers
// This file contains helper methods for parsing and applying builtin function operations
// e.g., {current git branch | replace '/' by '-'}

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
