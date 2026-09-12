package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// DockerStatement represents Docker operations
type DockerStatement struct {
	Options              map[string]string
	Operation            string
	Resource             string
	Name                 string
	ServiceName          string
	Token                lexer.Token
	ServiceScoped        bool
	ServiceNameIsLiteral bool
}

func (ds *DockerStatement) statementNode() {}
func (ds *DockerStatement) String() string {
	out := fmt.Sprintf("docker %s %s", ds.Operation, ds.Resource)
	if ds.ServiceScoped {
		if ds.ServiceNameIsLiteral {
			out += fmt.Sprintf(" in service \"%s\"", ds.ServiceName)
		} else {
			out += " in service " + ds.ServiceName
		}
	}
	if ds.Name != "" {
		out += fmt.Sprintf(" \"%s\"", ds.Name)
	}

	var outSb35 strings.Builder
	for key, value := range ds.Options {
		fmt.Fprintf(&outSb35, " %s \"%s\"", key, value)
	}
	out += outSb35.String()

	return out
}
