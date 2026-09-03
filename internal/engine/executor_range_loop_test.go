package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

// executeRangeTask parses and runs a single task through the engine, returning
// the captured output. configure can tweak the engine before execution.
func executeRangeTask(t *testing.T, input, taskName string, configure func(*Engine)) (string, error) {
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

func assertOutputContains(t *testing.T, output string, expected ...string) {
	t.Helper()
	for _, s := range expected {
		if !strings.Contains(output, s) {
			t.Errorf("Expected output to contain %q, but got:\n%s", s, output)
		}
	}
}

func assertOutputNotContains(t *testing.T, output string, unexpected ...string) {
	t.Helper()
	for _, s := range unexpected {
		if strings.Contains(output, s) {
			t.Errorf("Expected output to NOT contain %q, but got:\n%s", s, output)
		}
	}
}

// Regression: the stub executeRangeLoop hardcoded startInt=0/endInt=10/stepInt=1
// and ignored the parsed bounds, so `for $i in range 1 to 5 step 2` iterated
// 0..10. Real bounds must drive iteration.
func TestRangeLoopHonorsBounds(t *testing.T) {
	input := `version: 2.0

task "test":
	info "header"
	for $i in range 1 to 5 step 2:
		info "  Item {$i}"
	info "after"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	// Header must reflect the real range (1, 3, 5 -> 3 items).
	assertOutputContains(t, output, "from 1 to 5 step 2 (3 items)")
	// Iteration must visit 1, 3, 5 only - not the stub's hardcoded 0..10.
	assertOutputContains(t, output, "Item 1", "Item 3", "Item 5")
	assertOutputNotContains(t, output, "Item 0", "Item 2", "Item 4", "Item 6", "Item 7")

	// Sequential order: 1 before 3 before 5.
	if strings.Index(output, "Item 1") >= strings.Index(output, "Item 3") ||
		strings.Index(output, "Item 3") >= strings.Index(output, "Item 5") {
		t.Errorf("Expected items in ascending order 1, 3, 5, but got:\n%s", output)
	}
}

// A range without an explicit step defaults to step 1.
func TestRangeLoopDefaultStep(t *testing.T) {
	input := `version: 2.0

task "test":
	for $i in range 3 to 5:
		info "  Item {$i}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "from 3 to 5 step 1 (3 items)")
	assertOutputContains(t, output, "Item 3", "Item 4", "Item 5")
	assertOutputNotContains(t, output, "Item 0", "Item 1", "Item 2", "Item 6")
}

// Negative steps iterate downwards. The parser must accept `step -2`.
func TestRangeLoopDescendingWithNegativeStep(t *testing.T) {
	input := `version: 2.0

task "test":
	for $i in range 5 to 1 step -2:
		info "  Item {$i}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "from 5 to 1 step -2 (3 items)")
	assertOutputContains(t, output, "Item 5", "Item 3", "Item 1")
	assertOutputNotContains(t, output, "Item 0", "Item 2", "Item 4", "Item 6")

	// Descending order: 5 before 3 before 1.
	if strings.Index(output, "Item 5") >= strings.Index(output, "Item 3") ||
		strings.Index(output, "Item 3") >= strings.Index(output, "Item 1") {
		t.Errorf("Expected items in descending order 5, 3, 1, but got:\n%s", output)
	}
}

// A step of zero would loop forever; it must be rejected up front.
func TestRangeLoopRejectsZeroStep(t *testing.T) {
	input := `version: 2.0

task "test":
	for $i in range 1 to 5 step 0:
		info "  Item {$i}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error for step 0, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), "step") {
		t.Errorf("Expected error message mentioning step, got: %v", err)
	}
}

// Non-numeric bounds must produce a clear error instead of silently iterating
// the hardcoded stub range.
func TestRangeLoopRejectsNonNumericBounds(t *testing.T) {
	tests := []struct {
		name     string
		loopLine string
		errPart  string
	}{
		{
			// Fractional bounds pass the lexer (NUMBER "1.5") and the parser,
			// but are not valid integer range bounds.
			name:     "non-integer start",
			loopLine: `	for $i in range 1.5 to 5:`,
			errPart:  "1.5",
		},
		{
			name:     "non-integer end",
			loopLine: `	for $i in range 1 to 5.5:`,
			errPart:  "5.5",
		},
		{
			name:     "non-integer step",
			loopLine: `	for $i in range 1 to 5 step 1.5:`,
			errPart:  "1.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `version: 2.0

task "test":
` + tt.loopLine + `
		info "  Item {$i}"
`
			output, err := executeRangeTask(t, input, "test", nil)
			if err == nil {
				t.Fatalf("Expected error for loop line %q, got none. Output:\n%s", tt.loopLine, output)
			}
			if !strings.Contains(err.Error(), tt.errPart) {
				t.Errorf("Expected error message to mention %q, got: %v", tt.errPart, err)
			}
		})
	}
}

// Dry-run mode must report the real range without executing the body.
func TestRangeLoopDryRunReportsRealRange(t *testing.T) {
	input := `version: 2.0

task "test":
	for $i in range 1 to 5 step 2:
		info "  Item {$i}"
	info "after"
`

	output, err := executeRangeTask(t, input, "test", func(e *Engine) {
		e.SetDryRun(true)
	})
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "[DRY RUN] Would execute range loop from 1 to 5 step 2 (3 items)")
	assertOutputNotContains(t, output, "Item 1", "Item 3", "Item 5")
}

// Filters must apply to the real range items.
func TestRangeLoopAppliesFilter(t *testing.T) {
	input := `version: 2.0

task "test":
	for $i in range 1 to 5 where $i != "3":
		info "  Item {$i}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	// The stub iterates 0..10, so Item 0/Item 6/Item 7 would still appear;
	// the real range 1..5 (minus 3) must not contain them.
	assertOutputContains(t, output, "Item 1", "Item 2", "Item 4", "Item 5")
	assertOutputNotContains(t, output, "Item 0", "Item 3", "Item 6", "Item 7")
}

// `in parallel` must run over the real range items without error.
func TestRangeLoopParallel(t *testing.T) {
	input := `version: 2.0

task "test":
	for $i in range 1 to 3 in parallel:
		info "  Item {$i}"
`

	output, err := executeRangeTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	// Header reflects the real range before the parallel dispatch.
	assertOutputContains(t, output, "from 1 to 3 step 1 (3 items)")
}
