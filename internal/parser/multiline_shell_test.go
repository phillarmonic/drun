package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_MultilineShellStatements(t *testing.T) {
	input := `version: 2.0

task "multiline shell test":
  info "Starting multiline shell test"
  
  run:
    echo "First command"
    echo "Second command"
    pwd
  
  exec:
    ls -la
    date
  
  shell:
    export VAR=test
    echo $VAR
  
  capture as $result:
    echo "Captured output"
    whoami
    hostname
  
  success "Multiline shell test completed"`

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
	if task.Name != "multiline shell test" {
		t.Errorf("Expected task name 'multiline shell test', got %s", task.Name)
	}

	// Should have 6 statements: info, run, exec, shell, capture, success
	if len(task.Body) != 6 {
		t.Fatalf("Expected 6 statements in task body, got %d", len(task.Body))
	}

	// Check run statement (multiline)
	runStmt, ok := task.Body[1].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 1 to be ShellStatement, got %T", task.Body[1])
	}
	if runStmt.Action != "run" {
		t.Errorf("Expected action 'run', got %s", runStmt.Action)
	}
	if !runStmt.IsMultiline {
		t.Errorf("Expected IsMultiline to be true")
	}
	expectedRunCommands := []string{
		"echo \"First command\"",
		"echo \"Second command\"",
		"pwd",
	}
	if len(runStmt.Commands) != len(expectedRunCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedRunCommands), len(runStmt.Commands))
	}
	for i, expected := range expectedRunCommands {
		if i < len(runStmt.Commands) && runStmt.Commands[i] != expected {
			t.Errorf("Expected command %d to be %s, got %s", i, expected, runStmt.Commands[i])
		}
	}

	// Check exec statement (multiline)
	execStmt, ok := task.Body[2].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 2 to be ShellStatement, got %T", task.Body[2])
	}
	if execStmt.Action != "exec" {
		t.Errorf("Expected action 'exec', got %s", execStmt.Action)
	}
	if !execStmt.IsMultiline {
		t.Errorf("Expected IsMultiline to be true")
	}
	expectedExecCommands := []string{"ls -la", "date"}
	if len(execStmt.Commands) != len(expectedExecCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedExecCommands), len(execStmt.Commands))
	}

	// Check shell statement (multiline)
	shellStmt, ok := task.Body[3].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 3 to be ShellStatement, got %T", task.Body[3])
	}
	if shellStmt.Action != "shell" {
		t.Errorf("Expected action 'shell', got %s", shellStmt.Action)
	}
	if !shellStmt.IsMultiline {
		t.Errorf("Expected IsMultiline to be true")
	}

	// Check capture statement (multiline)
	captureStmt, ok := task.Body[4].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 4 to be ShellStatement, got %T", task.Body[4])
	}
	if captureStmt.Action != "capture" {
		t.Errorf("Expected action 'capture', got %s", captureStmt.Action)
	}
	if !captureStmt.IsMultiline {
		t.Errorf("Expected IsMultiline to be true")
	}
	if captureStmt.CaptureVar != "result" {
		t.Errorf("Expected capture variable 'result', got %s", captureStmt.CaptureVar)
	}
	expectedCaptureCommands := []string{
		"echo \"Captured output\"",
		"whoami",
		"hostname",
	}
	if len(captureStmt.Commands) != len(expectedCaptureCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCaptureCommands), len(captureStmt.Commands))
	}
}

func TestParser_MixedSingleLineAndMultilineShell(t *testing.T) {
	input := `version: 2.0

task "mixed shell test":
  run "single line command"
  
  run:
    echo "multiline command 1"
    echo "multiline command 2"
  
  capture "whoami" as $user
  
  capture as $info:
    echo "User: $(whoami)"
    echo "Date: $(date)"
  
  info "User is {$user}, info: {$info}"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]

	// Check single-line run
	runSingle, ok := task.Body[0].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 0 to be ShellStatement")
	}
	if runSingle.IsMultiline {
		t.Errorf("Expected single-line run to have IsMultiline=false")
	}
	if runSingle.Command != "single line command" {
		t.Errorf("Expected command 'single line command', got %s", runSingle.Command)
	}

	// Check multiline run
	runMulti, ok := task.Body[1].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 1 to be ShellStatement")
	}
	if !runMulti.IsMultiline {
		t.Errorf("Expected multiline run to have IsMultiline=true")
	}
	if len(runMulti.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(runMulti.Commands))
	}

	// Check single-line capture
	captureSingle, ok := task.Body[2].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 2 to be ShellStatement")
	}
	if captureSingle.IsMultiline {
		t.Errorf("Expected single-line capture to have IsMultiline=false")
	}
	if captureSingle.CaptureVar != "user" {
		t.Errorf("Expected capture variable 'user', got %s", captureSingle.CaptureVar)
	}

	// Check multiline capture
	captureMulti, ok := task.Body[3].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 3 to be ShellStatement")
	}
	if !captureMulti.IsMultiline {
		t.Errorf("Expected multiline capture to have IsMultiline=true")
	}
	if captureMulti.CaptureVar != "info" {
		t.Errorf("Expected capture variable 'info', got %s", captureMulti.CaptureVar)
	}
}

func TestParser_MultilineShellWithComments(t *testing.T) {
	input := `version: 2.0

task "shell with comments":
  run:
    # This is a comment
    echo "First command"
    # Another comment
    echo "Second command"
    # Final comment`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	runStmt, ok := task.Body[0].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 0 to be ShellStatement")
	}

	// Comments should be filtered out, only actual commands should remain
	expectedCommands := []string{
		"echo \"First command\"",
		"echo \"Second command\"",
	}
	if len(runStmt.Commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands (comments filtered), got %d", len(expectedCommands), len(runStmt.Commands))
	}
	for i, expected := range expectedCommands {
		if i < len(runStmt.Commands) && runStmt.Commands[i] != expected {
			t.Errorf("Expected command %d to be %s, got %s", i, expected, runStmt.Commands[i])
		}
	}
}

func TestParser_EmptyMultilineShell(t *testing.T) {
	input := `version: 2.0

task "empty multiline":
  run:
    # Only comments
    # No actual commands`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	runStmt, ok := task.Body[0].(*ast.ShellStatement)
	if !ok {
		t.Fatalf("Expected statement 0 to be ShellStatement")
	}

	// Should have no commands (only comments were present)
	if len(runStmt.Commands) != 0 {
		t.Errorf("Expected 0 commands (only comments), got %d", len(runStmt.Commands))
	}
}
