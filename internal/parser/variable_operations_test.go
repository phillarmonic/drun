package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_LetStatement(t *testing.T) {
	input := `version: 2.0

task "let_test":
  let name = "John"
  let count = 42
  let active = true
  
  info "Variable assignment completed"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("program should have 1 task. got=%d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Body) != 4 {
		t.Fatalf("task should have 4 statements. got=%d", len(task.Body))
	}

	// Check first let statement
	letStmt, ok := task.Body[0].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("first statement should be VariableStatement. got=%T", task.Body[0])
	}

	if letStmt.Operation != "let" {
		t.Errorf("operation not 'let'. got=%q", letStmt.Operation)
	}

	if letStmt.Variable != "name" {
		t.Errorf("variable not 'name'. got=%q", letStmt.Variable)
	}

	if letStmt.Value != "John" {
		t.Errorf("value not 'John'. got=%q", letStmt.Value)
	}

	// Check second let statement
	letStmt2, ok := task.Body[1].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("second statement should be VariableStatement. got=%T", task.Body[1])
	}

	if letStmt2.Variable != "count" {
		t.Errorf("variable not 'count'. got=%q", letStmt2.Variable)
	}

	if letStmt2.Value != "42" {
		t.Errorf("value not '42'. got=%q", letStmt2.Value)
	}
}

func TestParser_SetStatement(t *testing.T) {
	input := `version: 2.0

task "set_test":
  set message to "Hello World"
  set counter to 100
  
  info "Variable setting completed"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 3 {
		t.Fatalf("task should have 3 statements. got=%d", len(task.Body))
	}

	// Check first set statement
	setStmt, ok := task.Body[0].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("first statement should be VariableStatement. got=%T", task.Body[0])
	}

	if setStmt.Operation != "set" {
		t.Errorf("operation not 'set'. got=%q", setStmt.Operation)
	}

	if setStmt.Variable != "message" {
		t.Errorf("variable not 'message'. got=%q", setStmt.Variable)
	}

	if setStmt.Value != "Hello World" {
		t.Errorf("value not 'Hello World'. got=%q", setStmt.Value)
	}
}

func TestParser_TransformStatement(t *testing.T) {
	input := `version: 2.0

task "transform_test":
  transform mytext with uppercase
  transform mylist with join ","
  transform name with concat " Smith"
  
  info "Variable transformation completed"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 4 {
		t.Fatalf("task should have 4 statements. got=%d", len(task.Body))
	}

	// Check first transform statement
	transformStmt, ok := task.Body[0].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("first statement should be VariableStatement. got=%T", task.Body[0])
	}

	if transformStmt.Operation != "transform" {
		t.Errorf("operation not 'transform'. got=%q", transformStmt.Operation)
	}

	if transformStmt.Variable != "mytext" {
		t.Errorf("variable not 'mytext'. got=%q", transformStmt.Variable)
	}

	if transformStmt.Function != "uppercase" {
		t.Errorf("function not 'uppercase'. got=%q", transformStmt.Function)
	}

	// Check second transform statement with arguments
	transformStmt2, ok := task.Body[1].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("second statement should be VariableStatement. got=%T", task.Body[1])
	}

	if transformStmt2.Function != "join" {
		t.Errorf("function not 'join'. got=%q", transformStmt2.Function)
	}

	if len(transformStmt2.Arguments) != 1 {
		t.Fatalf("should have 1 argument. got=%d", len(transformStmt2.Arguments))
	}

	if transformStmt2.Arguments[0] != "," {
		t.Errorf("argument not ','. got=%q", transformStmt2.Arguments[0])
	}

	// Check third transform statement with string argument
	transformStmt3, ok := task.Body[2].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("third statement should be VariableStatement. got=%T", task.Body[2])
	}

	if transformStmt3.Function != "concat" {
		t.Errorf("function not 'concat'. got=%q", transformStmt3.Function)
	}

	if len(transformStmt3.Arguments) != 1 {
		t.Fatalf("should have 1 argument. got=%d", len(transformStmt3.Arguments))
	}

	if transformStmt3.Arguments[0] != " Smith" {
		t.Errorf("argument not ' Smith'. got=%q", transformStmt3.Arguments[0])
	}
}

func TestParser_MixedVariableOperations(t *testing.T) {
	input := `version: 2.0

task "mixed_test":
  let firstName = "John"
  let lastName = "Doe"
  set fullName to "Unknown"
  transform fullName with concat firstName " " lastName
  transform fullName with uppercase
  
  info "Mixed variable operations completed"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 6 {
		t.Fatalf("task should have 6 statements. got=%d", len(task.Body))
	}

	// Verify the sequence of operations
	operations := []string{"let", "let", "set", "transform", "transform"}
	variables := []string{"firstName", "lastName", "fullName", "fullName", "fullName"}

	for i, expectedOp := range operations {
		varStmt, ok := task.Body[i].(*ast.VariableStatement)
		if !ok {
			t.Fatalf("statement %d should be VariableStatement. got=%T", i, task.Body[i])
		}

		if varStmt.Operation != expectedOp {
			t.Errorf("statement %d operation not '%s'. got=%q", i, expectedOp, varStmt.Operation)
		}

		if varStmt.Variable != variables[i] {
			t.Errorf("statement %d variable not '%s'. got=%q", i, variables[i], varStmt.Variable)
		}
	}
}

