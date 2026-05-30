package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_OrchestrateRecreateAction(t *testing.T) {
	input := `version: 2.0

task "bounce":
  orchestrate "platform" recreate with cache "false"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("expected 1 statement in task body, got %d", len(task.Body))
	}

	stmt, ok := task.Body[0].(*ast.OrchestrationActionStatement)
	if !ok {
		t.Fatalf("expected OrchestrationActionStatement, got %T", task.Body[0])
	}

	if stmt.Action != "recreate" {
		t.Fatalf("expected action 'recreate', got %q", stmt.Action)
	}

	if stmt.Options["cache"] != "false" {
		t.Fatalf("expected cache option to be 'false', got %q", stmt.Options["cache"])
	}
}

func TestParser_OrchestrateBuildCacheOption(t *testing.T) {
	input := `version: 2.0

task "build-stack":
  orchestrate "platform" build with cache "true"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	stmt, ok := task.Body[0].(*ast.OrchestrationActionStatement)
	if !ok {
		t.Fatalf("expected OrchestrationActionStatement, got %T", task.Body[0])
	}

	if stmt.Action != "build" {
		t.Fatalf("expected action 'build', got %q", stmt.Action)
	}

	if stmt.Options["cache"] != "true" {
		t.Fatalf("expected cache option to be 'true', got %q", stmt.Options["cache"])
	}
}
