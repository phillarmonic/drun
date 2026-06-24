package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

func TestMultiToolDetectionLogic(t *testing.T) {
	tests := []struct {
		name           string
		script         string
		expectInOutput string
		expectError    bool
	}{
		{
			name: "all tools available - is available (should execute if body)",
			script: `version: 2.0

task "test":
  if git,go is available:
    info "Both git and go are available"
  else:
    error "One or both tools missing"`,
			expectInOutput: "Both git and go are available",
			expectError:    false,
		},
		{
			name: "one tool missing - is available (should execute else body)",
			script: `version: 2.0

task "test":
  if git,"nonexistent-tool-xyz" is available:
    error "Should not reach here"
  else:
    info "One tool is missing as expected"`,
			expectInOutput: "One tool is missing as expected",
			expectError:    false,
		},
		{
			name: "one tool missing - is not available (should execute if body)",
			script: `version: 2.0

task "test":
  if git,"nonexistent-tool-xyz" is not available:
    info "At least one tool is not available"
  else:
    error "Should not reach here"`,
			expectInOutput: "At least one tool is not available",
			expectError:    false,
		},
		{
			name: "all tools available - is not available (should execute else body)",
			script: `version: 2.0

task "test":
  if git,go is not available:
    error "Should not reach here"
  else:
    info "All tools are available"`,
			expectInOutput: "All tools are available",
			expectError:    false,
		},
		{
			name: "three tools mixed availability",
			script: `version: 2.0

task "test":
  if git,"docker-compose","nonexistent-xyz" is not available:
    info "At least one tool is missing"
  else:
    error "Should not reach here"`,
			expectInOutput: "At least one tool is missing",
			expectError:    false,
		},
		{
			name: "quoted tool available with version constraint",
			script: `version: 2.0

task "test":
  if "go" is available and version >= "1.0":
    info "Go is available and version is sufficient"
  else:
    error "Should not reach here"`,
			expectInOutput: "Go is available and version is sufficient",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)

			l := lexer.NewLexer(tt.script)
			p := parser.NewParser(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			err := engine.Execute(program, "test")

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			outputStr := output.String()
			if tt.expectInOutput != "" && !strings.Contains(outputStr, tt.expectInOutput) {
				t.Errorf("expected output to contain %q, but got:\n%s", tt.expectInOutput, outputStr)
			}
		})
	}
}

func TestDetectionRunningLogic(t *testing.T) {
	originalPath := os.Getenv("PATH")
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "docker")

	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"compose\" ] && [ \"$2\" = \"version\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"info\" ]; then\n" +
		"  exit 1\n" +
		"fi\n" +
		"exit 0\n"

	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(tmpDir, "docker.bat")
		script = "@echo off\r\n" +
			"if \"%1\"==\"compose\" if \"%2\"==\"version\" exit /b 0\r\n" +
			"if \"%1\"==\"info\" exit /b 1\r\n" +
			"exit /b 0\r\n"
	}

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake docker: %v", err)
	}

	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+originalPath)

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	l := lexer.NewLexer(`version: 2.0

task "test":
  if docker is not running:
    info "Docker daemon is not reachable"
  else:
    error "Should not reach here"`)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if err := engine.Execute(program, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output.String(), "Docker daemon is not reachable") {
		t.Fatalf("expected output to mention docker daemon not reachable, got:\n%s", output.String())
	}
}
