package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_DetectProjectType(t *testing.T) {
	input := `version: 2.0

task "analyze_project":
  detect project type
  success "Project analysis completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "analyze_project", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Detected project types:",
		"✅ Project analysis completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DetectToolVersion(t *testing.T) {
	input := `version: 2.0

task "check_go_version":
  detect go version
  success "Go version detected!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "check_go_version", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Detected go version:",
		"✅ Go version detected!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_IfToolAvailable(t *testing.T) {
	input := `version: 2.0

task "docker_check":
  if docker is available:
    info "Docker is available for use"
  else:
    warn "Docker is not available"
    
  success "Docker availability check completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "docker_check", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Checking if docker is available:",
		"✅ Docker availability check completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should contain either the if or else branch
	if !strings.Contains(outputStr, "Docker is available for use") &&
		!strings.Contains(outputStr, "Docker is not available") {
		t.Errorf("Expected output to contain either if or else branch execution")
	}
}

func TestEngine_IfVersionComparison(t *testing.T) {
	input := `version: 2.0

task "node_version_check":
  if node version >= "16":
    info "Node version is 16 or higher"
  else:
    warn "Node version is below 16"
    
  success "Node version check completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "node_version_check", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Checking node version",
		"✅ Node version check completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should contain either the if or else branch
	if !strings.Contains(outputStr, "Node version is 16 or higher") &&
		!strings.Contains(outputStr, "Node version is below 16") {
		t.Errorf("Expected output to contain either if or else branch execution")
	}
}

func TestEngine_WhenEnvironment(t *testing.T) {
	input := `version: 2.0

task "ci_specific_task":
  when in ci environment:
    info "Running in CI environment"
    warn "Using CI-specific configuration"
    
  success "Environment check completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "ci_specific_task", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Checking if in ci environment:",
		"✅ Environment check completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_MultipleDetectionStatements(t *testing.T) {
	input := `version: 2.0

task "comprehensive_check":
  detect project type
  if docker is available:
    info "Docker detected"
  when in local environment:
    info "Running locally"
  if go version >= "1.20":
    info "Go 1.20+ detected"
    
  success "All checks completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "comprehensive_check", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Detected project types:",
		"🔍 Checking if docker is available:",
		"🔍 Checking if in local environment:",
		"🔍 Checking go version",
		"✅ All checks completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DetectionDryRun(t *testing.T) {
	input := `version: 2.0

task "detection_test":
  detect project type
  if docker is available:
    info "Docker is available"
  if node version >= "16":
    info "Node 16+ available"
  when in ci environment:
    info "Running in CI"
    
  success "Detection test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.ExecuteWithParams(program, "detection_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] Would detect project types:",
		"[DRY RUN] Would check if docker is available:",
		"[DRY RUN] Would check if node version",
		"[DRY RUN] Would check if in ci environment:",
		"[DRY RUN] success: Detection test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_DetectionWithVariableInterpolation(t *testing.T) {
	input := `version: 2.0

task "version_check":
  requires min_version from ["16", "18", "20"]
  
  if node version >= "{min_version}":
    info "Node version meets minimum requirement of {min_version}"
  else:
    warn "Node version is below minimum requirement of {min_version}"
    
  success "Version check with parameter completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"min_version": "18",
	}

	err = engine.ExecuteWithParams(program, "version_check", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Checking node version",
		">= 18:",
		"✅ Version check with parameter completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should contain either the if or else branch with interpolated version
	if !strings.Contains(outputStr, "minimum requirement of 18") {
		t.Errorf("Expected output to contain interpolated version parameter")
	}
}
