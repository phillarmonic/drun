package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParseTaskCallWithDashedName(t *testing.T) {
	input := `version: 2.0

task "run-action":
  info "hello"

task "caller":
  call task run-action
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(program.Tasks))
	}

	caller := program.Tasks[1]
	if len(caller.Body) == 0 {
		t.Fatalf("expected caller task to have a body")
	}

	callStmt, ok := caller.Body[0].(*ast.TaskCallStatement)
	if !ok {
		t.Fatalf("expected first statement to be TaskCallStatement, got %T", caller.Body[0])
	}

	if callStmt.TaskName != "run-action" {
		t.Fatalf("expected task name 'run-action', got %q", callStmt.TaskName)
	}
}

func TestParseDependencyWithDashedName(t *testing.T) {
	input := `version: 2.0

task "run-action":
  info "hello"

task "dependent":
  depends on run-action
  info "done"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(program.Tasks))
	}

	dependent := program.Tasks[1]
	if len(dependent.Dependencies) != 1 {
		t.Fatalf("expected one dependency group, got %d", len(dependent.Dependencies))
	}

	group := dependent.Dependencies[0]
	if len(group.Dependencies) != 1 {
		t.Fatalf("expected one dependency item, got %d", len(group.Dependencies))
	}

	if group.Dependencies[0].Name != "run-action" {
		t.Fatalf("expected dependency name 'run-action', got %q", group.Dependencies[0].Name)
	}
}

func TestParseTaskCallWithNumericParameter(t *testing.T) {
	input := `version: 2.0

task "fuzz":
  requires $iterations
  info "Iterations: {$iterations}"

task "caller":
  call task fuzz with iterations=100
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(program.Tasks))
	}

	caller := program.Tasks[1]
	if len(caller.Body) == 0 {
		t.Fatalf("expected caller task to have a body")
	}

	callStmt, ok := caller.Body[0].(*ast.TaskCallStatement)
	if !ok {
		t.Fatalf("expected first statement to be TaskCallStatement, got %T", caller.Body[0])
	}

	if got := callStmt.Parameters["iterations"]; got != "100" {
		t.Fatalf("expected iterations parameter to be %q, got %q", "100", got)
	}
}

func TestParseTaskCallWithFollowedBySetStatement(t *testing.T) {
	// Regression: the lexer does not emit NEWLINE tokens between statements
	// inside a block, so the "with" parameter loop used to swallow a following
	// statement whose first token is a keyword (e.g. "set $x to ...") as
	// another parameter name. Parameter names must stay on the same source
	// line as the "with" keyword.
	input := `version: 2.0

task "inner":
  given $a defaults to "x"
  info "a={$a}"

task "caller":
  call task inner with a="1"
  set $v to "done"
  info "{$v}"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(program.Tasks))
	}

	caller := program.Tasks[1]
	if len(caller.Body) != 3 {
		t.Fatalf("expected caller task to have 3 statements, got %d", len(caller.Body))
	}

	callStmt, ok := caller.Body[0].(*ast.TaskCallStatement)
	if !ok {
		t.Fatalf("expected first statement to be TaskCallStatement, got %T", caller.Body[0])
	}

	if got := callStmt.Parameters["a"]; got != "1" {
		t.Fatalf("expected a parameter to be %q, got %q", "1", got)
	}

	if _, ok := caller.Body[1].(*ast.VariableStatement); !ok {
		t.Fatalf("expected second statement to be VariableStatement, got %T", caller.Body[1])
	}
}
