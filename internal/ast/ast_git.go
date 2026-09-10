package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// GitStatement represents Git operations
type GitStatement struct {
	Options   map[string]string
	Operation string
	Resource  string
	Name      string
	Token     lexer.Token
}

func (gs *GitStatement) statementNode() {}
func (gs *GitStatement) String() string {
	out := "git " + gs.Operation

	if gs.Resource != "" {
		out += " " + gs.Resource
	}

	if gs.Name != "" {
		out += fmt.Sprintf(" \"%s\"", gs.Name)
	}

	var outSb30 strings.Builder
	for key, value := range gs.Options {
		fmt.Fprintf(&outSb30, " %s \"%s\"", key, value)
	}
	out += outSb30.String()

	return out
}
