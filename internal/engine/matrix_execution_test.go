package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

func TestEngine_ArrayLiteralExecution(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		expectedOutput []string
	}{
		{
			name: "simple array literal loop",
			input: `version: 2.0
task "test":
	for each $item in ["apple", "banana", "cherry"]:
		info "Processing {$item}"`,
			taskName: "test",
			expectedOutput: []string{
				"Processing apple",
				"Processing banana",
				"Processing cherry",
			},
		},
		{
			name: "empty array",
			input: `version: 2.0
task "test":
	for each $item in []:
		info "Processing {$item}"`,
			taskName: "test",
			expectedOutput: []string{
				"No items to process in loop",
			},
		},
		{
			name: "single item array",
			input: `version: 2.0
task "test":
	for each $item in ["single"]:
		info "Processing {$item}"`,
			taskName: "test",
			expectedOutput: []string{
				"Processing single",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)

			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, tt.taskName)
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
				}
			}
		})
	}
}

func TestEngine_MatrixExecution(t *testing.T) {
	input := `version: 2.0
task "matrix":
	for each $os in ["linux", "darwin"]:
		info "Building for OS: {$os}"
		for each $arch in ["amd64", "arm64"]:
			info "  Architecture: {$arch}"
			step "Compile {$os}-{$arch}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "matrix")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that all combinations are executed
	expectedCombinations := []string{
		"Building for OS: linux",
		"Architecture: amd64",
		"Compile linux-amd64",
		"Architecture: arm64",
		"Compile linux-arm64",
		"Building for OS: darwin",
		"Compile darwin-amd64",
		"Compile darwin-arm64",
	}

	for _, expected := range expectedCombinations {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
		}
	}
}

func TestEngine_ParallelMatrixExecution(t *testing.T) {
	input := `version: 2.0
task "parallel matrix":
	for each $region in ["us-east", "eu-west"] in parallel:
		info "Deploying to {$region}"
		for each $service in ["api", "web"]:
			step "Deploy {$service} to {$region}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "parallel matrix")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check parallel execution indicators
	if !strings.Contains(outputStr, "Would execute 2 items in parallel") {
		t.Error("Expected parallel execution indicator")
	}

	// Check that both regions are processed
	if !strings.Contains(outputStr, "$region = us-east") {
		t.Error("Expected us-east region processing")
	}
	if !strings.Contains(outputStr, "$region = eu-west") {
		t.Error("Expected eu-west region processing")
	}
}

func TestEngine_ProjectArraySettings(t *testing.T) {
	input := `version: 2.0
project "test" version "1.0":
	set platforms as list to ["linux", "darwin", "windows"]
	set environments as list to ["dev", "staging"]

task "deploy":
	for each $platform in platforms:
		info "Platform: {$platform}"
	for each $env in environments:
		info "Environment: {$env}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "deploy")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that project arrays are accessible
	expectedPlatforms := []string{"linux", "darwin", "windows"}
	for _, platform := range expectedPlatforms {
		expected := "Platform: " + platform
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
		}
	}

	expectedEnvironments := []string{"dev", "staging"}
	for _, env := range expectedEnvironments {
		expected := "Environment: " + env
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
		}
	}
}

func TestEngine_ArrayLiteralParsing(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple array",
			input:    `["item1", "item2", "item3"]`,
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []string{},
		},
		{
			name:     "single item",
			input:    `["single"]`,
			expected: []string{"single"},
		},
		{
			name:     "array with spaces in items",
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
			result := engine.parseArrayLiteralString(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("Missing item at index %d", i)
					continue
				}
				if result[i] != expected {
					t.Errorf("Item %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestEngine_ComplexMatrixScenario(t *testing.T) {
	input := `version: 2.0
project "complex" version "1.0":
	set databases as list to ["postgres", "mysql"]
	set versions as list to ["13", "14"]

task "complex matrix":
	info "Starting complex matrix"
	for each $db in databases:
		info "Testing database: {$db}"
		for each $version in versions:
			info "  Version: {$version}"
			for each $test_type in ["unit", "integration"]:
				step "Run {$test_type} tests on {$db}:{$version}"
	success "Matrix completed"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "complex matrix")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check all combinations are executed (2 databases × 2 versions × 2 test types = 8 combinations)
	expectedCombinations := []string{
		"Run unit tests on postgres:13",
		"Run integration tests on postgres:13",
		"Run unit tests on postgres:14",
		"Run integration tests on postgres:14",
		"Run unit tests on mysql:13",
		"Run integration tests on mysql:13",
		"Run unit tests on mysql:14",
		"Run integration tests on mysql:14",
	}

	for _, expected := range expectedCombinations {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
		}
	}
}
