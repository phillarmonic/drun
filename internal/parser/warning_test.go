package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_WarningAction(t *testing.T) {
	input := `version: 2.0

task "test":
  warning "Mind the release"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	action, ok := program.Tasks[0].Body[0].(*ast.ActionStatement)
	if !ok {
		t.Fatalf("task body statement is not an ActionStatement: %T", program.Tasks[0].Body[0])
	}
	if action.Action != "warning" {
		t.Errorf("action = %q, want %q", action.Action, "warning")
	}
	if action.Message != "Mind the release" {
		t.Errorf("message = %q, want %q", action.Message, "Mind the release")
	}
}
