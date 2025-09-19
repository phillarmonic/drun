package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_TwoMultilineShell(t *testing.T) {
	input := `version: 2.0

task "two multiline":
  run:
    echo "first"
  
  exec:
    echo "second"`

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
	if len(task.Body) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(task.Body))
	}

	// Check first statement
	runStmt, ok := task.Body[0].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected first statement to be ShellStatement, got %T", task.Body[0])
	}
	if runStmt.Action != "run" {
		t.Errorf("Expected first action 'run', got %s", runStmt.Action)
	}

	// Check second statement
	execStmt, ok := task.Body[1].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected second statement to be ShellStatement, got %T", task.Body[1])
	}
	if execStmt.Action != "exec" {
		t.Errorf("Expected second action 'exec', got %s", execStmt.Action)
	}
}
