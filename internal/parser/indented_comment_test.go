package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_CommentsBeforeIndentedBodiesAreIndentationNeutral(t *testing.T) {
	input := "version: 2.0\n\ntask \"test\":\n\t  \t# comment before the task body\n    if true:\n\t\t  # comment before the nested body\n        info \"works\""

	p := NewParser(lexer.NewLexer(input))
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	if len(program.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(program.Tasks))
	}
}

func TestParser_CommentAtOuterIndentEndsStructuredBodyAtNextCodeLine(t *testing.T) {
	input := `version: 2.0

task "test":
    replace in "example.env":
        "OLD" with "NEW"
            # deliberately irregular comment indentation
    run "true"`

	p := NewParser(lexer.NewLexer(input))
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	if len(program.Tasks) != 1 || len(program.Tasks[0].Body) != 2 {
		t.Fatalf("expected one task with two statements, got %#v", program.Tasks)
	}
}
