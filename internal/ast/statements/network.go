package statements

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/lexer"
)

// NetworkStatement represents network operations (health checks, port testing, ping)
type NetworkStatement struct {
	Token     lexer.Token
	Action    string
	Target    string
	Port      string
	Options   map[string]string
	Condition string
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

	for key, value := range ns.Options {
		out += fmt.Sprintf(" %s %s", key, value)
	}

	if ns.Condition != "" {
		out += fmt.Sprintf(" expect %s", ns.Condition)
	}

	return out
}
