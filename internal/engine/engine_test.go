package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_HelloWorld(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello from drun v2! 👋"

task "hello world":
  step "Starting hello world example"
  info "Welcome to the semantic task runner!"
  success "Hello world completed successfully!"`

	// Test executing the "hello" task
	var output bytes.Buffer
	err := ExecuteString(input, "hello", &output)
	if err != nil {
		t.Fatalf("ExecuteString failed: %v", err)
	}

	expectedOutput := "ℹ️  Hello from drun v2! 👋\n"
	if output.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output.String())
	}

	// Test executing the "hello world" task
	output.Reset()
	err = ExecuteString(input, "hello world", &output)
	if err != nil {
		t.Fatalf("ExecuteString failed: %v", err)
	}

	expectedLines := []string{
		"🚀 Starting hello world example",
		"ℹ️  Welcome to the semantic task runner!",
		"✅ Hello world completed successfully!",
	}

	outputLines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(outputLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d lines", len(expectedLines), len(outputLines))
	}

	for i, expected := range expectedLines {
		if outputLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, outputLines[i])
		}
	}
}

func TestEngine_TaskNotFound(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello!"`

	var output bytes.Buffer
	err := ExecuteString(input, "nonexistent", &output)
	if err == nil {
		t.Fatal("Expected error for nonexistent task, got nil")
	}

	expectedError := "task 'nonexistent' not found"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

func TestEngine_DryRun(t *testing.T) {
	input := `version: 2.0

task "test":
  step "This is a step"
  info "This is info"
  success "This is success"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.Execute(program, "test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()
	expectedParts := []string{
		"[DRY RUN] Would execute task: test",
		"[DRY RUN] step: This is a step",
		"[DRY RUN] info: This is info",
		"[DRY RUN] success: This is success",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ListTasks(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Simple task"

task "greet" means "Greet someone by name":
  step "Greeting process"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	engine := NewEngine(nil)
	tasks := engine.ListTasks(program)

	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	// Check first task
	if tasks[0].Name != "hello" {
		t.Errorf("Expected first task name 'hello', got %q", tasks[0].Name)
	}
	if tasks[0].Description != "No description" {
		t.Errorf("Expected first task description 'No description', got %q", tasks[0].Description)
	}

	// Check second task
	if tasks[1].Name != "greet" {
		t.Errorf("Expected second task name 'greet', got %q", tasks[1].Name)
	}
	if tasks[1].Description != "Greet someone by name" {
		t.Errorf("Expected second task description 'Greet someone by name', got %q", tasks[1].Description)
	}
}

func TestEngine_AllActions(t *testing.T) {
	input := `version: 2.0

task "test all actions":
  step "Starting test"
  info "This is information"
  warn "This is a warning"
  error "This is an error"
  success "This is success"`

	var output bytes.Buffer
	err := ExecuteString(input, "test all actions", &output)
	if err != nil {
		t.Fatalf("ExecuteString failed: %v", err)
	}

	expectedLines := []string{
		"🚀 Starting test",
		"ℹ️  This is information",
		"⚠️  This is a warning",
		"❌ This is an error",
		"✅ This is success",
	}

	outputLines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(outputLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d lines", len(expectedLines), len(outputLines))
	}

	for i, expected := range expectedLines {
		if outputLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, outputLines[i])
		}
	}
}

func TestEngine_FailAction(t *testing.T) {
	input := `version: 2.0

task "fail test":
  info "Before fail"
  fail "This task should fail"
  info "After fail"`

	var output bytes.Buffer
	err := ExecuteString(input, "fail test", &output)
	if err == nil {
		t.Fatal("Expected error from fail action, got nil")
	}

	expectedError := "task failed: This task should fail"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}

	// Should have executed the info before fail, but not after
	outputStr := output.String()
	if !strings.Contains(outputStr, "ℹ️  Before fail") {
		t.Error("Expected to see 'Before fail' in output")
	}
	if !strings.Contains(outputStr, "💥 This task should fail") {
		t.Error("Expected to see fail message in output")
	}
	if strings.Contains(outputStr, "After fail") {
		t.Error("Should not see 'After fail' in output since task should have failed")
	}
}

func TestEngine_ParseError(t *testing.T) {
	input := `version: 2.0

task "invalid"
  info "Missing colon"`

	var output bytes.Buffer
	err := ExecuteString(input, "invalid", &output)
	if err == nil {
		t.Fatal("Expected parse error, got nil")
	}

	if !strings.Contains(err.Error(), "parse errors") {
		t.Errorf("Expected parse error, got %q", err.Error())
	}
}
