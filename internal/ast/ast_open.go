package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// OpenStatement represents opening a URL or file in the OS default handler
// (open url "https://example.com", open url "docs/index.html")
type OpenStatement struct {
	Noun  string // Always "url" today; reserved for future nouns such as "file"
	URL   string // Raw target; may contain {variable} interpolation
	Token lexer.Token
}

func (os *OpenStatement) statementNode() {}
func (os *OpenStatement) String() string {
	return fmt.Sprintf("open %s %q", os.Noun, os.URL)
}
