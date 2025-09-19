package engine

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// VariableOperation represents a single operation on a variable
type VariableOperation struct {
	Type string   // "without", "filtered", "sorted", etc.
	Args []string // operation arguments
}

// VariableOperationChain represents a chain of operations on a variable
type VariableOperationChain struct {
	Variable   string              // the base variable name (e.g., "$files", "$version")
	Operations []VariableOperation // chain of operations to apply
}

// parseVariableOperations parses a variable expression with operations
// Examples:
//   - "$version without prefix 'v'"
//   - "$files filtered by extension '.js' | sorted by name"
//   - "$path basename | without extension"
func (e *Engine) parseVariableOperations(expr string) (*VariableOperationChain, error) {
	// Split by pipe (|) to get operation chain
	parts := strings.Split(expr, "|")

	// First part should be the variable
	firstPart := strings.TrimSpace(parts[0])

	// Check if this looks like a variable operation
	if !strings.Contains(firstPart, " ") {
		// Simple variable reference, no operations
		return nil, nil
	}

	// Parse the first part to extract variable and first operation
	tokens := strings.Fields(firstPart)
	if len(tokens) < 2 {
		return nil, nil
	}

	// First token should be the variable
	variable := tokens[0]
	// Accept $variables or bare identifiers (loop variables)
	if !strings.HasPrefix(variable, "$") && !isValidIdentifier(variable) {
		return nil, nil
	}

	chain := &VariableOperationChain{
		Variable:   variable,
		Operations: []VariableOperation{},
	}

	// Parse first operation from remaining tokens
	if len(tokens) > 1 {
		op, err := e.parseOperation(tokens[1:])
		if err != nil {
			return nil, err
		}
		if op != nil {
			chain.Operations = append(chain.Operations, *op)
		}
	}

	// Parse remaining operations from pipe-separated parts
	for i := 1; i < len(parts); i++ {
		partTokens := strings.Fields(strings.TrimSpace(parts[i]))
		if len(partTokens) > 0 {
			op, err := e.parseOperation(partTokens)
			if err != nil {
				return nil, err
			}
			if op != nil {
				chain.Operations = append(chain.Operations, *op)
			}
		}
	}

	return chain, nil
}

// parseOperation parses a single operation from tokens
func (e *Engine) parseOperation(tokens []string) (*VariableOperation, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	opType := tokens[0]
	args := []string{}

	switch opType {
	case "without":
		// "without prefix 'v'" or "without suffix '.tmp'"
		if len(tokens) >= 3 {
			args = append(args, tokens[1]) // "prefix" or "suffix"
			// Join remaining tokens and remove quotes
			argValue := strings.Join(tokens[2:], " ")
			argValue = strings.Trim(argValue, `"'`)
			args = append(args, argValue)
		}

	case "filtered":
		// "filtered by extension '.js'" or "filtered by name 'test'"
		if len(tokens) >= 4 && tokens[1] == "by" {
			args = append(args, tokens[2]) // "extension", "name", etc.
			argValue := strings.Join(tokens[3:], " ")
			argValue = strings.Trim(argValue, `"'`)
			args = append(args, argValue)
		}

	case "split":
		// "split by ':'" or "split by ':' | first"
		if len(tokens) >= 3 && tokens[1] == "by" {
			argValue := strings.Join(tokens[2:], " ")
			argValue = strings.Trim(argValue, `"'`)
			args = append(args, argValue)
		}

	case "sorted":
		// "sorted by name" or "sorted by size"
		if len(tokens) >= 3 && tokens[1] == "by" {
			args = append(args, tokens[2])
		} else {
			// Default sort
			args = append(args, "name")
		}

	case "reversed", "unique", "first", "last", "basename", "dirname", "extension":
		// No arguments needed

	default:
		return nil, fmt.Errorf("unknown operation: %s", opType)
	}

	return &VariableOperation{
		Type: opType,
		Args: args,
	}, nil
}

// applyVariableOperations applies a chain of operations to a value
func (e *Engine) applyVariableOperations(value string, chain *VariableOperationChain, ctx *ExecutionContext) (string, error) {
	currentValue := value

	for _, op := range chain.Operations {
		newValue, err := e.applyVariableOperation(currentValue, op, ctx)
		if err != nil {
			return "", fmt.Errorf("operation '%s' failed: %v", op.Type, err)
		}
		currentValue = newValue
	}

	return currentValue, nil
}

