package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_SimpleDependency(t *testing.T) {
	input := `version: 2.0

task "deploy":
  depends on build
  
  info "Deploying application"`

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
	if len(task.Dependencies) != 1 {
		t.Fatalf("task should have 1 dependency group. got=%d", len(task.Dependencies))
	}

	dep := task.Dependencies[0]
	if len(dep.Dependencies) != 1 {
		t.Fatalf("dependency group should have 1 dependency. got=%d", len(dep.Dependencies))
	}

	if dep.Dependencies[0].Name != "build" {
		t.Errorf("dependency name not 'build'. got=%q", dep.Dependencies[0].Name)
	}

	if dep.Dependencies[0].Parallel {
		t.Errorf("dependency should not be parallel by default")
	}
}

func TestParser_SequentialDependencies(t *testing.T) {
	input := `version: 2.0

task "deploy":
  depends on build and test and package
  
  info "Deploying application"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Dependencies) != 1 {
		t.Fatalf("task should have 1 dependency group. got=%d", len(task.Dependencies))
	}

	dep := task.Dependencies[0]
	if len(dep.Dependencies) != 3 {
		t.Fatalf("dependency group should have 3 dependencies. got=%d", len(dep.Dependencies))
	}

	if !dep.Sequential {
		t.Errorf("dependency group should be sequential (using 'and')")
	}

	expectedNames := []string{"build", "test", "package"}
	for i, expected := range expectedNames {
		if dep.Dependencies[i].Name != expected {
			t.Errorf("dependency %d name not '%s'. got=%q", i, expected, dep.Dependencies[i].Name)
		}
	}
}

func TestParser_ParallelDependencies(t *testing.T) {
	input := `version: 2.0

task "deploy":
  depends on lint, test, security_scan
  
  info "Deploying application"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Dependencies) != 1 {
		t.Fatalf("task should have 1 dependency group. got=%d", len(task.Dependencies))
	}

	dep := task.Dependencies[0]
	if len(dep.Dependencies) != 3 {
		t.Fatalf("dependency group should have 3 dependencies. got=%d", len(dep.Dependencies))
	}

	if dep.Sequential {
		t.Errorf("dependency group should be parallel (using ',')")
	}

	expectedNames := []string{"lint", "test", "security_scan"}
	for i, expected := range expectedNames {
		if dep.Dependencies[i].Name != expected {
			t.Errorf("dependency %d name not '%s'. got=%q", i, expected, dep.Dependencies[i].Name)
		}
	}
}

func TestParser_MultipleDependencyGroups(t *testing.T) {
	input := `version: 2.0

task "deploy":
  depends on build and test
  depends on lint, security_scan
  
  info "Deploying application"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	if len(task.Dependencies) != 2 {
		t.Fatalf("task should have 2 dependency groups. got=%d", len(task.Dependencies))
	}

	// First group: sequential
	dep1 := task.Dependencies[0]
	if !dep1.Sequential {
		t.Errorf("first dependency group should be sequential")
	}
	if len(dep1.Dependencies) != 2 {
		t.Fatalf("first dependency group should have 2 dependencies. got=%d", len(dep1.Dependencies))
	}

	// Second group: parallel
	dep2 := task.Dependencies[1]
	if dep2.Sequential {
		t.Errorf("second dependency group should be parallel")
	}
	if len(dep2.Dependencies) != 2 {
		t.Fatalf("second dependency group should have 2 dependencies. got=%d", len(dep2.Dependencies))
	}
}

func TestParser_DependencyWithParameters(t *testing.T) {
	input := `version: 2.0

task "deploy":
  depends on build
  requires environment from ["dev", "staging", "production"]
  
  info "Deploying to {environment}"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]

	// Check dependencies
	if len(task.Dependencies) != 1 {
		t.Fatalf("task should have 1 dependency group. got=%d", len(task.Dependencies))
	}

	// Check parameters
	if len(task.Parameters) != 1 {
		t.Fatalf("task should have 1 parameter. got=%d", len(task.Parameters))
	}

	if task.Parameters[0].Name != "environment" {
		t.Errorf("parameter name not 'environment'. got=%q", task.Parameters[0].Name)
	}
}
