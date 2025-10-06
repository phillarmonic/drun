package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/lexer"
)

// DockerStatement represents Docker operations
type DockerStatement struct {
	Token     lexer.Token
	Operation string
	Resource  string
	Name      string
	Options   map[string]string
}

func (ds *DockerStatement) statementNode() {}
func (ds *DockerStatement) String() string {
	out := fmt.Sprintf("docker %s %s", ds.Operation, ds.Resource)
	if ds.Name != "" {
		out += fmt.Sprintf(" \"%s\"", ds.Name)
	}

	for key, value := range ds.Options {
		out += fmt.Sprintf(" %s \"%s\"", key, value)
	}

	return out
}
