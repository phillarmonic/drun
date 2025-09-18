package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_TypedParameters(t *testing.T) {
	input := `version: 2.0

task "typed params":
  requires count as number
  given enabled as boolean defaults to "true"
  accepts items as list
  
  info "Count: {count}"
  info "Enabled: {enabled}"
  info "Items: {items}"
  success "Typed parameters test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"count":   "42",
		"enabled": "false",
		"items":   "apple,banana,cherry",
	}

	err = engine.ExecuteWithParams(program, "typed params", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that typed parameters were processed correctly
	expectedParts := []string{
		"ℹ️  Count: 42",
		"ℹ️  Enabled: false",
		"ℹ️  Items: apple,banana,cherry",
		"✅ Typed parameters test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_NumberValidation(t *testing.T) {
	input := `version: 2.0

task "number test":
  requires port as number
  
  info "Port: {port}"
  success "Number test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Test valid number
	params := map[string]string{
		"port": "8080",
	}

	err = engine.ExecuteWithParams(program, "number test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed for valid number: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "ℹ️  Port: 8080") {
		t.Errorf("Expected valid number output, got %q", outputStr)
	}

	// Test invalid number
	output.Reset()
	params = map[string]string{
		"port": "not-a-number",
	}

	err = engine.ExecuteWithParams(program, "number test", params)
	if err == nil {
		t.Error("Expected error for invalid number, but got none")
	}

	if !strings.Contains(err.Error(), "invalid number value") {
		t.Errorf("Expected number validation error, got %q", err.Error())
	}
}

func TestEngine_BooleanValidation(t *testing.T) {
	input := `version: 2.0

task "boolean test":
  requires debug as boolean
  
  info "Debug: {debug}"
  success "Boolean test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	// Test various boolean values
	testCases := []struct {
		input    string
		expected string
	}{
		{"true", "true"},
		{"false", "false"},
		{"yes", "true"},
		{"no", "false"},
		{"1", "true"},
		{"0", "false"},
		{"on", "true"},
		{"off", "false"},
	}

	for _, tc := range testCases {
		var output bytes.Buffer
		engine := NewEngine(&output)

		params := map[string]string{
			"debug": tc.input,
		}

		err = engine.ExecuteWithParams(program, "boolean test", params)
		if err != nil {
			t.Fatalf("ExecuteWithParams failed for boolean %q: %v", tc.input, err)
		}

		outputStr := output.String()
		expectedOutput := "ℹ️  Debug: " + tc.expected
		if !strings.Contains(outputStr, expectedOutput) {
			t.Errorf("Expected %q for input %q, got %q", expectedOutput, tc.input, outputStr)
		}
	}

	// Test invalid boolean
	var output bytes.Buffer
	engine := NewEngine(&output)
	params := map[string]string{
		"debug": "maybe",
	}

	err = engine.ExecuteWithParams(program, "boolean test", params)
	if err == nil {
		t.Error("Expected error for invalid boolean, but got none")
	}

	if !strings.Contains(err.Error(), "invalid boolean value") {
		t.Errorf("Expected boolean validation error, got %q", err.Error())
	}
}

func TestEngine_ListParameters(t *testing.T) {
	input := `version: 2.0

task "list test":
  requires environments as list
  
  info "Environments: {environments}"
  
  for each env in environments:
    info "Processing environment: {env}"
  
  success "List test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"environments": "dev,staging,production",
	}

	err = engine.ExecuteWithParams(program, "list test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that list parameter was processed correctly
	expectedParts := []string{
		"ℹ️  Environments: dev,staging,production",
		"ℹ️  Processing environment: dev",
		"ℹ️  Processing environment: staging",
		"ℹ️  Processing environment: production",
		"✅ List test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_TypeInference(t *testing.T) {
	input := `version: 2.0

task "inference test":
  given count defaults to "42"
  given enabled defaults to "true"
  given items defaults to "a,b,c"
  
  info "Count: {count}"
  info "Enabled: {enabled}"
  info "Items: {items}"
  success "Type inference test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	// Run without providing parameters to use defaults
	err = engine.Execute(program, "inference test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that type inference worked with defaults
	expectedParts := []string{
		"ℹ️  Count: 42",
		"ℹ️  Enabled: true",
		"ℹ️  Items: a,b,c",
		"✅ Type inference test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_TypedConstraints(t *testing.T) {
	input := `version: 2.0

task "constraint test":
  requires environment as string from ["dev", "staging", "production"]
  requires port as number
  
  info "Environment: {environment}"
  info "Port: {port}"
  success "Constraint test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	// Test valid constraint
	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"environment": "dev",
		"port":        "8080",
	}

	err = engine.ExecuteWithParams(program, "constraint test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed for valid constraint: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "ℹ️  Environment: dev") {
		t.Errorf("Expected valid constraint output, got %q", outputStr)
	}

	// Test invalid constraint
	output.Reset()
	params = map[string]string{
		"environment": "invalid",
		"port":        "8080",
	}

	err = engine.ExecuteWithParams(program, "constraint test", params)
	if err == nil {
		t.Error("Expected error for invalid constraint, but got none")
	}

	if !strings.Contains(err.Error(), "not in allowed values") {
		t.Errorf("Expected constraint validation error, got %q", err.Error())
	}
}

func TestEngine_MixedTypesWithConditionals(t *testing.T) {
	input := `version: 2.0

task "mixed types":
  requires count as number
  given debug as boolean defaults to "false"
  
  info "Starting with count: {count}"
  
  when debug is "true":
    info "Debug mode is enabled"
  
  when debug is "false":
    info "Debug mode is disabled"
  
  success "Mixed types test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	// Test with debug enabled
	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"count": "5",
		"debug": "true",
	}

	err = engine.ExecuteWithParams(program, "mixed types", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"ℹ️  Starting with count: 5",
		"ℹ️  Debug mode is enabled",
		"✅ Mixed types test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should not contain the false branch
	if strings.Contains(outputStr, "Debug mode is disabled") {
		t.Errorf("Should not contain false branch output")
	}
}
