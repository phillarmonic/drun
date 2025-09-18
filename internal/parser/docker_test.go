package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_DockerBuildImage(t *testing.T) {
	input := `version: 2.0

task "build":
  docker build image "myapp:latest" from "Dockerfile"
  
  success "Image built successfully!"`

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

	// Check Docker statement
	dockerStmt, ok := task.Body[0].(*ast.DockerStatement)
	if !ok {
		t.Fatalf("first statement should be DockerStatement. got=%T", task.Body[0])
	}

	if dockerStmt.Operation != "build" {
		t.Errorf("docker operation not 'build'. got=%q", dockerStmt.Operation)
	}

	if dockerStmt.Resource != "image" {
		t.Errorf("docker resource not 'image'. got=%q", dockerStmt.Resource)
	}

	if dockerStmt.Name != "myapp:latest" {
		t.Errorf("docker image name not 'myapp:latest'. got=%q", dockerStmt.Name)
	}

	if dockerStmt.Options["from"] != "Dockerfile" {
		t.Errorf("docker 'from' option not 'Dockerfile'. got=%q", dockerStmt.Options["from"])
	}
}

func TestParser_DockerPushImage(t *testing.T) {
	input := `version: 2.0

task "push":
  docker push image "myapp:latest" to "registry.example.com"
  
  success "Image pushed successfully!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	dockerStmt, ok := task.Body[0].(*ast.DockerStatement)
	if !ok {
		t.Fatalf("first statement should be DockerStatement. got=%T", task.Body[0])
	}

	if dockerStmt.Operation != "push" {
		t.Errorf("docker operation not 'push'. got=%q", dockerStmt.Operation)
	}

	if dockerStmt.Resource != "image" {
		t.Errorf("docker resource not 'image'. got=%q", dockerStmt.Resource)
	}

	if dockerStmt.Name != "myapp:latest" {
		t.Errorf("docker image name not 'myapp:latest'. got=%q", dockerStmt.Name)
	}

	if dockerStmt.Options["to"] != "registry.example.com" {
		t.Errorf("docker 'to' option not 'registry.example.com'. got=%q", dockerStmt.Options["to"])
	}
}

func TestParser_DockerRunContainer(t *testing.T) {
	input := `version: 2.0

task "run":
  docker run container "webapp" from "myapp:latest"
  
  success "Container started successfully!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	dockerStmt, ok := task.Body[0].(*ast.DockerStatement)
	if !ok {
		t.Fatalf("first statement should be DockerStatement. got=%T", task.Body[0])
	}

	if dockerStmt.Operation != "run" {
		t.Errorf("docker operation not 'run'. got=%q", dockerStmt.Operation)
	}

	if dockerStmt.Resource != "container" {
		t.Errorf("docker resource not 'container'. got=%q", dockerStmt.Resource)
	}

	if dockerStmt.Name != "webapp" {
		t.Errorf("docker container name not 'webapp'. got=%q", dockerStmt.Name)
	}

	if dockerStmt.Options["from"] != "myapp:latest" {
		t.Errorf("docker 'from' option not 'myapp:latest'. got=%q", dockerStmt.Options["from"])
	}
}

func TestParser_DockerCompose(t *testing.T) {
	input := `version: 2.0

task "start_services":
  docker compose up
  
  success "Services started successfully!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	dockerStmt, ok := task.Body[0].(*ast.DockerStatement)
	if !ok {
		t.Fatalf("first statement should be DockerStatement. got=%T", task.Body[0])
	}

	if dockerStmt.Operation != "compose" {
		t.Errorf("docker operation not 'compose'. got=%q", dockerStmt.Operation)
	}

	if dockerStmt.Resource != "compose" {
		t.Errorf("docker resource not 'compose'. got=%q", dockerStmt.Resource)
	}

	if dockerStmt.Options["command"] != "up" {
		t.Errorf("docker compose command not 'up'. got=%q", dockerStmt.Options["command"])
	}
}

func TestParser_DockerTagImage(t *testing.T) {
	input := `version: 2.0

task "tag":
  docker tag image "myapp:latest" as "myapp:v1.0.0"
  
  success "Image tagged successfully!"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	task := program.Tasks[0]
	dockerStmt, ok := task.Body[0].(*ast.DockerStatement)
	if !ok {
		t.Fatalf("first statement should be DockerStatement. got=%T", task.Body[0])
	}

	if dockerStmt.Operation != "tag" {
		t.Errorf("docker operation not 'tag'. got=%q", dockerStmt.Operation)
	}

	if dockerStmt.Resource != "image" {
		t.Errorf("docker resource not 'image'. got=%q", dockerStmt.Resource)
	}

	if dockerStmt.Name != "myapp:latest" {
		t.Errorf("docker image name not 'myapp:latest'. got=%q", dockerStmt.Name)
	}

	if dockerStmt.Options["as"] != "myapp:v1.0.0" {
		t.Errorf("docker 'as' option not 'myapp:v1.0.0'. got=%q", dockerStmt.Options["as"])
	}
}

func TestParser_DockerMultipleOperations(t *testing.T) {
	input := `version: 2.0

task "deploy":
  docker build image "myapp:latest" from "Dockerfile"
  docker push image "myapp:latest" to "registry.example.com"
  docker run container "webapp" from "myapp:latest"
  
  success "Deployment completed!"`

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

	// Check all three Docker statements
	operations := []string{"build", "push", "run"}
	for i, expectedOp := range operations {
		dockerStmt, ok := task.Body[i].(*ast.DockerStatement)
		if !ok {
			t.Fatalf("statement %d should be DockerStatement. got=%T", i, task.Body[i])
		}

		if dockerStmt.Operation != expectedOp {
			t.Errorf("docker operation %d not '%s'. got=%q", i, expectedOp, dockerStmt.Operation)
		}
	}
}
