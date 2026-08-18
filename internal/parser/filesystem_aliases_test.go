package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_DirectoryActionNounsAreInterchangeable(t *testing.T) {
	input := `version: 2.0

task "filesystem aliases":
  create folder "one"
  create directory "two"
  create dir "three"
  delete folder "one"
  delete directory "two"
  delete dir "three"
  check if folder "one" exists
  check if directory "two" exists
  check if dir "three" exists
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Tasks) != 1 || len(program.Tasks[0].Body) != 9 {
		t.Fatalf("expected one task with nine statements, got %#v", program.Tasks)
	}
	for i, statement := range program.Tasks[0].Body {
		file, ok := statement.(*ast.FileStatement)
		if !ok {
			t.Fatalf("statement %d is %T, want *ast.FileStatement", i, statement)
		}
		if !file.IsDir {
			t.Errorf("statement %d should target a directory", i)
		}
	}
}

func TestParser_DoesNotExistCondition(t *testing.T) {
	input := `version: 2.0

task "missing aliases":
  if folder "cache" does not exist:
    info "missing"
`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	conditional, ok := program.Tasks[0].Body[0].(*ast.ConditionalStatement)
	if !ok {
		t.Fatalf("first statement is %T, want *ast.ConditionalStatement", program.Tasks[0].Body[0])
	}
	if conditional.Condition != `folder cache does not exist` {
		t.Fatalf("condition = %q", conditional.Condition)
	}
}
