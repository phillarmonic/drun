package engine

import (
	"bytes"
	"testing"
)

func TestStepActionRendersMultilineBox(t *testing.T) {
	input := `version: 2.0

task "fuzz":
  given $iterations defaults to 50
  step "Executing semantic fuzz tests against example-based inputs
Iterations: {$iterations}"`

	var buf bytes.Buffer
	if err := ExecuteString(input, "fuzz", &buf); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	expected := "" +
		"┌────────────────────────────────────────────────────────────┐\n" +
		"│ Executing semantic fuzz tests against example-based inputs │\n" +
		"│ Iterations: 50                                             │\n" +
		"└────────────────────────────────────────────────────────────┘\n"

	if got := buf.String(); got != expected {
		t.Fatalf("unexpected step box output:\n%s", got)
	}
}

func TestStepActionLineBreakFlagsStillWrapMultilineBox(t *testing.T) {
	input := `version: 2.0

task "fuzz":
  given $iterations defaults to 50
  step "Phase 1\nIterations: {$iterations}" add line break before add line break after`

	var buf bytes.Buffer
	if err := ExecuteString(input, "fuzz", &buf); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	expected := "" +
		"\n" +
		"┌────────────────┐\n" +
		"│ Phase 1        │\n" +
		"│ Iterations: 50 │\n" +
		"└────────────────┘\n" +
		"\n"

	if got := buf.String(); got != expected {
		t.Fatalf("unexpected step box output with line breaks:\n%s", got)
	}
}
