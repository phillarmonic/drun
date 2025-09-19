package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_GlobalsNamespace(t *testing.T) {
	input := `version: 2.0

project "test-app" version "2.1.0":
  set api_url to "https://api.example.com"
  set timeout to "30s"
  set debug_mode to "true"

task "test_globals":
  info "Project: {$globals.project}"
  info "Version: {$globals.version}"
  info "API URL: {$globals.api_url}"
  info "Timeout: {$globals.timeout}"
  info "Debug: {$globals.debug_mode}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "test_globals")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedLines := []string{
		"ℹ️  Project: test-app",
		"ℹ️  Version: 2.1.0",
		"ℹ️  API URL: https://api.example.com",
		"ℹ️  Timeout: 30s",
		"ℹ️  Debug: true",
	}

	outputLines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(outputLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d lines\nOutput: %s", len(expectedLines), len(outputLines), output.String())
	}

	for i, expected := range expectedLines {
		if outputLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, outputLines[i])
		}
	}
}

func TestEngine_GlobalsVsTaskVariables(t *testing.T) {
	input := `version: 2.0

project "conflict-test" version "1.0.0":
  set api_url to "https://global-api.com"
  set env to "production"

task "test_scoping":
  set $api_url to "https://task-api.com"
  set $local_var to "task-only"
  
  info "Global API: {$globals.api_url}"
  info "Task API: {$api_url}"
  info "Global env: {$globals.env}"
  info "Local var: {$local_var}"
  info "Project name: {$globals.project}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "test_scoping")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedLines := []string{
		"📝 Set variable $api_url to https://task-api.com",
		"📝 Set variable $local_var to task-only",
		"ℹ️  Global API: https://global-api.com",
		"ℹ️  Task API: https://task-api.com",
		"ℹ️  Global env: production",
		"ℹ️  Local var: task-only",
		"ℹ️  Project name: conflict-test",
	}

	outputLines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(outputLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d lines\nOutput: %s", len(expectedLines), len(outputLines), output.String())
	}

	for i, expected := range expectedLines {
		if outputLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, outputLines[i])
		}
	}
}

func TestEngine_GlobalsNonExistentKey(t *testing.T) {
	input := `version: 2.0

project "test-app" version "1.0.0":
  set existing_key to "value"

task "test_missing":
  info "Existing: {$globals.existing_key}"
  info "Missing: {$globals.non_existent_key}"
  info "Also missing: {$globals.another_missing}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "test_missing")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedLines := []string{
		"ℹ️  Existing: value",
		"ℹ️  Missing: {$globals.non_existent_key}",
		"ℹ️  Also missing: {$globals.another_missing}",
	}

	outputLines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(outputLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d lines\nOutput: %s", len(expectedLines), len(outputLines), output.String())
	}

	for i, expected := range expectedLines {
		if outputLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, outputLines[i])
		}
	}
}

func TestEngine_GlobalsWithoutProject(t *testing.T) {
	input := `version: 2.0

task "test_no_project":
  info "Should not resolve: {$globals.anything}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "test_no_project")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expected := "ℹ️  Should not resolve: {$globals.anything}"
	output_str := strings.TrimSpace(output.String())

	if output_str != expected {
		t.Errorf("Expected %q, got %q", expected, output_str)
	}
}

func TestEngine_GlobalsInDryRun(t *testing.T) {
	input := `version: 2.0

project "dry-test" version "3.0.0":
  set api_endpoint to "https://dry-run-api.com"

task "test_dry_globals":
  info "Testing {$globals.project} v{$globals.version}"
  step "Connecting to {$globals.api_endpoint}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.Execute(program, "test_dry_globals")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedLines := []string{
		"[DRY RUN] Execution order: [test_dry_globals]",
		"[DRY RUN] Would execute task: test_dry_globals",
		"[DRY RUN] info: Testing dry-test v3.0.0",
		"[DRY RUN] step: Connecting to https://dry-run-api.com",
	}

	outputLines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(outputLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d lines\nOutput: %s", len(expectedLines), len(outputLines), output.String())
	}

	for i, expected := range expectedLines {
		if outputLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, outputLines[i])
		}
	}
}
