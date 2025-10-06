package hooks

import (
	"github.com/phillarmonic/drun/internal/ast"
)

// Manager manages lifecycle hooks for drun execution
type Manager struct {
	setupHooks    []ast.Statement // on drun setup hooks
	teardownHooks []ast.Statement // on drun teardown hooks
	beforeHooks   []ast.Statement // before any task hooks
	afterHooks    []ast.Statement // after any task hooks
}

// NewManager creates a new hook manager
func NewManager() *Manager {
	return &Manager{
		setupHooks:    []ast.Statement{},
		teardownHooks: []ast.Statement{},
		beforeHooks:   []ast.Statement{},
		afterHooks:    []ast.Statement{},
	}
}

// RegisterSetupHook registers a setup hook statement
func (m *Manager) RegisterSetupHook(stmt ast.Statement) {
	m.setupHooks = append(m.setupHooks, stmt)
}

// RegisterSetupHooks registers multiple setup hook statements
func (m *Manager) RegisterSetupHooks(stmts []ast.Statement) {
	m.setupHooks = append(m.setupHooks, stmts...)
}

// RegisterTeardownHook registers a teardown hook statement
func (m *Manager) RegisterTeardownHook(stmt ast.Statement) {
	m.teardownHooks = append(m.teardownHooks, stmt)
}

// RegisterTeardownHooks registers multiple teardown hook statements
func (m *Manager) RegisterTeardownHooks(stmts []ast.Statement) {
	m.teardownHooks = append(m.teardownHooks, stmts...)
}

// RegisterBeforeHook registers a before-task hook statement
func (m *Manager) RegisterBeforeHook(stmt ast.Statement) {
	m.beforeHooks = append(m.beforeHooks, stmt)
}

// RegisterBeforeHooks registers multiple before-task hook statements
func (m *Manager) RegisterBeforeHooks(stmts []ast.Statement) {
	m.beforeHooks = append(m.beforeHooks, stmts...)
}

// RegisterAfterHook registers an after-task hook statement
func (m *Manager) RegisterAfterHook(stmt ast.Statement) {
	m.afterHooks = append(m.afterHooks, stmt)
}

// RegisterAfterHooks registers multiple after-task hook statements
func (m *Manager) RegisterAfterHooks(stmts []ast.Statement) {
	m.afterHooks = append(m.afterHooks, stmts...)
}

// GetSetupHooks returns all setup hooks
func (m *Manager) GetSetupHooks() []ast.Statement {
	return m.setupHooks
}

// GetTeardownHooks returns all teardown hooks
func (m *Manager) GetTeardownHooks() []ast.Statement {
	return m.teardownHooks
}

// GetBeforeHooks returns all before-task hooks
func (m *Manager) GetBeforeHooks() []ast.Statement {
	return m.beforeHooks
}

// GetAfterHooks returns all after-task hooks
func (m *Manager) GetAfterHooks() []ast.Statement {
	return m.afterHooks
}

// Clear clears all registered hooks
func (m *Manager) Clear() {
	m.setupHooks = []ast.Statement{}
	m.teardownHooks = []ast.Statement{}
	m.beforeHooks = []ast.Statement{}
	m.afterHooks = []ast.Statement{}
}
