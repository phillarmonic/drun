package engine

import (
	"strings"
	"testing"
)

// assertOrdered verifies that expected strings appear in output in order.
func assertOrdered(t *testing.T, output string, expected ...string) {
	t.Helper()
	prev := -1
	for _, s := range expected {
		idx := strings.Index(output, s)
		if idx < 0 {
			t.Errorf("Expected output to contain %q, but got:\n%s", s, output)
			continue
		}
		if idx < prev {
			t.Errorf("Expected %q to appear after the previous expected string, but got:\n%s", s, output)
		}
		prev = idx
	}
}

// Regression: executeEachLoop split variable strings with strings.Fields only,
// so "alpha,beta,gamma" (comma separated, no spaces) iterated as ONE item.
// Values containing commas must iterate per comma-separated item.
func TestEachLoopSplitsCommaSeparatedVariable(t *testing.T) {
	input := `version: 2.0

task "test":
	set $items to "alpha,beta,gamma"
	for each $item in $items:
		info "Item {$item}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Item alpha", "Item beta", "Item gamma")
	assertOutputNotContains(t, output, "Item alpha,beta,gamma")
	assertOrdered(t, output, "Item alpha", "Item beta", "Item gamma")
}

// Commas followed by spaces must not leak whitespace into the items.
func TestEachLoopTrimsCommaSeparatedItems(t *testing.T) {
	input := `version: 2.0

task "test":
	set $items to "alpha, beta, gamma"
	for each $item in $items:
		info "Item {$item}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Item alpha", "Item beta", "Item gamma")
	assertOutputNotContains(t, output, "Item alpha,", "Item  beta")
}

// Values without commas must keep splitting on whitespace (example 36 style).
func TestEachLoopKeepsWhitespaceSplitting(t *testing.T) {
	input := `version: 2.0

task "test":
	set $items to "alpha beta gamma"
	for each $item in $items:
		info "Item {$item}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Item alpha", "Item beta", "Item gamma")
}

// Regression: `given $items defaults to "alpha,beta,gamma"` iterated as ONE
// item. A comma-separated default must iterate per item.
func TestEachLoopSplitsCommaSeparatedParameterDefault(t *testing.T) {
	input := `version: 2.0

task "test":
	given $items defaults to "alpha,beta,gamma"
	for each $item in $items:
		info "Item {$item}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Item alpha", "Item beta", "Item gamma")
	assertOutputNotContains(t, output, "Item alpha,beta,gamma")
	assertOrdered(t, output, "Item alpha", "Item beta", "Item gamma")
}

// Array literals must keep working unchanged.
func TestEachLoopArrayLiteralUnchanged(t *testing.T) {
	input := `version: 2.0

task "test":
	for each $item in ["alpha", "beta"]:
		info "Item {$item}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Item alpha", "Item beta")
	assertOutputNotContains(t, output, "Item [")
}

// A comma-separated project setting accessed via $globals must iterate per item.
func TestEachLoopSplitsCommaSeparatedProjectSetting(t *testing.T) {
	input := `version: 2.0
project "p" version "1.0":
	set tags to "one,two,three"

task "test":
	for each $t in $globals.tags:
		info "Tag: {$t}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Tag: one", "Tag: two", "Tag: three")
	assertOutputNotContains(t, output, "Tag: one,two,three")
}

// A typed list-of-strings parameter must iterate its elements, not the whole
// comma-joined value as a single item.
func TestEachLoopListParameterIteratesElements(t *testing.T) {
	input := `version: 2.0

task "test":
	accepts $items as list of strings
	for each $item in $items:
		info "Item {$item}"
`

	output, err := runTaskWithParams(t, input, "test", map[string]string{"items": "alpha,beta"})
	if err != nil {
		t.Fatalf("Execution error: %v\noutput:\n%s", err, output)
	}

	assertOutputContains(t, output, "Item alpha", "Item beta")
	assertOutputNotContains(t, output, "Item alpha,beta")
}
