package statement

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
)

func TestFromAST_Action(t *testing.T) {
	astAction := &ast.ActionStatement{
		Action:          "info",
		Message:         "test message",
		LineBreakBefore: true,
		LineBreakAfter:  false,
	}

	domainStmt, err := FromAST(astAction)
	if err != nil {
		t.Fatalf("FromAST() error = %v", err)
	}

	action, ok := domainStmt.(*Action)
	if !ok {
		t.Fatalf("Expected *Action, got %T", domainStmt)
	}

	if action.ActionType != "info" {
		t.Errorf("ActionType = %v, want info", action.ActionType)
	}
	if action.Message != "test message" {
		t.Errorf("Message = %v, want test message", action.Message)
	}
	if !action.LineBreakBefore {
		t.Error("LineBreakBefore should be true")
	}
	if action.LineBreakAfter {
		t.Error("LineBreakAfter should be false")
	}
}

func TestFromAST_Shell(t *testing.T) {
	astShell := &ast.ShellStatement{
		Action:     "run",
		Command:    "echo hello",
		CaptureVar: "output",
		Attached:   true,
	}

	domainStmt, err := FromAST(astShell)
	if err != nil {
		t.Fatalf("FromAST() error = %v", err)
	}

	shell, ok := domainStmt.(*Shell)
	if !ok {
		t.Fatalf("Expected *Shell, got %T", domainStmt)
	}

	if shell.Action != "run" {
		t.Errorf("Action = %v, want run", shell.Action)
	}
	if shell.Command != "echo hello" {
		t.Errorf("Command = %v, want echo hello", shell.Command)
	}
	if shell.CaptureVar != "output" {
		t.Errorf("CaptureVar = %v, want output", shell.CaptureVar)
	}
	if !shell.Attached {
		t.Error("Attached should be true")
	}
}

func TestFromAST_RequiresToolsPreservesTaskRefs(t *testing.T) {
	domainStmt, err := FromAST(&ast.RequiresToolsStatement{
		Tools: []ast.ToolRequirement{{Name: "gosec"}},
		TaskSources: []ast.TaskToolSources{
			{Tasks: []string{"build", "lint"}},
			{Tasks: []string{"integration"}},
		},
	})
	if err != nil {
		t.Fatalf("FromAST() error = %v", err)
	}

	requiresTools, ok := domainStmt.(*RequiresTools)
	if !ok {
		t.Fatalf("Expected *RequiresTools, got %T", domainStmt)
	}

	if len(requiresTools.Tools) != 1 || requiresTools.Tools[0].Name != "gosec" {
		t.Fatalf("unexpected direct tools: %#v", requiresTools.Tools)
	}

	wantRefs := []string{"build", "lint", "integration"}
	if len(requiresTools.TaskRefs) != len(wantRefs) {
		t.Fatalf("TaskRefs length = %d, want %d", len(requiresTools.TaskRefs), len(wantRefs))
	}
	for i, want := range wantRefs {
		if got := requiresTools.TaskRefs[i]; got != want {
			t.Fatalf("TaskRefs[%d] = %q, want %q", i, got, want)
		}
	}
}

func TestFromAST_GitEnsureVersion(t *testing.T) {
	domainStmt, err := FromAST(&ast.GitEnsureVersionStatement{
		Candidate: "$release_version", CandidateIsVariable: true, Source: "runtime",
		AccessMethod: "ssh", TagFormat: "runtime-{version}", CaptureVar: "latest_version",
	})
	if err != nil {
		t.Fatal(err)
	}
	guard, ok := domainStmt.(*GitEnsureVersion)
	if !ok {
		t.Fatalf("statement = %T", domainStmt)
	}
	if guard.Type() != TypeGitEnsureVersion || guard.Candidate != "$release_version" || guard.Source != "runtime" || guard.TagFormat != "runtime-{version}" || guard.CaptureVar != "latest_version" {
		t.Fatalf("guard = %#v", guard)
	}
}

func TestFromAST_Conditional(t *testing.T) {
	astCond := &ast.ConditionalStatement{
		Type:      "when",
		Condition: "env == prod",
		Body: []ast.Statement{
			&ast.ActionStatement{Action: "info", Message: "production"},
		},
		ElseBody: []ast.Statement{
			&ast.ActionStatement{Action: "info", Message: "development"},
		},
	}

	domainStmt, err := FromAST(astCond)
	if err != nil {
		t.Fatalf("FromAST() error = %v", err)
	}

	cond, ok := domainStmt.(*Conditional)
	if !ok {
		t.Fatalf("Expected *Conditional, got %T", domainStmt)
	}

	if cond.ConditionType != "when" {
		t.Errorf("ConditionType = %v, want when", cond.ConditionType)
	}
	if cond.Condition != "env == prod" {
		t.Errorf("Condition = %v, want env == prod", cond.Condition)
	}
	if len(cond.Body) != 1 {
		t.Errorf("Body length = %v, want 1", len(cond.Body))
	}
	if len(cond.ElseBody) != 1 {
		t.Errorf("ElseBody length = %v, want 1", len(cond.ElseBody))
	}
}

func TestFromASTList_SkipsNil(t *testing.T) {
	astList := []ast.Statement{
		&ast.ActionStatement{Action: "info", Message: "first"},
		&ast.ParameterStatement{Name: "test"}, // Should be skipped (returns nil)
		&ast.ActionStatement{Action: "info", Message: "second"},
	}

	domainList, err := FromASTList(astList)
	if err != nil {
		t.Fatalf("FromASTList() error = %v", err)
	}

	// Should have 2 items (parameter is skipped)
	if len(domainList) != 2 {
		t.Errorf("Result length = %v, want 2 (parameter should be skipped)", len(domainList))
	}
}
