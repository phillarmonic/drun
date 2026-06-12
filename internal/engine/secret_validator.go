package engine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/engine/planner"
)

// SecretReference represents a reference to a secret found in the AST
type SecretReference struct {
	Key        string
	Namespace  string
	HasDefault bool
}

// secretPattern matches secret() function calls in interpolation strings
// Matches: {secret('key')}, {secret('key', 'default')}, {secret('key', ”, 'namespace')}
var secretPattern = regexp.MustCompile(`\{secret\(['"]([^'"]+)['"](?:,\s*['"]([^'"]*)['"])?(?:,\s*['"]([^'"]*)['"])?\)\}`)

// extractSecretReferences walks through the AST and extracts all secret() references
func extractSecretReferences(program *ast.Program) []SecretReference {
	var references []SecretReference
	seen := make(map[string]bool) // Track unique references to avoid duplicates

	// Helper function to extract secrets from a string
	extractFromString := func(s string) {
		matches := secretPattern.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			key := match[1]
			defaultValue := ""
			namespace := ""

			// Parse arguments: the regex captures all groups, so:
			// secret('key') -> match[1]='key', match[2]='', match[3]=''
			// secret('key', 'default') -> match[1]='key', match[2]='default', match[3]=''
			// secret('key', '', 'namespace') -> match[1]='key', match[2]='', match[3]='namespace'
			// secret('key', 'default', 'namespace') -> match[1]='key', match[2]='default', match[3]='namespace'
			if len(match) > 2 {
				defaultValue = match[2]
			}
			if len(match) > 3 {
				namespace = match[3]
			}

			// Create unique key for deduplication
			uniqueKey := fmt.Sprintf("%s:%s", namespace, key)
			if seen[uniqueKey] {
				continue
			}
			seen[uniqueKey] = true

			references = append(references, SecretReference{
				Key:        key,
				Namespace:  namespace,
				HasDefault: defaultValue != "",
			})
		}
	}

	// Walk through all tasks
	for _, task := range program.Tasks {
		extractFromTask(task, extractFromString)
	}

	// Walk through all templates
	for _, template := range program.Templates {
		extractFromTaskTemplate(template, extractFromString)
	}

	// Walk through all services
	for _, service := range program.Services {
		extractFromService(service, extractFromString)
	}

	// Walk through orchestrations
	for _, orchestration := range program.Orchestrations {
		extractFromOrchestration(orchestration, extractFromString)
	}

	return references
}

// extractFromTask extracts secret references from a task
func extractFromTask(task *ast.TaskStatement, extractFromString func(string)) {
	// Extract from description
	if task.Description != "" {
		extractFromString(task.Description)
	}

	// Extract from parameters
	for _, param := range task.Parameters {
		if param.DefaultValue != "" {
			extractFromString(param.DefaultValue)
		}
	}

	// Extract from body statements
	for _, stmt := range task.Body {
		extractFromStatement(stmt, extractFromString)
	}
}

// extractFromTaskTemplate extracts secret references from a task template
func extractFromTaskTemplate(template *ast.TaskTemplateStatement, extractFromString func(string)) {
	if template.Description != "" {
		extractFromString(template.Description)
	}

	for _, param := range template.Parameters {
		if param.DefaultValue != "" {
			extractFromString(param.DefaultValue)
		}
	}

	for _, stmt := range template.Body {
		extractFromStatement(stmt, extractFromString)
	}
}

// extractFromService extracts secret references from a service
func extractFromService(service *ast.ServiceStatement, extractFromString func(string)) {
	// Services are project-level settings, extract from their fields
	if service.Description != "" {
		extractFromString(service.Description)
	}
	if service.Path != "" {
		extractFromString(service.Path)
	}
	if service.PreTask != "" {
		extractFromString(service.PreTask)
	}
	if service.PostTask != "" {
		extractFromString(service.PostTask)
	}
	for _, value := range service.Environment {
		extractFromString(value)
	}
	// Note: ServiceStatement has nested configs (Repository, HealthCheck, etc.)
	// but those are typically not string-based, so we skip them for now
}

// extractFromOrchestration extracts secret references from an orchestration
func extractFromOrchestration(orchestration *ast.OrchestrateStatement, extractFromString func(string)) {
	// Orchestrations are project-level settings, extract from their fields
	if orchestration.Description != "" {
		extractFromString(orchestration.Description)
	}
	if orchestration.PreTask != "" {
		extractFromString(orchestration.PreTask)
	}
	if orchestration.PostTask != "" {
		extractFromString(orchestration.PostTask)
	}
	if orchestration.GitSSHKey != "" {
		extractFromString(orchestration.GitSSHKey)
	}
	// Note: OrchestrateActionStatement appears in task bodies, handled in extractFromStatement
}

