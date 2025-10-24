package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
)

func TestResolveCacheOption(t *testing.T) {
	t.Run("default true when no option set", func(t *testing.T) {
		result := resolveCacheOption(map[string]string{}, true)
		if !result {
			t.Fatalf("expected default true, got false")
		}
	})

	t.Run("cache false overrides default", func(t *testing.T) {
		result := resolveCacheOption(map[string]string{"cache": "false"}, true)
		if result {
			t.Fatalf("expected false when cache option is false")
		}
	})

	t.Run("no_cache true flips to false", func(t *testing.T) {
		result := resolveCacheOption(map[string]string{"no_cache": "true"}, true)
		if result {
			t.Fatalf("expected false when no_cache is true")
		}
	})

	t.Run("cache true wins over no_cache", func(t *testing.T) {
		opts := map[string]string{
			"cache":    "true",
			"no_cache": "true",
		}
		result := resolveCacheOption(opts, false)
		if !result {
			t.Fatalf("expected true when cache option explicitly true")
		}
	})
}

func TestBuildServiceWithOutput_NoCacheFlag(t *testing.T) {
	scriptDir := t.TempDir()
	argsFile := filepath.Join(scriptDir, "args.txt")

	scriptPath := filepath.Join(scriptDir, "docker")
	scriptContent := "#!/bin/sh\nprintf '%s' \"$*\" > \"$DRUN_TEST_ARGS_FILE\"\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write mock docker script: %v", err)
	}

	origPath := os.Getenv("PATH")
	if origPath != "" {
		t.Setenv("PATH", fmt.Sprintf("%s%c%s", scriptDir, os.PathListSeparator, origPath))
	} else {
		t.Setenv("PATH", scriptDir)
	}
	t.Setenv("DRUN_TEST_ARGS_FILE", argsFile)

	serviceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(serviceDir, "docker-compose.yml"), []byte("version: '3'"), 0o644); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	engine := NewEngine(io.Discard)
	service := &ast.ServiceStatement{
		Name: "web",
		Path: serviceDir,
	}

	if err := engine.buildServiceWithOutput(service, true); err != nil {
		t.Fatalf("buildServiceWithOutput with cache failed: %v", err)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read args file: %v", err)
	}

	argsStr := string(data)
	if strings.Contains(argsStr, "--no-cache") {
		t.Fatalf("did not expect --no-cache when cache enabled, got args: %s", argsStr)
	}

	if err := engine.buildServiceWithOutput(service, false); err != nil {
		t.Fatalf("buildServiceWithOutput without cache failed: %v", err)
	}

	data, err = os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read args file: %v", err)
	}

	argsStr = string(data)
	if !strings.Contains(argsStr, "--no-cache") {
		t.Fatalf("expected --no-cache flag, got args: %s", argsStr)
	}
}
