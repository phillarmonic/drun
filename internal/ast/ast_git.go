package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/lexer"
)

// GitStatement represents Git operations
type GitStatement struct {
	Token     lexer.Token
	Operation string
	Resource  string
	Name      string
	Options   map[string]string
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

	for key, value := range gs.Options {
		out += fmt.Sprintf(" %s \"%s\"", key, value)
	}

	return out
}