// extractFromStatement extracts secret references from a statement
func extractFromStatement(stmt ast.Statement, extractFromString func(string)) {
	switch s := stmt.(type) {
	case *ast.ActionStatement:
		if s.Message != "" {
			extractFromString(s.Message)
		}

	case *ast.ShellStatement:
		if s.Command != "" {
			extractFromString(s.Command)
		}
		for _, cmd := range s.Commands {
			extractFromString(cmd)
		}
		if s.ServiceName != "" && !s.ServiceNameIsLiteral {
			extractFromString(s.ServiceName)
		}

	case *ast.VariableStatement:
		if s.Value != nil {
			extractFromString(s.Value.String())
		}
		for _, arg := range s.Arguments {
			extractFromString(arg)
		}

	case *ast.OrchestrationActionStatement:
		if s.GroupName != "" {
			extractFromString(s.GroupName)
		}
		for _, value := range s.Options {
			extractFromString(value)
		}
		for _, filter := range s.ServiceFilters {
			extractFromString(filter)
		}

	case *ast.FileStatement:
		if s.Target != "" {
			extractFromString(s.Target)
		}
		if s.Source != "" {
			extractFromString(s.Source)
		}
		if s.Content != "" {
			extractFromString(s.Content)
		}
		for _, value := range s.Replacements {
			extractFromString(value)
		}

	case *ast.DockerStatement:
		if s.Name != "" {
			extractFromString(s.Name)
		}
		if s.Resource != "" {
			extractFromString(s.Resource)
		}
		if s.ServiceName != "" && !s.ServiceNameIsLiteral {
			extractFromString(s.ServiceName)
		}
		for _, value := range s.Options {
			extractFromString(value)
		}

	case *ast.GitStatement:
		if s.Name != "" {
			extractFromString(s.Name)
		}
		if s.Resource != "" {
			extractFromString(s.Resource)
		}
		for _, value := range s.Options {
			extractFromString(value)
		}

	case *ast.HTTPStatement:
		if s.URL != "" {
			extractFromString(s.URL)
		}
		if s.Body != "" {
			extractFromString(s.Body)
		}
		for _, value := range s.Headers {
			extractFromString(value)
		}
		for _, value := range s.Auth {
			extractFromString(value)
		}
		for _, value := range s.Options {
			extractFromString(value)
		}

	case *ast.DownloadStatement:
		if s.URL != "" {
			extractFromString(s.URL)
		}
		if s.Path != "" {
			extractFromString(s.Path)
		}
		if s.ExtractTo != "" {
			extractFromString(s.ExtractTo)
		}
		for _, value := range s.Headers {
			extractFromString(value)
		}
		for _, value := range s.Auth {
			extractFromString(value)
		}
		for _, value := range s.Options {
			extractFromString(value)
		}

	case *ast.NetworkStatement:
		if s.Target != "" {
			extractFromString(s.Target)
		}
		if s.Port != "" {
			extractFromString(s.Port)
		}
		if s.Condition != "" {
			extractFromString(s.Condition)
		}
		for _, value := range s.Options {
			extractFromString(value)
		}

	case *ast.DetectionStatement:
		if s.Target != "" {
			extractFromString(s.Target)
		}
		if s.Condition != "" {
			extractFromString(s.Condition)
		}
		if s.Value != "" {
			extractFromString(s.Value)
		}
		for _, alt := range s.Alternatives {
			extractFromString(alt)
		}
		for _, stmt := range s.Body {
			extractFromStatement(stmt, extractFromString)
		}
		for _, stmt := range s.ElseBody {
			extractFromStatement(stmt, extractFromString)
		}

	case *ast.ConditionalStatement:
		if s.Condition != "" {
			extractFromString(s.Condition)
		}
		for _, stmt := range s.Body {
			extractFromStatement(stmt, extractFromString)
		}
		for _, stmt := range s.ElseBody {
			extractFromStatement(stmt, extractFromString)
		}

	case *ast.LoopStatement:
		if s.Iterable != "" {
			extractFromString(s.Iterable)
		}
		if s.RangeStart != "" {
			extractFromString(s.RangeStart)
		}
		if s.RangeEnd != "" {
			extractFromString(s.RangeEnd)
		}
		if s.RangeStep != "" {
			extractFromString(s.RangeStep)
		}
		if s.Filter != nil && s.Filter.Value != "" {
			extractFromString(s.Filter.Value)
		}
		for _, stmt := range s.Body {
			extractFromStatement(stmt, extractFromString)
		}

	case *ast.TryStatement:
		for _, stmt := range s.TryBody {
			extractFromStatement(stmt, extractFromString)
		}
		for _, clause := range s.CatchClauses {
			for _, stmt := range clause.Body {
				extractFromStatement(stmt, extractFromString)
			}
		}
		for _, stmt := range s.FinallyBody {
			extractFromStatement(stmt, extractFromString)
		}

	case *ast.ThrowStatement:
		if s.Message != "" {
			extractFromString(s.Message)
		}

	case *ast.BreakStatement:
		if s.Condition != "" {
			extractFromString(s.Condition)
		}

	case *ast.ContinueStatement:
		if s.Condition != "" {
			extractFromString(s.Condition)
		}

	case *ast.TaskCallStatement:
		if s.TaskName != "" {
			extractFromString(s.TaskName)
		}
		for _, value := range s.Parameters {
			extractFromString(value)
		}

	case *ast.TaskFromTemplateStatement:
		if s.TemplateName != "" {
			extractFromString(s.TemplateName)
		}
		for _, value := range s.Overrides {
			extractFromString(value)
		}

	case *ast.SecretStatement:
		// Secret statements themselves don't need validation (they set secrets)
		// But we might want to check if the value being set contains secret references
		if s.Value != nil {
			extractFromString(s.Value.String())
		}
		if s.Default != nil {
			extractFromString(s.Default.String())
		}
		if s.Namespace != "" {
			extractFromString(s.Namespace)
		}
		if s.Pattern != "" {
			extractFromString(s.Pattern)
		}
	}
}

