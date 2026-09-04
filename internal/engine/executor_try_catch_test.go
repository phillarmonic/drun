package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

// executeErrorTask parses and runs a single task through the engine, returning
// the captured output. configure can tweak the engine before execution.
func executeErrorTask(t *testing.T, input, taskName string, configure func(*Engine)) (string, error) {
	t.Helper()

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	if configure != nil {
		configure(engine)
	}

	err := engine.Execute(program, taskName)
	return output.String(), err
}

// Regression: executeTry wiped any error raised inside a matching catch body
// (handled=true then tryError=nil), so the task continued past the try block.
// An error raised in a catch body must propagate and fail the task.
func TestCatchBodyErrorPropagates(t *testing.T) {
	input := `version: 2.0

task "test":
	try:
		throw "original failure"
	catch:
		info "  In catch"
		throw "failure inside catch body"
	finally:
		info "  Finally ran"
	info "AFTER TRY"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error from catch body, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "failure inside catch body") {
		t.Errorf("Expected the catch-body error message, got: %v", err)
	}

	assertOutputContains(t, output, "In catch", "Finally ran")
	assertOutputNotContains(t, output, "AFTER TRY", "Error handled successfully")
}

// Control: a catch body that recovers still swallows the original error and
// execution continues past the try block.
func TestCatchBodyRecoveryStillHandles(t *testing.T) {
	input := `version: 2.0

task "test":
	try:
		throw "original failure"
	catch:
		info "  Recovered"
	info "AFTER TRY"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Recovered", "AFTER TRY", "Error handled successfully")
}

// Typed catch interplay: when no catch clause matches, the original error is
// reported as unhandled and propagates; statements after the try do not run.
func TestUnmatchedTypedCatchReportsOriginalError(t *testing.T) {
	input := `version: 2.0

task "test":
	try:
		throw "boom"
	catch FileNotFoundError:
		info "  Should not reach typed catch"
	info "AFTER TRY"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected unhandled error, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("Expected original error message, got: %v", err)
	}

	assertOutputContains(t, output, "Unhandled error")
	assertOutputNotContains(t, output, "Should not reach typed catch", "AFTER TRY")
}

// A finally block still runs when the catch body itself fails, and the
// catch-body error is what propagates.
func TestFinallyRunsWhenCatchBodyFails(t *testing.T) {
	input := `version: 2.0

task "test":
	try:
		throw "original failure"
	catch:
		throw "cleanup failed"
	finally:
		info "  Cleanup attempted"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "cleanup failed") {
		t.Errorf("Expected catch-body error to propagate through finally, got: %v", err)
	}
	assertOutputContains(t, output, "Cleanup attempted")
}

// Regression: rethrow returned a generic "rethrown error" and never
// propagated. It must re-raise the ORIGINAL caught error with its message
// intact, fail the task, and skip statements after the try block.
func TestRethrowReRaisesOriginalError(t *testing.T) {
	input := `version: 2.0

task "test":
	try:
		throw "disk full"
	catch:
		info "  Handling..."
		rethrow
	info "AFTER TRY"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected rethrown error, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("Expected the original error message to survive rethrow, got: %v", err)
	}
	if strings.Contains(err.Error(), "rethrown error") {
		t.Errorf("Expected the original error, not the generic placeholder, got: %v", err)
	}

	assertOutputContains(t, output, "Handling...", "Rethrowing current error")
	assertOutputNotContains(t, output, "AFTER TRY", "Error handled successfully")
}

// Rethrow only makes sense while a catch body is handling an error; outside a
// catch it must fail with a clear message instead of returning a generic one.
func TestRethrowOutsideCatchErrorsClearly(t *testing.T) {
	input := `version: 2.0

task "test":
	rethrow
	info "AFTER RETHROW"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error for rethrow outside catch, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "no active") && !strings.Contains(err.Error(), "outside a catch") && !strings.Contains(err.Error(), "not in a catch") {
		t.Errorf("Expected a clear 'rethrow outside catch' style error, got: %v", err)
	}
	if strings.Contains(err.Error(), "rethrown error") {
		t.Errorf("Expected a helpful error, not the generic placeholder, got: %v", err)
	}
	assertOutputNotContains(t, output, "AFTER RETHROW")
}

// ignore is valid inside a catch block, where it is an explicit
// documentation-only no-op that marks the caught error as handled.
func TestIgnoreInsideCatchIsAllowed(t *testing.T) {
	input := `version: 2.0

task "test":
	try:
		throw "expected hiccup"
	catch:
		info "  In catch"
		ignore
	info "AFTER TRY"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "In catch", "AFTER TRY", "Ignoring current error")
}

// Regression: bare ignore outside a catch was a silent no-op that suppressed
// nothing. It must now fail with a clear, helpful error.
func TestBareIgnoreOutsideCatchErrorsClearly(t *testing.T) {
	input := `version: 2.0

task "test":
	info "  Before"
	ignore
	info "AFTER IGNORE"
`

	output, err := executeErrorTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error for bare ignore outside a catch, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "inside a catch") && !strings.Contains(err.Error(), "catch block") {
		t.Errorf("Expected a helpful error pointing at catch blocks, got: %v", err)
	}
	assertOutputNotContains(t, output, "AFTER IGNORE")
}
