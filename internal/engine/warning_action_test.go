package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

func TestEngine_ExecutesWarningAction(t *testing.T) {
	input := `version: 2.0

task "test":
  warning "Mind the release"`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	var output bytes.Buffer
	if err := NewEngine(&output).Execute(program, "test"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, "⚠️  Mind the release") {
		t.Errorf("output = %q, want warning message", got)
	}
}
