package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/lexer"
)

// ActionStatement represents an action call (info, step, success, etc.)
type ActionStatement struct {
	Token           lexer.Token
	Action          string
	Message         string
	LineBreakBefore bool
	LineBreakAfter  bool
}

func (as *ActionStatement) statementNode() {}
func (as *ActionStatement) String() string {
	suffix := ""
	if as.LineBreakBefore {
		suffix += " add line break before"
	}
	if as.LineBreakAfter {
		suffix += " add line break after"
	}
	return fmt.Sprintf("%s \"%s\"%s", as.Action, as.Message, suffix)
}
