package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_RangeLoop(t *testing.T) {
	input := `version: 2.0

task "range_test":
  for i in range 1 to 5:
    info "Processing item {i}"
    
  success "Range loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "range_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Executing range loop from 1 to 5 step 1",
		"🔄 Executing",
		"items sequentially",
		"✅ Range loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_FilteredLoop(t *testing.T) {
	input := `version: 2.0

task "filtered_test":
  requires items as string
  
  for each item in items where item contains "test":
    info "Processing test item: {item}"
    
  success "Filtered loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "test1,normal,test2,other,test3",
	}

	err = engine.ExecuteWithParams(program, "filtered_test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Filter applied:",
		"items match condition",
		"✅ Filtered loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_LineLoop(t *testing.T) {
	input := `version: 2.0

task "line_test":
  for each line text in file "data.txt":
    info "Processing line: {text}"
    
  success "Line loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "line_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"📄 Reading lines from file: data.txt",
		"lines)",
		"✅ Line loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_MatchLoop(t *testing.T) {
	input := `version: 2.0

task "match_test":
  for each match result in pattern "[0-9]+":
    info "Found number: {result}"
    
  success "Match loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.ExecuteWithParams(program, "match_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Finding matches for pattern: [0-9]+",
		"matches)",
		"✅ Match loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_BreakStatement(t *testing.T) {
	input := `version: 2.0

task "break_test":
  requires items as string
  
  for each item in items:
    info "Processing: {item}"
    break
    info "This should not be reached"
    
  success "Break test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "item1,item2,item3",
	}

	err = engine.ExecuteWithParams(program, "break_test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Breaking loop",
		"✅ Break test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should not contain the message after break
	if strings.Contains(outputStr, "This should not be reached") {
		t.Errorf("Break statement did not work - found text that should not be reached")
	}
}

func TestEngine_ContinueStatement(t *testing.T) {
	input := `version: 2.0

task "continue_test":
  requires items as string
  
  for each item in items:
    continue
    info "This should not be reached"
    
  success "Continue test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "item1,item2,item3",
	}

	err = engine.ExecuteWithParams(program, "continue_test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Continuing loop",
		"✅ Continue test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should not contain the message after continue
	if strings.Contains(outputStr, "This should not be reached") {
		t.Errorf("Continue statement did not work - found text that should not be reached")
	}
}

func TestEngine_ConditionalBreak(t *testing.T) {
	input := `version: 2.0

task "conditional_break_test":
  requires items as string
  
  for each item in items:
    info "Processing: {item}"
    break when item == "stop"
    
  success "Conditional break test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "item1,stop,item3",
	}

	err = engine.ExecuteWithParams(program, "conditional_break_test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Breaking loop (condition:",
		"✅ Conditional break test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ConditionalContinue(t *testing.T) {
	input := `version: 2.0

task "conditional_continue_test":
  requires items as string
  
  for each item in items:
    continue if item == "skip"
    info "Processing: {item}"
    
  success "Conditional continue test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "item1,skip,item3",
	}

	err = engine.ExecuteWithParams(program, "conditional_continue_test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔄 Continuing loop (condition:",
		"✅ Conditional continue test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_AdvancedControlFlowDryRun(t *testing.T) {
	input := `version: 2.0

task "advanced_test":
  for i in range 1 to 10 step 2:
    info "Processing {i}"
  
  for each line text in file "data.txt":
    info "Line: {text}"
  
  for each match num in pattern "[0-9]+":
    info "Number: {num}"
    
  success "Advanced control flow test completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.ExecuteWithParams(program, "advanced_test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"[DRY RUN] Would execute range loop from 1 to 10 step 2",
		"[DRY RUN] Would read lines from file: data.txt",
		"[DRY RUN] Would find matches for pattern: [0-9]+",
		"[DRY RUN] success: Advanced control flow test completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ComplexFilteredLoop(t *testing.T) {
	input := `version: 2.0

task "complex_filter_test":
  requires items as string
  
  for each item in items where item ends with ".js":
    info "Processing JavaScript file: {item}"
    
  success "Complex filtered loop completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"items": "app.js,style.css,main.js,readme.txt,utils.js",
	}

	err = engine.ExecuteWithParams(program, "complex_filter_test", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	expectedParts := []string{
		"🔍 Filter applied:",
		"items match condition",
		"ends with",
		"✅ Complex filtered loop completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}
