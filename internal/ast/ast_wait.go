package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// WaitStatement represents a fixed-duration wait (wait 5 seconds, wait {retries} minutes)
type WaitStatement struct {
	Token lexer.Token
	Value string // Raw value: a number literal or a {variable} interpolation
	Unit  string // Normalized singular unit: "second", "minute", "hour"
}

func (ws *WaitStatement) statementNode() {}
func (ws *WaitStatement) String() string {
	unit := ws.Unit
	if ws.Value != "1" {
		unit += "s"
	}
	return fmt.Sprintf("wait %s %s", ws.Value, unit)
}
