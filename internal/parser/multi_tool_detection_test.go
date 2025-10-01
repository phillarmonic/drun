package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestMultiToolAvailability(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTools []string // first is Target, rest are Alternatives
		condition     string
	}{
		{
			name: "single tool available",
			input: `version: 2.0

task "test":
  if docker is available:
    info "ok"`,
			expectedTools: []string{"docker"},
			condition:     "available",
		},
		{
			name: "two tools available with comma",
			input: `version: 2.0

task "test":
  if docker,"docker-compose" is available:
    info "ok"`,
			expectedTools: []string{"docker", "docker-compose"},
			condition:     "available",
		},
		{
			name: "multiple tools not available",
			input: `version: 2.0

task "test":
  if docker,"docker-compose",kubectl is not available:
    error "missing"`,
			expectedTools: []string{"docker", "docker-compose", "kubectl"},
			condition:     "not_available",
		},
		{
			name: "mixed quoted and unquoted tools",
			input: `version: 2.0

task "test":
  if docker,"docker compose" is available:
    info "ok"`,
			expectedTools: []string{"docker", "docker compose"},
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

			// Check target
			if stmt.Target != tt.expectedTools[0] {
				t.Errorf("expected target '%s', got '%s'", tt.expectedTools[0], stmt.Target)
			}

			// Check alternatives
			expectedAlternatives := tt.expectedTools[1:]
			if len(stmt.Alternatives) != len(expectedAlternatives) {
				t.Errorf("expected %d alternatives, got %d", len(expectedAlternatives), len(stmt.Alternatives))
			}

			for i, alt := range expectedAlternatives {
				if i >= len(stmt.Alternatives) {
					t.Errorf("missing alternative at index %d", i)
					continue
				}
				if stmt.Alternatives[i] != alt {
					t.Errorf("alternative[%d]: expected '%s', got '%s'", i, alt, stmt.Alternatives[i])
				}
			}
		})
	}
}

func TestMultiToolAvailabilityWithElse(t *testing.T) {
	input := `version: 2.0

task "test":
  if docker,"docker-compose" is not available:
    error "You don't seem to have docker on your machine."
  else:
    info "Docker detected."
`

	l := lexer.NewLexer(input)
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

	// Check the if statement
	ifStmt, ok := task.Body[0].(*ast.DetectionStatement)
	if !ok {
		t.Fatalf("expected DetectionStatement, got %T", task.Body[0])
	}

	if ifStmt.Target != "docker" {
		t.Errorf("expected target 'docker', got '%s'", ifStmt.Target)
	}

	if len(ifStmt.Alternatives) != 1 {
		t.Fatalf("expected 1 alternative, got %d", len(ifStmt.Alternatives))
	}

	if ifStmt.Alternatives[0] != "docker-compose" {
		t.Errorf("expected alternative 'docker-compose', got '%s'", ifStmt.Alternatives[0])
	}

	if ifStmt.Condition != "not_available" {
		t.Errorf("expected condition 'not_available', got '%s'", ifStmt.Condition)
	}

	// Check body
	if len(ifStmt.Body) != 1 {
		t.Fatalf("expected 1 statement in if body, got %d", len(ifStmt.Body))
	}

	// Check else body
	if len(ifStmt.ElseBody) != 1 {
		t.Fatalf("expected 1 statement in else body, got %d", len(ifStmt.ElseBody))
	}
}
