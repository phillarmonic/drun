package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/lexer"
)

// SnippetStatement represents a reusable code block
type SnippetStatement struct {
	Token lexer.Token
	Name  string
	Body  []Statement
}

func (ss *SnippetStatement) statementNode()      {}
func (ss *SnippetStatement) projectSettingNode() {}
func (ss *SnippetStatement) String() string {
	out := fmt.Sprintf("snippet \"%s\":", ss.Name)
	for _, stmt := range ss.Body {
		out += "\n  " + stmt.String()
	}
	return out
}

// UseSnippetStatement represents using a snippet
type UseSnippetStatement struct {
	Token       lexer.Token
	SnippetName string
}

func (uss *UseSnippetStatement) statementNode() {}
func (uss *UseSnippetStatement) String() string {
	return fmt.Sprintf("use snippet \"%s\"", uss.SnippetName)
}
