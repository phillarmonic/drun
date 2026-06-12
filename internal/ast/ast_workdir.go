package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// ChangeWorkdirStatement represents a working directory change within a task.
// Syntax: use workdir "path"
// The path may contain variable interpolation. Relative paths are resolved
// against the original working directory (not chained).
type ChangeWorkdirStatement struct {
	Token lexer.Token
	Path  string
}

func (cws *ChangeWorkdirStatement) statementNode() {}
func (cws *ChangeWorkdirStatement) String() string {
	return fmt.Sprintf("use workdir %q", cws.Path)
}
