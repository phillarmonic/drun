package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

// TestParser_NewlineHandling tests that the parser correctly handles NEWLINE tokens
// at the top level, which was causing issues on Windows GitHub Actions.
func TestParser_NewlineHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "newline_after_comments",
			input: `# Comment
# Another comment
version: 2.0

project "test":

task "hello":
	info "test"
`,
		},
		{
			name: "multiple_newlines_between_sections",
			input: `version: 2.0


project "test":


task "hello":
	info "test"
`,
		},
		{
			name: "newlines_at_start",
			input: `

version: 2.0

task "hello":
	info "test"
`,
		},
		{
			name: "clipboarder_style",
			input: `# Clipboarder - Cross-platform clipboard library development automation
# Built with drun v2 - https://github.com/phillarmonic/drun
version: 2.0

project "clipboarder" version "1.0":

task "usage" means "Show available tasks and usage information":
	info "🚀 Clipboarder Development Tasks"
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

			// For tests with project statements, check they were parsed
			if tt.name == "clipboarder_style" || tt.name == "multiple_newlines_between_sections" || tt.name == "newline_after_comments" {
				if program.Project == nil {
					t.Error("Project statement not parsed")
				}
			}

			// Check that at least one task was parsed
			if len(program.Tasks) == 0 {
				t.Error("No tasks were parsed")
			}
		})
	}
}

// TestParser_NewlineTokensSkipped verifies that NEWLINE tokens are properly skipped
// in the main parsing loop and skipComments function.
func TestParser_NewlineTokensSkipped(t *testing.T) {
	input := `version: 2.0

task "test":
	info "hello"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)

	// Parse the program
	program := p.ParseProgram()

	// Should have no errors
	if len(p.Errors()) > 0 {
		t.Errorf("Expected no errors, got: %v", p.Errors())
	}

	// Should have parsed successfully
	if program == nil {
		t.Fatal("Expected program to be parsed")
	}

	if program.Version == nil {
		t.Error("Expected version to be parsed")
	}

	if len(program.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(program.Tasks))
	}
}
