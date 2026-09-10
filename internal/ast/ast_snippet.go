package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// SnippetStatement represents a reusable code block
type SnippetStatement struct {
	Name        string
	Annotations []Annotation
	Body        []Statement
	Token       lexer.Token
}

func (ss *SnippetStatement) statementNode()      {}
func (ss *SnippetStatement) projectSettingNode() {}
func (ss *SnippetStatement) String() string {
	var out string
	var outSb21 strings.Builder
	for _, annotation := range ss.Annotations {
		outSb21.WriteString(annotation.String() + "\n")
	}
	out += outSb21.String()
	out += fmt.Sprintf("snippet \"%s\":", ss.Name)
	var outSb25 strings.Builder
	for _, stmt := range ss.Body {
		outSb25.WriteString("\n  " + stmt.String())
	}
	out += outSb25.String()
	return out
}

// UseSnippetStatement represents using a snippet
type UseSnippetStatement struct {
	SnippetName string
	Token       lexer.Token
}

func (uss *UseSnippetStatement) statementNode() {}
func (uss *UseSnippetStatement) String() string {
	return fmt.Sprintf("use snippet \"%s\"", uss.SnippetName)
}
