package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

func TestEngine_StrictVariableChecking(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		allowUndefined bool
		expectError    bool
		errorContains  string
	}{
		{
			name: "undefined variable in info statement - strict mode",
			input: `version: 2.0
task "test":
	info "Hello {$undefined_var}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$undefined_var}",
		},
		{
			name: "undefined variable in info statement - allow undefined",
			input: `version: 2.0
task "test":
	info "Hello {$undefined_var}"`,
			taskName:       "test",
			allowUndefined: true,
			expectError:    false,
		},
		{
			name: "undefined variable in shell command - strict mode",
			input: `version: 2.0
task "test":
	run "echo {$missing_var}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$missing_var}",
		},
		{
			name: "undefined variable in shell command - allow undefined",
			input: `version: 2.0
task "test":
	run "echo {$missing_var}"`,
			taskName:       "test",
			allowUndefined: true,
			expectError:    false,
		},
		{
			name: "multiple undefined variables - strict mode",
			input: `version: 2.0
task "test":
	info "Hello {$var1} and {$var2}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variables: {$var1}, {$var2}",
		},
		{
			name: "defined variable works in strict mode",
			input: `version: 2.0
task "test":
	let $name = "world"
	info "Hello {$name}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    false,
		},
		{
			name: "mixed defined and undefined variables - strict mode",
			input: `version: 2.0
task "test":
	let $defined = "exists"
	info "Defined: {$defined}, Undefined: {$undefined}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$undefined}",
		},
		{
			name: "undefined variable in step statement - strict mode",
			input: `version: 2.0
task "test":
	step "Processing {$item}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "in step statement: undefined variable: {$item}",
		},
		{
			name: "undefined variable in success statement - strict mode",
			input: `version: 2.0
task "test":
	success "Completed {$task_name}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "in success statement: undefined variable: {$task_name}",
		},
		{
			name: "undefined variable in warn statement - strict mode",
			input: `version: 2.0
task "test":
	warn "Warning: {$issue}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "in warn statement: undefined variable: {$issue}",
		},
		{
			name: "undefined variable in error statement - strict mode",
			input: `version: 2.0
task "test":
	error "Error: {$problem}"`,
			taskName:       "test",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "in error statement: undefined variable: {$problem}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)
			engine.SetAllowUndefinedVars(tt.allowUndefined)

			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, tt.taskName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestEngine_StrictVariablesInLoops(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		allowUndefined bool
		expectError    bool
		errorContains  string
	}{
		{
			name: "undefined variable in loop body - strict mode",
			input: `version: 2.0
task "test":
	for each $item in ["a", "b"]:
		info "Item: {$item}, Undefined: {$missing}"`,
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$missing}",
		},
		{
			name: "undefined variable in loop body - allow undefined",
			input: `version: 2.0
task "test":
	for each $item in ["a", "b"]:
		info "Item: {$item}, Undefined: {$missing}"`,
			allowUndefined: true,
			expectError:    false,
		},
		{
			name: "loop variable correctly defined",
			input: `version: 2.0
task "test":
	for each $item in ["a", "b"]:
		info "Processing: {$item}"`,
			allowUndefined: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)
			engine.SetAllowUndefinedVars(tt.allowUndefined)

			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, "test")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestEngine_StrictVariablesInConditionals(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		allowUndefined bool
		expectError    bool
		errorContains  string
	}{
		{
			name: "undefined variable in when condition - strict mode",
			input: `version: 2.0
task "test":
	when $undefined_var is "test":
		info "Should not reach here"`,
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$undefined_var}",
		},
		{
			name: "undefined variable in when body - strict mode",
			input: `version: 2.0
task "test":
	let $platform = "linux"
	when $platform is "linux":
		info "Platform: {$platform}, Version: {$undefined_version}"`,
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$undefined_version}",
		},
		{
			name: "undefined variable in otherwise body - strict mode",
			input: `version: 2.0
task "test":
	let $platform = "windows"
	when $platform is "linux":
		info "Linux detected"
	otherwise:
		info "Other platform: {$platform}, Arch: {$undefined_arch}"`,
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$undefined_arch}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)
			engine.SetAllowUndefinedVars(tt.allowUndefined)

			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, "test")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestEngine_StrictVariablesGlobalAccess(t *testing.T) {
	input := `version: 2.0
project "test" version "1.0":
	set app_name to "myapp"

task "test":
	info "App: {$globals.app_name}, Version: {$globals.version}, Undefined: {$globals.undefined}"`

	tests := []struct {
		name           string
		allowUndefined bool
		expectError    bool
		errorContains  string
	}{
		{
			name:           "undefined global variable - strict mode",
			allowUndefined: false,
			expectError:    true,
			errorContains:  "undefined variable: {$globals.undefined}",
		},
		{
			name:           "undefined global variable - allow undefined",
			allowUndefined: true,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)
			engine.SetAllowUndefinedVars(tt.allowUndefined)

			lexer := lexer.NewLexer(input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, "test")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestEngine_StrictVariablesErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "single undefined variable",
			input: `version: 2.0
task "test":
	info "Value: {$missing}"`,
			expectedError: "task 'test' failed: in info statement: undefined variable: {$missing}",
		},
		{
			name: "multiple undefined variables",
			input: `version: 2.0
task "test":
	info "Values: {$var1}, {$var2}, {$var3}"`,
			expectedError: "task 'test' failed: in info statement: undefined variables: {$var1}, {$var2}, {$var3}",
		},
		{
			name: "undefined variable in shell command",
			input: `version: 2.0
task "test":
	run "echo {$missing}"`,
			expectedError: "task 'test' failed: in shell command: undefined variable: {$missing}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)
			engine.SetAllowUndefinedVars(false) // Strict mode

			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, "test")

			if err == nil {
				t.Errorf("Expected error but got none")
			} else if err.Error() != tt.expectedError {
				t.Errorf("Expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}