// extractSecretsSetInPlan extracts all secrets that will be set in the execution plan
func extractSecretsSetInPlan(plan *planner.ExecutionPlan, projectName string) map[string]bool {
	secretsSet := make(map[string]bool)

	// Helper to add a secret to the set
	addSecret := func(namespace, key string) {
		if namespace == "" {
			if projectName != "" {
				namespace = projectName
			} else {
				namespace = "default"
			}
		}
		secretsSet[fmt.Sprintf("%s:%s", namespace, key)] = true
	}

	// Walk through all tasks in the execution plan
	for _, taskName := range plan.ExecutionOrder {
		taskPlan, err := plan.GetTask(taskName)
		if err != nil {
			continue
		}

		// Extract secrets set in this task's body
		for _, stmt := range taskPlan.Body {
			if secretStmt, ok := stmt.(*statement.Secret); ok {
				if secretStmt.Operation == "set" {
					namespace := secretStmt.Namespace
					if namespace == "" {
						namespace = projectName
					}
					addSecret(namespace, secretStmt.Key)
				}
			}
		}
	}

	// Also check setup hooks
	if plan.Hooks != nil {
		for _, stmt := range plan.Hooks.SetupHooks {
			if secretStmt, ok := stmt.(*statement.Secret); ok {
				if secretStmt.Operation == "set" {
					namespace := secretStmt.Namespace
					if namespace == "" {
						namespace = projectName
					}
					addSecret(namespace, secretStmt.Key)
				}
			}
		}
	}

	return secretsSet
}

// validateSecrets checks if all referenced secrets exist before execution
// It skips validation for secrets that are set in the execution plan
func (e *Engine) validateSecrets(program *ast.Program, plan *planner.ExecutionPlan, projectName string) error {
	if e.secretsManager == nil {
		// If secrets manager is not initialized, skip validation
		// This allows scripts to run even if secrets manager fails to initialize
		return nil
	}

	// Skip validation in dry-run mode - we're not actually executing, so secrets don't need to exist
	// This allows example files and test scripts to validate syntax without requiring secrets
	if e.dryRun {
		// In dry-run mode, we skip secret validation since we're not actually executing
		// The dryRun flag is checked here to avoid requiring secrets for syntax validation
		return nil
	}

	references := extractSecretReferences(program)
	if len(references) == 0 {
		// No secret references found, nothing to validate
		return nil
	}

	// Extract secrets that will be set in the execution plan
	secretsSet := extractSecretsSetInPlan(plan, projectName)

	var missingSecrets []string

	for _, ref := range references {
		// Skip secrets with default values - they're optional
		if ref.HasDefault {
			continue
		}

		// Determine namespace
		namespace := ref.Namespace
		if namespace == "" {
			// Use project name as default namespace
			if projectName != "" {
				namespace = projectName
			} else {
				namespace = "default"
			}
		}

		// Check if this secret will be set during execution
		secretKey := fmt.Sprintf("%s:%s", namespace, ref.Key)
		if secretsSet[secretKey] {
			// Secret will be set during execution, skip validation
			continue
		}

		// Check if secret exists
		exists, err := e.secretsManager.Exists(namespace, ref.Key)
		if err != nil {
			// If there's an error checking existence, we'll treat it as missing
			// This ensures we fail fast rather than continuing with uncertainty
			missingSecrets = append(missingSecrets, fmt.Sprintf("%s:%s (error checking: %v)", namespace, ref.Key, err))
			continue
		}

		if !exists {
			missingSecrets = append(missingSecrets, fmt.Sprintf("%s:%s", namespace, ref.Key))
		}
	}

	if len(missingSecrets) > 0 {
		if len(missingSecrets) == 1 {
			return fmt.Errorf("secret not found: %s\n\nUse 'xdrun cmd:secret set \"%s\" to \"value\"' to set it", missingSecrets[0], strings.Split(missingSecrets[0], ":")[1])
		}
		return fmt.Errorf("secrets not found: %s\n\nUse 'xdrun cmd:secret set' to set missing secrets", strings.Join(missingSecrets, ", "))
	}

	return nil
}
