package lsp

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/engine"
)

func TestAppendTaskCompletionsIncludesVariantDetails(t *testing.T) {
	items := appendTaskCompletions(nil, map[string]struct{}{}, mustParseProgram(t, `version: 2.0

@platform("linux")
task "shell":
  info "linux"

@platform("mac")
task "shell":
  info "mac"
`))

	foundLinux := false
	foundMac := false
	for _, item := range items {
		if item.Label != "shell" {
			continue
		}
		switch item.Detail {
		case "Task [linux]":
			foundLinux = true
		case "Task [mac]":
			foundMac = true
		}
	}

	if !foundLinux || !foundMac {
		t.Fatalf("expected linux and mac variant completion details, got %#v", items)
	}
}

func mustParseProgram(t *testing.T, input string) *ast.Program {
	t.Helper()
	program, err := engine.ParseStringWithFilename(input, "<test>")
	if err != nil {
		t.Fatalf("ParseStringWithFilename() error = %v", err)
	}
	return program
}
