package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_FileValueStatements(t *testing.T) {
	source := `version: 2.0
project "test" version "1.0.0":
task "versions":
  get property "pluginVersion" from "gradle.properties" as $plugin_version
  check json "/version" in "package.json" equals "{$globals.version}"
  check yaml "chart.version" in "Chart.yaml" differs from "{$previous}"
  update toml "package.version" in "Cargo.toml" to "{$next}" or add as string
  update match "(?m)^VERSION=(?P<value>.+)$" in "VERSION" to "{$next}" or fail
`
	l := lexer.NewLexer(source)
	p := NewParser(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)
	if len(program.Tasks) != 1 || len(program.Tasks[0].Body) != 5 {
		t.Fatalf("expected five file value statements, got %#v", program.Tasks)
	}
	first, ok := program.Tasks[0].Body[0].(*ast.FileValueStatement)
	if !ok || first.Operation != "get" || first.Format != "property" || first.CaptureVar != "plugin_version" {
		t.Fatalf("unexpected get statement: %#v", program.Tasks[0].Body[0])
	}
	last := program.Tasks[0].Body[4].(*ast.FileValueStatement)
	if last.Format != "match" || last.MissingPolicy != "fail" {
		t.Fatalf("unexpected regex update: %#v", last)
	}
}

func TestParser_FileValueRejectsRegexAdd(t *testing.T) {
	source := `version: 2.0
task "bad":
  update match "(?P<value>.+)" in "VERSION" to "2" or add as string
`
	l := lexer.NewLexer(source)
	p := NewParser(l)
	_ = p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatal("expected regex add error")
	}
}

func TestParser_ProjectVersionCheckAndUpdate(t *testing.T) {
	source := `version: 2.0
project "test" version "1.0.0":
task "versions":
  check project version equals "{$expected}"
  check project version differs from "0.9.0"
  update project version to "{$next}"
`
	p := NewParser(lexer.NewLexer(source))
	program := p.ParseProgram()
	checkParserErrors(t, p)
	if len(program.Tasks) != 1 || len(program.Tasks[0].Body) != 3 {
		t.Fatalf("unexpected project version statements: %#v", program.Tasks)
	}
	check := program.Tasks[0].Body[0].(*ast.FileValueStatement)
	if check.Format != "drun" || check.Selector != "project.version" || check.Comparison != "equals" || check.Expected != "{$expected}" {
		t.Fatalf("unexpected project version check: %#v", check)
	}
	update := program.Tasks[0].Body[2].(*ast.FileValueStatement)
	if update.Format != "drun" || update.Selector != "project.version" || update.Value != "{$next}" || update.MissingPolicy != "fail" {
		t.Fatalf("unexpected project version update: %#v", update)
	}
	if got := update.String(); got != `update project version to "{$next}"` {
		t.Fatalf("String() = %q", got)
	}
}
