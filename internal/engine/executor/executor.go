package executor

import (
	"fmt"
	"io"

	"github.com/phillarmonic/drun/internal/domain/statement"
	"github.com/phillarmonic/drun/internal/domain/task"
	"github.com/phillarmonic/drun/internal/engine/hooks"
)

// DomainStatementExecutor defines the interface for executing domain statements
// The context parameter is intentionally interface{} to avoid circular dependencies
type DomainStatementExecutor interface {
	ExecuteDomainStatement(stmt statement.Statement, ctx interface{}) error
}

// Executor handles execution of tasks and hooks
type Executor struct {
	output             io.Writer
	dryRun             bool
	domainStmtExecutor DomainStatementExecutor
}

// NewExecutor creates a new task executor
func NewExecutor(output io.Writer, dryRun bool, domainStmtExecutor DomainStatementExecutor) *Executor {
	return &Executor{
		output:             output,
		dryRun:             dryRun,
		domainStmtExecutor: domainStmtExecutor,
	}
}

// ExecuteTask executes a single task using domain statements
func (ex *Executor) ExecuteTask(domainTask *task.Task, ctx interface{}) error {
	if ex.dryRun {
		_, _ = fmt.Fprintf(ex.output, "[DRY RUN] Would execute task: %s\n", domainTask.Name)
		if domainTask.Description != "" {
			_, _ = fmt.Fprintf(ex.output, "[DRY RUN] Description: %s\n", domainTask.Description)
		}
	}

	// Execute each domain statement directly
	for _, stmt := range domainTask.Body {
		if err := ex.domainStmtExecutor.ExecuteDomainStatement(stmt, ctx); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteHooks executes a list of domain statement hooks
func (ex *Executor) ExecuteHooks(hookType string, domainHooks []statement.Statement, ctx interface{}, failFast bool) error {
	for _, hook := range domainHooks {
		if err := ex.domainStmtExecutor.ExecuteDomainStatement(hook, ctx); err != nil {
			if failFast {
				return fmt.Errorf("%s hook failed: %w", hookType, err)
			}
			_, _ = fmt.Fprintf(ex.output, "⚠️  %s hook failed: %v\n", hookType, err)
		}
	}
	return nil
}

// ExecuteSetupHooks executes setup hooks (fail-fast)
func (ex *Executor) ExecuteSetupHooks(hookMgr *hooks.Manager, ctx interface{}) error {
	if hookMgr == nil {
		return nil
	}
	return ex.ExecuteHooks("setup", hookMgr.GetSetupHooks(), ctx, true)
}

// ExecuteTeardownHooks executes teardown hooks (best-effort)
func (ex *Executor) ExecuteTeardownHooks(hookMgr *hooks.Manager, ctx interface{}) error {
	if hookMgr == nil {
		return nil
	}
	return ex.ExecuteHooks("teardown", hookMgr.GetTeardownHooks(), ctx, false)
}

// ExecuteBeforeHooks executes before-task hooks (fail-fast)
func (ex *Executor) ExecuteBeforeHooks(hookMgr *hooks.Manager, ctx interface{}) error {
	if hookMgr == nil {
		return nil
	}
	return ex.ExecuteHooks("before", hookMgr.GetBeforeHooks(), ctx, true)
}

// ExecuteAfterHooks executes after-task hooks (best-effort)
func (ex *Executor) ExecuteAfterHooks(hookMgr *hooks.Manager, ctx interface{}) error {
	if hookMgr == nil {
		return nil
	}
	return ex.ExecuteHooks("after", hookMgr.GetAfterHooks(), ctx, false)
}
