package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// Annotation represents a declaration decorator like @platform("linux").
type Annotation struct {
	Name  string
	Args  []string
	Token lexer.Token
}

func (a Annotation) String() string {
	var out strings.Builder
	fmt.Fprintf(&out, "@%s(", a.Name)
	for i, arg := range a.Args {
		if i > 0 {
			out.WriteString(", ")
		}
		fmt.Fprintf(&out, "%q", arg)
	}
	out.WriteString(")")
	return out.String()
}
