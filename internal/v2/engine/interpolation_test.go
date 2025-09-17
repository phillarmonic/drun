package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_VariableInterpolation(t *testing.T) {
	input := `version: 2.0

task "greet":
  requires name
  given title defaults to "friend"
  
  info "Hello, {title} {name}! Nice to meet you."
  step "Processing greeting for {name}"
  success "Greeting completed for {title} {name}!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with provided parameters
	params := map[string]string{
		"name":  "Alice",
		"title": "Ms.",
	}

	err = engine.ExecuteWithParams(program, "greet", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	expectedLines := []string{
		"ℹ️  Hello, Ms. Alice! Nice to meet you.",
		"🚀 Processing greeting for Alice",
		"✅ Greeting completed for Ms. Alice!",
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

func TestEngine_DefaultParameters(t *testing.T) {
	input := `version: 2.0

task "greet":
  requires name
  given title defaults to "friend"
  
  info "Hello, {title} {name}!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with only required parameter (title should use default)
	params := map[string]string{
		"name": "Bob",
	}

	err = engine.ExecuteWithParams(program, "greet", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	expected := "ℹ️  Hello, friend Bob!"
	actual := strings.TrimSpace(output.String())

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

func TestEngine_MissingRequiredParameter(t *testing.T) {
	input := `version: 2.0

task "greet":
  requires name
  
  info "Hello, {name}!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test without required parameter
	params := map[string]string{}

	err = engine.ExecuteWithParams(program, "greet", params)
	if err == nil {
		t.Fatal("Expected error for missing required parameter, got nil")
	}

	expectedError := "required parameter 'name' not provided"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

func TestEngine_InterpolationDryRun(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires environment
  
  step "Deploying to {environment}"
  success "Deployment to {environment} completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	params := map[string]string{
		"environment": "staging",
	}

	err = engine.ExecuteWithParams(program, "deploy", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()
	expectedParts := []string{
		"[DRY RUN] Would execute task: deploy",
		"[DRY RUN] step: Deploying to staging",
		"[DRY RUN] success: Deployment to staging completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
