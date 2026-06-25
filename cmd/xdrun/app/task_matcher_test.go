package app

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
)

func TestResolvePartialTaskNameDeduplicatesPlatformVariants(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "shell"},
			{Name: "shell"},
			{Name: "serve"},
		},
	}

	got, err := ResolvePartialTaskName("she", program)
	if err != nil {
		t.Fatalf("ResolvePartialTaskName() error = %v", err)
	}
	if got != "shell" {
		t.Fatalf("expected shell, got %q", got)
	}
}
