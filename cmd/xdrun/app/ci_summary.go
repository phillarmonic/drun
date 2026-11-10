package app

import "github.com/phillarmonic/drun/v2/internal/engine"

// normalizeCISummary validates a --ci-summary value. The empty string selects
// the default (text). Validation lives in the engine so the CLI and the engine
// agree on the accepted set.
func normalizeCISummary(format string) (string, error) {
	normalized, err := engine.NormalizeCISummaryFormat(format)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}
