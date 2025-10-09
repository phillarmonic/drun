package builtins

import (
	"fmt"
)

// getSecret retrieves a secret from the secrets manager
// Syntax:
//
//	{secret('key')} - get secret from current project namespace
//	{secret('key', 'default')} - get secret with default value
//	{secret('key', '', 'namespace')} - get secret from specific namespace
func getSecret(ctx Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("secret key required")
	}

	key := args[0]
	defaultValue := ""
	namespace := ""

	// Parse arguments
	if len(args) > 1 {
		defaultValue = args[1]
	}
	if len(args) > 2 {
		namespace = args[2]
	}

	// If no explicit namespace, use project name from context
	if namespace == "" && ctx != nil {
		namespace = ctx.GetProjectName()
	}
	if namespace == "" {
		namespace = "default"
	}

	// Check if context has secrets manager
	if ctx == nil || ctx.GetSecretsManager() == nil {
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", fmt.Errorf("secrets manager not available")
	}

	secretsMgr := ctx.GetSecretsManager()

	// Try to get the secret
	value, err := secretsMgr.Get(namespace, key)
	if err != nil {
		// If secret not found and we have a default, use it
		if defaultValue != "" {
			return defaultValue, nil
		}
		return "", fmt.Errorf("secret %s:%s not found: %w", namespace, key, err)
	}

	return value, nil
}
