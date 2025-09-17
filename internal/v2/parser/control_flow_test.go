package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/lexer"
)

func TestParser_SimpleWhenStatement(t *testing.T) {
	input := `version: 2.0

task "deploy":
  requires environment
  
  step "Starting deployment"
  
  when environment is "production":
    warn "Deploying to production!"
    step "Extra validation"
  
  success "Deployment completed!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) != 0 {
		t.Fatalf("parser has %d errors", len(parser.Errors()))
		for _, err := range parser.Errors() {
			t.Errorf("parser error: %q", err)
		}
	}

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("program.Tasks does not contain 1 task. got=%d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if task.Name != "deploy" {
		t.Errorf("task.Name not 'deploy'. got=%q", task.Name)
	}

	// Should have 3 statements: step, when, success
	if len(task.Body) != 3 {
		t.Fatalf("task.Body does not contain 3 statements. got=%d", len(task.Body))
	}

	// Check the when statement
	whenStmt, ok := task.Body[1].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("task.Body[1] is not a ConditionalStatement")
	}

	if whenStmt.Type != "when" {
		t.Errorf("whenStmt.Type not 'when'. got=%q", whenStmt.Type)
	}

	if whenStmt.Condition != `environment is production` {
		t.Errorf("whenStmt.Condition not correct. got=%q", whenStmt.Condition)
	}

	// Check when body has 2 statements
	if len(whenStmt.Body) != 2 {
		t.Fatalf("whenStmt.Body does not contain 2 statements. got=%d", len(whenStmt.Body))
	}
}

func TestParser_SimpleIfElseStatement(t *testing.T) {
	input := `version: 2.0

task "test":
  given skip_tests defaults to "false"
  
  if skip_tests is "false":
    step "Running tests"
    info "Tests passed"
  else:
    warn "Skipping tests"
  
  success "Done!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) != 0 {
		t.Fatalf("parser has %d errors", len(parser.Errors()))
		for _, err := range parser.Errors() {
			t.Errorf("parser error: %q", err)
		}
	}

	task := program.Tasks[0]

	// Should have 2 statements: if, success
	if len(task.Body) != 2 {
		t.Fatalf("task.Body does not contain 2 statements. got=%d", len(task.Body))
	}

	// Check the if statement
	ifStmt, ok := task.Body[0].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("task.Body[0] is not a ConditionalStatement")
	}

	if ifStmt.Type != "if" {
		t.Errorf("ifStmt.Type not 'if'. got=%q", ifStmt.Type)
	}

	if ifStmt.Condition != `skip_tests is false` {
		t.Errorf("ifStmt.Condition not correct. got=%q", ifStmt.Condition)
	}

	// Check if body has 2 statements
	if len(ifStmt.Body) != 2 {
		t.Fatalf("ifStmt.Body does not contain 2 statements. got=%d", len(ifStmt.Body))
	}

	// Check else body has 1 statement
	if len(ifStmt.ElseBody) != 1 {
		t.Fatalf("ifStmt.ElseBody does not contain 1 statement. got=%d", len(ifStmt.ElseBody))
	}
}
