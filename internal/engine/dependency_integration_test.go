package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_SimpleDependencyExecution(t *testing.T) {
	input := `version: 2.0

task "build":
  info "Building application"
  success "Build completed!"

task "deploy":
  depends on build
  
  info "Deploying application"
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

	// Check that build runs before deploy
	buildIdx := strings.Index(outputStr, "Building application")
	deployIdx := strings.Index(outputStr, "Deploying application")

	if buildIdx == -1 {
		t.Errorf("Expected build task to run")
	}
	if deployIdx == -1 {
		t.Errorf("Expected deploy task to run")
	}
	if buildIdx >= deployIdx {
		t.Errorf("Build task should run before deploy task")
	}

	expectedParts := []string{
		"ℹ️  Building application",
		"✅ Build completed!",
		"ℹ️  Deploying application",
		"✅ Deployment completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ChainedDependencies(t *testing.T) {
	input := `version: 2.0

task "install":
  info "Installing dependencies"

task "build":
  depends on install
  info "Building application"

task "test":
  depends on build
  info "Running tests"

task "deploy":
  depends on test
  info "Deploying application"`

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

	// Check execution order
	installIdx := strings.Index(outputStr, "Installing dependencies")
	buildIdx := strings.Index(outputStr, "Building application")
	testIdx := strings.Index(outputStr, "Running tests")
	deployIdx := strings.Index(outputStr, "Deploying application")

	if installIdx >= buildIdx || buildIdx >= testIdx || testIdx >= deployIdx {
		t.Errorf("Tasks should execute in order: install -> build -> test -> deploy")
	}
}

func TestEngine_ParallelDependencies(t *testing.T) {
	input := `version: 2.0

task "lint":
  info "Running linter"

task "test":
  info "Running tests"

task "security_scan":
  info "Running security scan"

task "deploy":
  depends on lint, test, security_scan
  info "Deploying application"`

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

	// All dependencies should run before deploy
	lintIdx := strings.Index(outputStr, "Running linter")
	testIdx := strings.Index(outputStr, "Running tests")
	securityIdx := strings.Index(outputStr, "Running security scan")
	deployIdx := strings.Index(outputStr, "Deploying application")

	if lintIdx >= deployIdx || testIdx >= deployIdx || securityIdx >= deployIdx {
		t.Errorf("All dependencies should run before deploy")
	}

	// Check that all tasks are executed
	expectedParts := []string{
		"ℹ️  Running linter",
		"ℹ️  Running tests",
		"ℹ️  Running security scan",
		"ℹ️  Deploying application",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q", part)
		}
	}
}

func TestEngine_ComplexDependencies(t *testing.T) {
	input := `version: 2.0

task "install":
  info "Installing dependencies"

task "build":
  depends on install
  info "Building application"

task "lint":
  depends on install
  info "Running linter"

task "test":
  depends on build
  info "Running tests"

task "security_scan":
  depends on lint
  info "Running security scan"

task "deploy":
  depends on test, security_scan
  info "Deploying application"`

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

	// Verify execution order constraints
	installIdx := strings.Index(outputStr, "Installing dependencies")
	buildIdx := strings.Index(outputStr, "Building application")
	lintIdx := strings.Index(outputStr, "Running linter")
	testIdx := strings.Index(outputStr, "Running tests")
	securityIdx := strings.Index(outputStr, "Running security scan")
	deployIdx := strings.Index(outputStr, "Deploying application")

	// install must come before build and lint
	if installIdx >= buildIdx || installIdx >= lintIdx {
		t.Errorf("install should come before build and lint")
	}

	// build must come before test
	if buildIdx >= testIdx {
		t.Errorf("build should come before test")
	}

	// lint must come before security_scan
	if lintIdx >= securityIdx {
		t.Errorf("lint should come before security_scan")
	}

	// test and security_scan must come before deploy
	if testIdx >= deployIdx || securityIdx >= deployIdx {
		t.Errorf("test and security_scan should come before deploy")
	}
}

func TestEngine_DependencyWithParameters(t *testing.T) {
	input := `version: 2.0

task "build":
  requires environment from ["dev", "staging", "production"]
  info "Building for {environment}"

task "deploy":
  depends on build
  requires environment from ["dev", "staging", "production"]
  info "Deploying to {environment}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "deploy", map[string]string{
		"environment": "staging",
	})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  Building for staging",
		"ℹ️  Deploying to staging",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_CircularDependencyError(t *testing.T) {
	input := `version: 2.0

task "build":
  depends on test
  info "Building application"

task "test":
  depends on build
  info "Running tests"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "build", map[string]string{})
	if err == nil {
		t.Fatalf("Expected circular dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

func TestEngine_MissingDependencyError(t *testing.T) {
	input := `version: 2.0

task "deploy":
  depends on build
  info "Deploying application"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "deploy", map[string]string{})
	if err == nil {
		t.Fatalf("Expected missing dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "non-existent task") {
		t.Errorf("Expected missing dependency error, got: %v", err)
	}
}

func TestEngine_DependencyDryRun(t *testing.T) {
	input := `version: 2.0

task "build":
  info "Building application"

task "deploy":
  depends on build
  info "Deploying application"`

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
		"[DRY RUN] Execution order: [build deploy]",
		"[DRY RUN] Would execute task: build",
		"[DRY RUN] info: Building application",
		"[DRY RUN] Would execute task: deploy",
		"[DRY RUN] info: Deploying application",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DependencyWithProjectHooks(t *testing.T) {
	input := `version: 2.0

project "webapp":
  before any task:
    info "🚀 Starting task execution"
  
  after any task:
    info "✅ Task execution completed"

task "build":
  info "Building application"

task "deploy":
  depends on build
  info "Deploying application"`

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

	// Hooks should only run for the target task (deploy), not dependencies
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")

	// Should have: hook, build, hook, deploy, hook
	expectedPattern := []string{
		"🚀 Starting task execution",
		"Building application",
		"✅ Task execution completed",
		"Deploying application",
	}

	for _, expected := range expectedPattern {
		found := false
		for _, line := range lines {
			if strings.Contains(line, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %q in output", expected)
		}
	}
}
