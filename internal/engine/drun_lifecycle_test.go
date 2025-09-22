package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

func TestDrunLifecycleHooks(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		expectedOutput []string // strings that should be present in output
		shouldFail     bool
	}{
		{
			name: "drun setup and teardown hooks execution",
			input: `version: 2.0

project "myapp":
  on drun setup:
    info "🚀 Starting drun execution pipeline"
    info "📊 Tool version: v2.0"
  
  on drun teardown:
    info "🏁 Drun execution pipeline completed"
    info "📊 Total execution time: 5s"

task "deploy":
  info "Deploying application"`,
			taskName: "deploy",
			expectedOutput: []string{
				"🚀 Starting drun execution pipeline",
				"📊 Tool version: v2.0",
				"Deploying application",
				"🏁 Drun execution pipeline completed",
				"📊 Total execution time: 5s",
			},
			shouldFail: false,
		},
		{
			name: "mixed lifecycle hooks - old and new syntax",
			input: `version: 2.0

project "myapp":
  on drun setup:
    info "Tool starting up"
  
  before any task:
    info "Task starting"
  
  after any task:
    info "Task completed"
  
  on drun teardown:
    info "Tool shutting down"

task "test":
  info "Running test"`,
			taskName: "test",
			expectedOutput: []string{
				"Tool starting up",
				"Task starting",
				"Running test",
				"Task completed",
				"Tool shutting down",
			},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a buffer to capture output
			var output bytes.Buffer

			// Parse the input
			l := lexer.NewLexer(tt.input)
			p := parser.NewParser(l)
			program := p.ParseProgram()

			// Check for parse errors
			if len(p.Errors()) > 0 {
				t.Fatalf("Parse errors: %v", p.Errors())
			}

			// Create engine and execute
			eng := NewEngine(&output)
			err := eng.Execute(program, tt.taskName)

			// Check if execution should fail
			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected execution to fail, but it succeeded")
				}
				return
			}

			// Check for unexpected errors
			if err != nil {
				t.Fatalf("Unexpected execution error: %v", err)
			}

			outputStr := output.String()

			// Check expected messages are present and in correct order
			lastIndex := -1
			for _, expected := range tt.expectedOutput {
				index := strings.Index(outputStr, expected)
				if index == -1 {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				} else if index < lastIndex {
					t.Errorf("Expected %q to appear after previous message, but order is wrong in:\n%s", expected, outputStr)
				}
				lastIndex = index
			}
		})
	}
}
