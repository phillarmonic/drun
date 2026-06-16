package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_RequiresTools_ProjectLevel(t *testing.T) {
	input := `version: 2.0

project "Test":
  requires tools:
    gosec >= "2.27" <= "3.0"
    golangci-lint >= 2.12
    docker
`
	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if len(program.Project.Settings) != 1 {
		t.Fatalf("expected 1 project setting, got %d", len(program.Project.Settings))
	}

	stmt, ok := program.Project.Settings[0].(*ast.RequiresToolsStatement)
	if !ok {
		t.Fatalf("expected RequiresToolsStatement, got %T", program.Project.Settings[0])
	}

	if len(stmt.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(stmt.Tools))
	}

	// Check gosec
	tool1 := stmt.Tools[0]
	if tool1.Name != "gosec" {
		t.Errorf("tool1 name wrong. expected gosec, got %s", tool1.Name)
	}
	if len(tool1.Constraints) != 2 {
		t.Fatalf("tool1 constraints wrong. expected 2, got %d", len(tool1.Constraints))
	}
	if tool1.Constraints[0].Operator != ">=" || tool1.Constraints[0].Version != "2.27" {
		t.Errorf("tool1 constraint 0 wrong: %s %s", tool1.Constraints[0].Operator, tool1.Constraints[0].Version)
	}
	if tool1.Constraints[1].Operator != "<=" || tool1.Constraints[1].Version != "3.0" {
		t.Errorf("tool1 constraint 1 wrong: %s %s", tool1.Constraints[1].Operator, tool1.Constraints[1].Version)
	}

	// Check golangci-lint
	tool2 := stmt.Tools[1]
	if tool2.Name != "golangci-lint" {
		t.Errorf("tool2 name wrong. expected golangci-lint, got %s", tool2.Name)
	}
	if len(tool2.Constraints) != 1 {
		t.Fatalf("tool2 constraints wrong. expected 1, got %d", len(tool2.Constraints))
	}
	if tool2.Constraints[0].Operator != ">=" || tool2.Constraints[0].Version != "2.12" {
		t.Errorf("tool2 constraint 0 wrong: %s %s", tool2.Constraints[0].Operator, tool2.Constraints[0].Version)
	}

	// Check docker
	tool3 := stmt.Tools[2]
	if tool3.Name != "docker" {
		t.Errorf("tool3 name wrong. expected docker, got %s", tool3.Name)
	}
	if len(tool3.Constraints) != 0 {
		t.Errorf("tool3 constraints wrong. expected 0, got %d", len(tool3.Constraints))
	}
}

func TestParser_RequiresTools_TaskLevel(t *testing.T) {
	input := `version: 2.0

task "security":
  requires tools:
    gosec >= "2.27"
`
	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if len(program.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Body) != 1 {
		t.Fatalf("expected 1 statement in task body, got %d", len(task.Body))
	}

	stmt, ok := task.Body[0].(*ast.RequiresToolsStatement)
	if !ok {
		t.Fatalf("expected RequiresToolsStatement, got %T", task.Body[0])
	}

	if len(stmt.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(stmt.Tools))
	}

	tool1 := stmt.Tools[0]
	if tool1.Name != "gosec" {
		t.Errorf("tool name wrong. expected gosec, got %s", tool1.Name)
	}
	if len(tool1.Constraints) != 1 {
		t.Fatalf("tool constraints wrong. expected 1, got %d", len(tool1.Constraints))
	}
	if tool1.Constraints[0].Operator != ">=" || tool1.Constraints[0].Version != "2.27" {
		t.Errorf("tool constraint wrong: %s %s", tool1.Constraints[0].Operator, tool1.Constraints[0].Version)
	}
}

func TestParser_RequiresTools_ProjectLevel_CRLF(t *testing.T) {
	input := "# comment\r\n" +
		"version: 2.0\r\n" +
		"project \"POG\" version \"1.0\":\r\n" +
		"\trequires tools:\r\n" +
		"\t\tgo >= 1.26\r\n" +
		"\t\tgolangci-lint >= 2.12\r\n" +
		"\t\tgosec\r\n" +
		"\r\n" +
		"task \"default\":\r\n" +
		"\tinfo \"ok\"\r\n"

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if program.Project == nil {
		t.Fatal("expected project statement to be parsed")
	}

	if len(program.Project.Settings) != 1 {
		t.Fatalf("expected 1 project setting, got %d", len(program.Project.Settings))
	}

	stmt, ok := program.Project.Settings[0].(*ast.RequiresToolsStatement)
	if !ok {
		t.Fatalf("expected RequiresToolsStatement, got %T", program.Project.Settings[0])
	}

	if len(stmt.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(stmt.Tools))
	}
}
