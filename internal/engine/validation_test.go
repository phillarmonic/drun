package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_ParameterConstraintValidation(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires environment from ["dev", "staging", "production"]
  
  step "Deploying to {environment}"
  success "Deployment completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test valid constraint value
	params := map[string]string{
		"environment": "staging",
	}

	err = engine.ExecuteWithParams(program, "deploy", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed with valid constraint: %v", err)
	}

	expectedOutput := "🚀 Deploying to staging\n✅ Deployment completed!\n"
	if output.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output.String())
	}
}

func TestEngine_InvalidConstraintValue(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires environment from ["dev", "staging", "production"]
  
  step "Deploying to {environment}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test invalid constraint value
	params := map[string]string{
		"environment": "testing", // Not in the allowed list
	}

	err = engine.ExecuteWithParams(program, "deploy", params)
	if err == nil {
		t.Fatal("Expected error for invalid constraint value, got nil")
	}

	expectedError := "parameter 'environment': value 'testing' is not in allowed values: [dev staging production]"
	if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
}

func TestEngine_MultipleConstraintValues(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires environment from ["dev", "staging", "production"]
  
  step "Deploying to {environment}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test all valid constraint values
	validValues := []string{"dev", "staging", "production"}

	for _, value := range validValues {
		output.Reset()
		params := map[string]string{
			"environment": value,
		}

		err = engine.ExecuteWithParams(program, "deploy", params)
		if err != nil {
			t.Fatalf("ExecuteWithParams failed with valid constraint '%s': %v", value, err)
		}

		expectedOutput := "🚀 Deploying to " + value + "\n"
		if !strings.Contains(output.String(), expectedOutput) {
			t.Errorf("Expected output to contain %q, got %q", expectedOutput, output.String())
		}
	}
}

func TestEngine_NoConstraints(t *testing.T) {
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

	// Test parameter without constraints (should accept any value)
	params := map[string]string{
		"name": "AnyValueShouldWork123!@#",
	}

	err = engine.ExecuteWithParams(program, "greet", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed with unconstrained parameter: %v", err)
	}

	expectedOutput := "ℹ️  Hello, AnyValueShouldWork123!@#!\n"
	if output.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output.String())
	}
}

func TestEngine_DefaultValueUsage(t *testing.T) {
	input := `version: 2.0

task "deploy":
  given environment defaults to "dev"
  
  step "Deploying to {environment}"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test with no parameters (should use default value)
	params := map[string]string{}

	err = engine.ExecuteWithParams(program, "deploy", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed with default value: %v", err)
	}

	expectedOutput := "🚀 Deploying to dev\n"
	if output.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output.String())
	}
}
