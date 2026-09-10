package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func parseTaskWithPrompt(t *testing.T, input string) *ast.TaskStatement {
	t.Helper()
	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Tasks) != 1 {
		t.Fatalf("program should have 1 task. got=%d", len(program.Tasks))
	}
	return program.Tasks[0]
}

func TestParser_ConfirmHappyPaths(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		wantQuestion  string
		wantDefault   string
		wantResultVar string
		wantHasDef    bool
	}{
		{
			name:         "bare gate",
			statement:    `confirm "Deploy to production?"`,
			wantQuestion: "Deploy to production?",
		},
		{
			name:          "with result variable",
			statement:     `confirm "Run database migrations?" as $migrate`,
			wantQuestion:  "Run database migrations?",
			wantResultVar: "migrate",
		},
		{
			name:         "with quoted default",
			statement:    `confirm "Delete build cache?" defaults to "no"`,
			wantQuestion: "Delete build cache?",
			wantDefault:  "no",
			wantHasDef:   true,
		},
		{
			name:         "with bare no default",
			statement:    `confirm "Continue?" defaults to no`,
			wantQuestion: "Continue?",
			wantDefault:  "no",
			wantHasDef:   true,
		},
		{
			name:         "with bare yes default",
			statement:    `confirm "Install extras?" defaults to yes`,
			wantQuestion: "Install extras?",
			wantDefault:  "yes",
			wantHasDef:   true,
		},
		{
			name:          "default and result variable",
			statement:     `confirm "Proceed?" defaults to "yes" as $proceed`,
			wantQuestion:  "Proceed?",
			wantDefault:   "yes",
			wantHasDef:    true,
			wantResultVar: "proceed",
		},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n  " + tt.statement + "\n"
		task := parseTaskWithPrompt(t, input)

		if len(task.Body) != 1 {
			t.Fatalf("%q: task should have 1 statement. got=%d", tt.statement, len(task.Body))
		}
		confirmStmt, ok := task.Body[0].(*ast.ConfirmStatement)
		if !ok {
			t.Fatalf("%q: statement should be ConfirmStatement. got=%T", tt.statement, task.Body[0])
		}
		if confirmStmt.Question != tt.wantQuestion {
			t.Errorf("%q: question not %q. got=%q", tt.statement, tt.wantQuestion, confirmStmt.Question)
		}
		if confirmStmt.HasDefault != tt.wantHasDef {
			t.Errorf("%q: HasDefault not %v. got=%v", tt.statement, tt.wantHasDef, confirmStmt.HasDefault)
		}
		if confirmStmt.DefaultValue != tt.wantDefault {
			t.Errorf("%q: DefaultValue not %q. got=%q", tt.statement, tt.wantDefault, confirmStmt.DefaultValue)
		}
		if confirmStmt.ResultVar != tt.wantResultVar {
			t.Errorf("%q: ResultVar not %q. got=%q", tt.statement, tt.wantResultVar, confirmStmt.ResultVar)
		}
	}
}

func TestParser_PromptHappyPaths(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		wantQuestion  string
		wantDefault   string
		wantResultVar string
		wantHasDef    bool
	}{
		{
			name:          "with result variable",
			statement:     `prompt "Which environment?" as $environment`,
			wantQuestion:  "Which environment?",
			wantResultVar: "environment",
		},
		{
			name:          "with quoted default and result variable",
			statement:     `prompt "Release notes?" defaults to "n/a" as $notes`,
			wantQuestion:  "Release notes?",
			wantDefault:   "n/a",
			wantHasDef:    true,
			wantResultVar: "notes",
		},
		{
			name:          "with bare yes default and result variable",
			statement:     `prompt "Deploy?" defaults to yes as $deploy`,
			wantQuestion:  "Deploy?",
			wantDefault:   "yes",
			wantHasDef:    true,
			wantResultVar: "deploy",
		},
		{
			name:          "interpolated question",
			statement:     `prompt "Which branch for {$project}?" as $branch`,
			wantQuestion:  "Which branch for {$project}?",
			wantResultVar: "branch",
		},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n  " + tt.statement + "\n"
		task := parseTaskWithPrompt(t, input)

		if len(task.Body) != 1 {
			t.Fatalf("%q: task should have 1 statement. got=%d", tt.statement, len(task.Body))
		}
		promptStmt, ok := task.Body[0].(*ast.PromptStatement)
		if !ok {
			t.Fatalf("%q: statement should be PromptStatement. got=%T", tt.statement, task.Body[0])
		}
		if promptStmt.Question != tt.wantQuestion {
			t.Errorf("%q: question not %q. got=%q", tt.statement, tt.wantQuestion, promptStmt.Question)
		}
		if promptStmt.HasDefault != tt.wantHasDef {
			t.Errorf("%q: HasDefault not %v. got=%v", tt.statement, tt.wantHasDef, promptStmt.HasDefault)
		}
		if promptStmt.DefaultValue != tt.wantDefault {
			t.Errorf("%q: DefaultValue not %q. got=%q", tt.statement, tt.wantDefault, promptStmt.DefaultValue)
		}
		if promptStmt.ResultVar != tt.wantResultVar {
			t.Errorf("%q: ResultVar not %q. got=%q", tt.statement, tt.wantResultVar, promptStmt.ResultVar)
		}
	}
}

