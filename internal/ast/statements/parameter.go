package statements

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// ParameterStatement represents parameter declarations (requires, given, accepts)
type ParameterStatement struct {
	Token        lexer.Token
	Type         string
	Name         string
	DefaultValue string
	HasDefault   bool
	Constraints  []string
	DataType     string
	Required     bool
	Variadic     bool
	MinValue     *float64
	MaxValue     *float64
	Pattern      string
	PatternMacro string
	EmailFormat  bool
}

func (ps *ParameterStatement) statementNode() {}
func (ps *ParameterStatement) String() string {
	var out strings.Builder
	out.WriteString(ps.Type)
	out.WriteString(" ")
	out.WriteString(ps.Name)

	if ps.DefaultValue != "" {
		out.WriteString(" defaults to ")
		out.WriteString(ps.DefaultValue)
	}

	if len(ps.Constraints) > 0 {
		out.WriteString(" from [")
		out.WriteString(strings.Join(ps.Constraints, ", "))
		out.WriteString("]")
	}

	if ps.DataType != "" && ps.DataType != "string" {
		out.WriteString(" as ")
		out.WriteString(ps.DataType)
	}

	return out.String()
}

// ProjectParameterStatement represents a shared parameter defined at project level
type ProjectParameterStatement struct {
	Token        lexer.Token
	Name         string
	DefaultValue string
	HasDefault   bool
	Constraints  []string
	DataType     string
	MinValue     *float64
	MaxValue     *float64
	Pattern      string
	PatternMacro string
	EmailFormat  bool
}

func (pps *ProjectParameterStatement) statementNode()      {}
func (pps *ProjectParameterStatement) projectSettingNode() {}
func (pps *ProjectParameterStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("parameter $%s", pps.Name))

	if pps.DataType != "" {
		out.WriteString(fmt.Sprintf(" as %s", pps.DataType))
	}

	if len(pps.Constraints) > 0 {
		out.WriteString(" from [")
		for i, c := range pps.Constraints {
			if i > 0 {
				out.WriteString(", ")
			}
			out.WriteString(fmt.Sprintf("\"%s\"", c))
		}
		out.WriteString("]")
	}

	if pps.HasDefault {
		out.WriteString(fmt.Sprintf(" defaults to \"%s\"", pps.DefaultValue))
	}

	return out.String()
}
