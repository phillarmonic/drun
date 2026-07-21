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
    gosec >= "2.27" <= "3.0" provision
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
	if !tool1.AutoProvision {
		t.Errorf("tool1 AutoProvision wrong. expected true")
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
	if tool2.AutoProvision {
		t.Errorf("tool2 AutoProvision wrong. expected false")
	}

	// Check docker
	tool3 := stmt.Tools[2]
	if tool3.Name != "docker" {
		t.Errorf("tool3 name wrong. expected docker, got %s", tool3.Name)
	}
	if len(tool3.Constraints) != 0 {
		t.Errorf("tool3 constraints wrong. expected 0, got %d", len(tool3.Constraints))
	}
	if tool3.AutoProvision {
		t.Errorf("tool3 AutoProvision wrong. expected false")
	}
}

func TestParser_RequiresTools_TaskLevel(t *testing.T) {
	input := `version: 2.0

task "security":
  requires tools:
    gosec >= "2.27" provision
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
	if !tool1.AutoProvision {
		t.Errorf("tool AutoProvision wrong. expected true")
	}
}

func TestParser_RequiresTools_FromTasksClause(t *testing.T) {
	input := `version: 2.0

task "security":
  requires tools:
    from tasks:
      build
      "integration test"
      lint-check
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

	if len(stmt.Tools) != 0 {
		t.Fatalf("expected no direct tools, got %d", len(stmt.Tools))
	}
	if len(stmt.TaskSources) != 1 {
		t.Fatalf("expected 1 task source clause, got %d", len(stmt.TaskSources))
	}

	expectedTasks := []string{"build", "integration test", "lint-check"}
	if len(stmt.TaskSources[0].Tasks) != len(expectedTasks) {
		t.Fatalf("expected %d task sources, got %d", len(expectedTasks), len(stmt.TaskSources[0].Tasks))
	}
	for i, expected := range expectedTasks {
		if got := stmt.TaskSources[0].Tasks[i]; got != expected {
			t.Fatalf("task source %d wrong: expected %q, got %q", i, expected, got)
		}
	}
}

func TestParser_RequiresTools_DirectToolsAndFromTasksClause(t *testing.T) {
	input := `version: 2.0

task "security":
  requires tools:
    gosec >= "2.27" provision
    from tasks:
      lint
      test-unit
    docker
`
	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	task := program.Tasks[0]
	stmt, ok := task.Body[0].(*ast.RequiresToolsStatement)
	if !ok {
		t.Fatalf("expected RequiresToolsStatement, got %T", task.Body[0])
	}

	if len(stmt.Tools) != 2 {
		t.Fatalf("expected 2 direct tools, got %d", len(stmt.Tools))
	}
	if stmt.Tools[0].Name != "gosec" || !stmt.Tools[0].AutoProvision {
		t.Fatalf("unexpected first tool: %#v", stmt.Tools[0])
	}
	if stmt.Tools[1].Name != "docker" {
		t.Fatalf("unexpected second tool: %#v", stmt.Tools[1])
	}

	if len(stmt.TaskSources) != 1 {
		t.Fatalf("expected 1 task source clause, got %d", len(stmt.TaskSources))
	}
	if got := stmt.TaskSources[0].Tasks; len(got) != 2 || got[0] != "lint" || got[1] != "test-unit" {
		t.Fatalf("unexpected task sources: %#v", got)
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

func TestParser_ProvisioningSources_ProjectLevel(t *testing.T) {
	input := `version: 2.0

project "Test":
  provisioning sources:
    "./.drun/provisionings.yaml"
    "https://example.com/drun/provisionings.yaml"
    "github:acme/devx-catalog/catalog/provisionings.yaml@main"
`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)
	program := parser.ParseProgram()

	checkParserErrors(t, parser)

	if len(program.Project.Settings) != 1 {
		t.Fatalf("expected 1 project setting, got %d", len(program.Project.Settings))
	}

	stmt, ok := program.Project.Settings[0].(*ast.ProvisioningSourcesStatement)
	if !ok {
		t.Fatalf("expected ProvisioningSourcesStatement, got %T", program.Project.Settings[0])
	}

	if len(stmt.Sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(stmt.Sources))
	}

	if stmt.Sources[0] != "./.drun/provisionings.yaml" {
		t.Fatalf("unexpected first source: %q", stmt.Sources[0])
	}

	expected := "provisioning sources:\n  \"./.drun/provisionings.yaml\"\n  \"https://example.com/drun/provisionings.yaml\"\n  \"github:acme/devx-catalog/catalog/provisionings.yaml@main\""
	if got := stmt.String(); got != expected {
		t.Fatalf("unexpected String() output:\nexpected:\n%s\n\ngot:\n%s", expected, got)
	}
}
