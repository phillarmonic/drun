package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_SingleMultilineShell(t *testing.T) {
	input := `version: 2.0

task "simple multiline":
  run:
    echo "hello"
    echo "world"`

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
	if len(task.Body) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(task.Body))
	}

	runStmt, ok := task.Body[0].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected ShellStatement, got %T", task.Body[0])
	}

	if !runStmt.IsMultiline {
		t.Errorf("Expected IsMultiline to be true")
	}

	if runStmt.Action != "run" {
		t.Errorf("Expected action 'run', got %s", runStmt.Action)
	}

	expectedCommands := []string{"echo \"hello\"", "echo \"world\""}
	if len(runStmt.Commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(runStmt.Commands))
	}

	for i, expected := range expectedCommands {
		if i < len(runStmt.Commands) && runStmt.Commands[i] != expected {
			t.Errorf("Expected command %d to be '%s', got '%s'", i, expected, runStmt.Commands[i])
		}
	}
}
