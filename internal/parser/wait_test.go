package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_WaitDurationUnits(t *testing.T) {
	tests := []struct {
		input     string
		wantValue string
		wantUnit  string
	}{
		{"wait 5 seconds", "5", "second"},
		{"wait 1 second", "1", "second"},
		{"wait 2 minutes", "2", "minute"},
		{"wait 1 minute", "1", "minute"},
		{"wait 3 hours", "3", "hour"},
		{"wait 1 hour", "1", "hour"},
		{"wait 0.5 seconds", "0.5", "second"},
		{"wait {$backoff} seconds", "{$backoff}", "second"},
		{"wait {backoff} minutes", "{backoff}", "minute"},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n  " + tt.input + "\n"

		l := lexer.NewLexer(input)
		p := NewParser(l)
		program := p.ParseProgram()

		checkParserErrors(t, p)

		if len(program.Tasks) != 1 {
			t.Fatalf("%q: program should have 1 task. got=%d", tt.input, len(program.Tasks))
		}

		task := program.Tasks[0]
		if len(task.Body) != 1 {
			t.Fatalf("%q: task should have 1 statement. got=%d", tt.input, len(task.Body))
		}

		waitStmt, ok := task.Body[0].(*ast.WaitStatement)
		if !ok {
			t.Fatalf("%q: statement should be WaitStatement. got=%T", tt.input, task.Body[0])
		}

		if waitStmt.Value != tt.wantValue {
			t.Errorf("%q: wait value not %q. got=%q", tt.input, tt.wantValue, waitStmt.Value)
		}

		if waitStmt.Unit != tt.wantUnit {
			t.Errorf("%q: wait unit not %q. got=%q", tt.input, tt.wantUnit, waitStmt.Unit)
		}
	}
}

func TestParser_WaitForServiceStillParses(t *testing.T) {
	input := `version: 2.0

task "demo":
  wait for service at "https://api.local/health" to be ready timeout "60s"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("task should have 1 statement. got=%d", len(task.Body))
	}

	networkStmt, ok := task.Body[0].(*ast.NetworkStatement)
	if !ok {
		t.Fatalf("statement should be NetworkStatement. got=%T", task.Body[0])
	}

	if networkStmt.Action != "wait_for_service" {
		t.Errorf("network action not 'wait_for_service'. got=%q", networkStmt.Action)
	}
}

func TestParser_WaitDurationInControlFlow(t *testing.T) {
	input := `version: 2.0

task "demo":
  for each $attempt in ["1", "2"]:
    try:
      run "./flaky.sh"
      break
    catch:
      wait {$attempt} seconds`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("task should have 1 statement. got=%d", len(task.Body))
	}

	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("statement should be LoopStatement. got=%T", task.Body[0])
	}

	if len(loopStmt.Body) != 1 {
		t.Fatalf("loop body should have 1 statement. got=%d", len(loopStmt.Body))
	}

	tryStmt, ok := loopStmt.Body[0].(*ast.TryStatement)
	if !ok {
		t.Fatalf("loop body statement should be TryStatement. got=%T", loopStmt.Body[0])
	}

	if len(tryStmt.CatchClauses) != 1 {
		t.Fatalf("try should have 1 catch clause. got=%d", len(tryStmt.CatchClauses))
	}

	catchBody := tryStmt.CatchClauses[0].Body
	if len(catchBody) != 1 {
		t.Fatalf("catch body should have 1 statement. got=%d", len(catchBody))
	}

	waitStmt, ok := catchBody[0].(*ast.WaitStatement)
	if !ok {
		t.Fatalf("catch body statement should be WaitStatement. got=%T", catchBody[0])
	}

	if waitStmt.Value != "{$attempt}" || waitStmt.Unit != "second" {
		t.Errorf("unexpected wait statement. got=%q %q", waitStmt.Value, waitStmt.Unit)
	}
}

func TestParser_WaitDurationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing unit", "wait 5"},
		{"invalid unit", "wait 5 days"},
		{"missing value", "wait seconds"},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n  " + tt.input + "\n"

		l := lexer.NewLexer(input)
		p := NewParser(l)
		p.ParseProgram()

		if len(p.Errors()) == 0 {
			t.Errorf("%s (%q): expected parser errors, got none", tt.name, tt.input)
		}
	}
}
