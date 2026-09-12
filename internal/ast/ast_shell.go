package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// ShellStatement represents shell command execution
type ShellStatement struct {
	Action               string
	Command              string
	CaptureVar           string
	ServiceName          string
	Commands             []string
	Token                lexer.Token
	Attached             bool
	StreamOutput         bool
	IsMultiline          bool
	ServiceScoped        bool
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
				prefix += " in service " + ss.ServiceName
			}
		}
		out = prefix + ":"
		if ss.CaptureVar != "" {
			out = fmt.Sprintf("%s as %s:", prefix, ss.CaptureVar)
		}
		var outSb40 strings.Builder
		for _, cmd := range ss.Commands {
			outSb40.WriteString("\n  " + cmd)
		}
		out += outSb40.String()
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
