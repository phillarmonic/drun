package parser

import (
	"fmt"
	"testing"

	"github.com/phillarmonic/drun/internal/v2/lexer"
)

func TestParser_LifecycleDebugTokens(t *testing.T) {
	input := `version: 2.0

project "myapp":
  before any task:
    info "Starting task"`

	l := lexer.NewLexer(input)

	// Print all tokens to debug
	fmt.Println("=== LIFECYCLE TOKENS ===")
	for {
		tok := l.NextToken()
		fmt.Printf("Type: %s, Literal: %q, Line: %d, Column: %d\n",
			tok.Type, tok.Literal, tok.Line, tok.Column)
		if tok.Type == lexer.EOF {
			break
		}
	}
}
