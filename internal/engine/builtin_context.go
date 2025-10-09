package engine

import (
	"github.com/phillarmonic/drun/internal/builtins"
)

// BuiltinContext implements builtins.Context interface for the engine
type BuiltinContext struct {
	execCtx        *ExecutionContext
	secretsManager SecretsManager
}

// GetProjectName returns the current project name
func (bc *BuiltinContext) GetProjectName() string {
	if bc.execCtx != nil && bc.execCtx.Project != nil {
		return bc.execCtx.Project.Name
	}
	return ""
}

// GetSecretsManager returns the secrets manager
func (bc *BuiltinContext) GetSecretsManager() builtins.SecretsManager {
	// Return nil if no secrets manager
	if bc.secretsManager == nil {
		return nil
	}
	// The SecretsManager interface matches, so we can return it directly
	return bc.secretsManager
}
