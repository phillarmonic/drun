package lexer

import (
	"fmt"
	"testing"
)

func TestLexer_GitPushTokens(t *testing.T) {
	input := `git push to remote "origin" branch "main"`

	l := NewLexer(input)

	// Print all tokens to debug
	fmt.Println("=== GIT PUSH TOKENS ===")
	for {
		tok := l.NextToken()
		fmt.Printf("Type: %s, Literal: %q, Line: %d, Column: %d\n",
			tok.Type, tok.Literal, tok.Line, tok.Column)
		if tok.Type == EOF {
			break
		}
	}
}
