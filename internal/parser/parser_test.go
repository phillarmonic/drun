package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_HelloWorld(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello from drun v2! 👋"

task "hello world":
  step "Starting hello world example"
  info "Welcome to the semantic task runner!"
  success "Hello world completed successfully!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	// Check version
	if program.Version == nil {
		t.Fatalf("program.Version is nil")
	}

	if program.Version.Value != "2.0" {
		t.Errorf("program.Version.Value wrong. expected=2.0, got=%s", program.Version.Value)
	}

	// Check tasks
	if len(program.Tasks) != 2 {
		t.Fatalf("program.Tasks does not contain 2 tasks. got=%d", len(program.Tasks))
	}

	// Check first task
	task1 := program.Tasks[0]
	if task1.Name != "hello" {
		t.Errorf("task1.Name wrong. expected=hello, got=%s", task1.Name)
	}

	if len(task1.Body) != 1 {
		t.Errorf("task1.Body wrong length. expected=1, got=%d", len(task1.Body))
	}

	action1, ok := task1.Body[0].(*ast.ActionStatement)
	if !ok {
		t.Errorf("task1.Body[0] is not an ActionStatement")
	} else {
		if action1.Action != "info" {
			t.Errorf("task1.Body[0].Action wrong. expected=info, got=%s", action1.Action)
		}

		if action1.Message != "Hello from drun v2! 👋" {
			t.Errorf("task1.Body[0].Message wrong. expected='Hello from drun v2! 👋', got=%s", action1.Message)
		}
	}

	// Check second task
	task2 := program.Tasks[1]
	if task2.Name != "hello world" {
		t.Errorf("task2.Name wrong. expected='hello world', got=%s", task2.Name)
	}

	if len(task2.Body) != 3 {
		t.Errorf("task2.Body wrong length. expected=3, got=%d", len(task2.Body))
	}

	expectedActions := []struct {
		action  string
		message string
	}{
		{"step", "Starting hello world example"},
		{"info", "Welcome to the semantic task runner!"},
		{"success", "Hello world completed successfully!"},
	}

	for i, expected := range expectedActions {
		action, ok := task2.Body[i].(*ast.ActionStatement)
		if !ok {
			t.Errorf("task2.Body[%d] is not an ActionStatement", i)
			continue
		}

		if action.Action != expected.action {
			t.Errorf("task2.Body[%d].Action wrong. expected=%s, got=%s", i, expected.action, action.Action)
		}

		if action.Message != expected.message {
			t.Errorf("task2.Body[%d].Message wrong. expected=%s, got=%s", i, expected.message, action.Message)
		}
	}
}

