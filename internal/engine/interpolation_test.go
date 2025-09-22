package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
	"github.com/phillarmonic/drun/internal/types"
)

func TestVariableInterpolation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		params         map[string]string
		expectedOutput []string // strings that should be present in output
		shouldFail     bool
	}{
		{
			name: "basic parameter interpolation with $ prefix",
			input: `version: 2.0

task "greet":
  requires $name
  given $title defaults to "friend"
  
  info "Hello, {$title} {$name}! Nice to meet you."
  step "Processing greeting for {$name}"
  success "Greeting completed for {$title} {$name}!"`,
			taskName: "greet",
			params:   map[string]string{"name": "Andy"},
			expectedOutput: []string{
				"Hello, friend Andy! Nice to meet you.",
				"Processing greeting for Andy",
				"Greeting completed for friend Andy!",
			},
		},
		{
			name: "parameter interpolation with custom values",
			input: `version: 2.0

task "greet":
  requires $name
  given $title defaults to "friend"
  
  info "Hello, {$title} {$name}! Nice to meet you."
  step "Processing greeting for {$name}"
  success "Greeting completed for {$title} {$name}!"`,
			taskName: "greet",
			params:   map[string]string{"name": "Bob", "title": "buddy"},
			expectedOutput: []string{
				"Hello, buddy Bob! Nice to meet you.",
				"Processing greeting for Bob",
				"Greeting completed for buddy Bob!",
			},
		},
		{
			name: "multiple parameter types interpolation",
			input: `version: 2.0

task "deploy":
  requires $environment from ["dev", "staging", "production"]
  given $app_version defaults to "latest"
  
  step "Deploying version {$app_version} to {$environment}"
  info "Environment: {$environment}"
  info "Version: {$app_version}"
  success "Deployment to {$environment} completed!"`,
			taskName: "deploy",
			params:   map[string]string{"environment": "staging", "app_version": "v1.2.3"},
			expectedOutput: []string{
				"Deploying version v1.2.3 to staging",
				"Environment: staging",
				"Version: v1.2.3",
				"Deployment to staging completed!",
			},
		},
		{
			name: "interpolation with default values only",
			input: `version: 2.0

task "backup":
  requires $source_path
  given $backup_name defaults to "backup-2024-01-01"
  
  step "Creating backup: {$backup_name}"
  info "Source: {$source_path}"
  info "Backup: {$backup_name}"
  success "Backup created: {$backup_name}"`,
			taskName: "backup",
			params:   map[string]string{"source_path": "/home/user/data"},
			expectedOutput: []string{
				"Creating backup: backup-2024-01-01",
				"Source: /home/user/data",
				"Backup: backup-2024-01-01",
				"Backup created: backup-2024-01-01",
			},
		},
		{
			name: "mixed interpolation patterns",
			input: `version: 2.0

task "mixed":
  requires $name
  given $prefix defaults to "Mr."
  given $suffix defaults to "Jr."
  
  info "Full name: {$prefix} {$name} {$suffix}"
  step "Processing {$name} with prefix {$prefix}"
  success "Done with {$prefix} {$name} {$suffix}!"`,
			taskName: "mixed",
			params:   map[string]string{"name": "Smith", "suffix": "Sr."},
			expectedOutput: []string{
				"Full name: Mr. Smith Sr.",
				"Processing Smith with prefix Mr.",
				"Done with Mr. Smith Sr.!",
			},
		},
		{
			name: "no interpolation needed",
			input: `version: 2.0

task "simple":
  info "This is a simple message"
  step "No variables here"
  success "All done!"`,
			taskName: "simple",
			params:   map[string]string{},
			expectedOutput: []string{
				"This is a simple message",
				"No variables here",
				"All done!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			if program == nil {
				t.Fatalf("ParseProgram() returned nil")
			}

			// Create engine with buffer to capture output
			var output bytes.Buffer
			engine := NewEngine(&output)

			// Execute the task
			err := engine.ExecuteWithParams(program, tt.taskName, tt.params)

			if tt.shouldFail && err == nil {
				t.Fatalf("Expected execution to fail, but it succeeded")
			}

			if !tt.shouldFail && err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			// Check output contains expected strings
			outputStr := output.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}
		})
	}
}

func TestVariableInterpolationEdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		taskName         string
		params           map[string]string
		expectedOutput   []string
		unexpectedOutput []string // strings that should NOT be present in output
	}{
		{
			name: "undefined variable should remain as placeholder",
			input: `version: 2.0

task "undefined":
  requires $name
  
  info "Hello {$name}, undefined: {$undefined_var}"`,
			taskName: "undefined",
			params:   map[string]string{"name": "Alice"},
			expectedOutput: []string{
				"Hello Alice, undefined: {$undefined_var}",
			},
		},
		{
			name: "empty parameter value",
			input: `version: 2.0

task "empty":
  requires $name
  given $title defaults to ""
  
  info "Name: '{$name}', Title: '{$title}'"`,
			taskName: "empty",
			params:   map[string]string{"name": ""},
			expectedOutput: []string{
				"Name: '', Title: ''",
			},
		},
		{
			name: "special characters in parameter values",
			input: `version: 2.0

task "special":
  requires $message
  
  info "Message: {$message}"`,
			taskName: "special",
			params:   map[string]string{"message": "Hello! @#$%^&*(){}[]|\\:;\"'<>,.?/~`"},
			expectedOutput: []string{
				"Message: Hello! @#$%^&*(){}[]|\\:;\"'<>,.?/~`",
			},
		},
		{
			name: "multiple same variable in one string",
			input: `version: 2.0

task "repeat":
  requires $word
  
  info "{$word} {$word} {$word}!"`,
			taskName: "repeat",
			params:   map[string]string{"word": "echo"},
			expectedOutput: []string{
				"echo echo echo!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			// Create engine with buffer to capture output
			var output bytes.Buffer
			engine := NewEngine(&output)

			// Execute the task
			err := engine.ExecuteWithParams(program, tt.taskName, tt.params)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			// Check output contains expected strings
			outputStr := output.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}

			// Check output does NOT contain unexpected strings
			for _, unexpected := range tt.unexpectedOutput {
				if strings.Contains(outputStr, unexpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", unexpected, outputStr)
				}
			}
		})
	}
}

