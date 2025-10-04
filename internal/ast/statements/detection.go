package statements

import (
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// DetectionStatement represents smart detection operations
type DetectionStatement struct {
	Token        lexer.Token
	Type         string
	Target       string
	Alternatives []string
	Condition    string
	Value        string
	CaptureVar   string
	Body         []ast.Statement
	ElseBody     []ast.Statement
}

func (ds *DetectionStatement) statementNode() {}
func (ds *DetectionStatement) String() string {
	var out strings.Builder

	switch ds.Type {
	case "detect":
		out.WriteString("detect " + ds.Target)
		if ds.Condition != "" {
			out.WriteString(" " + ds.Condition)
		}
		if ds.Value != "" {
			out.WriteString(" " + ds.Value)
		}
	case "detect_available":
		out.WriteString("detect available " + ds.Target)
		for _, alt := range ds.Alternatives {
			out.WriteString(" or \"" + alt + "\"")
		}
		if ds.CaptureVar != "" {
			out.WriteString(" as " + ds.CaptureVar)
		}
	case "if_available":
		if len(ds.Alternatives) > 0 {
			tools := append([]string{ds.Target}, ds.Alternatives...)
			out.WriteString("if " + strings.Join(tools, ",") + " is available")
		} else {
			out.WriteString("if " + ds.Target + " is available")
		}
	case "when_environment":
		out.WriteString("when in " + ds.Target + " environment")
	case "if_version":
		out.WriteString("if " + ds.Target + " version " + ds.Condition + " " + ds.Value)
	default:
		out.WriteString(ds.Type + " " + ds.Target)
	}

	if len(ds.Body) > 0 {
		out.WriteString(":")
		for _, stmt := range ds.Body {
			out.WriteString("\n  " + stmt.String())
		}
	}

	if len(ds.ElseBody) > 0 {
		out.WriteString("\nelse:")
		for _, stmt := range ds.ElseBody {
			out.WriteString("\n  " + stmt.String())
		}
	}

	return out.String()
}
