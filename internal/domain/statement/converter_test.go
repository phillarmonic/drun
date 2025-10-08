package statement

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
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

func TestToAST_RoundTrip(t *testing.T) {
	// Create a domain statement
	domainAction := &Action{
		ActionType: "step",
		Message:    "test step",
	}

	// Convert to AST
	astStmt, err := ToAST(domainAction)
	if err != nil {
		t.Fatalf("ToAST() error = %v", err)
	}

	astAction, ok := astStmt.(*ast.ActionStatement)
	if !ok {
		t.Fatalf("Expected *ast.ActionStatement, got %T", astStmt)
	}

	if astAction.Action != "step" {
		t.Errorf("Action = %v, want step", astAction.Action)
	}
	if astAction.Message != "test step" {
		t.Errorf("Message = %v, want test step", astAction.Message)
	}

	// Convert back to domain
	domainStmt2, err := FromAST(astAction)
	if err != nil {
		t.Fatalf("FromAST() error = %v", err)
	}

	action2, ok := domainStmt2.(*Action)
	if !ok {
		t.Fatalf("Expected *Action, got %T", domainStmt2)
	}

	if action2.ActionType != domainAction.ActionType {
		t.Errorf("Round trip failed: ActionType = %v, want %v", action2.ActionType, domainAction.ActionType)
	}
	if action2.Message != domainAction.Message {
		t.Errorf("Round trip failed: Message = %v, want %v", action2.Message, domainAction.Message)
	}
}
