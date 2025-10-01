package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestRequiresWithDefaultValue(t *testing.T) {
	input := `
version: 2.0

task "build":
	requires $cache from ["yes", "no"] defaults to "no"
	info "Cache is {$cache}"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser has errors: %v", p.Errors())
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(task.Parameters))
	}

	param := task.Parameters[0]
	if param.Type != "requires" {
		t.Errorf("expected parameter type 'requires', got '%s'", param.Type)
	}

	if param.Name != "cache" {
		t.Errorf("expected parameter name 'cache', got '%s'", param.Name)
	}

	if !param.Required {
		t.Error("expected parameter to be required")
	}

	if param.DefaultValue != `no` {
		t.Errorf("expected default value 'no', got '%s'", param.DefaultValue)
	}

	if len(param.Constraints) != 2 {
		t.Fatalf("expected 2 constraints, got %d", len(param.Constraints))
	}

	if param.Constraints[0] != "yes" || param.Constraints[1] != "no" {
		t.Errorf("expected constraints ['yes', 'no'], got %v", param.Constraints)
	}
}

func TestRequiresWithDefaultValueValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid default value",
			input: `
version: 2.0

task "build":
	requires $env from ["dev", "staging", "prod"] defaults to "dev"
`,
			shouldError: false,
		},
		{
			name: "invalid default value",
			input: `
version: 2.0

task "build":
	requires $env from ["dev", "staging", "prod"] defaults to "production"
`,
			shouldError: true,
			errorMsg:    "default value 'production' must be one of the allowed values: [dev, staging, prod]",
		},
		{
			name: "numeric default value",
			input: `
version: 2.0

task "build":
	requires $port from ["8080", "3000", "5000"] defaults to "3000"
`,
			shouldError: false,
		},
		{
			name: "invalid numeric default value",
			input: `
version: 2.0

task "build":
	requires $port from ["8080", "3000", "5000"] defaults to "9000"
`,
			shouldError: true,
			errorMsg:    "default value '9000' must be one of the allowed values: [8080, 3000, 5000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer(tt.input)
			p := NewParser(l)
			_ = p.ParseProgram()

			hasError := len(p.Errors()) > 0

			if hasError != tt.shouldError {
				if tt.shouldError {
					t.Errorf("expected parser error, but got none")
				} else {
					t.Errorf("unexpected parser error: %v", p.Errors())
				}
				return
			}

			if tt.shouldError && hasError {
				found := false
				for _, err := range p.Errors() {
					if err == tt.errorMsg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error message '%s', got %v", tt.errorMsg, p.Errors())
				}
			}
		})
	}
}

func TestRequiresWithoutConstraintsButWithDefault(t *testing.T) {
	input := `
version: 2.0

task "build":
	requires $image defaults to "base"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser has errors: %v", p.Errors())
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(task.Parameters))
	}

	param := task.Parameters[0]
	if param.DefaultValue != `base` {
		t.Errorf("expected default value 'base', got '%s'", param.DefaultValue)
	}

	if len(param.Constraints) != 0 {
		t.Errorf("expected no constraints, got %v", param.Constraints)
	}
}
