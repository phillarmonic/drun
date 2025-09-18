package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_LetStatement(t *testing.T) {
	input := `version: 2.0

task "let_test":
  let name = "John"
  let count = 42
  info "Name: {name}, Count: {count}"
  
  success "Let statement test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "let_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"📝 Set variable name = John",
		"📝 Set variable count = 42",
		"ℹ️  Name: John, Count: 42",
		"✅ Let statement test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_SetStatement(t *testing.T) {
	input := `version: 2.0

task "set_test":
  set message to "Hello World"
  set counter to 100
  info "Message: {message}, Counter: {counter}"
  
  success "Set statement test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "set_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"📝 Set variable message to Hello World",
		"📝 Set variable counter to 100",
		"ℹ️  Message: Hello World, Counter: 100",
		"✅ Set statement test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_TransformStatement(t *testing.T) {
	input := `version: 2.0

task "transform_test":
  let text = "hello world"
  transform text with uppercase
  info "Uppercase: {text}"
  
  let name = "John"
  transform name with concat " Doe"
  info "Full name: {name}"
  
  success "Transform statement test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "transform_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"📝 Set variable text = hello world",
		"🔄 Transformed variable text with uppercase: hello world -> HELLO WORLD",
		"ℹ️  Uppercase: HELLO WORLD",
		"📝 Set variable name = John",
		"🔄 Transformed variable name with concat: John -> John Doe",
		"ℹ️  Full name: John Doe",
		"✅ Transform statement test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_StringTransformations(t *testing.T) {
	input := `version: 2.0

task "string_transform_test":
  let text = "  Hello World  "
  transform text with trim
  info "Trimmed: '{text}'"
  
  transform text with lowercase
  info "Lowercase: {text}"
  
  transform text with replace "world" "Universe"
  info "Replaced: {text}"
  
  success "String transformations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "string_transform_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Transformed variable text with trim:",
		"ℹ️  Trimmed: 'Hello World'",
		"🔄 Transformed variable text with lowercase:",
		"ℹ️  Lowercase: hello world",
		"🔄 Transformed variable text with replace:",
		"ℹ️  Replaced: hello Universe",
		"✅ String transformations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_VariableOperationsInLoops(t *testing.T) {
	input := `version: 2.0

task "loop_variables_test":
  given items defaults to "apple,banana,cherry"
  
  for each item in items:
    let processed = item
    transform processed with uppercase
    info "Processed: {processed}"
  
  success "Loop variables test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "loop_variables_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"📝 Set variable processed = apple",
		"🔄 Transformed variable processed with uppercase: apple -> APPLE",
		"ℹ️  Processed: APPLE",
		"📝 Set variable processed = banana",
		"🔄 Transformed variable processed with uppercase: banana -> BANANA",
		"ℹ️  Processed: BANANA",
		"📝 Set variable processed = cherry",
		"🔄 Transformed variable processed with uppercase: cherry -> CHERRY",
		"ℹ️  Processed: CHERRY",
		"✅ Loop variables test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_VariableOperationsInConditionals(t *testing.T) {
	input := `version: 2.0

task "conditional_variables_test":
  let status = "pending"
  
  if status == "pending":
    set status to "processing"
    transform status with concat " - in progress"
    info "Status updated: {status}"
  
  success "Conditional variables test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "conditional_variables_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"📝 Set variable status = pending",
		"📝 Set variable status to processing",
		"🔄 Transformed variable status with concat: processing -> processing - in progress",
		"ℹ️  Status updated: processing - in progress",
		"✅ Conditional variables test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_VariableOperationsDryRun(t *testing.T) {
	input := `version: 2.0

task "dry_run_test":
  let name = "John"
  set message to "Hello"
  transform message with concat " {name}"
  info "Result: {message}"
  
  success "Dry run test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.ExecuteWithParams(program, "dry_run_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] Would set variable name = John",
		"[DRY RUN] Would set variable message to Hello",
		"[DRY RUN] Would transform variable message with concat:",
		"[DRY RUN] info: Result: Hello John",
		"[DRY RUN] success: Dry run test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_VariableTransformationFunctions(t *testing.T) {
	input := `version: 2.0

task "transformation_functions_test":
  let text = "Hello World"
  transform text with length
  info "Length: {text}"
  
  let greeting = "Hello"
  transform greeting with slice 0 4
  info "Sliced: {greeting}"
  
  success "Transformation functions test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "transformation_functions_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Transformed variable text with length: Hello World -> 11",
		"ℹ️  Length: 11",
		"🔄 Transformed variable greeting with slice: Hello -> Hell",
		"ℹ️  Sliced: Hell",
		"✅ Transformation functions test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_VariableOperationsErrorHandling(t *testing.T) {
	input := `version: 2.0

task "error_handling_test":
  transform nonexistent with uppercase
  
  success "This should not be reached"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "error_handling_test", map[string]string{})
	if err == nil {
		t.Fatalf("Expected error for nonexistent variable, but got none")
	}

	if !strings.Contains(err.Error(), "variable 'nonexistent' not found") {
		t.Errorf("Expected error about nonexistent variable, got: %v", err)
	}
}

func TestEngine_ComplexVariableOperations(t *testing.T) {
	input := `version: 2.0

task "complex_test":
  let firstName = "John"
  let lastName = "Doe"
  set fullName to "Unknown"
  
  # Build full name step by step
  set fullName to firstName
  transform fullName with concat " "
  transform fullName with concat lastName
  transform fullName with uppercase
  
  info "Full name: {fullName}"
  
  success "Complex variable operations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "complex_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"📝 Set variable firstName = John",
		"📝 Set variable lastName = Doe",
		"📝 Set variable fullName to Unknown",
		"📝 Set variable fullName to John",
		"🔄 Transformed variable fullName with concat: John -> John ",
		"🔄 Transformed variable fullName with concat: John  -> John Doe",
		"🔄 Transformed variable fullName with uppercase: John Doe -> JOHN DOE",
		"ℹ️  Full name: JOHN DOE",
		"✅ Complex variable operations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
