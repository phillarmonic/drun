package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_SimpleParameters(t *testing.T) {
	input := `version: 2.0

task "greet":
  requires name
  given title defaults to "friend"
  
  info "Hello, {title} {name}!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) != 0 {
		t.Fatalf("parser has %d errors", len(parser.Errors()))
		for _, err := range parser.Errors() {
			t.Errorf("parser error: %q", err)
		}
	}

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("program.Tasks does not contain 1 task. got=%d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "greet" {
		t.Errorf("task.Name not 'greet'. got=%q", task.Name)
	}

	if len(task.Parameters) != 2 {
		t.Fatalf("task.Parameters does not contain 2 parameters. got=%d", len(task.Parameters))
	}

	// Check first parameter (requires name)
	param1 := task.Parameters[0]
	if param1.Type != "requires" {
		t.Errorf("param1.Type not 'requires'. got=%q", param1.Type)
	}
	if param1.Name != "name" {
		t.Errorf("param1.Name not 'name'. got=%q", param1.Name)
	}
	if !param1.Required {
		t.Errorf("param1.Required should be true")
	}

	// Check second parameter (given title defaults to "friend")
	param2 := task.Parameters[1]
	if param2.Type != "given" {
		t.Errorf("param2.Type not 'given'. got=%q", param2.Type)
	}
	if param2.Name != "title" {
		t.Errorf("param2.Name not 'title'. got=%q", param2.Name)
	}
	if param2.Required {
		t.Errorf("param2.Required should be false")
	}
	if param2.DefaultValue != "friend" {
		t.Errorf("param2.DefaultValue not 'friend'. got=%q", param2.DefaultValue)
	}

	// Check that we still have the action statement
	if len(task.Body) != 1 {
		t.Fatalf("task.Body does not contain 1 statement. got=%d", len(task.Body))
	}
}

func TestParser_ParameterConstraints(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires $environment from ["dev", "staging", "production"]
  
  step "Deploying to {environment}"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) != 0 {
		t.Fatalf("parser has %d errors", len(parser.Errors()))
		for _, err := range parser.Errors() {
			t.Errorf("parser error: %q", err)
		}
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 1 {
		t.Fatalf("task.Parameters does not contain 1 parameter. got=%d", len(task.Parameters))
	}

	param := task.Parameters[0]
	if param.Name != "environment" {
		t.Errorf("param.Name not 'environment'. got=%q", param.Name)
	}

	expectedConstraints := []string{"dev", "staging", "production"}
	if len(param.Constraints) != len(expectedConstraints) {
		t.Fatalf("param.Constraints length mismatch. expected=%d, got=%d",
			len(expectedConstraints), len(param.Constraints))
	}

	for i, expected := range expectedConstraints {
		if param.Constraints[i] != expected {
			t.Errorf("param.Constraints[%d] not '%s'. got=%q", i, expected, param.Constraints[i])
		}
	}
}