func TestInterpolateVariablesFunction(t *testing.T) {
	// Test the interpolateVariables function directly
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		message  string
		params   map[string]string
		vars     map[string]string
		expected string
	}{
		{
			name:     "simple parameter interpolation",
			message:  "Hello {$name}!",
			params:   map[string]string{"name": "World"},
			expected: "Hello World!",
		},
		{
			name:     "multiple parameters",
			message:  "{$greeting} {$name}, how are you?",
			params:   map[string]string{"greeting": "Hi", "name": "Alice"},
			expected: "Hi Alice, how are you?",
		},
		{
			name:     "no interpolation needed",
			message:  "Static message",
			params:   map[string]string{},
			expected: "Static message",
		},
		{
			name:     "undefined variable",
			message:  "Hello {$undefined}!",
			params:   map[string]string{},
			expected: "Hello {$undefined}!",
		},
		{
			name:     "mixed defined and undefined",
			message:  "Hello {$name}, {$undefined} variable",
			params:   map[string]string{"name": "Bob"},
			expected: "Hello Bob, {$undefined} variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create execution context
			ctx := &ExecutionContext{
				Parameters: make(map[string]*types.Value),
				Variables:  make(map[string]string),
			}

			// Add parameters to context
			for name, value := range tt.params {
				typedValue, err := types.NewValue(types.StringType, value)
				if err != nil {
					t.Fatalf("Failed to create typed value: %v", err)
				}
				ctx.Parameters[name] = typedValue
			}

			// Add variables to context
			for name, value := range tt.vars {
				ctx.Variables[name] = value
			}

			// Test interpolation
			result := engine.interpolateVariables(tt.message, ctx)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEngine_EmptyKeywordConditions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		taskName    string
		params      map[string]string
		expected    []string // Expected output messages
		notExpected []string // Messages that should NOT appear
	}{
		{
			name: "empty list parameter with 'is empty' condition",
			input: `version: 2.0

task "test":
  given $features as list defaults to empty
  
  if $features is empty:
    info "Features list is empty"
  
  if $features is not empty:
    info "Features list has items"`,
			taskName:    "test",
			params:      map[string]string{},
			expected:    []string{"Features list is empty"},
			notExpected: []string{"Features list has items"},
		},
		{
			name: "non-empty list parameter with 'is not empty' condition",
			input: `version: 2.0

task "test":
  given $features as list defaults to empty
  
  if $features is empty:
    info "Features list is empty"
  
  if $features is not empty:
    info "Features list has items: {$features}"`,
			taskName:    "test",
			params:      map[string]string{"features": "auth,payments"},
			expected:    []string{"Features list has items: auth,payments"},
			notExpected: []string{"Features list is empty"},
		},
		{
			name: "empty string parameter with 'is empty' condition",
			input: `version: 2.0

task "test":
  given $name defaults to empty
  
  if $name is empty:
    info "Name is empty"
  
  if $name is not empty:
    info "Name is: {$name}"`,
			taskName:    "test",
			params:      map[string]string{},
			expected:    []string{"Name is empty"},
			notExpected: []string{"Name is:"},
		},
		{
			name: "non-empty string parameter with 'is not empty' condition",
			input: `version: 2.0

task "test":
  given $name defaults to empty
  
  if $name is empty:
    info "Name is empty"
  
  if $name is not empty:
    info "Name is: {$name}"`,
			taskName:    "test",
			params:      map[string]string{"name": "Alice"},
			expected:    []string{"Name is: Alice"},
			notExpected: []string{"Name is empty"},
		},
		{
			name: "empty keyword equivalent to empty string",
			input: `version: 2.0

task "test":
  given $value1 defaults to empty
  given $value2 defaults to ""
  
  if $value1 is empty:
    info "Value1 is empty using empty keyword"
  
  if $value2 is empty:
    info "Value2 is empty using empty string"
    
  if $value1 is "":
    info "Value1 equals empty string"
    
  if $value2 is "":
    info "Value2 equals empty string"`,
			taskName: "test",
			params:   map[string]string{},
			expected: []string{
				"Value1 is empty using empty keyword",
				"Value2 is empty using empty string",
				"Value1 equals empty string",
				"Value2 equals empty string",
			},
			notExpected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var output strings.Builder
			engine := NewEngine(&output)

			// Parse and execute
			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.ExecuteWithParams(program, tt.taskName, tt.params)
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()

			// Check expected messages are present
			for _, expected := range tt.expected {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}

			// Check that unexpected messages are not present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestEngine_FolderEmptyConditions(t *testing.T) {
	// Create temporary directories for testing
	tempDir := t.TempDir()
	emptyDir := filepath.Join(tempDir, "empty")
	nonEmptyDir := filepath.Join(tempDir, "nonempty")

	// Create empty directory
	err := os.MkdirAll(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	// Create non-empty directory with a file
	err = os.MkdirAll(nonEmptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create non-empty directory: %v", err)
	}
	testFile := filepath.Join(nonEmptyDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		taskName    string
		params      map[string]string
		expected    []string
		notExpected []string
	}{
		{
			name: "folder is empty - empty directory",
			input: fmt.Sprintf(`version: 2.0

task "test":
  if folder "%s" is empty:
    info "Folder is empty"
  
  if folder "%s" is not empty:
    info "Folder is not empty"`, emptyDir, emptyDir),
			taskName:    "test",
			params:      map[string]string{},
			expected:    []string{"Folder is empty"},
			notExpected: []string{"Folder is not empty"},
		},
		{
			name: "folder is not empty - non-empty directory",
			input: fmt.Sprintf(`version: 2.0

task "test":
  if folder "%s" is empty:
    info "Folder is empty"
  
  if folder "%s" is not empty:
    info "Folder is not empty"`, nonEmptyDir, nonEmptyDir),
			taskName:    "test",
			params:      map[string]string{},
			expected:    []string{"Folder is not empty"},
			notExpected: []string{"Folder is empty"},
		},
		{
			name: "directory keyword works",
			input: fmt.Sprintf(`version: 2.0

task "test":
  if directory "%s" is empty:
    info "Directory is empty"
  
  if directory "%s" is not empty:
    info "Directory is not empty"`, emptyDir, nonEmptyDir),
			taskName: "test",
			params:   map[string]string{},
			expected: []string{"Directory is empty", "Directory is not empty"},
		},
		{
			name: "dir keyword works",
			input: fmt.Sprintf(`version: 2.0

task "test":
  if dir "%s" is empty:
    info "Dir is empty"
  
  if dir "%s" is not empty:
    info "Dir is not empty"`, emptyDir, nonEmptyDir),
			taskName: "test",
			params:   map[string]string{},
			expected: []string{"Dir is empty", "Dir is not empty"},
		},
		{
			name: "non-existent folder is treated as empty",
			input: `version: 2.0

task "test":
  if folder "/tmp/non-existent-folder-xyz123" is empty:
    info "Non-existent folder is empty"
  
  if folder "/tmp/non-existent-folder-xyz123" is not empty:
    info "Non-existent folder is not empty"`,
			taskName:    "test",
			params:      map[string]string{},
			expected:    []string{"Non-existent folder is empty"},
			notExpected: []string{"Non-existent folder is not empty"},
		},
		{
			name: "folder path with variable interpolation",
			input: `version: 2.0

task "test":
  given $folder_path defaults to empty
  
  if folder "{$folder_path}" is empty:
    info "Variable folder is empty"`,
			taskName: "test",
			params:   map[string]string{"folder_path": emptyDir},
			expected: []string{"Variable folder is empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var output strings.Builder
			engine := NewEngine(&output)

			// Parse and execute
			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.ExecuteWithParams(program, tt.taskName, tt.params)
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()

			// Check expected messages are present
			for _, expected := range tt.expected {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}

			// Check that unexpected messages are not present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}
