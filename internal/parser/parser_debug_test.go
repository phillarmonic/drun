package parser

import (
	"fmt"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_Debug(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello from drun v2! 👋"

task "hello world":
  step "Starting hello world example"
  info "Welcome to the semantic task runner!"
  success "Hello world completed successfully!"`

	lexer := lexer.NewLexer(input)
	parser := NewParser(lexer)

	fmt.Printf("Initial tokens: cur=%s, peek=%s\n", parser.curToken.Type, parser.peekToken.Type)

	program := parser.ParseProgram()

	fmt.Printf("Errors: %v\n", parser.Errors())

	if program != nil {
		fmt.Printf("Program: %s\n", program.String())
	}
}
