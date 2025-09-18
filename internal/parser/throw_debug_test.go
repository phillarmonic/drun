package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_ThrowDebug(t *testing.T) {
	input := `version: 2.0

task "debug":
  throw "test error"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(task.Body))
	}

	stmt := task.Body[0]
	t.Logf("Statement type: %T", stmt)

	throwStmt, ok := stmt.(*ast.ThrowStatement)
	if !ok {
		t.Fatalf("Expected ThrowStatement, got %T", stmt)
	}

	if throwStmt.Action != "throw" {
		t.Errorf("Expected action 'throw', got %s", throwStmt.Action)
	}

	if throwStmt.Message != "test error" {
		t.Errorf("Expected message 'test error', got %s", throwStmt.Message)
	}
}
