package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// NetworkStatement represents network operations (health checks, port testing, ping)
type NetworkStatement struct {
	Options   map[string]string
	Action    string
	Target    string
	Port      string
	Condition string
	Token     lexer.Token
}

func (ns *NetworkStatement) statementNode() {}
func (ns *NetworkStatement) String() string {
	var out string

	switch ns.Action {
	case "health_check":
		out = fmt.Sprintf("check health of service at \"%s\"", ns.Target)
	case "wait_for_service":
		out = fmt.Sprintf("wait for service at \"%s\" to be ready", ns.Target)
	case "port_check":
		if ns.Port != "" {
			out = fmt.Sprintf("check if port %s is open on \"%s\"", ns.Port, ns.Target)
		} else {
			out = fmt.Sprintf("test connection to \"%s\"", ns.Target)
		}
	case "ping":
		out = fmt.Sprintf("ping host \"%s\"", ns.Target)
	}

	var outSb38 strings.Builder
	for key, value := range ns.Options {
		fmt.Fprintf(&outSb38, " %s %s", key, value)
	}
	out += outSb38.String()

	if ns.Condition != "" {
		out += " expect " + ns.Condition
	}

	return out
}
