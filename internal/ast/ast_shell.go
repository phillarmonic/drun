package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// ShellStatement represents shell command execution
type ShellStatement struct {
	Token                lexer.Token
	Action               string
	Command              string
	Commands             []string
	CaptureVar           string
	Attached             bool
	StreamOutput         bool
	IsMultiline          bool
	ServiceScoped        bool
	ServiceName          string
	ServiceNameIsLiteral bool
}

func (ss *ShellStatement) statementNode() {}
func (ss *ShellStatement) String() string {
	if ss.IsMultiline {
		var out string
		prefix := ss.Action
		if ss.ServiceScoped {
			if ss.ServiceNameIsLiteral {
				prefix += fmt.Sprintf(" in service \"%s\"", ss.ServiceName)
			} else {
				prefix += fmt.Sprintf(" in service %s", ss.ServiceName)
			}
		}
		out = prefix + ":"
		if ss.CaptureVar != "" {
			out = fmt.Sprintf("%s as %s:", prefix, ss.CaptureVar)
		}
		for _, cmd := range ss.Commands {
			out += fmt.Sprintf("\n  %s", cmd)
		}
		return out
	}

	var prefix string
	if ss.ServiceScoped {
		if ss.ServiceNameIsLiteral {
			prefix = fmt.Sprintf("%s in service \"%s\"", ss.Action, ss.ServiceName)
		} else {
			prefix = fmt.Sprintf("%s in service %s", ss.Action, ss.ServiceName)
		}
	} else {
		prefix = ss.Action
	}

	if ss.CaptureVar != "" {
		return fmt.Sprintf("%s \"%s\" as %s", prefix, ss.Command, ss.CaptureVar)
	}
	if ss.Attached {
		return fmt.Sprintf("%s \"%s\" attached", prefix, ss.Command)
	}
	return fmt.Sprintf("%s \"%s\"", prefix, ss.Command)
}
