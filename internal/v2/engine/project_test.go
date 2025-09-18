package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_ProjectDeclaration(t *testing.T) {
	input := `version: 2.0

project "myapp" version "1.0.0":
  set registry to "ghcr.io/company"
  set timeout to "5m"

task "deploy":
  info "Deploying {registry}/myapp:{version}"
  success "Deployment completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "deploy", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  Deploying ghcr.io/company/myapp:1.0.0",
		"✅ Deployment completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ProjectWithLifecycleHooks(t *testing.T) {
	input := `version: 2.0

project "webapp":
  set app_name to "webapp"
  
  before any task:
    info "🚀 Starting task execution for {app_name}"
  
  after any task:
    info "✅ Task execution completed for {app_name}"

task "build":
  info "Building {app_name}"
  success "Build completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "build", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  🚀 Starting task execution for webapp",
		"ℹ️  Building webapp",
		"✅ Build completed!",
		"ℹ️  ✅ Task execution completed for webapp",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Check execution order
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	if len(lines) < 4 {
		t.Fatalf("Expected at least 4 lines of output, got %d", len(lines))
	}

	// Before hook should be first
	if !strings.Contains(lines[0], "🚀 Starting task execution") {
		t.Errorf("Expected before hook to be first, got: %s", lines[0])
	}

	// After hook should be last
	if !strings.Contains(lines[len(lines)-1], "✅ Task execution completed") {
		t.Errorf("Expected after hook to be last, got: %s", lines[len(lines)-1])
	}
}

func TestEngine_ProjectDryRun(t *testing.T) {
	input := `version: 2.0

project "testapp":
  set environment to "production"
  
  before any task:
    info "Pre-deployment checks for {environment}"
  
  after any task:
    info "Post-deployment cleanup for {environment}"

task "deploy":
  info "Deploying to {environment}"
  success "Deployment to {environment} completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.ExecuteWithParams(program, "deploy", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] info: Pre-deployment checks for production",
		"[DRY RUN] Would execute task: deploy",
		"[DRY RUN] info: Deploying to production",
		"[DRY RUN] success: Deployment to production completed!",
		"[DRY RUN] info: Post-deployment cleanup for production",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ProjectWithoutHooks(t *testing.T) {
	input := `version: 2.0

project "simple":
  set name to "simple-app"

task "test":
  info "Testing {name}"
  success "Tests passed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  Testing simple-app",
		"✅ Tests passed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should not contain hook messages
	unexpectedParts := []string{
		"Starting task execution",
		"Task execution completed",
	}

	for _, part := range unexpectedParts {
		if strings.Contains(outputStr, part) {
			t.Errorf("Expected output to NOT contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_NoProject(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello without project!"
  success "Done!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "hello", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  Hello without project!",
		"✅ Done!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ProjectSettingsInParameters(t *testing.T) {
	input := `version: 2.0

project "config-app":
  set default_env to "development"
  set max_retries to "3"

task "configure":
  info "Default environment: {default_env}"
  info "Max retries: {max_retries}"
  success "Configuration loaded!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "configure", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  Default environment: development",
		"ℹ️  Max retries: 3",
		"✅ Configuration loaded!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
