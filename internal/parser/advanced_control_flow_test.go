package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_RangeLoop(t *testing.T) {
	input := `version: 2.0

task "range_test":
  for $i in range 1 to 10:
    info "Processing item {$i}"
    
  success "Range loop completed!"`

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
	if len(task.Body) != 2 {
		t.Fatalf("task should have 2 statements. got=%d", len(task.Body))
	}

	// Check Loop statement
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if loopStmt.Type != "range" {
		t.Errorf("loop type not 'range'. got=%q", loopStmt.Type)
	}

	if loopStmt.Variable != "$i" {
		t.Errorf("loop variable not '$i'. got=%q", loopStmt.Variable)
	}

	if loopStmt.RangeStart != "1" {
		t.Errorf("range start not '1'. got=%q", loopStmt.RangeStart)
	}

	if loopStmt.RangeEnd != "10" {
		t.Errorf("range end not '10'. got=%q", loopStmt.RangeEnd)
	}
}

func TestParser_RangeLoopWithStep(t *testing.T) {
	input := `version: 2.0

task "range_step_test":
  for $i in range 0 to 100 step 5:
    info "Processing item {$i}"
    
  success "Range loop with step completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if loopStmt.Type != "range" {
		t.Errorf("loop type not 'range'. got=%q", loopStmt.Type)
	}

	if loopStmt.RangeStart != "0" {
		t.Errorf("range start not '0'. got=%q", loopStmt.RangeStart)
	}

	if loopStmt.RangeEnd != "100" {
		t.Errorf("range end not '100'. got=%q", loopStmt.RangeEnd)
	}

	if loopStmt.RangeStep != "5" {
		t.Errorf("range step not '5'. got=%q", loopStmt.RangeStep)
	}
}

func TestParser_FilteredLoop(t *testing.T) {
	input := `version: 2.0

task "filtered_test":
  for each $item in $items where $item contains "test":
    info "Processing test item: {$item}"
    
  success "Filtered loop completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if loopStmt.Type != "each" {
		t.Errorf("loop type not 'each'. got=%q", loopStmt.Type)
	}

	if loopStmt.Variable != "$item" {
		t.Errorf("loop variable not '$item'. got=%q", loopStmt.Variable)
	}

	if loopStmt.Iterable != "$items" {
		t.Errorf("loop iterable not '$items'. got=%q", loopStmt.Iterable)
	}

	if loopStmt.Filter == nil {
		t.Fatalf("filter should not be nil")
	}

	if loopStmt.Filter.Variable != "$item" {
		t.Errorf("filter variable not '$item'. got=%q", loopStmt.Filter.Variable)
	}

	if loopStmt.Filter.Operator != "contains" {
		t.Errorf("filter operator not 'contains'. got=%q", loopStmt.Filter.Operator)
	}

	if loopStmt.Filter.Value != "test" {
		t.Errorf("filter value not 'test'. got=%q", loopStmt.Filter.Value)
	}
}

func TestParser_LineLoop(t *testing.T) {
	input := `version: 2.0

task "line_test":
  for each line text in file "data.txt":
    info "Processing line: {text}"
    
  success "Line loop completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if loopStmt.Type != "line" {
		t.Errorf("loop type not 'line'. got=%q", loopStmt.Type)
	}

	if loopStmt.Variable != "text" {
		t.Errorf("loop variable not 'text'. got=%q", loopStmt.Variable)
	}

	if loopStmt.Iterable != "data.txt" {
		t.Errorf("loop iterable not 'data.txt'. got=%q", loopStmt.Iterable)
	}
}

func TestParser_MatchLoop(t *testing.T) {
	input := `version: 2.0

task "match_test":
  for each match result in pattern "[0-9]+":
    info "Found number: {result}"
    
  success "Match loop completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if loopStmt.Type != "match" {
		t.Errorf("loop type not 'match'. got=%q", loopStmt.Type)
	}

	if loopStmt.Variable != "result" {
		t.Errorf("loop variable not 'result'. got=%q", loopStmt.Variable)
	}

	if loopStmt.Iterable != "[0-9]+" {
		t.Errorf("loop iterable not '[0-9]+'. got=%q", loopStmt.Iterable)
	}
}

