package engine

import (
	"testing"
)

func TestParseVariableOperations(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		expr     string
		expected *VariableOperationChain
		hasError bool
	}{
		{
			name: "simple without prefix",
			expr: "$version without prefix 'v'",
			expected: &VariableOperationChain{
				Variable: "$version",
				Operations: []VariableOperation{
					{Type: "without", Args: []string{"prefix", "v"}},
				},
			},
		},
		{
			name: "filtered by extension",
			expr: "$files filtered by extension '.js'",
			expected: &VariableOperationChain{
				Variable: "$files",
				Operations: []VariableOperation{
					{Type: "filtered", Args: []string{"extension", ".js"}},
				},
			},
		},
		{
			name: "chained operations",
			expr: "$files filtered by extension '.js' | sorted by name",
			expected: &VariableOperationChain{
				Variable: "$files",
				Operations: []VariableOperation{
					{Type: "filtered", Args: []string{"extension", ".js"}},
					{Type: "sorted", Args: []string{"name"}},
				},
			},
		},
		{
			name: "basename operation",
			expr: "$path basename",
			expected: &VariableOperationChain{
				Variable: "$path",
				Operations: []VariableOperation{
					{Type: "basename", Args: []string{}},
				},
			},
		},
		{
			name:     "simple variable - no operations",
			expr:     "$version",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.parseVariableOperations(tt.expr)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil result but got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
				return
			}

			if result.Variable != tt.expected.Variable {
				t.Errorf("expected variable %s but got %s", tt.expected.Variable, result.Variable)
			}

			if len(result.Operations) != len(tt.expected.Operations) {
				t.Errorf("expected %d operations but got %d", len(tt.expected.Operations), len(result.Operations))
				return
			}

			for i, op := range result.Operations {
				expectedOp := tt.expected.Operations[i]
				if op.Type != expectedOp.Type {
					t.Errorf("operation %d: expected type %s but got %s", i, expectedOp.Type, op.Type)
				}

				if len(op.Args) != len(expectedOp.Args) {
					t.Errorf("operation %d: expected %d args but got %d", i, len(expectedOp.Args), len(op.Args))
					continue
				}

				for j, arg := range op.Args {
					if arg != expectedOp.Args[j] {
						t.Errorf("operation %d arg %d: expected %s but got %s", i, j, expectedOp.Args[j], arg)
					}
				}
			}
		})
	}
}

func TestApplyVariableOperation(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name      string
		value     string
		operation VariableOperation
		expected  string
		hasError  bool
	}{
		{
			name:      "without prefix",
			value:     "v2.1.0",
			operation: VariableOperation{Type: "without", Args: []string{"prefix", "v"}},
			expected:  "2.1.0",
		},
		{
			name:      "without suffix",
			value:     "file.txt",
			operation: VariableOperation{Type: "without", Args: []string{"suffix", ".txt"}},
			expected:  "file",
		},
		{
			name:      "basename",
			value:     "/path/to/file.txt",
			operation: VariableOperation{Type: "basename", Args: []string{}},
			expected:  "file.txt",
		},
		{
			name:      "dirname",
			value:     "/path/to/file.txt",
			operation: VariableOperation{Type: "dirname", Args: []string{}},
			expected:  "/path/to",
		},
		{
			name:      "extension",
			value:     "file.txt",
			operation: VariableOperation{Type: "extension", Args: []string{}},
			expected:  "txt",
		},
		{
			name:      "filtered by extension",
			value:     "app.js test.js config.json",
			operation: VariableOperation{Type: "filtered", Args: []string{"extension", ".js"}},
			expected:  "app.js test.js",
		},
		{
			name:      "sorted by name",
			value:     "zebra apple banana",
			operation: VariableOperation{Type: "sorted", Args: []string{"name"}},
			expected:  "apple banana zebra",
		},
		{
			name:      "reversed",
			value:     "one two three",
			operation: VariableOperation{Type: "reversed", Args: []string{}},
			expected:  "three two one",
		},
		{
			name:      "unique",
			value:     "apple banana apple orange banana",
			operation: VariableOperation{Type: "unique", Args: []string{}},
			expected:  "apple banana orange",
		},
		{
			name:      "first",
			value:     "apple banana orange",
			operation: VariableOperation{Type: "first", Args: []string{}},
			expected:  "apple",
		},
		{
			name:      "last",
			value:     "apple banana orange",
			operation: VariableOperation{Type: "last", Args: []string{}},
			expected:  "orange",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.applyVariableOperation(tt.value, tt.operation, nil)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}

func TestApplyVariableOperations(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		value    string
		chain    *VariableOperationChain
		expected string
		hasError bool
	}{
		{
			name:  "chained filter and sort",
			value: "app.js test.js config.json package.json",
			chain: &VariableOperationChain{
				Variable: "$files",
				Operations: []VariableOperation{
					{Type: "filtered", Args: []string{"extension", ".js"}},
					{Type: "sorted", Args: []string{"name"}},
				},
			},
			expected: "app.js test.js",
		},
		{
			name:  "chained without prefix and suffix",
			value: "v2.1.0-beta",
			chain: &VariableOperationChain{
				Variable: "$version",
				Operations: []VariableOperation{
					{Type: "without", Args: []string{"prefix", "v"}},
					{Type: "without", Args: []string{"suffix", "-beta"}},
				},
			},
			expected: "2.1.0",
		},
		{
			name:  "path basename and extension",
			value: "/path/to/file.txt",
			chain: &VariableOperationChain{
				Variable: "$path",
				Operations: []VariableOperation{
					{Type: "basename", Args: []string{}},
					{Type: "without", Args: []string{"suffix", ".txt"}},
				},
			},
			expected: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.applyVariableOperations(tt.value, tt.chain, nil)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, result)
			}
		})
	}
}
