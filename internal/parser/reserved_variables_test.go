package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

// TestParser_ReservedVariableName_Let tests that reserved variable names are rejected in let statements
func TestParser_ReservedVariableName_Let(t *testing.T) {
	input := `version: 2.0

task "test":
  let $globals = "forbidden"
  info "test"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	// Should have errors
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected parser errors for reserved variable name, got none")
	}

	// Check that error message mentions reserved variable
	foundReservedError := false
	for _, err := range errors {
		if contains(err, "reserved variable name") && contains(err, "$globals") {
			foundReservedError = true
			break
		}
	}

	if !foundReservedError {
		t.Errorf("expected error about reserved variable name '$globals', got: %v", errors)
	}
}

// TestParser_ReservedVariableName_Set tests that reserved variable names are rejected in set statements
func TestParser_ReservedVariableName_Set(t *testing.T) {
	input := `version: 2.0

task "test":
  set $globals to "forbidden"
  info "test"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	// Should have errors
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected parser errors for reserved variable name, got none")
	}

	// Check that error message mentions reserved variable
	foundReservedError := false
	for _, err := range errors {
		if contains(err, "reserved variable name") && contains(err, "$globals") {
			foundReservedError = true
			break
		}
	}

	if !foundReservedError {
		t.Errorf("expected error about reserved variable name '$globals', got: %v", errors)
	}
}

// TestParser_ReservedVariableName_Capture tests that reserved variable names are rejected in capture statements
func TestParser_ReservedVariableName_Capture(t *testing.T) {
	input := `version: 2.0

task "test":
  capture from shell "echo test" as $globals
  info "test"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	// Should have errors
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected parser errors for reserved variable name, got none")
	}

	// Check that error message mentions reserved variable
	foundReservedError := false
	for _, err := range errors {
		if contains(err, "reserved variable name") && contains(err, "$globals") {
			foundReservedError = true
			break
		}
	}

	if !foundReservedError {
		t.Errorf("expected error about reserved variable name '$globals', got: %v", errors)
	}
}

// TestParser_ReservedVariableName_Transform tests that reserved variable names are rejected in transform statements
func TestParser_ReservedVariableName_Transform(t *testing.T) {
	input := `version: 2.0

task "test":
  let $value = "hello"
  transform $globals with uppercase
  info "test"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	// Should have errors
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected parser errors for reserved variable name, got none")
	}

	// Check that error message mentions reserved variable
	foundReservedError := false
	for _, err := range errors {
		if contains(err, "reserved variable name") && contains(err, "$globals") {
			foundReservedError = true
			break
		}
	}

	if !foundReservedError {
		t.Errorf("expected error about reserved variable name '$globals', got: %v", errors)
	}
}

// TestParser_AllowedVariableNames tests that non-reserved variable names work fine
func TestParser_AllowedVariableNames(t *testing.T) {
	input := `version: 2.0

task "test":
  let $global_config = "allowed"
  set $globals_copy to "also allowed"
  capture from shell "echo test" as $globalstuff
  info "All good"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatal("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 4 {
		t.Errorf("expected 4 statements in task body, got %d", len(task.Body))
	}
}

// TestParser_GlobalsAccessStillWorks tests that $globals.key syntax for accessing project settings still works
func TestParser_GlobalsAccessStillWorks(t *testing.T) {
	input := `version: 2.0

project "test":
  set api_url to "https://example.com"

task "test":
  info "URL: {$globals.api_url}"
  step "Testing {$globals.api_url}"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatal("ParseProgram() returned nil")
	}

	if program.Project == nil {
		t.Fatal("expected project declaration")
	}

	task := program.Tasks[0]
	if len(task.Body) != 2 {
		t.Errorf("expected 2 statements in task body, got %d", len(task.Body))
	}
}

// TestParser_ParamsAccessWorks tests that $params.key syntax for accessing project parameters works
func TestParser_ParamsAccessWorks(t *testing.T) {
	input := `version: 2.0

project "docker":
  parameter $registry as string defaults to "docker.io"
  parameter $namespace as string defaults to "mycompany"

task "test":
  info "Registry: {$params.registry}"
  info "Namespace: {$params.namespace}"
  step "Using {$params.registry}/{$params.namespace}"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatal("ParseProgram() returned nil")
	}

	if program.Project == nil {
		t.Fatal("expected project declaration")
	}

	task := program.Tasks[0]
	if len(task.Body) != 3 {
		t.Errorf("expected 3 statements in task body, got %d", len(task.Body))
	}
}

// TestParser_ReservedVariableName_Params tests that $params is protected
func TestParser_ReservedVariableName_Params(t *testing.T) {
	input := `version: 2.0

task "test":
  let $params = "forbidden"
  info "test"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	_ = p.ParseProgram()

	// Should have errors
	errors := p.Errors()
	if len(errors) == 0 {
		t.Fatal("expected parser errors for reserved variable name, got none")
	}

	// Check that error message mentions reserved variable
	foundReservedError := false
	for _, err := range errors {
		if contains(err, "reserved variable name") && contains(err, "$params") {
			foundReservedError = true
			break
		}
	}

	if !foundReservedError {
		t.Errorf("expected error about reserved variable name '$params', got: %v", errors)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