func TestParser_BreakStatement(t *testing.T) {
	input := `version: 2.0

task "break_test":
  for each $item in $items:
    if $item == "stop":
      break
    info "Processing: {$item}"
    
  success "Break test completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if len(loopStmt.Body) != 2 {
		t.Fatalf("loop body should have 2 statements. got=%d", len(loopStmt.Body))
	}

	// The break should be inside the if statement
	ifStmt, ok := loopStmt.Body[0].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("first loop statement should be ConditionalStatement. got=%T", loopStmt.Body[0])
	}

	if len(ifStmt.Body) != 1 {
		t.Fatalf("if body should have 1 statement. got=%d", len(ifStmt.Body))
	}

	breakStmt, ok := ifStmt.Body[0].(*ast.BreakStatement)
	if !ok {
		t.Fatalf("if body should contain BreakStatement. got=%T", ifStmt.Body[0])
	}

	if breakStmt.Condition != "" {
		t.Errorf("break condition should be empty. got=%q", breakStmt.Condition)
	}
}

func TestParser_ConditionalBreak(t *testing.T) {
	input := `version: 2.0

task "conditional_break_test":
  for each $item in $items:
    break when $item == "stop"
    info "Processing: {$item}"
    
  success "Conditional break test completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if len(loopStmt.Body) != 2 {
		t.Logf("Loop body statements: %d", len(loopStmt.Body))
		for i, stmt := range loopStmt.Body {
			t.Logf("Statement %d: %T - %s", i, stmt, stmt.String())
		}
		t.Fatalf("loop body should have 2 statements. got=%d", len(loopStmt.Body))
	}

	breakStmt, ok := loopStmt.Body[0].(*ast.BreakStatement)
	if !ok {
		t.Fatalf("first loop statement should be BreakStatement. got=%T", loopStmt.Body[0])
	}

	if breakStmt.Condition == "" {
		t.Errorf("break condition should not be empty")
	}
}

func TestParser_ContinueStatement(t *testing.T) {
	input := `version: 2.0

task "continue_test":
  for each $item in $items:
    continue if $item == "skip"
    info "Processing: {$item}"
    
  success "Continue test completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if len(loopStmt.Body) != 2 {
		t.Fatalf("loop body should have 2 statements. got=%d", len(loopStmt.Body))
	}

	continueStmt, ok := loopStmt.Body[0].(*ast.ContinueStatement)
	if !ok {
		t.Fatalf("first loop statement should be ContinueStatement. got=%T", loopStmt.Body[0])
	}

	if continueStmt.Condition == "" {
		t.Errorf("continue condition should not be empty")
	}
}

func TestParser_ComplexAdvancedLoop(t *testing.T) {
	input := `version: 2.0

task "complex_loop_test":
  for each $item in $items where $item ends with ".js" in parallel:
    continue if $item contains "test"
    info "Processing JavaScript file: {$item}"
    break when $item == "stop.js"
    
  success "Complex loop test completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	loopStmt, ok := task.Body[0].(*ast.LoopStatement)
	if !ok {
		t.Fatalf("first statement should be LoopStatement. got=%T", task.Body[0])
	}

	if loopStmt.Type != "each" {
		t.Errorf("loop type not 'each'. got=%q", loopStmt.Type)
	}

	if loopStmt.Filter == nil {
		t.Fatalf("filter should not be nil")
	}

	if loopStmt.Filter.Operator != "ends with" {
		t.Errorf("filter operator not 'ends with'. got=%q", loopStmt.Filter.Operator)
	}

	if !loopStmt.Parallel {
		t.Errorf("loop should be parallel")
	}

	if len(loopStmt.Body) != 3 {
		t.Fatalf("loop body should have 3 statements. got=%d", len(loopStmt.Body))
	}

	// Check continue statement
	continueStmt, ok := loopStmt.Body[0].(*ast.ContinueStatement)
	if !ok {
		t.Fatalf("first loop statement should be ContinueStatement. got=%T", loopStmt.Body[0])
	}

	if continueStmt.Condition == "" {
		t.Errorf("continue condition should not be empty")
	}

	// Check break statement
	breakStmt, ok := loopStmt.Body[2].(*ast.BreakStatement)
	if !ok {
		t.Fatalf("third loop statement should be BreakStatement. got=%T", loopStmt.Body[2])
	}

	if breakStmt.Condition == "" {
		t.Errorf("break condition should not be empty")
	}
}
