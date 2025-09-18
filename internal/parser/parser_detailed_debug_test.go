package parser

import (
	"fmt"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_DetailedDebug(t *testing.T) {
	input := `version: 2.0

task "hello":
  info "Hello from drun v2! 👋"

task "hello world":
  step "Starting hello world example"
  info "Welcome to the semantic task runner!"
  success "Hello world completed successfully!"`

	lexer := lexer.NewLexer(input)

	// Let's manually step through what the parser should do
	fmt.Println("=== Manual Token Processing ===")

	tokens := lexer.AllTokens()
	for i, tok := range tokens {
		fmt.Printf("%d: %s\n", i, tok)
	}

	fmt.Println("\n=== Expected Parser Flow ===")
	fmt.Println("1. Parse VERSION : NUMBER")
	fmt.Println("2. Parse TASK STRING COLON")
	fmt.Println("3. Parse INDENT")
	fmt.Println("4. Parse INFO STRING")
	fmt.Println("5. Expect DEDENT")
	fmt.Println("6. Parse TASK STRING COLON")
	fmt.Println("7. Parse INDENT")
	fmt.Println("8. Parse STEP STRING")
	fmt.Println("9. Expect DEDENT")
	fmt.Println("10. EOF")
}
