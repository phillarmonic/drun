package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

// TestParser_WindowsNewlineAfterColon tests the specific issue that was failing on Windows
// where NEWLINE tokens after colons in task definitions were causing parse errors.
func TestParser_WindowsNewlineAfterColon(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "task_with_newline_after_colon",
			input: `version: 2.0

task "hello":
	info "test"
`,
		},
		{
			name: "task_with_means_and_newline",
			input: `version: 2.0

task "greet" means "Say hello":
	info "Hello!"
`,
		},
		{
			name: "project_with_newline_after_colon",
			input: `version: 2.0

project "test" version "1.0":
	set env to "test"

task "hello":
	info "test"
`,
		},
		{
			name: "multiline_shell_with_newline",
			input: `version: 2.0

task "shell":
	run:
		echo "hello"
		echo "world"
`,
		},
		{
			name: "multiple_newlines_after_colon",
			input: `version: 2.0

task "test":

	info "test with extra blank line"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer(tt.input)
			p := NewParser(l)
			program := p.ParseProgram()

			// Check that parsing succeeded (no errors)
			if len(p.Errors()) > 0 {
				t.Errorf("Parser errors: %v", p.Errors())
			}

			// Check that we got a valid program
			if program == nil {
				t.Fatal("ParseProgram() returned nil")
			}

			// Check that version was parsed
			if program.Version == nil {
				t.Error("Version statement not parsed")
			}

			// Check that at least one task was parsed
			if len(program.Tasks) == 0 {
				t.Error("No tasks were parsed")
			}

			// For project tests, check project was parsed
			if tt.name == "project_with_newline_after_colon" {
				if program.Project == nil {
					t.Error("Project statement not parsed")
				}
			}
		})
	}
}

// TestParser_WindowsNewlineErrorRegression tests the specific error pattern that was
// occurring on Windows to ensure it doesn't regress.
func TestParser_WindowsNewlineErrorRegression(t *testing.T) {
	// This is the exact pattern from examples/01-hello-world.drun that was failing
	input := `# Hello World - Your First drun v2 Task
# This demonstrates the most basic semantic syntax

version: 2.0

task "hello":
	info "Hello from drun v2! 👋"

task "hello world":
	step "Starting hello world example"
	info "Welcome to the semantic task runner!"
	success "Hello world completed successfully!"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	// This should NOT produce the error:
	// "Error: expected next token to be INDENT, got NEWLINE instead"
	if len(p.Errors()) > 0 {
		t.Errorf("Unexpected parser errors (this was the Windows regression): %v", p.Errors())
	}

	if program == nil {
		t.Fatal("ParseProgram() returned nil")
	}

	// Should have parsed 2 tasks
	if len(program.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(program.Tasks))
	}

	// Check task names
	expectedTasks := []string{"hello", "hello world"}
	for i, task := range program.Tasks {
		if task.Name != expectedTasks[i] {
			t.Errorf("Expected task name %q, got %q", expectedTasks[i], task.Name)
		}
	}
}
