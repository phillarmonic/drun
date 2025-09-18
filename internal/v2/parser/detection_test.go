package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/lexer"
)

func TestParser_DetectProjectType(t *testing.T) {
	input := `version: 2.0

task "analyze_project":
  detect project type
  
  success "Project type detected!"`

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

	// Check Detection statement
	detectionStmt, ok := task.Body[0].(*ast.DetectionStatement)
	if !ok {
		t.Fatalf("first statement should be DetectionStatement. got=%T", task.Body[0])
	}

	if detectionStmt.Type != "detect" {
		t.Errorf("detection type not 'detect'. got=%q", detectionStmt.Type)
	}

	if detectionStmt.Target != "project" {
		t.Errorf("detection target not 'project'. got=%q", detectionStmt.Target)
	}

	if detectionStmt.Condition != "type" {
		t.Errorf("detection condition not 'type'. got=%q", detectionStmt.Condition)
	}
}

func TestParser_DetectToolVersion(t *testing.T) {
	input := `version: 2.0

task "check_node":
  detect node version
  
  success "Node version detected!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	detectionStmt, ok := task.Body[0].(*ast.DetectionStatement)
	if !ok {
		t.Fatalf("first statement should be DetectionStatement. got=%T", task.Body[0])
	}

	if detectionStmt.Type != "detect" {
		t.Errorf("detection type not 'detect'. got=%q", detectionStmt.Type)
	}

	if detectionStmt.Target != "node" {
		t.Errorf("detection target not 'node'. got=%q", detectionStmt.Target)
	}

	if detectionStmt.Condition != "version" {
		t.Errorf("detection condition not 'version'. got=%q", detectionStmt.Condition)
	}
}

func TestParser_IfToolAvailable(t *testing.T) {
	input := `version: 2.0

task "docker_task":
  if docker is available:
    info "Docker is available"
    
  success "Docker check completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	detectionStmt, ok := task.Body[0].(*ast.DetectionStatement)
	if !ok {
		t.Fatalf("first statement should be DetectionStatement. got=%T", task.Body[0])
	}

	if detectionStmt.Type != "if_available" {
		t.Errorf("detection type not 'if_available'. got=%q", detectionStmt.Type)
	}

	if detectionStmt.Target != "docker" {
		t.Errorf("detection target not 'docker'. got=%q", detectionStmt.Target)
	}

	if detectionStmt.Condition != "available" {
		t.Errorf("detection condition not 'available'. got=%q", detectionStmt.Condition)
	}

	if len(detectionStmt.Body) != 1 {
		t.Fatalf("detection body should have 1 statement. got=%d", len(detectionStmt.Body))
	}
}

func TestParser_IfVersionComparison(t *testing.T) {
	input := `version: 2.0

task "node_version_check":
  if node version >= "16":
    info "Node version is 16 or higher"
  else:
    warn "Node version is below 16"
    
  success "Node version check completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	detectionStmt, ok := task.Body[0].(*ast.DetectionStatement)
	if !ok {
		t.Fatalf("first statement should be DetectionStatement. got=%T", task.Body[0])
	}

	if detectionStmt.Type != "if_version" {
		t.Errorf("detection type not 'if_version'. got=%q", detectionStmt.Type)
	}

	if detectionStmt.Target != "node" {
		t.Errorf("detection target not 'node'. got=%q", detectionStmt.Target)
	}

	if detectionStmt.Condition != ">=" {
		t.Errorf("detection condition not '>='. got=%q", detectionStmt.Condition)
	}

	if detectionStmt.Value != "16" {
		t.Errorf("detection value not '16'. got=%q", detectionStmt.Value)
	}

	if len(detectionStmt.Body) != 1 {
		t.Fatalf("detection body should have 1 statement. got=%d", len(detectionStmt.Body))
	}

	if len(detectionStmt.ElseBody) != 1 {
		t.Fatalf("detection else body should have 1 statement. got=%d", len(detectionStmt.ElseBody))
	}
}

func TestParser_WhenEnvironment(t *testing.T) {
	input := `version: 2.0

task "ci_task":
  when in ci environment:
    info "Running in CI environment"
    
  success "Environment check completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	detectionStmt, ok := task.Body[0].(*ast.DetectionStatement)
	if !ok {
		t.Fatalf("first statement should be DetectionStatement. got=%T", task.Body[0])
	}

	if detectionStmt.Type != "when_environment" {
		t.Errorf("detection type not 'when_environment'. got=%q", detectionStmt.Type)
	}

	if detectionStmt.Target != "ci" {
		t.Errorf("detection target not 'ci'. got=%q", detectionStmt.Target)
	}

	if detectionStmt.Condition != "environment" {
		t.Errorf("detection condition not 'environment'. got=%q", detectionStmt.Condition)
	}

	if len(detectionStmt.Body) != 1 {
		t.Fatalf("detection body should have 1 statement. got=%d", len(detectionStmt.Body))
	}
}

func TestParser_MultipleDetectionStatements(t *testing.T) {
	input := `version: 2.0

task "comprehensive_check":
  detect project type
  if docker is available:
    info "Docker is available"
  when in production environment:
    warn "Running in production"
  if node version >= "18":
    info "Node 18+ detected"
    
  success "All checks completed!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Body) != 5 {
		t.Fatalf("task should have 5 statements. got=%d", len(task.Body))
	}

	// Check all four detection statements
	detectionTypes := []string{"detect", "if_available", "when_environment", "if_version"}
	for i, expectedType := range detectionTypes {
		detectionStmt, ok := task.Body[i].(*ast.DetectionStatement)
		if !ok {
			t.Fatalf("statement %d should be DetectionStatement. got=%T", i, task.Body[i])
		}

		if detectionStmt.Type != expectedType {
			t.Errorf("detection type %d not '%s'. got=%q", i, expectedType, detectionStmt.Type)
		}
	}
}
