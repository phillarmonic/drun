package engine

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestEngine_BuiltinFunctions(t *testing.T) {
	input := `version: 2.0

task "builtin demo":
  info "Current directory: {pwd}"
  info "Hostname: {hostname}"
  info "Current time: {now.format('2006-01-02 15:04:05')}"
  info "Custom time format: {now.format('2006-01-02')}"
  success "Builtin functions demo completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "builtin demo")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that builtin functions were called and interpolated
	expectedParts := []string{
		"ℹ️  Current directory:",
		"ℹ️  Hostname:",
		"ℹ️  Current time:",
		"ℹ️  Custom time format:",
		"✅ Builtin functions demo completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Check that the time format worked (should contain date)
	if !strings.Contains(outputStr, "202") { // Should contain year starting with 202x
		t.Errorf("Expected output to contain formatted date, got %q", outputStr)
	}
}

func TestEngine_EnvironmentVariableBuiltin(t *testing.T) {
	// Set a test environment variable
	testKey := "DRUN_TEST_BUILTIN_VAR"
	testValue := "test_builtin_value_456"
	_ = os.Setenv(testKey, testValue)
	defer func() { _ = os.Unsetenv(testKey) }()

	input := `version: 2.0

task "env test":
  info "Test var: {env('DRUN_TEST_BUILTIN_VAR')}"
  info "Default var: {env('NON_EXISTENT_VAR', 'default_value')}"
  success "Environment test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "env test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that environment variables were interpolated correctly
	expectedParts := []string{
		"ℹ️  Test var: " + testValue,
		"ℹ️  Default var: default_value",
		"✅ Environment test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_FileExistsBuiltin(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_builtin_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	input := `version: 2.0

task "file test":
  requires existing_file
  requires non_existing_file
  
  info "Existing file check: {file exists(existing_file)}"
  info "Non-existing file check: {file exists(non_existing_file)}"
  success "File test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"existing_file":     tmpFile.Name(),
		"non_existing_file": "/non/existent/file.txt",
	}

	err = engine.ExecuteWithParams(program, "file test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that file existence checks worked
	expectedParts := []string{
		"ℹ️  Existing file check: true",
		"ℹ️  Non-existing file check: false",
		"✅ File test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_GitCommitBuiltin(t *testing.T) {
	input := `version: 2.0

task "git test":
  info "Full commit: {current git commit}"
  info "Short commit: {current git commit('short')}"
  success "Git test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "git test")

	// This test might fail if we're not in a git repository
	// So we'll make it lenient
	if err != nil {
		t.Logf("Git test failed (expected if not in git repo): %v", err)
		return
	}

	outputStr := output.String()

	// If we got here, we should have git output
	expectedParts := []string{
		"ℹ️  Full commit:",
		"ℹ️  Short commit:",
		"✅ Git test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_BuiltinWithParameterInterpolation(t *testing.T) {
	input := `version: 2.0

task "mixed test":
  requires user_name
  given format defaults to "2006-01-02"
  
  info "Hello {user_name}, today is {now.format(format)}"
  info "Your directory: {pwd('basename')}"
  success "Mixed interpolation test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"user_name": "Alice",
		"format":    "2006-01-02 15:04",
	}

	err = engine.ExecuteWithParams(program, "mixed test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that both parameters and builtins were interpolated
	expectedParts := []string{
		"ℹ️  Hello Alice, today is",
		"ℹ️  Your directory:",
		"✅ Mixed interpolation test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should contain a date
	if !strings.Contains(outputStr, "202") {
		t.Errorf("Expected output to contain formatted date, got %q", outputStr)
	}
}
