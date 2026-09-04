package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

// writeTestFile creates a file under a temp dir with the given content,
// returning its absolute path in forward-slash form. drun string literals
// treat backslashes as escapes, so Windows temp paths must be embedded in
// scripts with '/' separators (the engine's file APIs accept them).
func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "lines.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	return filepath.ToSlash(path)
}

// executeLineTask runs a drun task and returns captured output and error.
func executeLineTask(t *testing.T, input, taskName string, configure func(*Engine)) (string, error) {
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

// Regression: executeLineLoop never opened the file; it iterated the fake
// []string{"line1","line2","line3"} and reported "(3 lines)". A real file's
// lines must be iterated in order with the real count.
func TestLineLoopReadsRealFileLines(t *testing.T) {
	path := writeTestFile(t, "alpha\nbeta\ngamma\ndelta\n")

	input := `version: 2.0

task "test":
	for each line text in file "` + path + `":
		info "  Line: {text}"
`

	output, err := executeLineTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	// Real line count in the header, and the real content in order.
	assertOutputContains(t, output, "(4 lines)")
	assertOutputContains(t, output, "Line: alpha", "Line: beta", "Line: gamma", "Line: delta")
	assertOutputNotContains(t, output, "Line: line1", "Line: line2", "Line: line3", "(3 lines)")

	if strings.Index(output, "Line: alpha") >= strings.Index(output, "Line: beta") ||
		strings.Index(output, "Line: beta") >= strings.Index(output, "Line: gamma") ||
		strings.Index(output, "Line: gamma") >= strings.Index(output, "Line: delta") {
		t.Errorf("Expected lines in file order, but got:\n%s", output)
	}
}

// A missing file must produce a clear error instead of iterating fake lines.
func TestLineLoopMissingFileErrors(t *testing.T) {
	// drun string literals treat backslashes as escapes, so embed the missing
	// path in forward-slash form (see writeTestFile).
	missing := filepath.ToSlash(filepath.Join(t.TempDir(), "no-such-file.txt"))

	input := `version: 2.0

task "test":
	for each line text in file "` + missing + `":
		info "  Line: {text}"
`

	output, err := executeLineTask(t, input, "test", nil)
	if err == nil {
		t.Fatalf("Expected error for missing file, got none. Output:\n%s", output)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("Expected error to mention the missing file %q, got: %v", missing, err)
	}
}

// Dry-run must announce the read without opening the file or running the body.
func TestLineLoopDryRunDoesNotRead(t *testing.T) {
	path := writeTestFile(t, "alpha\nbeta\n")

	input := `version: 2.0

task "test":
	for each line text in file "` + path + `":
		info "  Line: {text}"
`

	output, err := executeLineTask(t, input, "test", func(e *Engine) {
		e.SetDryRun(true)
	})
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "[DRY RUN] Would read lines from file: "+path)
	assertOutputNotContains(t, output, "Line: alpha", "Line: beta")
}

// Relative filenames resolve against the current use-workdir directory.
func TestLineLoopResolvesAgainstWorkdir(t *testing.T) {
	rawDir := t.TempDir()
	dir := filepath.ToSlash(rawDir)
	if err := os.WriteFile(filepath.Join(rawDir, "data.txt"), []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("writing workdir file: %v", err)
	}

	input := `version: 2.0

task "test":
	use workdir "` + dir + `"
	for each line text in file "data.txt":
		info "  Line: {text}"
`

	output, err := executeLineTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "(3 lines)")
	assertOutputContains(t, output, "Line: one", "Line: two", "Line: three")
	assertOutputNotContains(t, output, "Line: line1", "Line: line2", "Line: line3")
}

// Filters must apply to the real lines.
func TestLineLoopAppliesFilter(t *testing.T) {
	path := writeTestFile(t, "alpha\nbeta\ngamma\ndelta\n")

	input := `version: 2.0

task "test":
	for each line text in file "` + path + `" where text == "gamma":
		info "  Line: {text}"
`

	output, err := executeLineTask(t, input, "test", nil)
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	assertOutputContains(t, output, "Line: gamma")
	assertOutputNotContains(t, output, "Line: alpha", "Line: beta", "Line: delta", "Line: line1", "Line: line2", "Line: line3")
}
