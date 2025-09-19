package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestEngine_ProgressAndTimerFunctions(t *testing.T) {
	input := `version: 2.0

task "progress and timer demo":
  info "Starting progress and timer demo"
  
  # Start a timer
  info "{start timer('demo_timer')}"
  
  # Start progress indicator
  info "{start progress('Initializing system')}"
  
  # Update progress
  info "{update progress('25', 'Loading configuration')}"
  info "{update progress('50', 'Processing data')}"
  info "{update progress('75', 'Finalizing setup')}"
  
  # Show timer status
  info "{show elapsed time('demo_timer')}"
  
  # Finish progress
  info "{finish progress('System ready')}"
  
  # Stop timer
  info "{stop timer('demo_timer')}"
  
  # Show final timer status
  info "{show elapsed time('demo_timer')}"
  
  success "Progress and timer demo completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "progress and timer demo")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that progress functions were called and interpolated
	expectedParts := []string{
		"ℹ️  Starting progress and timer demo",
		"⏱️  Started timer 'demo_timer'",
		"📋 Initializing system",
		"📋 Loading configuration",
		"📋 Processing data",
		"📋 Finalizing setup",
		"Timer 'demo_timer' (running):",
		"✅ System ready (completed in",
		"⏹️  Stopped timer 'demo_timer'",
		"Timer 'demo_timer' (stopped):",
		"✅ Progress and timer demo completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Check that progress bars are present
	if !strings.Contains(outputStr, "[") || !strings.Contains(outputStr, "]") {
		t.Errorf("Expected output to contain progress bars, got %q", outputStr)
	}

	// Check that percentages are present
	expectedPercentages := []string{"(25%)", "(50%)", "(75%)"}
	for _, percentage := range expectedPercentages {
		if !strings.Contains(outputStr, percentage) {
			t.Errorf("Expected output to contain %q, got %q", percentage, outputStr)
		}
	}
}

func TestEngine_ProgressAndTimerWithCustomNames(t *testing.T) {
	input := `version: 2.0

task "custom names demo":
  info "Testing custom progress and timer names"
  
  # Start multiple timers
  info "{start timer('timer1')}"
  info "{start timer('timer2')}"
  
  # Start multiple progress indicators
  info "{start progress('Task A', 'progress_a')}"
  info "{start progress('Task B', 'progress_b')}"
  
  # Update different progress indicators
  info "{update progress('30', 'Task A in progress', 'progress_a')}"
  info "{update progress('60', 'Task B in progress', 'progress_b')}"
  
  # Show different timer statuses
  info "{show elapsed time('timer1')}"
  info "{show elapsed time('timer2')}"
  
  # Finish different progress indicators
  info "{finish progress('Task A completed', 'progress_a')}"
  info "{finish progress('Task B completed', 'progress_b')}"
  
  # Stop different timers
  info "{stop timer('timer1')}"
  info "{stop timer('timer2')}"
  
  success "Custom names demo completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "custom names demo")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that all custom names are handled correctly
	expectedParts := []string{
		"Started timer 'timer1'",
		"Started timer 'timer2'",
		"📋 Task A",
		"📋 Task B",
		"Task A in progress",
		"Task B in progress",
		"Timer 'timer1' (running):",
		"Timer 'timer2' (running):",
		"✅ Task A completed",
		"✅ Task B completed",
		"Stopped timer 'timer1'",
		"Stopped timer 'timer2'",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestEngine_ProgressAndTimerErrorHandling(t *testing.T) {
	input := `version: 2.0

task "error handling demo":
  info "Testing error handling for progress and timer functions"
  
  # This should work
  info "{start progress('Valid progress')}"
  
  # This should fail - no message
  info "{start progress()}"
  
  success "Error handling demo completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "error handling demo")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that valid progress was started
	if !strings.Contains(outputStr, "📋 Valid progress") {
		t.Errorf("Expected output to contain valid progress message, got %q", outputStr)
	}

	// Check that invalid function call was handled (should show the original placeholder)
	if !strings.Contains(outputStr, "{start progress()}") {
		t.Errorf("Expected output to contain original placeholder for invalid call, got %q", outputStr)
	}
}
