package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/docker"
)

// Domain: Docker Condition Evaluation
// This file contains helpers for evaluating Docker resource conditions:
//
//	if docker network "proxy" exists:
//	when docker network "{app_network}" not exists:
//
// The dispatcher routes on the resource keyword. Only `network` is supported
// today; future resources (container, image, volume, ...) add a pattern and a
// branch in evaluateDockerCondition without touching the call sites.

var dockerNetworkExistsConditionPattern = regexp.MustCompile(`^docker\s+network\s+(.+?)\s+(not\s+)?exists\s*$`)

// evaluateDockerCondition handles Docker resource conditions. It returns
// handled=false for any condition that is not a recognized Docker condition so
// the general evaluator can take over.
func (e *Engine) evaluateDockerCondition(condition string, ctx *ExecutionContext) (bool, bool, error) {
	condition = strings.TrimSpace(condition)
	if !strings.HasPrefix(condition, "docker ") {
		return false, false, nil
	}

	if match := dockerNetworkExistsConditionPattern.FindStringSubmatch(condition); match != nil {
		return e.evaluateDockerNetworkExistsCondition(match, ctx)
	}

	return false, false, nil
}

// evaluateDockerNetworkExistsCondition probes the Docker daemon for a network:
//
//	docker network "proxy" exists
//	docker network "proxy" not exists
//
// The network name supports interpolation.
func (e *Engine) evaluateDockerNetworkExistsCondition(match []string, ctx *ExecutionContext) (bool, bool, error) {
	name, err := e.resolveDockerConditionOperand(match[1], ctx)
	if err != nil {
		return false, true, fmt.Errorf("resolving network name: %w", err)
	}

	// Dry runs never talk to the daemon; treat the network as missing so
	// dependent branches are skipped rather than executed on a fabricated probe.
	if e.dryRun {
		return false, true, nil
	}

	exists, err := docker.NewNetworkManager().CheckNetworkExists(context.Background(), name)
	if err != nil {
		return false, true, fmt.Errorf("checking docker network %q: %w", name, err)
	}

	if strings.TrimSpace(match[2]) == "not" {
		return !exists, true, nil
	}
	return exists, true, nil
}

func (e *Engine) resolveDockerConditionOperand(operand string, ctx *ExecutionContext) (string, error) {
	value, err := e.interpolateVariablesWithError(strings.TrimSpace(operand), ctx)
	if err != nil {
		return "", err
	}
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	if value == "" {
		return "", fmt.Errorf("value is empty")
	}
	return value, nil
}
