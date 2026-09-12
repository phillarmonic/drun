package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
)

func TestFindDefaultTaskPrefersDefaultOverEarlierStart(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "start"},
			{Name: "default"},
		},
	}

	if got := FindDefaultTask(program); got != "default" {
		t.Fatalf("expected default task to be selected, got %q", got)
	}
}

func TestFindDefaultTaskFallsBackThroughPriorityList(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "build"},
			{Name: "help"},
			{Name: "deploy"},
		},
	}

	if got := FindDefaultTask(program); got != "help" {
		t.Fatalf("expected help task to be selected, got %q", got)
	}
}

func TestFindDefaultTaskReturnsEmptyWhenNoMatch(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "build"},
			{Name: "deploy"},
		},
	}

	if got := FindDefaultTask(program); got != "" {
		t.Fatalf("expected empty default task when no match, got %q", got)
	}
}

func TestFindDefaultTaskDoesNotAutoRunStart(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "start"},
		},
	}

	if got := FindDefaultTask(program); got != "" {
		t.Fatalf("expected empty default task when only start is defined, got %q", got)
	}
}

func TestExecuteTaskReturnsExitCodeErrorForRuntimeFailure(t *testing.T) {
	// A failing task must come back as an *exitCodeError so the deferred
	// engine cleanup runs before the process exits (the old code called
	// os.Exit inside ExecuteTask, skipping it).
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "spec.drun")
	source := "version: 2.0\n\ntask \"boom\":\n\trun \"exit 3\"\n"
	if err := os.WriteFile(specPath, []byte(source), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := ExecuteTask(specPath, false, false, false, "", false, false, false, false, true, false, []string{"boom"})
	if err == nil {
		t.Fatal("ExecuteTask() error = nil, want a runtime failure")
	}

	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) {
		t.Fatalf("ExecuteTask() error = %T (%v), want *exitCodeError", err, err)
	}
	if exitErr.code != 1 {
		t.Fatalf("exitCodeError.code = %d, want 1", exitErr.code)
	}
}

func TestExecuteTaskRejectsYesAndNoTogether(t *testing.T) {
	// The mutual-exclusion check must fire before any file lookup, so this
	// call needs no task file on disk.
	err := ExecuteTask("", false, false, false, "", false, false, false, false, true, true, nil)
	if err == nil {
		t.Fatal("ExecuteTask() error = nil, want a mutual-exclusion error")
	}
	if !strings.Contains(err.Error(), "--yes and --no cannot be used together") {
		t.Fatalf("ExecuteTask() error = %q, want a clear --yes/--no conflict message", err)
	}
}
