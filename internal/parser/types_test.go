package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_TypedParameters(t *testing.T) {
	input := `version: 2.0

task "typed test":
  requires count as number
  given enabled as boolean defaults to "true"
  accepts items as list
  requires env as string from ["dev", "staging"]`

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
	if len(task.Parameters) != 4 {
		t.Fatalf("Expected 4 parameters, got %d", len(task.Parameters))
	}

	// Check parameter types
	expectedParams := []struct {
		name           string
		dataType       string
		required       bool
		hasConstraints bool
	}{
		{"count", "number", true, false},
		{"enabled", "boolean", false, false},
		{"items", "list", false, false},
		{"env", "string", true, true},
	}

	for i, expected := range expectedParams {
		param := task.Parameters[i]

		if param.Name != expected.name {
			t.Errorf("Parameter %d: expected name %s, got %s", i, expected.name, param.Name)
		}

		if param.DataType != expected.dataType {
			t.Errorf("Parameter %d: expected type %s, got %s", i, expected.dataType, param.DataType)
		}

		if param.Required != expected.required {
			t.Errorf("Parameter %d: expected required %t, got %t", i, expected.required, param.Required)
		}

		hasConstraints := len(param.Constraints) > 0
		if hasConstraints != expected.hasConstraints {
			t.Errorf("Parameter %d: expected hasConstraints %t, got %t", i, expected.hasConstraints, hasConstraints)
		}
	}

	// Check specific constraint values for env parameter
	envParam := task.Parameters[3]
	expectedConstraints := []string{"dev", "staging"}
	if len(envParam.Constraints) != len(expectedConstraints) {
		t.Errorf("Expected %d constraints, got %d", len(expectedConstraints), len(envParam.Constraints))
	}

	for i, expected := range expectedConstraints {
		if i < len(envParam.Constraints) && envParam.Constraints[i] != expected {
			t.Errorf("Constraint %d: expected %s, got %s", i, expected, envParam.Constraints[i])
		}
	}
}

func TestParser_ListWithConstraints(t *testing.T) {
	input := `version: 2.0

task "list constraints":
  requires $environments as list from ["dev", "staging", "production"]`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(task.Parameters))
	}

	param := task.Parameters[0]

	if param.Name != "environments" {
		t.Errorf("Expected name 'environments', got %s", param.Name)
	}

	if param.DataType != "list" {
		t.Errorf("Expected type 'list', got %s", param.DataType)
	}

	if !param.Required {
		t.Errorf("Expected parameter to be required")
	}

	expectedConstraints := []string{"dev", "staging", "production"}
	if len(param.Constraints) != len(expectedConstraints) {
		t.Errorf("Expected %d constraints, got %d", len(expectedConstraints), len(param.Constraints))
	}

	for i, expected := range expectedConstraints {
		if i < len(param.Constraints) && param.Constraints[i] != expected {
			t.Errorf("Constraint %d: expected %s, got %s", i, expected, param.Constraints[i])
		}
	}
}
