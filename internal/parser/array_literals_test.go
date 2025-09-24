package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_ArrayLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple array literal",
			input:    `["item1", "item2", "item3"]`,
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "single item array",
			input:    `["single"]`,
			expected: []string{"single"},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []string{},
		},
		{
			name:     "array with spaces",
			input:    `["hello world", "test item"]`,
			expected: []string{"hello world", "test item"},
		},
		{
			name:     "array with special characters",
			input:    `["test@example.com", "user-123", "file.txt"]`,
			expected: []string{"test@example.com", "user-123", "file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := lexer.NewLexer(tt.input)
			parser := NewParser(lexer)

			// Parse as expression
			expr := parser.parseArrayLiteral()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			arrayLiteral, ok := expr.(*ast.ArrayLiteral)
			if !ok {
				t.Fatalf("Expected *ast.ArrayLiteral, got %T", expr)
			}

			if len(arrayLiteral.Elements) != len(tt.expected) {
				t.Fatalf("Expected %d elements, got %d", len(tt.expected), len(arrayLiteral.Elements))
			}

			for i, expectedElement := range tt.expected {
				literal, ok := arrayLiteral.Elements[i].(*ast.LiteralExpression)
				if !ok {
					t.Fatalf("Element %d is not a LiteralExpression, got %T", i, arrayLiteral.Elements[i])
				}

				if literal.Value != expectedElement {
					t.Errorf("Element %d: expected %q, got %q", i, expectedElement, literal.Value)
				}
			}
		})
	}
}

func TestParser_ForEachWithArrayLiterals(t *testing.T) {
	input := `version: 2.0

task "matrix test":
	for each $platform in ["linux", "darwin", "windows"]:
		info "Building for {$platform}"
		for each $arch in ["amd64", "arm64"]:
			step "Compiling {$platform}-{$arch}"
`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "matrix test" {
		t.Errorf("Expected task name 'matrix test', got %q", task.Name)
	}

	if len(task.Body) != 1 {
		t.Fatalf("Expected 1 statement in task body, got %d", len(task.Body))
	}

	// Check outer loop
	outerLoop, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("Expected LoopStatement, got %T", task.Body[0])
	}

	if outerLoop.Variable != "$platform" {
		t.Errorf("Expected loop variable '$platform', got %q", outerLoop.Variable)
	}

	if outerLoop.Iterable != `[linux, darwin, windows]` {
		t.Errorf("Expected array literal iterable, got %q", outerLoop.Iterable)
	}

	if len(outerLoop.Body) != 2 {
		t.Fatalf("Expected 2 statements in outer loop body, got %d", len(outerLoop.Body))
	}

	// Check inner loop
	innerLoop, ok := outerLoop.Body[1].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("Expected inner LoopStatement, got %T", outerLoop.Body[1])
	}

	if innerLoop.Variable != "$arch" {
		t.Errorf("Expected inner loop variable '$arch', got %q", innerLoop.Variable)
	}

	if innerLoop.Iterable != `[amd64, arm64]` {
		t.Errorf("Expected inner array literal iterable, got %q", innerLoop.Iterable)
	}
}

func TestParser_ProjectArraySettings(t *testing.T) {
	input := `version: 2.0

project "test" version "1.0":
	set platforms as list to ["linux", "darwin", "windows"]
	set environments as list to ["dev", "staging", "production"]
	set registry to "ghcr.io/company"

task "deploy":
	for each $platform in $globals.platforms:
		info "Deploying to {$platform}"
`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	// Check project
	if program.Project == nil {
		t.Fatalf("Expected project declaration")
	}

	if len(program.Project.Settings) != 3 {
		t.Fatalf("Expected 3 project settings, got %d", len(program.Project.Settings))
	}

	// Check array settings
	platformsSetting, ok := program.Project.Settings[0].(*ast.SetStatement)
	if !ok {
		t.Fatalf("Expected SetStatement, got %T", program.Project.Settings[0])
	}

	if platformsSetting.Key != "platforms" {
		t.Errorf("Expected setting key 'platforms', got %q", platformsSetting.Key)
	}

	// The value should be an ArrayLiteral expression
	arrayExpr, ok := platformsSetting.Value.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("Expected ArrayLiteral for platforms setting, got %T", platformsSetting.Value)
	}

	expectedPlatforms := []string{"linux", "darwin", "windows"}
	if len(arrayExpr.Elements) != len(expectedPlatforms) {
		t.Fatalf("Expected %d platforms, got %d", len(expectedPlatforms), len(arrayExpr.Elements))
	}

	for i, expected := range expectedPlatforms {
		literal, ok := arrayExpr.Elements[i].(*ast.LiteralExpression)
		if !ok {
			t.Fatalf("Platform %d is not a LiteralExpression, got %T", i, arrayExpr.Elements[i])
		}
		if literal.Value != expected {
			t.Errorf("Platform %d: expected %q, got %q", i, expected, literal.Value)
		}
	}

	// Check task uses project array
	if len(program.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	loop, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("Expected LoopStatement, got %T", task.Body[0])
	}

	if loop.Iterable != "$globals.platforms" {
		t.Errorf("Expected loop to iterate over '$globals.platforms', got %q", loop.Iterable)
	}
}

func TestParser_ParallelMatrixExecution(t *testing.T) {
	input := `version: 2.0

task "parallel matrix":
	for each $region in ["us-east", "eu-west"] in parallel:
		for each $service in ["api", "web", "worker"]:
			step "Deploy {$service} to {$region}"
`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	outerLoop, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("Expected LoopStatement, got %T", task.Body[0])
	}

	// Check parallel execution
	if !outerLoop.Parallel {
		t.Error("Expected outer loop to be parallel")
	}

	if outerLoop.Variable != "$region" {
		t.Errorf("Expected loop variable '$region', got %q", outerLoop.Variable)
	}

	// Check nested loop
	innerLoop, ok := outerLoop.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("Expected inner LoopStatement, got %T", outerLoop.Body[0])
	}

	if innerLoop.Parallel {
		t.Error("Expected inner loop to be sequential (not parallel)")
	}

	if innerLoop.Variable != "$service" {
		t.Errorf("Expected inner loop variable '$service', got %q", innerLoop.Variable)
	}
}
