package ast

import (
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// VariableStatement represents variable operations (let, set, transform)
type VariableStatement struct {
	Token     lexer.Token
	Operation string
	Variable  string
	Value     Expression
	Function  string
	Arguments []string
}

func (vs *VariableStatement) statementNode() {}
func (vs *VariableStatement) String() string {
	var out strings.Builder

	switch vs.Operation {
	case "let":
		out.WriteString("let ")
		out.WriteString(vs.Variable)
		out.WriteString(" = ")
		if vs.Value != nil {
			out.WriteString(vs.Value.String())
		}
	case "set":
		out.WriteString("set ")
		out.WriteString(vs.Variable)
		out.WriteString(" to ")
		if vs.Value != nil {
			out.WriteString(vs.Value.String())
		}
	case "transform":
		out.WriteString("transform ")
		out.WriteString(vs.Variable)
		out.WriteString(" with ")
		out.WriteString(vs.Function)
		if len(vs.Arguments) > 0 {
			out.WriteString(" ")
			out.WriteString(strings.Join(vs.Arguments, " "))
		}
	default:
		out.WriteString(vs.Operation)
		out.WriteString(" ")
		out.WriteString(vs.Variable)
		if vs.Value != nil {
			out.WriteString(" ")
			out.WriteString(vs.Value.String())
		}
	}

	return out.String()
}
