package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_WhenOtherwiseStatement(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		condition string
		hasElse   bool
	}{
		{
			name: "simple when-otherwise",
			input: `version: 2.0
task "test":
	when $platform is "windows":
		step "Windows build"
	otherwise:
		step "Non-Windows build"`,
			condition: "$platform is windows",
			hasElse:   true,
		},
		{
			name: "when without otherwise",
			input: `version: 2.0
task "test":
	when $env is "production":
		step "Production deployment"`,
			condition: "$env is production",
			hasElse:   false,
		},
		{
			name: "when with complex condition",
			input: `version: 2.0
task "test":
	when $version is not "1.0":
		step "New version handling"
	otherwise:
		step "Legacy version handling"`,
			condition: "$version is not 1.0",
			hasElse:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := lexer.NewLexer(tt.input)
			parser := NewParser(lexer)
			program := parser.ParseProgram()

			checkParserErrors(t, parser)

			if program == nil {
				t.Fatalf("ParseProgram() returned nil")
			}

			if len(program.Tasks) != 1 {
				t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
			}

			task := program.Tasks[0]
			if len(task.Body) != 1 {
				t.Fatalf("Expected 1 statement in task body, got %d", len(task.Body))
			}

			whenStmt, ok := task.Body[0].(*ast.ConditionalStatement)
			if !ok {
				t.Fatalf("Expected ConditionalStatement, got %T", task.Body[0])
			}

			if whenStmt.Type != "when" {
				t.Errorf("Expected statement type 'when', got %q", whenStmt.Type)
			}

			if whenStmt.Condition != tt.condition {
				t.Errorf("Expected condition %q, got %q", tt.condition, whenStmt.Condition)
			}

			if len(whenStmt.Body) == 0 {
				t.Error("Expected when body to have statements")
			}

			if tt.hasElse {
				if len(whenStmt.ElseBody) == 0 {
					t.Error("Expected otherwise body to have statements")
				}
			} else {
				if len(whenStmt.ElseBody) != 0 {
					t.Error("Expected no otherwise body")
				}
			}
		})
	}
}

func TestParser_NestedWhenOtherwise(t *testing.T) {
	input := `version: 2.0
task "nested test":
	when $platform is "windows":
		step "Windows detected"
		when $arch is "amd64":
			step "Windows x64"
		otherwise:
			step "Windows ARM"
	otherwise:
		step "Non-Windows platform"
		when $platform is "linux":
			step "Linux detected"
		otherwise:
			step "Other platform"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	outerWhen, ok := task.Body[0].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("Expected ConditionalStatement, got %T", task.Body[0])
	}

	// Check outer when has nested when in body
	if len(outerWhen.Body) != 2 {
		t.Fatalf("Expected 2 statements in outer when body, got %d", len(outerWhen.Body))
	}

	innerWhen, ok := outerWhen.Body[1].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("Expected nested ConditionalStatement, got %T", outerWhen.Body[1])
	}

	if innerWhen.Condition != "$arch is amd64" {
		t.Errorf("Expected inner condition '$arch is amd64', got %q", innerWhen.Condition)
	}

	// Check otherwise clause has nested when
	if len(outerWhen.ElseBody) != 2 {
		t.Fatalf("Expected 2 statements in otherwise body, got %d", len(outerWhen.ElseBody))
	}

	elseWhen, ok := outerWhen.ElseBody[1].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("Expected nested ConditionalStatement in else, got %T", outerWhen.ElseBody[1])
	}

	if elseWhen.Condition != "$platform is linux" {
		t.Errorf("Expected else condition '$platform is linux', got %q", elseWhen.Condition)
	}
}

func TestParser_WhenOtherwiseWithLoops(t *testing.T) {
	input := `version: 2.0
task "loop test":
	for each $platform in ["windows", "linux"]:
		when $platform is "windows":
			step "Building {$platform} with .exe"
		otherwise:
			step "Building {$platform} without .exe"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loop, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("Expected LoopStatement, got %T", task.Body[0])
	}

	if len(loop.Body) != 1 {
		t.Fatalf("Expected 1 statement in loop body, got %d", len(loop.Body))
	}

	whenStmt, ok := loop.Body[0].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("Expected ConditionalStatement in loop, got %T", loop.Body[0])
	}

	if whenStmt.Condition != "$platform is windows" {
		t.Errorf("Expected condition '$platform is windows', got %q", whenStmt.Condition)
	}

	if len(whenStmt.ElseBody) == 0 {
		t.Error("Expected otherwise clause in loop")
	}
}
