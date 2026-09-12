package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// ConfirmStatement represents an interactive y/n confirmation gate
// (confirm "Deploy to production?"). Without a result variable the bare form
// stops the task gracefully when declined; with `as $var` the answer is stored
// using drun's string-boolean convention ("true"/"false").
type ConfirmStatement struct {
	Question     string // Prompt text; may contain {variable} interpolation
	DefaultValue string // Raw default value (e.g. "no"); empty unless HasDefault is set
	ResultVar    string // Variable name without the $ prefix for `as $var`; empty for a bare gate
	Token        lexer.Token
	HasDefault   bool // Whether a `defaults to` value was declared
}

func (cs *ConfirmStatement) statementNode() {}
func (cs *ConfirmStatement) String() string {
	out := fmt.Sprintf("confirm %q", cs.Question)
	if cs.HasDefault {
		out += fmt.Sprintf(" defaults to %q", cs.DefaultValue)
	}
	if cs.ResultVar != "" {
		out += " as $" + cs.ResultVar
	}
	return out
}

// PromptStatement represents a free-form interactive text prompt
// (prompt "Which environment?" as $environment). The typed answer is stored in
// the result variable; `defaults to` supplies the value used when the user
// presses enter with no input.
type PromptStatement struct {
	Question     string // Prompt text; may contain {variable} interpolation
	DefaultValue string // Raw default value; empty unless HasDefault is set
	ResultVar    string // Variable name without the $ prefix receiving the answer
	Token        lexer.Token
	HasDefault   bool // Whether a `defaults to` value was declared
}

func (ps *PromptStatement) statementNode() {}
func (ps *PromptStatement) String() string {
	out := fmt.Sprintf("prompt %q", ps.Question)
	if ps.HasDefault {
		out += fmt.Sprintf(" defaults to %q", ps.DefaultValue)
	}
	if ps.ResultVar != "" {
		out += " as $" + ps.ResultVar
	}
	return out
}
