package statements

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/lexer"
)

// ShellStatement represents shell command execution
type ShellStatement struct {
	Token        lexer.Token
	Action       string
	Command      string
	Commands     []string
	CaptureVar   string
	StreamOutput bool
	IsMultiline  bool
}

func (ss *ShellStatement) statementNode() {}
func (ss *ShellStatement) String() string {
	if ss.IsMultiline {
		var out string
		if ss.CaptureVar != "" {
			out = fmt.Sprintf("%s as %s:", ss.Action, ss.CaptureVar)
		} else {
			out = fmt.Sprintf("%s:", ss.Action)
		}
		for _, cmd := range ss.Commands {
			out += fmt.Sprintf("\n  %s", cmd)
		}
		return out
	}

	if ss.CaptureVar != "" {
		return fmt.Sprintf("%s \"%s\" as %s", ss.Action, ss.Command, ss.CaptureVar)
	}
	return fmt.Sprintf("%s \"%s\"", ss.Action, ss.Command)
}
