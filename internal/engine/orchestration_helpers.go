package engine

import (
	"fmt"
	"strings"
)

func (e *Engine) resolveOrchestrateServicesBuiltin(execCtx *ExecutionContext, args []string) (string, error) {
	if execCtx == nil || execCtx.Program == nil {
		return "", fmt.Errorf("no orchestration context available")
	}

	groupName := strings.TrimSpace(args[0])
	if groupName == "" {
		return "", fmt.Errorf("orchestrate services requires an orchestration name")
	}

	var orchestrationServices []string
	for _, orch := range execCtx.Program.Orchestrations {
		if orch.Name == groupName {
			orchestrationServices = orch.Services
			break
		}
	}

	if orchestrationServices == nil {
		return "", fmt.Errorf("orchestration '%s' not found", groupName)
	}

	quoted := make([]string, 0, len(orchestrationServices))
	for _, svc := range orchestrationServices {
		trimmed := strings.TrimSpace(svc)
		if trimmed == "" {
			continue
		}
		quoted = append(quoted, fmt.Sprintf("%q", trimmed))
	}

	return fmt.Sprintf("[%s]", strings.Join(quoted, ", ")), nil
}
