package engine

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileComparisonConditionsCompareExactBytes(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{
		"left file.bin":      {0x00, 0x01, '\n', 0xff},
		"equal file.bin":     {0x00, 0x01, '\n', 0xff},
		"different file.bin": {0x00, 0x01, '\r', '\n', 0xff},
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	input := `version: 2.0

task "compare":
  use workdir "` + dir + `"
  if file "left file.bin" matches file "equal file.bin":
    info "MATCH_TRUE"
  else:
    info "MATCH_FALSE"
  if file "left file.bin" not matches file "different file.bin":
    info "NOT_MATCH_TRUE"
  else:
    info "NOT_MATCH_FALSE"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var output bytes.Buffer
	engine := NewEngine(&output)
	if err := engine.Execute(program, "compare"); err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	for _, expected := range []string{"MATCH_TRUE", "NOT_MATCH_TRUE"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output %q does not contain %q", output.String(), expected)
		}
	}
	for _, unexpected := range []string{"MATCH_FALSE", "NOT_MATCH_FALSE"} {
		if strings.Contains(output.String(), unexpected) {
			t.Errorf("output %q unexpectedly contains %q", output.String(), unexpected)
		}
	}
}

func TestFileComparisonConditionInterpolatesPaths(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"left.bin", "right.bin"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("same\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{
		OriginalWorkingDir: dir,
		Variables:          map[string]string{"left": "left.bin", "right": "right.bin"},
	}
	result, handled, err := engine.evaluateFileComparisonCondition(
		"file {$left} matches file {$right}", ctx,
	)
	if err != nil {
		t.Fatalf("comparison failed: %v", err)
	}
	if !handled {
		t.Fatal("file comparison condition was not handled")
	}
	if !result {
		t.Fatal("equal interpolated files should match")
	}
}

func TestFileComparisonConditionReportsReadErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "left.bin"), []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{OriginalWorkingDir: dir}
	_, handled, err := engine.evaluateFileComparisonCondition(
		"file left.bin matches file missing.bin", ctx,
	)
	if !handled {
		t.Fatal("file comparison condition was not handled")
	}
	if err == nil || !strings.Contains(err.Error(), `reading right file "missing.bin"`) {
		t.Fatalf("error = %v, want an actionable right-file read error", err)
	}
}

func TestFileComparisonConditionIgnoresOtherConditions(t *testing.T) {
	result, handled, err := NewEngine(io.Discard).evaluateFileComparisonCondition(
		"file package.json exists", &ExecutionContext{},
	)
	if err != nil || handled || result {
		t.Fatalf("result, handled, err = %v, %v, %v; want false, false, nil", result, handled, err)
	}
}
