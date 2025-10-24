package engine

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/types"
)

func TestResolveServiceContextLiteral(t *testing.T) {
	baseDir := t.TempDir()
	serviceRelPath := filepath.Join("services", "api")
	serviceAbsPath := filepath.Join(baseDir, serviceRelPath)

	ctx := &ExecutionContext{
		CurrentFile: filepath.Join(baseDir, ".drun", "spec.drun"),
		Program: &ast.Program{
			Services: []*ast.ServiceStatement{
				{Name: "api", Path: serviceRelPath},
			},
		},
	}

	eng := NewEngine(io.Discard)

	svcCtx, err := eng.resolveServiceContext("api", true, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svcCtx.Name != "api" {
		t.Fatalf("expected service name 'api', got %s", svcCtx.Name)
	}

	if svcCtx.Path != serviceAbsPath {
		t.Fatalf("expected path %s, got %s", serviceAbsPath, svcCtx.Path)
	}
}

func TestResolveServiceContextParameter(t *testing.T) {
	baseDir := t.TempDir()
	ctx := &ExecutionContext{
		CurrentFile: filepath.Join(baseDir, ".drun", "spec.drun"),
		Parameters:  map[string]*types.Value{},
		Program: &ast.Program{
			Services: []*ast.ServiceStatement{
				{Name: "api", Path: "services/api"},
			},
		},
	}

	value, err := types.NewValue(types.StringType, "api")
	if err != nil {
		t.Fatalf("creating parameter value: %v", err)
	}
	ctx.Parameters["servicename"] = value

	eng := NewEngine(io.Discard)

	svcCtx, err := eng.resolveServiceContext("$servicename", false, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svcCtx.Name != "api" {
		t.Fatalf("expected service name 'api', got %s", svcCtx.Name)
	}
}
