package lexer

import (
	"fmt"
	"testing"
)

func TestLexer_StringWithKeywords(t *testing.T) {
	input := `task "build":`

	l := NewLexer(input)

	// Print all tokens to debug
	fmt.Println("=== STRING WITH KEYWORDS TOKENS ===")
	for {
		tok := l.NextToken()
		fmt.Printf("Type: %s, Literal: %q, Line: %d, Column: %d\n",
			tok.Type, tok.Literal, tok.Line, tok.Column)
		if tok.Type == EOF {
			break
		}
	}
}
