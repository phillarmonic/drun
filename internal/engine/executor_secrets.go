package engine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Domain: Secret Operations Execution
// This file contains executors for:
// - Secret set/get/delete operations
// - Secret existence checks
// - Secret listing

// executeSecret executes secret operation statements
func (e *Engine) executeSecret(secretStmt *statement.Secret, ctx *ExecutionContext) error {
	switch secretStmt.Operation {
	case "set":
		return e.executeSecretSet(secretStmt, ctx)
	case "get":
		return e.executeSecretGet(secretStmt, ctx)
	case "delete":
		return e.executeSecretDelete(secretStmt, ctx)
	case "exists":
		return e.executeSecretExists(secretStmt, ctx)
	case "list":
		return e.executeSecretList(secretStmt, ctx)
	default:
		return fmt.Errorf("unknown secret operation: %s", secretStmt.Operation)
	}
}

// executeSecretSet executes "secret set" statements
func (e *Engine) executeSecretSet(secretStmt *statement.Secret, ctx *ExecutionContext) error {
	// Interpolate the value
	interpolatedValue := e.interpolateVariables(secretStmt.Value, ctx)

	// Determine namespace (use project name if not specified)
	namespace := secretStmt.Namespace
	if namespace == "" {
		if ctx.Project != nil && ctx.Project.Name != "" {
			namespace = ctx.Project.Name
		} else {
			namespace = "default"
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set secret %s:%s = [REDACTED]\n", namespace, secretStmt.Key)
		return nil
	}

	// Store the secret using secrets manager
	if e.secretsManager != nil {
		if err := e.secretsManager.Set(namespace, secretStmt.Key, interpolatedValue); err != nil {
			return fmt.Errorf("failed to set secret %s:%s: %w", namespace, secretStmt.Key, err)
		}
		_, _ = fmt.Fprintf(e.output, "🔐 Secret %s stored securely (namespace: %s)\n", secretStmt.Key, namespace)
	} else {
		return fmt.Errorf("secrets manager not initialized")
	}

	return nil
}

// executeSecretGet executes "secret get" statements
func (e *Engine) executeSecretGet(secretStmt *statement.Secret, ctx *ExecutionContext) error {
	// Determine namespace (use project name if not specified)
	namespace := secretStmt.Namespace
	if namespace == "" {
		if ctx.Project != nil && ctx.Project.Name != "" {
			namespace = ctx.Project.Name
		} else {
			namespace = "default"
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would get secret %s:%s\n", namespace, secretStmt.Key)
		return nil
	}

	// Retrieve the secret using secrets manager
	var value string
	if e.secretsManager != nil {
		val, err := e.secretsManager.Get(namespace, secretStmt.Key)
		if err != nil {
			// Check if we have a default value
			if secretStmt.Default != "" {
				interpolatedDefault := e.interpolateVariables(secretStmt.Default, ctx)
				value = interpolatedDefault
				_, _ = fmt.Fprintf(e.output, "🔓 Secret %s not found, using default value (namespace: %s)\n", secretStmt.Key, namespace)
			} else {
				return fmt.Errorf("failed to get secret %s:%s: %w", namespace, secretStmt.Key, err)
			}
		} else {
			value = val
			_, _ = fmt.Fprintf(e.output, "🔓 Retrieved secret %s (namespace: %s)\n", secretStmt.Key, namespace)
		}
	} else {
		return fmt.Errorf("secrets manager not initialized")
	}

	// Secret get is typically used in variable assignments, but since we're in a domain
	// statement, we would need to return the value somehow. For now, we'll just log it.
	// In the future, we might want to support: let $var = secret get "key"
	// For now, this is a placeholder
	_ = value

	return nil
}

// executeSecretDelete executes "secret delete" statements
func (e *Engine) executeSecretDelete(secretStmt *statement.Secret, ctx *ExecutionContext) error {
	// Determine namespace (use project name if not specified)
	namespace := secretStmt.Namespace
	if namespace == "" {
		if ctx.Project != nil && ctx.Project.Name != "" {
			namespace = ctx.Project.Name
		} else {
			namespace = "default"
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would delete secret %s:%s\n", namespace, secretStmt.Key)
		return nil
	}

	// Delete the secret using secrets manager
	if e.secretsManager != nil {
		if err := e.secretsManager.Delete(namespace, secretStmt.Key); err != nil {
			return fmt.Errorf("failed to delete secret %s:%s: %w", namespace, secretStmt.Key, err)
		}
		_, _ = fmt.Fprintf(e.output, "🗑️  Secret %s deleted (namespace: %s)\n", secretStmt.Key, namespace)
	} else {
		return fmt.Errorf("secrets manager not initialized")
	}

	return nil
}

// executeSecretExists executes "secret exists" statements
func (e *Engine) executeSecretExists(secretStmt *statement.Secret, ctx *ExecutionContext) error {
	// Determine namespace (use project name if not specified)
	namespace := secretStmt.Namespace
	if namespace == "" {
		if ctx.Project != nil && ctx.Project.Name != "" {
			namespace = ctx.Project.Name
		} else {
			namespace = "default"
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if secret %s:%s exists\n", namespace, secretStmt.Key)
		return nil
	}

	// Check if secret exists using secrets manager
	if e.secretsManager != nil {
		exists, err := e.secretsManager.Exists(namespace, secretStmt.Key)
		if err != nil {
			return fmt.Errorf("failed to check secret %s:%s: %w", namespace, secretStmt.Key, err)
		}

		if exists {
			_, _ = fmt.Fprintf(e.output, "✅ Secret %s exists (namespace: %s)\n", secretStmt.Key, namespace)
		} else {
			_, _ = fmt.Fprintf(e.output, "❌ Secret %s does not exist (namespace: %s)\n", secretStmt.Key, namespace)
		}
	} else {
		return fmt.Errorf("secrets manager not initialized")
	}

	return nil
}

// executeSecretList executes "secret list" statements
func (e *Engine) executeSecretList(secretStmt *statement.Secret, ctx *ExecutionContext) error {
	// Determine namespace (use project name if not specified)
	namespace := secretStmt.Namespace
	if namespace == "" {
		if ctx.Project != nil && ctx.Project.Name != "" {
			namespace = ctx.Project.Name
		} else {
			namespace = "default"
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would list secrets in namespace: %s\n", namespace)
		return nil
	}

	// List secrets using secrets manager
	if e.secretsManager != nil {
		keys, err := e.secretsManager.List(namespace)
		if err != nil {
			return fmt.Errorf("failed to list secrets in namespace %s: %w", namespace, err)
		}

		// Filter by pattern if specified
		if secretStmt.Pattern != "" {
			var filtered []string
			pattern := secretStmt.Pattern
			for _, key := range keys {
				// Simple wildcard matching (replace * with .*)
				regexPattern := strings.ReplaceAll(pattern, "*", ".*")
				if matched, _ := regexp.MatchString("^"+regexPattern+"$", key); matched {
					filtered = append(filtered, key)
				}
			}
			keys = filtered
		}

		if len(keys) == 0 {
			_, _ = fmt.Fprintf(e.output, "📋 No secrets found in namespace: %s\n", namespace)
		} else {
			_, _ = fmt.Fprintf(e.output, "📋 Secrets in namespace %s:\n", namespace)
			for _, key := range keys {
				_, _ = fmt.Fprintf(e.output, "   - %s\n", key)
			}
		}
	} else {
		return fmt.Errorf("secrets manager not initialized")
	}

	return nil
}
