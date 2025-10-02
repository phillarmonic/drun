package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
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
