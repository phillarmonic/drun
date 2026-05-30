package hooks

import (
	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Manager manages lifecycle hooks for drun execution
type Manager struct {
	setupHooks    []statement.Statement // on drun setup hooks
	teardownHooks []statement.Statement // on drun teardown hooks
	beforeHooks   []statement.Statement // before any task hooks
	afterHooks    []statement.Statement // after any task hooks
}

// NewManager creates a new hook manager
func NewManager() *Manager {
	return &Manager{
		setupHooks:    []statement.Statement{},
		teardownHooks: []statement.Statement{},
		beforeHooks:   []statement.Statement{},
		afterHooks:    []statement.Statement{},
	}
}

// RegisterSetupHook registers a setup hook statement
func (m *Manager) RegisterSetupHook(stmt statement.Statement) {
	m.setupHooks = append(m.setupHooks, stmt)
}

// RegisterSetupHooks registers multiple setup hook statements
func (m *Manager) RegisterSetupHooks(stmts []statement.Statement) {
	m.setupHooks = append(m.setupHooks, stmts...)
}

// RegisterTeardownHook registers a teardown hook statement
func (m *Manager) RegisterTeardownHook(stmt statement.Statement) {
	m.teardownHooks = append(m.teardownHooks, stmt)
}

// RegisterTeardownHooks registers multiple teardown hook statements
func (m *Manager) RegisterTeardownHooks(stmts []statement.Statement) {
	m.teardownHooks = append(m.teardownHooks, stmts...)
}

// RegisterBeforeHook registers a before-task hook statement
func (m *Manager) RegisterBeforeHook(stmt statement.Statement) {
	m.beforeHooks = append(m.beforeHooks, stmt)
}

// RegisterBeforeHooks registers multiple before-task hook statements
func (m *Manager) RegisterBeforeHooks(stmts []statement.Statement) {
	m.beforeHooks = append(m.beforeHooks, stmts...)
}

// RegisterAfterHook registers an after-task hook statement
func (m *Manager) RegisterAfterHook(stmt statement.Statement) {
	m.afterHooks = append(m.afterHooks, stmt)
}

// RegisterAfterHooks registers multiple after-task hook statements
func (m *Manager) RegisterAfterHooks(stmts []statement.Statement) {
	m.afterHooks = append(m.afterHooks, stmts...)
}

// GetSetupHooks returns all setup hooks
func (m *Manager) GetSetupHooks() []statement.Statement {
	return m.setupHooks
}

// GetTeardownHooks returns all teardown hooks
func (m *Manager) GetTeardownHooks() []statement.Statement {
	return m.teardownHooks
}

// GetBeforeHooks returns all before-task hooks
func (m *Manager) GetBeforeHooks() []statement.Statement {
	return m.beforeHooks
}

// GetAfterHooks returns all after-task hooks
func (m *Manager) GetAfterHooks() []statement.Statement {
	return m.afterHooks
}

// Clear clears all registered hooks
func (m *Manager) Clear() {
	m.setupHooks = []statement.Statement{}
	m.teardownHooks = []statement.Statement{}
	m.beforeHooks = []statement.Statement{}
	m.afterHooks = []statement.Statement{}
}
