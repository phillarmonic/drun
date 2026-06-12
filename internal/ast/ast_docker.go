package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// DockerStatement represents Docker operations
type DockerStatement struct {
	Token                lexer.Token
	Operation            string
	Resource             string
	Name                 string
	Options              map[string]string
	ServiceScoped        bool
	ServiceName          string
	ServiceNameIsLiteral bool
}

func (ds *DockerStatement) statementNode() {}
func (ds *DockerStatement) String() string {
	out := fmt.Sprintf("docker %s %s", ds.Operation, ds.Resource)
	if ds.ServiceScoped {
		if ds.ServiceNameIsLiteral {
			out += fmt.Sprintf(" in service \"%s\"", ds.ServiceName)
		} else {
			out += fmt.Sprintf(" in service %s", ds.ServiceName)
		}
	}
	if ds.Name != "" {
		out += fmt.Sprintf(" \"%s\"", ds.Name)
	}

	for key, value := range ds.Options {
		out += fmt.Sprintf(" %s \"%s\"", key, value)
	}

	return out
}
