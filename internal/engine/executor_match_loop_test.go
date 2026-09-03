package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
	"github.com/phillarmonic/drun/v2/internal/types"
)

// executeMatchTask parses and runs a single task through the engine, returning
// the captured output. configure can tweak the engine before execution.
func executeMatchTask(t *testing.T, input, taskName string, configure func(*Engine)) (string, error) {
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

// Regression: executeMatchLoop iterated the fake ["match1","match2"] and the
// subject-less grammar had nothing to match. Real matches of the pattern over
// the subject variable must drive iteration.
func TestMatchLoopIteratesRegexMatchesOverSubject(t *testing.T) {
	input := `version: 2.0

task "test":
	given $log defaults to "abc123def456"
	for each match num in pattern "[0-9]+" of $log:
		info "  Match: {num}"
`

	output, err := executeMatchTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "(2 matches)")
	assertOutputContains(t, output, "Match: 123", "Match: 456")
	assertOutputNotContains(t, output, "Match: match1", "Match: match2", "Match: 789")

	if strings.Index(output, "Match: 123") >= strings.Index(output, "Match: 456") {
		t.Errorf("Expected matches in subject order, but got:\n%s", output)
	}
}

// The subject can be a set variable rather than a parameter default.
func TestMatchLoopSubjectFromSetVariable(t *testing.T) {
	input := `version: 2.0

task "test":
	set $haystack to "x=1 y=22 z=333"
	for each match num in pattern "[0-9]+" of $haystack:
		info "  Match: {num}"
`

	output, err := executeMatchTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Match: 1", "Match: 22", "Match: 333")
	assertOutputNotContains(t, output, "Match: match1", "Match: match2")
}

// An invalid regular expression must error clearly instead of fake matches.
func TestMatchLoopInvalidPatternErrors(t *testing.T) {
	input := `version: 2.0

task "test":
	given $log defaults to "abc123def456"
	for each match num in pattern "[0-9]+(" of $log:
		info "  Match: {num}"
`

	output, err := executeMatchTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error for invalid pattern, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "[0-9]+(") {
		t.Errorf("Expected error to mention the pattern, got: %v", err)
	}
}

// An undefined subject variable must error clearly.
func TestMatchLoopMissingSubjectVariableErrors(t *testing.T) {
	input := `version: 2.0

task "test":
	for each match num in pattern "[0-9]+" of $ghost:
		info "  Match: {num}"
`

	output, err := executeMatchTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error for missing subject variable, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "$ghost") {
		t.Errorf("Expected error to mention the subject variable $ghost, got: %v", err)
	}
}

// Dry-run must announce pattern and subject without compiling or iterating.
func TestMatchLoopDryRun(t *testing.T) {
	input := `version: 2.0

task "test":
	given $log defaults to "abc123def456"
	for each match num in pattern "[0-9]+" of $log:
		info "  Match: {num}"
`

	output, err := executeMatchTask(t, input, "test", func(e *Engine) {
		e.SetDryRun(true)
	})
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "[DRY RUN] Would find matches for pattern: [0-9]+ in $log")
	assertOutputNotContains(t, output, "Match: 123", "Match: 456")
}

// Engine guard: a match loop constructed without a subject is rejected, never
// executed as a fake stub.
func TestMatchLoopEngineRejectsMissingSubject(t *testing.T) {
	var output bytes.Buffer
	engine := NewEngine(&output)
	ctx := &ExecutionContext{
		Parameters: make(map[string]*types.Value),
		Variables:  make(map[string]string),
	}

	stmt := &statement.Loop{LoopType: "match", Variable: "num", Iterable: "[0-9]+"}
	err := engine.executeMatchLoop(stmt, ctx)
	if err == nil {
		t.Fatalf("Expected error for subject-less match loop, got none")
	}
	if !strings.Contains(err.Error(), "no subject") {
		t.Errorf("Expected error mentioning a missing subject, got: %v", err)
	}
}
