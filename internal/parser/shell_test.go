package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_ShellStatements(t *testing.T) {
	input := `version: 2.0

task "shell test":
  info "Starting"
  run "echo 'hello'"
  exec "date"
  shell "pwd"
  capture from shell "whoami" as $user
  success "Done"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "shell test" {
		t.Errorf("Expected task name 'shell test', got %s", task.Name)
	}

	// Should have 6 statements: info, run, exec, shell, capture, success
	if len(task.Body) != 6 {
		t.Fatalf("Expected 6 statements in task body, got %d", len(task.Body))
	}

	// Check each statement type
	statements := []struct {
		index    int
		expected string
		isShell  bool
	}{
		{0, "info", false},
		{1, "run", true},
		{2, "exec", true},
		{3, "shell", true},
		{4, "capture", true},
		{5, "success", false},
	}

	for _, stmt := range statements {
		if stmt.expected == "capture" {
			// Capture is now a variable statement
			varStmt, ok := task.Body[stmt.index].(*ast.VariableStatement)
			if !ok {
				t.Errorf("Expected statement %d to be VariableStatement for capture, got %T", stmt.index, task.Body[stmt.index])
				continue
			}
			if varStmt.Operation != "capture_shell" {
				t.Errorf("Expected operation 'capture_shell', got %s", varStmt.Operation)
			}
		} else if stmt.isShell {
			shellStmt, ok := task.Body[stmt.index].(*ast.ShellStatement)
			if !ok {
				t.Errorf("Expected statement %d to be ShellStatement, got %T", stmt.index, task.Body[stmt.index])
				continue
			}
			if shellStmt.Action != stmt.expected {
				t.Errorf("Expected shell action %s, got %s", stmt.expected, shellStmt.Action)
			}
		} else {
			actionStmt, ok := task.Body[stmt.index].(*ast.ActionStatement)
			if !ok {
				t.Errorf("Expected statement %d to be ActionStatement, got %T", stmt.index, task.Body[stmt.index])
				continue
			}
			if actionStmt.Action != stmt.expected {
				t.Errorf("Expected action %s, got %s", stmt.expected, actionStmt.Action)
			}
		}
	}

	// Check capture statement specifically
	captureStmt, ok := task.Body[4].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("Expected capture statement to be VariableStatement")
	}
	if captureStmt.Variable != "$user" {
		t.Errorf("Expected capture variable '$user', got %s", captureStmt.Variable)
	}
}

func TestParser_ShellStatementTypes(t *testing.T) {
	tests := []struct {
		input          string
		expectedAction string
		expectedCmd    string
		expectedVar    string
		streamOutput   bool
	}{
		{`run "echo hello"`, "run", "echo hello", "", true},
		{`exec "date"`, "exec", "date", "", true},
		{`shell "pwd"`, "shell", "pwd", "", true},
		{`capture from shell "whoami" as $user`, "capture", "whoami", "user", false},
	}

	for _, test := range tests {
		input := `version: 2.0

task "test":
  ` + test.input

		l := lexer.NewLexer(input)
		p := NewParser(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors for %s: %v", test.input, p.Errors())
		}

		task := program.Tasks[0]
		if len(task.Body) != 1 {
			t.Fatalf("Expected 1 statement, got %d", len(task.Body))
		}

		if test.expectedAction == "capture" {
			// Capture is now a variable statement
			varStmt, ok := task.Body[0].(*ast.VariableStatement)
			if !ok {
				t.Fatalf("Expected VariableStatement for capture, got %T", task.Body[0])
			}

			if varStmt.Operation != "capture_shell" {
				t.Errorf("Expected operation 'capture_shell', got %s", varStmt.Operation)
			}

			if varStmt.Variable != "$"+test.expectedVar {
				t.Errorf("Expected capture var $%s, got %s", test.expectedVar, varStmt.Variable)
			}

			literalExpr, ok := varStmt.Value.(*ast.LiteralExpression)
			if !ok {
				t.Fatalf("Expected LiteralExpression for capture value, got %T", varStmt.Value)
			}

			if literalExpr.Value != test.expectedCmd {
				t.Errorf("Expected command %s, got %s", test.expectedCmd, literalExpr.Value)
			}
		} else {
			// Regular shell statement
			shellStmt, ok := task.Body[0].(*ast.ShellStatement)
			if !ok {
				t.Fatalf("Expected ShellStatement, got %T", task.Body[0])
			}

			if shellStmt.Action != test.expectedAction {
				t.Errorf("Expected action %s, got %s", test.expectedAction, shellStmt.Action)
			}

			if shellStmt.Command != test.expectedCmd {
				t.Errorf("Expected command %s, got %s", test.expectedCmd, shellStmt.Command)
			}

			if shellStmt.StreamOutput != test.streamOutput {
				t.Errorf("Expected stream output %v, got %v", test.streamOutput, shellStmt.StreamOutput)
			}
		}
	}
}