func TestParser_ConfirmPromptInControlFlow(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		statement string
	}{
		{"confirm in if", "  if $dry_run is \"true\":\n    confirm \"Deploy to production?\" as $migrate\n", "confirm"},
		{"confirm in for", "  for each $page in [\"index\"]:\n    confirm \"Publish {$page}?\" defaults to \"no\"\n", "confirm"},
		{"prompt in when", "  when $env is \"prod\":\n    prompt \"Release notes?\" defaults to \"n/a\" as $notes\n", "prompt"},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n" + tt.input
		task := parseTaskWithPrompt(t, input)

		if len(task.Body) != 1 {
			t.Fatalf("%s: task should have 1 statement. got=%d", tt.name, len(task.Body))
		}
		switch bodyStmt := task.Body[0].(type) {
		case *ast.ConditionalStatement:
			if len(bodyStmt.Body) != 1 {
				t.Fatalf("%s: control flow body should have 1 statement. got=%d", tt.name, len(bodyStmt.Body))
			}
		case *ast.LoopStatement:
			if len(bodyStmt.Body) != 1 {
				t.Fatalf("%s: loop body should have 1 statement. got=%d", tt.name, len(bodyStmt.Body))
			}
		default:
			t.Fatalf("%s: statement should be conditional or loop. got=%T", tt.name, task.Body[0])
		}
	}
}

func TestParser_PromptInLifecycleHook(t *testing.T) {
	input := `version: 2.0

project "demo":
  before any task:
    confirm "Run preflight checks?" as $preflight
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if program == nil || program.Project == nil {
		t.Fatalf("ParseProgram() returned nil project")
	}
	if len(program.Project.Settings) != 1 {
		t.Fatalf("project should have 1 lifecycle hook. got=%d", len(program.Project.Settings))
	}
	hook, ok := program.Project.Settings[0].(*ast.LifecycleHook)
	if !ok {
		t.Fatalf("project.Settings[0] is not *ast.LifecycleHook. got=%T", program.Project.Settings[0])
	}
	if len(hook.Body) != 1 {
		t.Fatalf("hook body should have 1 statement. got=%d", len(hook.Body))
	}
	confirmStmt, ok := hook.Body[0].(*ast.ConfirmStatement)
	if !ok {
		t.Fatalf("hook body statement should be ConfirmStatement. got=%T", hook.Body[0])
	}
	if confirmStmt.ResultVar != "preflight" {
		t.Errorf("ResultVar not %q. got=%q", "preflight", confirmStmt.ResultVar)
	}
}

func TestParser_ConfirmPromptString(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      string
	}{
		{"bare confirm", `confirm "Deploy?"`, `confirm "Deploy?"`},
		{"confirm with default and var", `confirm "Go?" defaults to "no" as $go`, `confirm "Go?" defaults to "no" as $go`},
		{"prompt with default and var", `prompt "Env?" defaults to "dev" as $env`, `prompt "Env?" defaults to "dev" as $env`},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n  " + tt.statement + "\n"
		task := parseTaskWithPrompt(t, input)

		if len(task.Body) != 1 {
			t.Fatalf("%q: task should have 1 statement. got=%d", tt.statement, len(task.Body))
		}
		if got := task.Body[0].String(); got != tt.want {
			t.Errorf("%s: String() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestParser_ConfirmPromptErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"confirm without question", `confirm`},
		{"prompt without question", `prompt`},
		{"confirm defaults without value", `confirm "Go?" defaults to`},
		{"confirm defaults missing to", `confirm "Go?" defaults "yes"`},
		{"confirm with non-variable after as", `confirm "Go?" as env`},
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