// applyVariableOperation applies a single operation to a value
func (e *Engine) applyVariableOperation(value string, op VariableOperation, ctx *ExecutionContext) (string, error) {
	switch op.Type {
	case "without":
		return e.applyWithoutOperation(value, op.Args)

	case "filtered":
		return e.applyFilteredOperation(value, op.Args)

	case "sorted":
		return e.applySortedOperation(value, op.Args)

	case "reversed":
		return e.applyReversedOperation(value)

	case "unique":
		return e.applyUniqueOperation(value)

	case "first":
		return e.applyFirstOperation(value)

	case "last":
		return e.applyLastOperation(value)

	case "basename":
		return e.applyBasenameOperation(value)

	case "dirname":
		return e.applyDirnameOperation(value)

	case "extension":
		return e.applyExtensionOperation(value)

	case "split":
		return e.applySplitOperation(value, op.Args)

	default:
		return "", fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// String operations
func (e *Engine) applyWithoutOperation(value string, args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("without operation requires 2 arguments")
	}

	operation := args[0] // "prefix" or "suffix"
	target := args[1]    // the string to remove

	switch operation {
	case "prefix":
		if strings.HasPrefix(value, target) {
			return value[len(target):], nil
		}
		return value, nil

	case "suffix":
		if strings.HasSuffix(value, target) {
			return value[:len(value)-len(target)], nil
		}
		return value, nil

	default:
		return "", fmt.Errorf("unknown without operation: %s", operation)
	}
}

// Array operations (assuming space-separated values for now)
func (e *Engine) applyFilteredOperation(value string, args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("filtered operation requires 2 arguments")
	}

	filterType := args[0]  // "extension", "name", etc.
	filterValue := args[1] // the value to filter by

	// Split value into array
	items := strings.Fields(value)
	var filtered []string

	switch filterType {
	case "extension":
		for _, item := range items {
			if strings.HasSuffix(item, filterValue) {
				filtered = append(filtered, item)
			}
		}

	case "name":
		// Filter by name containing the value
		for _, item := range items {
			if strings.Contains(item, filterValue) {
				filtered = append(filtered, item)
			}
		}

	case "prefix":
		for _, item := range items {
			if strings.HasPrefix(item, filterValue) {
				filtered = append(filtered, item)
			}
		}

	case "suffix":
		for _, item := range items {
			if strings.HasSuffix(item, filterValue) {
				filtered = append(filtered, item)
			}
		}

	default:
		return "", fmt.Errorf("unknown filter type: %s", filterType)
	}

	return strings.Join(filtered, " "), nil
}

func (e *Engine) applySortedOperation(value string, args []string) (string, error) {
	items := strings.Fields(value)

	sortType := "name"
	if len(args) > 0 {
		sortType = args[0]
	}

	switch sortType {
	case "name":
		sort.Strings(items)

	case "length":
		sort.Slice(items, func(i, j int) bool {
			return len(items[i]) < len(items[j])
		})

	default:
		return "", fmt.Errorf("unknown sort type: %s", sortType)
	}

	return strings.Join(items, " "), nil
}

func (e *Engine) applyReversedOperation(value string) (string, error) {
	items := strings.Fields(value)

	// Reverse the slice
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	return strings.Join(items, " "), nil
}

func (e *Engine) applyUniqueOperation(value string) (string, error) {
	items := strings.Fields(value)
	seen := make(map[string]bool)
	var unique []string

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			unique = append(unique, item)
		}
	}

	return strings.Join(unique, " "), nil
}

func (e *Engine) applyFirstOperation(value string) (string, error) {
	items := strings.Fields(value)
	if len(items) > 0 {
		return items[0], nil
	}
	return "", nil
}

func (e *Engine) applyLastOperation(value string) (string, error) {
	items := strings.Fields(value)
	if len(items) > 0 {
		return items[len(items)-1], nil
	}
	return "", nil
}

// Path operations
func (e *Engine) applyBasenameOperation(value string) (string, error) {
	return filepath.Base(value), nil
}

func (e *Engine) applyDirnameOperation(value string) (string, error) {
	return filepath.Dir(value), nil
}

func (e *Engine) applyExtensionOperation(value string) (string, error) {
	ext := filepath.Ext(value)
	// Remove the leading dot
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}
	return ext, nil
}

// String split operation
func (e *Engine) applySplitOperation(value string, args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("split operation requires 1 argument (delimiter)")
	}

	delimiter := args[0]
	parts := strings.Split(value, delimiter)

	// Return space-separated parts (so they can be used with first, last, etc.)
	return strings.Join(parts, " "), nil
}

// isValidIdentifier checks if a string is a valid identifier (for loop variables)
func isValidIdentifier(s string) bool {
	// Simple check: starts with letter, contains only letters, numbers, underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_]*$`, s)
	return matched
}
