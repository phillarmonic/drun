package engine

import (
	"bytes"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

func TestTaskCall(t *testing.T) {
	input := `
version: 2.0

task "hello":
  info "Hello from hello task"

task "greet":
  requires $name
  info "Hello, {$name}!"

task "main":
  info "Starting main task"
  call task "hello"
  call task "greet" with name="World"
  info "Main task completed"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	err := engine.Execute(program, "main")
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output := buf.String()
	expected := []string{
		"Starting main task",
		"Hello from hello task",
		"Hello, World!",
		"Main task completed",
	}

	for _, exp := range expected {
		if !contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got: %s", exp, output)
		}
	}
}

func TestTaskCallWithParameters(t *testing.T) {
	input := `
version: 2.0

task "multiply":
  requires $a
  requires $b
  info "Result: {$a} * {$b}"

task "calculator":
  call task "multiply" with a="5" b="3"
  info "Calculation done"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	err := engine.Execute(program, "calculator")
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output := buf.String()
	if !contains(output, "Result: 5 * 3") {
		t.Errorf("Expected output to contain 'Result: 5 * 3', got: %s", output)
	}
	if !contains(output, "Calculation done") {
		t.Errorf("Expected output to contain 'Calculation done', got: %s", output)
	}
}

func TestTaskCallNotFound(t *testing.T) {
	input := `
version: 2.0

task "main":
  call task "nonexistent"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	err := engine.Execute(program, "main")
	if err == nil {
		t.Fatal("Expected error for nonexistent task, got nil")
	}

	if !contains(err.Error(), "task 'nonexistent' not found") {
		t.Errorf("Expected error about task not found, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
