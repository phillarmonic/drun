package main
import (
	"fmt"
	"os"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)
func main() {
	src, _ := os.ReadFile(".drun/spec.drun")
	l := lexer.NewLexer(string(src))
	for {
		tok := l.NextToken()
		if tok.Type == lexer.EOF { break }
		if tok.Line >= 12 && tok.Line <= 15 {
			fmt.Printf("Line %d: Type=%v Literal=%q\n", tok.Line, tok.Type, tok.Literal)
		}
	}
}
