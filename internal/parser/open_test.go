package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_OpenURLHappyPaths(t *testing.T) {
	tests := []struct {
		input   string
		wantURL string
	}{
		{`open url "https://example.com"`, "https://example.com"},
		{`open url "{$base}/docs"`, "{$base}/docs"},
		{`open url "./report.html"`, "./report.html"},
		{`open url "file:///tmp/output.html"`, "file:///tmp/output.html"},
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

		openStmt, ok := task.Body[0].(*ast.OpenStatement)
		if !ok {
			t.Fatalf("%q: statement should be OpenStatement. got=%T", tt.input, task.Body[0])
		}

		if openStmt.Noun != "url" {
			t.Errorf("%q: noun not %q. got=%q", tt.input, "url", openStmt.Noun)
		}

		if openStmt.URL != tt.wantURL {
			t.Errorf("%q: URL not %q. got=%q", tt.input, tt.wantURL, openStmt.URL)
		}
	}
}

func TestParser_OpenURLInControlFlow(t *testing.T) {
	input := `version: 2.0

task "demo":
  if $open_browser is "true":
    open url "https://example.com"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("task should have 1 statement. got=%d", len(task.Body))
	}

	condStmt, ok := task.Body[0].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("statement should be ConditionalStatement. got=%T", task.Body[0])
	}

	if len(condStmt.Body) != 1 {
		t.Fatalf("if body should have 1 statement. got=%d", len(condStmt.Body))
	}

	openStmt, ok := condStmt.Body[0].(*ast.OpenStatement)
	if !ok {
		t.Fatalf("if body statement should be OpenStatement. got=%T", condStmt.Body[0])
	}

	if openStmt.URL != "https://example.com" {
		t.Errorf("unexpected URL. got=%q", openStmt.URL)
	}
}

func TestParser_OpenURLInTryCatch(t *testing.T) {
	input := `version: 2.0

task "demo":
  try:
    open url "https://example.com"
  catch:
    warn "could not open"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("task should have 1 statement. got=%d", len(task.Body))
	}

	tryStmt, ok := task.Body[0].(*ast.TryStatement)
	if !ok {
		t.Fatalf("statement should be TryStatement. got=%T", task.Body[0])
	}

	if len(tryStmt.TryBody) != 1 {
		t.Fatalf("try body should have 1 statement. got=%d", len(tryStmt.TryBody))
	}

	_, ok = tryStmt.TryBody[0].(*ast.OpenStatement)
	if !ok {
		t.Fatalf("try body statement should be OpenStatement. got=%T", tryStmt.TryBody[0])
	}
}

func TestParser_OpenURLInForEach(t *testing.T) {
	input := `version: 2.0

task "demo":
  for each $page in ["index", "about"]:
    open url "https://example.com/{$page}"
`

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

	openStmt, ok := loopStmt.Body[0].(*ast.OpenStatement)
	if !ok {
		t.Fatalf("loop body statement should be OpenStatement. got=%T", loopStmt.Body[0])
	}

	if openStmt.URL != "https://example.com/{$page}" {
		t.Errorf("unexpected URL. got=%q", openStmt.URL)
	}
}

func TestParser_OpenURLErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing noun", `open "https://example.com"`},
		{"unknown noun", `open file "report.html"`},
		{"missing target", "open url\n"},
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

func TestParser_OpenDoesNotBreakPortCheck(t *testing.T) {
	input := `version: 2.0

task "demo":
  check if port 8080 is open on "localhost"
`

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

	if networkStmt.Action != "port_check" {
		t.Errorf("network action not 'port_check'. got=%q", networkStmt.Action)
	}
}

func TestParser_OpenURLString(t *testing.T) {
	input := "version: 2.0\n\ntask \"demo\":\n  open url \"https://example.com\"\n"

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	openStmt := program.Tasks[0].Body[0].(*ast.OpenStatement)
	got := openStmt.String()
	want := `open url "https://example.com"`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
