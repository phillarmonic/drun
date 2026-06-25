package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_TaskAnnotations(t *testing.T) {
	input := `version: 2.0

@platform("linux", "mac")
task "shell":
  info "ok"
`

	lex := lexer.NewLexer(input)
	p := NewParser(lex)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(program.Tasks))
	}
	if len(program.Tasks[0].Annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(program.Tasks[0].Annotations))
	}
	if got := program.Tasks[0].Annotations[0].Name; got != "platform" {
		t.Fatalf("expected annotation name platform, got %q", got)
	}
	if got := len(program.Tasks[0].Annotations[0].Args); got != 2 {
		t.Fatalf("expected 2 annotation args, got %d", got)
	}
}

func TestParser_SnippetAnnotations(t *testing.T) {
	input := `version: 2.0

project "demo":
  @platform("mac")
  snippet "setup":
    info "hi"
`

	p := NewParser(lexer.NewLexer(input))
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if program.Project == nil {
		t.Fatal("expected project")
	}
	if len(program.Project.Settings) != 1 {
		t.Fatalf("expected 1 project setting, got %d", len(program.Project.Settings))
	}
}