func TestParser_TaskWithMeans(t *testing.T) {
	input := `version: 2.0

task "greet" means "Greet someone by name":
  info "Hello there!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("program.Tasks does not contain 1 task. got=%d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "greet" {
		t.Errorf("task.Name wrong. expected=greet, got=%s", task.Name)
	}

	if task.Description != "Greet someone by name" {
		t.Errorf("task.Description wrong. expected='Greet someone by name', got=%s", task.Description)
	}
}

func TestParser_BasicVersion(t *testing.T) {
	input := `version: 2.0`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if program.Version == nil {
		t.Fatalf("program.Version is nil")
	}

	if program.Version.Value != "2.0" {
		t.Errorf("program.Version.Value wrong. expected=2.0, got=%s", program.Version.Value)
	}

	if len(program.Tasks) != 0 {
		t.Errorf("program.Tasks should be empty. got=%d", len(program.Tasks))
	}
}

func TestParser_Errors(t *testing.T) {
	tests := []struct {
		input         string
		expectedError string
	}{
		{
			"task \"hello\":",
			"expected version statement",
		},
		{
			"version 2.0",
			"expected next token to be COLON",
		},
		{
			"version:",
			"expected next token to be NUMBER",
		},
	}

	for _, tt := range tests {
		lexer := lexer.NewLexer(tt.input)
		parser := NewParser(lexer)
		parser.ParseProgram()

		errors := parser.Errors()
		if len(errors) == 0 {
			t.Errorf("expected parser errors for input %q, got none", tt.input)
			continue
		}

		found := false
		for _, err := range errors {
			if containsString(err, tt.expectedError) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected error containing %q, got %v", tt.expectedError, errors)
		}
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParser_EmptyKeyword(t *testing.T) {
	input := `version: 2.0

task "test":
  given $features as list defaults to empty
  given $name defaults to ""
  
  if $features is empty:
    info "Features is empty"
    
  if $features is not empty:
    info "Features: {$features}"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	// Check version
	if program.Version == nil {
		t.Fatalf("program.Version is nil")
	}

	if program.Version.Value != "2.0" {
		t.Errorf("program.Version.Value wrong. expected=2.0, got=%s", program.Version.Value)
	}

	// Check tasks
	if len(program.Tasks) != 1 {
		t.Fatalf("program.Tasks does not contain 1 task. got=%d", len(program.Tasks))
	}

	// Check task
	task := program.Tasks[0]
	if task.Name != "test" {
		t.Errorf("task.Name wrong. expected=test, got=%s", task.Name)
	}

	// Check parameters
	if len(task.Parameters) != 2 {
		t.Fatalf("task.Parameters does not contain 2 parameters. got=%d", len(task.Parameters))
	}

	// Check first parameter (features with empty default)
	featuresParam := task.Parameters[0]
	if featuresParam.Name != "features" {
		t.Errorf("featuresParam.Name wrong. expected=features, got=%s", featuresParam.Name)
	}
	if featuresParam.DataType != "list" {
		t.Errorf("featuresParam.DataType wrong. expected=list, got=%s", featuresParam.DataType)
	}
	if featuresParam.DefaultValue != "" {
		t.Errorf("featuresParam.DefaultValue wrong. expected=empty string, got=%s", featuresParam.DefaultValue)
	}
	if featuresParam.Required {
		t.Errorf("featuresParam.Required wrong. expected=false, got=%t", featuresParam.Required)
	}

	// Check second parameter (name with empty string default)
	nameParam := task.Parameters[1]
	if nameParam.Name != "name" {
		t.Errorf("nameParam.Name wrong. expected=name, got=%s", nameParam.Name)
	}
	if nameParam.DefaultValue != "" {
		t.Errorf("nameParam.DefaultValue wrong. expected=empty string, got=%s", nameParam.DefaultValue)
	}
	if nameParam.Required {
		t.Errorf("nameParam.Required wrong. expected=false, got=%t", nameParam.Required)
	}

	// Check task body - should have 2 if statements
	if len(task.Body) != 2 {
		t.Fatalf("task.Body does not contain 2 statements. got=%d", len(task.Body))
	}

	// Check first if statement (is empty)
	firstIf, ok := task.Body[0].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("task.Body[0] is not a ConditionalStatement")
	}
	if firstIf.Type != "if" {
		t.Errorf("firstIf.Type wrong. expected=if, got=%s", firstIf.Type)
	}
	if firstIf.Condition != "$features is empty" {
		t.Errorf("firstIf.Condition wrong. expected='$features is empty', got=%s", firstIf.Condition)
	}

	// Check second if statement (is not empty)
	secondIf, ok := task.Body[1].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("task.Body[1] is not a ConditionalStatement")
	}
	if secondIf.Type != "if" {
		t.Errorf("secondIf.Type wrong. expected=if, got=%s", secondIf.Type)
	}
	if secondIf.Condition != "$features is not empty" {
		t.Errorf("secondIf.Condition wrong. expected='$features is not empty', got=%s", secondIf.Condition)
	}
}

func TestParser_EscapedQuotes(t *testing.T) {
	input := `version: 2.0

task "test with \"escaped\" quotes":
  info "This has \"quotes\" inside"
  run "echo \"Hello World\""`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	// Check that we have one task
	if len(program.Tasks) != 1 {
		t.Fatalf("program.Tasks does not contain 1 task. got=%d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "test with \"escaped\" quotes" {
		t.Errorf("task.Name wrong. expected='test with \"escaped\" quotes', got=%s", task.Name)
	}

	// Check the info statement
	infoStmt, ok := task.Body[0].(*ast.ActionStatement)
	if !ok {
		t.Fatalf("task.Body[0] is not an ActionStatement")
	}
	if infoStmt.Action != "info" {
		t.Errorf("infoStmt.Action wrong. expected=info, got=%s", infoStmt.Action)
	}
	if infoStmt.Message != "This has \"quotes\" inside" {
		t.Errorf("infoStmt.Message wrong. expected='This has \"quotes\" inside', got=%s", infoStmt.Message)
	}

	// Check the run statement
	runStmt, ok := task.Body[1].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("task.Body[1] is not a ShellStatement, got %T", task.Body[1])
	}
	if runStmt.Command != "echo \"Hello World\"" {
		t.Errorf("runStmt.Command wrong. expected='echo \"Hello World\"', got=%s", runStmt.Command)
	}
}
