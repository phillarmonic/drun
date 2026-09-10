package engine

import (
	"strconv"
	"strings"
	"testing"
)

// Regression: executeCaptureStatement stored varStmt.Value verbatim without
// interpolating, so the documented timing idiom `capture start from now`
// stored the literal string "now". A bare `now` source must store the `now`
// builtin's epoch-seconds timestamp.
func TestCaptureFromNowStoresEpochSeconds(t *testing.T) {
	input := `version: 2.0

task "test":
	capture start_time from now
	capture end_time from now
	info "Start_time: {start_time}"
	info "End_time: {end_time}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputNotContains(t, output, "Start_time: now")

	start := extractCaptureValue(t, output, "Start_time: ")
	end := extractCaptureValue(t, output, "End_time: ")

	startSec, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		t.Fatalf("Expected epoch-seconds start timestamp, got %q: %v\noutput:\n%s", start, err, output)
	}
	endSec, err := strconv.ParseInt(end, 10, 64)
	if err != nil {
		t.Fatalf("Expected epoch-seconds end timestamp, got %q: %v\noutput:\n%s", end, err, output)
	}
	if endSec < startSec {
		t.Errorf("Expected end timestamp >= start timestamp, got %d < %d", endSec, startSec)
	}
}

// Regression: capture-from-expression must interpolate its value the same way
// let/set do, storing the resolved text instead of the raw expression.
func TestCaptureInterpolatesExpressionLikeLetSet(t *testing.T) {
	input := `version: 2.0

task "test":
	set $who to "World"
	capture greeting from {$who}
	info "Greeting: {greeting}"
`

	// Verbose mode prints the captured value as stored, exposing whether the
	// raw expression or the interpolated text was persisted.
	output, err := executeRangeTask(t, input, "test", func(engine *Engine) {
		engine.SetVerbose(true)
	})
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Captured greeting: World", "Greeting: World")
	assertOutputNotContains(t, output, "Captured greeting: {$who}")
}

// extractCaptureValue returns the text after prefix on the line that contains
// prefix (e.g. "Start_time: " -> the captured value, ignoring leading output
// decorations such as the info icon).
func extractCaptureValue(t *testing.T, output, prefix string) string {
	t.Helper()
	for line := range strings.SplitSeq(output, "\n") {
		if _, after, ok := strings.Cut(line, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	t.Fatalf("Expected output line containing %q, but got:\n%s", prefix, output)
	return ""
}
