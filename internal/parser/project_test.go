package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_ProjectDeclaration(t *testing.T) {
	input := `version: 2.0

project "myapp":
  set registry to "ghcr.io/company"
  set timeout to "5m"
  include "shared/common.drun"

task "hello":
  info "Hello from {registry}!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if program.Version == nil {
		t.Fatalf("program.Version is nil")
	}

	if program.Project == nil {
		t.Fatalf("program.Project is nil")
	}

	// Check project name
	if program.Project.Name != "myapp" {
		t.Errorf("project.Name not 'myapp'. got=%q", program.Project.Name)
	}

	// Check project settings
	if len(program.Project.Settings) != 3 {
		t.Fatalf("project should have 3 settings. got=%d", len(program.Project.Settings))
	}

	// Check first setting (set registry)
	setSetting, ok := program.Project.Settings[0].(*ast.SetStatement)
	if !ok {
		t.Fatalf("project.Settings[0] is not *ast.SetStatement. got=%T", program.Project.Settings[0])
	}

	if setSetting.Key != "registry" {
		t.Errorf("setSetting.Key not 'registry'. got=%q", setSetting.Key)
	}

	if setSetting.Value.String() != "ghcr.io/company" {
		t.Errorf("setSetting.Value not 'ghcr.io/company'. got=%q", setSetting.Value)
	}

	// Check include setting
	includeSetting, ok := program.Project.Settings[2].(*ast.IncludeStatement)
	if !ok {
		t.Fatalf("project.Settings[2] is not *ast.IncludeStatement. got=%T", program.Project.Settings[2])
	}

	if includeSetting.Path != "shared/common.drun" {
		t.Errorf("includeSetting.Path not 'shared/common.drun'. got=%q", includeSetting.Path)
	}

	// Check tasks still work
	if len(program.Tasks) != 1 {
		t.Fatalf("program should have 1 task. got=%d", len(program.Tasks))
	}

	if program.Tasks[0].Name != "hello" {
		t.Errorf("task.Name not 'hello'. got=%q", program.Tasks[0].Name)
	}
}

func TestParser_ProjectWithVersion(t *testing.T) {
	input := `version: 2.0

project "webapp" version "1.0.0":
  set registry to "docker.io"

task "build":
  info "Building webapp v{version}"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if program.Project == nil {
		t.Fatalf("program.Project is nil")
	}

	// Check project name and version
	if program.Project.Name != "webapp" {
		t.Errorf("project.Name not 'webapp'. got=%q", program.Project.Name)
	}

	if program.Project.Version != "1.0.0" {
		t.Errorf("project.Version not '1.0.0'. got=%q", program.Project.Version)
	}
}

func TestParser_ProjectWithLifecycleHooks(t *testing.T) {
	input := `version: 2.0

project "myapp":
  before any task:
    info "Starting task execution"
  
  after any task:
    info "Task completed"

task "deploy":
  info "Deploying application"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if program.Project == nil {
		t.Fatalf("program.Project is nil")
	}

	// Check lifecycle hooks
	if len(program.Project.Settings) != 2 {
		t.Fatalf("project should have 2 lifecycle hooks. got=%d", len(program.Project.Settings))
	}

	// Check before hook
	beforeHook, ok := program.Project.Settings[0].(*ast.LifecycleHook)
	if !ok {
		t.Fatalf("project.Settings[0] is not *ast.LifecycleHook. got=%T", program.Project.Settings[0])
	}

	if beforeHook.Type != "before" {
		t.Errorf("beforeHook.Type not 'before'. got=%q", beforeHook.Type)
	}

	if beforeHook.Scope != "any" {
		t.Errorf("beforeHook.Scope not 'any'. got=%q", beforeHook.Scope)
	}

	if len(beforeHook.Body) != 1 {
		t.Fatalf("beforeHook should have 1 statement. got=%d", len(beforeHook.Body))
	}

	// Check after hook
	afterHook, ok := program.Project.Settings[1].(*ast.LifecycleHook)
	if !ok {
		t.Fatalf("project.Settings[1] is not *ast.LifecycleHook. got=%T", program.Project.Settings[1])
	}

	if afterHook.Type != "after" {
		t.Errorf("afterHook.Type not 'after'. got=%q", afterHook.Type)
	}
}

func TestParser_ProjectWithDrunLifecycleHooks(t *testing.T) {
	input := `version: 2.0

project "myapp":
  on drun setup:
    info "🚀 Starting drun execution pipeline"
    info "📊 Tool version: v2.0"
  
  on drun teardown:
    info "🏁 Drun execution pipeline completed"
    info "📊 Total execution time: 5s"

task "deploy":
  info "Deploying application"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if program.Project == nil {
		t.Fatalf("program.Project is nil")
	}

	// Check lifecycle hooks
	if len(program.Project.Settings) != 2 {
		t.Fatalf("project should have 2 drun lifecycle hooks. got=%d", len(program.Project.Settings))
	}

	// Check setup hook
	setupHook, ok := program.Project.Settings[0].(*ast.LifecycleHook)
	if !ok {
		t.Fatalf("project.Settings[0] is not *ast.LifecycleHook. got=%T", program.Project.Settings[0])
	}

	if setupHook.Type != "setup" {
		t.Errorf("setupHook.Type not 'setup'. got=%q", setupHook.Type)
	}

	if setupHook.Scope != "drun" {
		t.Errorf("setupHook.Scope not 'drun'. got=%q", setupHook.Scope)
	}

	if len(setupHook.Body) != 2 {
		t.Fatalf("setupHook should have 2 statements. got=%d", len(setupHook.Body))
	}

	// Check teardown hook
	teardownHook, ok := program.Project.Settings[1].(*ast.LifecycleHook)
	if !ok {
		t.Fatalf("project.Settings[1] is not *ast.LifecycleHook. got=%T", program.Project.Settings[1])
	}

	if teardownHook.Type != "teardown" {
		t.Errorf("teardownHook.Type not 'teardown'. got=%q", teardownHook.Type)
	}

	if teardownHook.Scope != "drun" {
		t.Errorf("teardownHook.Scope not 'drun'. got=%q", teardownHook.Scope)
	}

	if len(teardownHook.Body) != 2 {
		t.Fatalf("teardownHook should have 2 statements. got=%d", len(teardownHook.Body))
	}
}

func TestParser_ProjectOptional(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello without project!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	// Project should be nil (optional)
	if program.Project != nil {
		t.Errorf("program.Project should be nil when not specified. got=%v", program.Project)
	}

	// Tasks should still work
	if len(program.Tasks) != 1 {
		t.Fatalf("program should have 1 task. got=%d", len(program.Tasks))
	}

	if program.Tasks[0].Name != "hello" {
		t.Errorf("task.Name not 'hello'. got=%q", program.Tasks[0].Name)
	}
}
