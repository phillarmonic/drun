package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_WhenStatement(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires environment
  
  step "Starting deployment"
  
  when environment is "production":
    warn "Deploying to production!"
    step "Extra validation"
  
  success "Deployment completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with production environment (should execute when block)
	params := map[string]string{
		"environment": "production",
	}

	err = engine.ExecuteWithParams(program, "deploy", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()
	expectedParts := []string{
		"🚀 Starting deployment",
		"⚠️  Deploying to production!",
		"🚀 Extra validation",
		"✅ Deployment completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_WhenStatementNotExecuted(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires environment
  
  step "Starting deployment"
  
  when environment is "production":
    warn "Deploying to production!"
    step "Extra validation"
  
  success "Deployment completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with dev environment (should NOT execute when block)
	params := map[string]string{
		"environment": "dev",
	}

	err = engine.ExecuteWithParams(program, "deploy", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Should contain these
	expectedParts := []string{
		"🚀 Starting deployment",
		"✅ Deployment completed!",
	}

	// Should NOT contain these
	notExpectedParts := []string{
		"⚠️  Deploying to production!",
		"🚀 Extra validation",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	for _, part := range notExpectedParts {
		if strings.Contains(outputStr, part) {
			t.Errorf("Expected output to NOT contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_IfElseStatement(t *testing.T) {
	input := `version: 2.0

task "test":
  given skip_tests defaults to "false"
  
  if skip_tests is "false":
    step "Running tests"
    info "Tests passed"
  else:
    warn "Skipping tests"
  
  success "Done!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with skip_tests=false (should execute if block)
	params := map[string]string{
		"skip_tests": "false",
	}

	err = engine.ExecuteWithParams(program, "test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()
	expectedParts := []string{
		"🚀 Running tests",
		"ℹ️  Tests passed",
		"✅ Done!",
	}

	notExpectedParts := []string{
		"⚠️  Skipping tests",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	for _, part := range notExpectedParts {
		if strings.Contains(outputStr, part) {
			t.Errorf("Expected output to NOT contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_IfElseStatementElseBranch(t *testing.T) {
	input := `version: 2.0

task "test":
  given skip_tests defaults to "false"
  
  if skip_tests is "false":
    step "Running tests"
    info "Tests passed"
  else:
    warn "Skipping tests"
  
  success "Done!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with skip_tests=true (should execute else block)
	params := map[string]string{
		"skip_tests": "true",
	}

	err = engine.ExecuteWithParams(program, "test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()
	expectedParts := []string{
		"⚠️  Skipping tests",
		"✅ Done!",
	}

	notExpectedParts := []string{
		"🚀 Running tests",
		"ℹ️  Tests passed",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	for _, part := range notExpectedParts {
		if strings.Contains(outputStr, part) {
			t.Errorf("Expected output to NOT contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_SimpleLoop(t *testing.T) {
	input := `version: 2.0

task "process":
  accepts files as list
  
  step "Processing files"
  
  for each item in files:
    step "Processing: {item}"
    info "Completed: {item}"
  
  success "All files processed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with a list of files
	params := map[string]string{
		"files": "file1.txt,file2.txt,file3.txt",
	}

	err = engine.ExecuteWithParams(program, "process", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()
	expectedParts := []string{
		"🚀 Processing files",
		"🚀 Processing: file1.txt",
		"ℹ️  Completed: file1.txt",
		"🚀 Processing: file2.txt",
		"ℹ️  Completed: file2.txt",
		"🚀 Processing: file3.txt",
		"ℹ️  Completed: file3.txt",
		"✅ All files processed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
