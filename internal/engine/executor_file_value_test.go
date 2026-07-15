package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

func TestExecuteFileValueGetCheckUpdateAndDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gradle.properties")
	if err := os.WriteFile(path, []byte("pluginVersion=1.0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &ExecutionContext{Variables: map[string]string{"next": "1.0.2"}}
	out := &bytes.Buffer{}
	e := NewEngine(out)
	if err := e.executeFileValue(&statement.FileValue{Operation: "get", Format: "property", Selector: "pluginVersion", Target: path, CaptureVar: "current"}, ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Variables["current"] != "1.0.1" {
		t.Fatalf("capture = %q", ctx.Variables["current"])
	}
	err := e.executeFileValue(&statement.FileValue{Operation: "check", Format: "property", Selector: "pluginVersion", Target: path, Comparison: "equals", Expected: "9.9.9"}, ctx)
	if err == nil || !strings.Contains(err.Error(), "actual \"1.0.1\"") || !strings.Contains(err.Error(), path) {
		t.Fatalf("check error = %v", err)
	}
	e.SetDryRun(true)
	if err := e.executeFileValue(&statement.FileValue{Operation: "update", Format: "property", Selector: "pluginVersion", Target: path, Value: "{$next}", MissingPolicy: "fail"}, ctx); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "pluginVersion=1.0.1\n" {
		t.Fatal("dry run mutated file")
	}
	e.SetDryRun(false)
	if err := e.executeFileValue(&statement.FileValue{Operation: "update", Format: "property", Selector: "pluginVersion", Target: path, Value: "{$next}", MissingPolicy: "fail"}, ctx); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != "pluginVersion=1.0.2\n" {
		t.Fatalf("update = %q", data)
	}
}