func TestParser_VariableOperationsInControlFlow(t *testing.T) {
	input := `version: 2.0

task "control_flow_test":
  given items defaults to "a,b,c"
  
  for each item in items:
    let processed = item
    transform processed with uppercase
    info "Processed: {processed}"
  
  if true:
    set result to "success"
    transform result with concat " - completed"
    
  success "Control flow with variables completed"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 3 {
		t.Fatalf("task should have 3 statements. got=%d", len(task.Body))
	}

	// Check loop body contains variable operations
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if len(loopStmt.Body) != 3 {
		t.Fatalf("loop body should have 3 statements. got=%d", len(loopStmt.Body))
	}

	// Check let statement in loop
	letStmt, ok := loopStmt.Body[0].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("first loop statement should be VariableStatement. got=%T", loopStmt.Body[0])
	}

	if letStmt.Operation != "let" {
		t.Errorf("operation not 'let'. got=%q", letStmt.Operation)
	}

	// Check transform statement in loop
	transformStmt, ok := loopStmt.Body[1].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("second loop statement should be VariableStatement. got=%T", loopStmt.Body[1])
	}

	if transformStmt.Operation != "transform" {
		t.Errorf("operation not 'transform'. got=%q", transformStmt.Operation)
	}

	// Check if body contains variable operations
	ifStmt, ok := task.Body[1].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("second statement should be ConditionalStatement. got=%T", task.Body[1])
	}

	if len(ifStmt.Body) != 2 {
		t.Fatalf("if body should have 2 statements. got=%d", len(ifStmt.Body))
	}

	// Check set statement in if
	setStmt, ok := ifStmt.Body[0].(*ast.VariableStatement)
	if !ok {
		t.Fatalf("first if statement should be VariableStatement. got=%T", ifStmt.Body[0])
	}

	if setStmt.Operation != "set" {
		t.Errorf("operation not 'set'. got=%q", setStmt.Operation)
	}
}

func TestParser_VariableOperationStringRepresentation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `let name = "John"`,
			expected: `let name = John`,
		},
		{
			input:    `set counter to 42`,
			expected: `set counter to 42`,
		},
		{
			input:    `transform text with uppercase`,
			expected: `transform text with uppercase`,
		},
		{
			input:    `transform list with join ","`,
			expected: `transform list with join ,`,
		},
	}

	for _, tt := range tests {
		input := `version: 2.0

task "test":
  ` + tt.input

		l := lexer.NewLexer(input)
		p := NewParser(l)
		program := p.ParseProgram()

		checkParserErrors(t, p)

		if program == nil {
			t.Fatalf("ParseProgram() returned nil")
		}

		task := program.Tasks[0]
		if len(task.Body) != 1 {
			t.Fatalf("task should have 1 statement. got=%d", len(task.Body))
		}

		varStmt, ok := task.Body[0].(*ast.VariableStatement)
		if !ok {
			t.Fatalf("statement should be VariableStatement. got=%T", task.Body[0])
		}

		if varStmt.String() != tt.expected {
			t.Errorf("String() not %q. got=%q", tt.expected, varStmt.String())
		}
	}
}
