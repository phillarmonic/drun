package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v2/lexer"
)

func TestParser_HelloWorld(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello from drun v2! 👋"

task "hello world":
  step "Starting hello world example"
  info "Welcome to the semantic task runner!"
  success "Hello world completed successfully!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	// Check version
	if program.Version == nil {
		t.Fatalf("program.Version is nil")
	}

	if program.Version.Value != "2.0" {
		t.Errorf("program.Version.Value wrong. expected=2.0, got=%s", program.Version.Value)
	}

	// Check tasks
	if len(program.Tasks) != 2 {
		t.Fatalf("program.Tasks does not contain 2 tasks. got=%d", len(program.Tasks))
	}

	// Check first task
	task1 := program.Tasks[0]
	if task1.Name != "hello" {
		t.Errorf("task1.Name wrong. expected=hello, got=%s", task1.Name)
	}

	if len(task1.Body) != 1 {
		t.Errorf("task1.Body wrong length. expected=1, got=%d", len(task1.Body))
	}

	if task1.Body[0].Action != "info" {
		t.Errorf("task1.Body[0].Action wrong. expected=info, got=%s", task1.Body[0].Action)
	}

	if task1.Body[0].Message != "Hello from drun v2! 👋" {
		t.Errorf("task1.Body[0].Message wrong. expected='Hello from drun v2! 👋', got=%s", task1.Body[0].Message)
	}

	// Check second task
	task2 := program.Tasks[1]
	if task2.Name != "hello world" {
		t.Errorf("task2.Name wrong. expected='hello world', got=%s", task2.Name)
	}

	if len(task2.Body) != 3 {
		t.Errorf("task2.Body wrong length. expected=3, got=%d", len(task2.Body))
	}

	expectedActions := []struct {
		action  string
		message string
	}{
		{"step", "Starting hello world example"},
		{"info", "Welcome to the semantic task runner!"},
		{"success", "Hello world completed successfully!"},
	}

	for i, expected := range expectedActions {
		if task2.Body[i].Action != expected.action {
			t.Errorf("task2.Body[%d].Action wrong. expected=%s, got=%s", i, expected.action, task2.Body[i].Action)
		}

		if task2.Body[i].Message != expected.message {
			t.Errorf("task2.Body[%d].Message wrong. expected=%s, got=%s", i, expected.message, task2.Body[i].Message)
		}
	}
}

func TestParser_TaskWithMeans(t *testing.T) {
	input := `version: 2.0

task "greet" means "Greet someone by name":
  info "Hello there!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("program.Tasks does not contain 1 task. got=%d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "greet" {
		t.Errorf("task.Name wrong. expected=greet, got=%s", task.Name)
	}

	if task.Description != "Greet someone by name" {
		t.Errorf("task.Description wrong. expected='Greet someone by name', got=%s", task.Description)
	}
}

func TestParser_BasicVersion(t *testing.T) {
	input := `version: 2.0`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if program.Version == nil {
		t.Fatalf("program.Version is nil")
	}

	if program.Version.Value != "2.0" {
		t.Errorf("program.Version.Value wrong. expected=2.0, got=%s", program.Version.Value)
	}

	if len(program.Tasks) != 0 {
		t.Errorf("program.Tasks should be empty. got=%d", len(program.Tasks))
	}
}

func TestParser_Errors(t *testing.T) {
	tests := []struct {
		input         string
		expectedError string
	}{
		{
			"task \"hello\":",
			"expected version statement",
		},
		{
			"version 2.0",
			"expected next token to be COLON",
		},
		{
			"version:",
			"expected next token to be NUMBER",
		},
	}

	for _, tt := range tests {
		lexer := lexer.NewLexer(tt.input)
		parser := NewParser(lexer)
		parser.ParseProgram()

		errors := parser.Errors()
		if len(errors) == 0 {
			t.Errorf("expected parser errors for input %q, got none", tt.input)
			continue
		}

		found := false
		for _, err := range errors {
			if containsString(err, tt.expectedError) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected error containing %q, got %v", tt.expectedError, errors)
		}
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
