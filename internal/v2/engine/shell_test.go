package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_ShellExecution(t *testing.T) {
	input := `version: 2.0

task "shell demo":
  info "Starting shell command demo"
  run "echo 'Hello from shell!'"
  exec "date +%Y-%m-%d"
  shell "ls -la /tmp | head -3"
  success "Shell commands completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "shell demo")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that shell commands were executed
	expectedParts := []string{
		"ℹ️  Starting shell command demo",
		"🏃 Running: echo 'Hello from shell!'",
		"Hello from shell!",
		"⚡ Executing: date +%Y-%m-%d",
		"🐚 Shell: ls -la /tmp | head -3",
		"✅ Command completed successfully",
		"✅ Shell commands completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_CaptureCommand(t *testing.T) {
	input := `version: 2.0

task "capture demo":
  info "Testing command capture"
  capture "echo 'captured output'" as result
  info "Captured value: {result}"
  success "Capture test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "capture demo")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that capture worked
	expectedParts := []string{
		"ℹ️  Testing command capture",
		"📥 Capturing: echo 'captured output'",
		"📦 Captured output in variable 'result'",
		"ℹ️  Captured value: captured output",
		"✅ Capture test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ShellWithParameters(t *testing.T) {
	input := `version: 2.0

task "shell with params":
  requires message
  given count defaults to "3"
  
  info "Shell command with parameters"
  run "echo '{message}' | head -{count}"
  success "Parameterized shell completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"message": "Hello World",
		"count":   "1",
	}

	err = engine.ExecuteWithParams(program, "shell with params", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that parameters were interpolated in shell commands
	expectedParts := []string{
		"ℹ️  Shell command with parameters",
		"🏃 Running: echo 'Hello World' | head -1",
		"Hello World",
		"✅ Parameterized shell completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ShellDryRun(t *testing.T) {
	input := `version: 2.0

task "dry run test":
  info "Testing dry run mode"
  run "echo 'this should not execute'"
  capture "date" as current_date
  exec "rm -rf /important/files"
  success "Dry run completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.Execute(program, "dry run test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that commands were not actually executed
	expectedParts := []string{
		"[DRY RUN] Would execute task: dry run test",
		"[DRY RUN] info: Testing dry run mode",
		"[DRY RUN] Would execute shell command: echo 'this should not execute'",
		"[DRY RUN] Would execute shell command: date",
		"[DRY RUN] Would capture output as: current_date",
		"[DRY RUN] Would execute shell command: rm -rf /important/files",
		"[DRY RUN] success: Dry run completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Make sure actual command execution indicators are not present
	unexpectedParts := []string{
		"🏃 Running:",
		"⚡ Executing:",
		"📥 Capturing:",
		"✅ Command completed successfully",
	}

	for _, part := range unexpectedParts {
		if strings.Contains(outputStr, part) {
			t.Errorf("Did not expect output to contain %q in dry run mode, got %q", part, outputStr)
		}
	}
}

func TestEngine_ShellWithConditionals(t *testing.T) {
	input := `version: 2.0

task "conditional shell":
  requires environment from ["dev", "prod"]
  
  info "Conditional shell execution"
  
  when environment is "dev":
    run "echo 'Development environment detected'"
    exec "echo 'Running dev-specific commands'"
  
  when environment is "prod":
    run "echo 'Production environment detected'"
    exec "echo 'Running production commands'"
  
  success "Conditional shell completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	// Test dev environment
	var devOutput bytes.Buffer
	devEngine := NewEngine(&devOutput)

	devParams := map[string]string{
		"environment": "dev",
	}

	err = devEngine.ExecuteWithParams(program, "conditional shell", devParams)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed for dev: %v", err)
	}

	devOutputStr := devOutput.String()

	// Check dev-specific output
	if !strings.Contains(devOutputStr, "Development environment detected") {
		t.Errorf("Expected dev output to contain development message")
	}
	if !strings.Contains(devOutputStr, "Running dev-specific commands") {
		t.Errorf("Expected dev output to contain dev commands")
	}
	if strings.Contains(devOutputStr, "Production environment detected") {
		t.Errorf("Did not expect dev output to contain production message")
	}

	// Test prod environment
	var prodOutput bytes.Buffer
	prodEngine := NewEngine(&prodOutput)

	prodParams := map[string]string{
		"environment": "prod",
	}

	err = prodEngine.ExecuteWithParams(program, "conditional shell", prodParams)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed for prod: %v", err)
	}

	prodOutputStr := prodOutput.String()

	// Check prod-specific output
	if !strings.Contains(prodOutputStr, "Production environment detected") {
		t.Errorf("Expected prod output to contain production message")
	}
	if !strings.Contains(prodOutputStr, "Running production commands") {
		t.Errorf("Expected prod output to contain production commands")
	}
	if strings.Contains(prodOutputStr, "Development environment detected") {
		t.Errorf("Did not expect prod output to contain development message")
	}
}

func TestEngine_CaptureAndUse(t *testing.T) {
	input := `version: 2.0

task "capture and use":
  info "Testing capture and variable usage"
  capture "echo 'test-value-123'" as test_var
  capture "date +%Y" as current_year
  info "Test variable: {test_var}"
  info "Current year: {current_year}"
  run "echo 'Using captured: {test_var} in {current_year}'"
  success "Capture and use completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "capture and use")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that captured variables were used correctly
	expectedParts := []string{
		"ℹ️  Testing capture and variable usage",
		"📥 Capturing: echo 'test-value-123'",
		"📦 Captured output in variable 'test_var'",
		"📥 Capturing: date +%Y",
		"📦 Captured output in variable 'current_year'",
		"ℹ️  Test variable: test-value-123",
		"ℹ️  Current year: 202", // Should contain current year starting with 202x
		"🏃 Running: echo 'Using captured: test-value-123 in 202",
		"Using captured: test-value-123 in 202",
		"✅ Capture and use completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
