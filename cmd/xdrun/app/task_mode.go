package app

import (
	"fmt"
	"strings"
)

func normalizeRuntimeTaskMode(mode string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "", "ci", "normal":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported runtime task mode %q (supported: ci, normal)", mode)
	}
}
