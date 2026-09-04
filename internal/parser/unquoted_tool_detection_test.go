package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// Regression: an unquoted hyphenated tool name (`if my-fake-tool is available`)
// lexes as one dashed IDENT, but isDetectionContext only routed tool keywords
// and STRINGs to detection parsing, so the statement fell into the generic
// conditional and silently evaluated false even when the tool existed.
func TestUnquotedHyphenatedToolAvailability(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTools []string // first is Target, rest are Alternatives
		condition     string
	}{
		{
			name: "single unquoted hyphenated tool available",
			input: `version: 2.0

task "test":
  if my-fake-tool is available:
    info "ok"`,
			expectedTools: []string{"my-fake-tool"},
			condition:     "available",
		},
		{
			name: "single unquoted hyphenated tool not available",
			input: `version: 2.0

task "test":
  if my-fake-tool is not available:
    error "missing"`,
			expectedTools: []string{"my-fake-tool"},
			condition:     "not_available",
		},
		{
			name: "unquoted identifier with are running",
			input: `version: 2.0

task "test":
  if gofmt-fixer are running:
    info "ok"`,
			expectedTools: []string{"gofmt-fixer"},
			condition:     "running",
		},
		{
			name: "mixed keyword and unquoted hyphenated tools",
			input: `version: 2.0

task "test":
  if docker,my-fake-tool is not available:
    error "missing"`,
			expectedTools: []string{"docker", "my-fake-tool"},
			condition:     "not_available",
		},
		{
			name: "quoted hyphenated tool still works",
			input: `version: 2.0

task "test":
  if "my-fake-tool" is available:
    info "ok"`,
			expectedTools: []string{"my-fake-tool"},
			condition:     "available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.NewLexer(tt.input)
			p := NewParser(l)
			program := p.ParseProgram()

			checkParserErrors(t, p)

			if len(program.Tasks) != 1 {
				t.Fatalf("expected 1 task, got %d", len(program.Tasks))
			}

			task := program.Tasks[0]
			if len(task.Body) != 1 {
				t.Fatalf("expected 1 statement in task body, got %d", len(task.Body))
			}

			stmt, ok := task.Body[0].(*ast.DetectionStatement)
			if !ok {
				t.Fatalf("expected DetectionStatement, got %T", task.Body[0])
			}

			if stmt.Type != "if_available" {
				t.Errorf("expected type 'if_available', got '%s'", stmt.Type)
			}
			if stmt.Condition != tt.condition {
				t.Errorf("expected condition '%s', got '%s'", tt.condition, stmt.Condition)
			}
			if stmt.Target != tt.expectedTools[0] {
				t.Errorf("expected target '%s', got '%s'", tt.expectedTools[0], stmt.Target)
			}

			expectedAlternatives := tt.expectedTools[1:]
			if len(stmt.Alternatives) != len(expectedAlternatives) {
				t.Fatalf("expected %d alternatives, got %d", len(expectedAlternatives), len(stmt.Alternatives))
			}
			for i, alt := range expectedAlternatives {
				if stmt.Alternatives[i] != alt {
					t.Errorf("alternative[%d]: expected '%s', got '%s'", i, alt, stmt.Alternatives[i])
				}
			}
		})
	}
}

// Guard: a generic comparison of a bare identifier against a quoted value
// (`if my_mode is "dev"`) must stay a conditional, not become a detection.
func TestUnquotedIdentifierComparisonStaysConditional(t *testing.T) {
	input := `version: 2.0

task "test":
  if my_mode is "dev":
    info "ok"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("expected 1 statement in task body, got %d", len(task.Body))
	}

	if _, ok := task.Body[0].(*ast.ConditionalStatement); !ok {
		t.Fatalf("expected ConditionalStatement, got %T", task.Body[0])
	}
}

// Guard: `if tags is not empty:` (a generic conditional against a non-tool
// property) must stay a conditional even though the subject is a bare
// identifier and the phrase contains "is not".
func TestUnquotedIdentifierIsNotEmptyStaysConditional(t *testing.T) {
	input := `version: 2.0

task "test":
  if tags is not empty:
    info "ok"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("expected 1 statement in task body, got %d", len(task.Body))
	}

	if _, ok := task.Body[0].(*ast.ConditionalStatement); !ok {
		t.Fatalf("expected ConditionalStatement, got %T", task.Body[0])
	}
}
