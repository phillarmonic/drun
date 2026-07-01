package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// GitValidateStatement represents inline git validation within a task.
// It validates the current git state against the project's git policy.
type GitValidateStatement struct {
	Token  lexer.Token
	Target string // "branch_name", "commit_message", "signed_commits", "all"
	Value  string // optional explicit value to validate (e.g. commit message text)
}

func (gv *GitValidateStatement) statementNode() {}
func (gv *GitValidateStatement) String() string {
	out := "git validate " + gv.Target
	if gv.Value != "" {
		out += fmt.Sprintf(" \"%s\"", gv.Value)
	}
	return out
}
