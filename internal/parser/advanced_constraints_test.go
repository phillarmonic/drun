package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_AdvancedParameterConstraints(t *testing.T) {
	input := `version: 2.0

task "advanced_params":
  requires $port as number between 1000 and 9999
  requires $version as string matching pattern "v\d+\.\d+\.\d+"
  requires $email as string matching email format
  accepts $flags as list
  given $timeout as number between 1 and 300 defaults to "30"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 5 {
		t.Fatalf("Expected 5 parameters, got %d", len(task.Parameters))
	}

	// Test port parameter (range constraint)
	portParam := task.Parameters[0]
	if portParam.Name != "port" {
		t.Errorf("Expected parameter name 'port', got '%s'", portParam.Name)
	}
	if portParam.DataType != "number" {
		t.Errorf("Expected parameter type 'number', got '%s'", portParam.DataType)
	}
	if portParam.MinValue == nil || *portParam.MinValue != 1000 {
		t.Errorf("Expected MinValue 1000, got %v", portParam.MinValue)
	}
	if portParam.MaxValue == nil || *portParam.MaxValue != 9999 {
		t.Errorf("Expected MaxValue 9999, got %v", portParam.MaxValue)
	}

	// Test version parameter (pattern constraint)
	versionParam := task.Parameters[1]
	if versionParam.Name != "version" {
		t.Errorf("Expected parameter name 'version', got '%s'", versionParam.Name)
	}
	if versionParam.DataType != "string" {
		t.Errorf("Expected parameter type 'string', got '%s'", versionParam.DataType)
	}
	expectedPattern := `v\d+\.\d+\.\d+`
	if versionParam.Pattern != expectedPattern {
		t.Errorf("Expected pattern '%s', got '%s'", expectedPattern, versionParam.Pattern)
	}

	// Test email parameter (email format constraint)
	emailParam := task.Parameters[2]
	if emailParam.Name != "email" {
		t.Errorf("Expected parameter name 'email', got '%s'", emailParam.Name)
	}
	if emailParam.DataType != "string" {
		t.Errorf("Expected parameter type 'string', got '%s'", emailParam.DataType)
	}
	if !emailParam.EmailFormat {
		t.Errorf("Expected EmailFormat to be true")
	}

	// Test flags parameter (variadic/list)
	flagsParam := task.Parameters[3]
	if flagsParam.Name != "flags" {
		t.Errorf("Expected parameter name 'flags', got '%s'", flagsParam.Name)
	}
	if flagsParam.DataType != "list" {
		t.Errorf("Expected parameter type 'list', got '%s'", flagsParam.DataType)
	}
	if !flagsParam.Variadic {
		t.Errorf("Expected Variadic to be true for list parameters")
	}

	// Test timeout parameter (range constraint with default)
	timeoutParam := task.Parameters[4]
	if timeoutParam.Name != "timeout" {
		t.Errorf("Expected parameter name 'timeout', got '%s'", timeoutParam.Name)
	}
	if timeoutParam.DataType != "number" {
		t.Errorf("Expected parameter type 'number', got '%s'", timeoutParam.DataType)
	}
	if timeoutParam.MinValue == nil || *timeoutParam.MinValue != 1 {
		t.Errorf("Expected MinValue 1, got %v", timeoutParam.MinValue)
	}
	if timeoutParam.MaxValue == nil || *timeoutParam.MaxValue != 300 {
		t.Errorf("Expected MaxValue 300, got %v", timeoutParam.MaxValue)
	}
	if timeoutParam.DefaultValue != "30" {
		t.Errorf("Expected DefaultValue '30', got '%s'", timeoutParam.DefaultValue)
	}
}

func TestParser_PatternConstraintVariations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "version pattern",
			input:    `requires $version as string matching pattern "v\d+\.\d+\.\d+"`,
			expected: `v\d+\.\d+\.\d+`,
		},
		{
			name:     "uuid pattern",
			input:    `requires $id as string matching pattern "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"`,
			expected: `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `version: 2.0
task "test":
  ` + tt.input

			l := lexer.NewLexer(input)
			p := NewParser(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			param := program.Tasks[0].Parameters[0]
			if param.Pattern != tt.expected {
				t.Errorf("Expected pattern '%s', got '%s'", tt.expected, param.Pattern)
			}
		})
	}
}

func TestParser_EmailFormatConstraint(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "email format",
			input: `requires $email as string matching email format`,
		},
		{
			name:  "email without format keyword",
			input: `requires $email as string matching email`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `version: 2.0
task "test":
  ` + tt.input

			l := lexer.NewLexer(input)
			p := NewParser(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			param := program.Tasks[0].Parameters[0]
			if !param.EmailFormat {
				t.Errorf("Expected EmailFormat to be true")
			}
		})
	}
}

func TestParser_RangeConstraintVariations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minValue float64
		maxValue float64
	}{
		{
			name:     "integer range",
			input:    `requires $port as number between 1000 and 9999`,
			minValue: 1000,
			maxValue: 9999,
		},
		{
			name:     "decimal range",
			input:    `requires $ratio as number between 0.1 and 1.0`,
			minValue: 0.1,
			maxValue: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `version: 2.0
task "test":
  ` + tt.input

			l := lexer.NewLexer(input)
			p := NewParser(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			param := program.Tasks[0].Parameters[0]
			if param.MinValue == nil || *param.MinValue != tt.minValue {
				t.Errorf("Expected MinValue %f, got %v", tt.minValue, param.MinValue)
			}
			if param.MaxValue == nil || *param.MaxValue != tt.maxValue {
				t.Errorf("Expected MaxValue %f, got %v", tt.maxValue, param.MaxValue)
			}
		})
	}
}
