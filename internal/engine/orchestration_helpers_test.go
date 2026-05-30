package engine

import (
	"io"
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
)

func TestResolveOrchestrateServicesBuiltin(t *testing.T) {
	eng := NewEngine(io.Discard)
	execCtx := &ExecutionContext{
		Program: &ast.Program{
			Orchestrations: []*ast.OrchestrateStatement{
				{
					Name:     "stack",
					Services: []string{"api", "worker", "web"},
				},
			},
		},
	}

	got, err := eng.resolveOrchestrateServicesBuiltin(execCtx, []string{"stack"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `["api", "worker", "web"]`
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestResolveOrchestrateServicesBuiltinErrors(t *testing.T) {
	eng := NewEngine(io.Discard)

	if _, err := eng.resolveOrchestrateServicesBuiltin(&ExecutionContext{}, []string{"missing"}); err == nil {
		t.Fatalf("expected error for missing program context")
	}

	execCtx := &ExecutionContext{Program: &ast.Program{}}
	if _, err := eng.resolveOrchestrateServicesBuiltin(execCtx, []string{"missing"}); err == nil {
		t.Fatalf("expected error for missing orchestration")
	}
}
