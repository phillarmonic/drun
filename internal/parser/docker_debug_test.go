package parser

import (
	"fmt"
	"testing"

	lexer2 "github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_DockerRunDebugTokens(t *testing.T) {
	input := `version: 2.0

task "run":
  docker run container "webapp" from "myapp:latest"`

	l := lexer2.NewLexer(input)

	// Print all tokens to debug
	fmt.Println("=== DOCKER RUN TOKENS ===")
	for {
		tok := l.NextToken()
		fmt.Printf("Type: %s, Literal: %q, Line: %d, Column: %d\n",
			tok.Type, tok.Literal, tok.Line, tok.Column)
		if tok.Type == lexer2.EOF {
			break
		}
	}
}
