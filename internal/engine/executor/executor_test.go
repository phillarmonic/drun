package executor

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/statement"
	"github.com/phillarmonic/drun/internal/domain/task"
	"github.com/phillarmonic/drun/internal/engine/hooks"
)

// MockStatementExecutor implements DomainStatementExecutor for testing
type MockStatementExecutor struct {
	ExecutedDomainStatements []statement.Statement
	ShouldFail               bool
}

func (m *MockStatementExecutor) ExecuteDomainStatement(stmt statement.Statement, ctx interface{}) error {
	m.ExecutedDomainStatements = append(m.ExecutedDomainStatements, stmt)
	if m.ShouldFail {
		return fmt.Errorf("mock execution error")
	}
	return nil
}

// MockExecutionContext implements ExecutionContext for testing
type MockExecutionContext struct {
	CurrentTask string
	Output      io.Writer
}

func (m *MockExecutionContext) GetCurrentTask() string {
	return m.CurrentTask
}

func (m *MockExecutionContext) GetOutput() io.Writer {
	return m.Output
}

func TestExecutor_ExecuteTask(t *testing.T) {
	output := &bytes.Buffer{}
	mockExec := &MockStatementExecutor{}
	executor := NewExecutor(output, false, mockExec)

	// Create a test task with domain statements
	domainTask, err := task.NewTask(&ast.TaskStatement{
		Name:        "test-task",
		Description: "Test task",
		Body: []ast.Statement{
			&ast.ActionStatement{Action: "info", Message: "test"},
		},
	}, "", "test.drun")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	ctx := &MockExecutionContext{
		CurrentTask: "test-task",
		Output:      output,
	}

	// Execute the task
	if err := executor.ExecuteTask(domainTask, ctx); err != nil {
		t.Errorf("ExecuteTask() error = %v", err)
	}

	// Verify statement was executed
	if len(mockExec.ExecutedDomainStatements) != 1 {
		t.Errorf("Expected 1 statement to be executed, got %d", len(mockExec.ExecutedDomainStatements))
	}
}

func TestExecutor_ExecuteTaskDryRun(t *testing.T) {
	output := &bytes.Buffer{}
	mockExec := &MockStatementExecutor{}
	executor := NewExecutor(output, true, mockExec)

	domainTask, err := task.NewTask(&ast.TaskStatement{
		Name:        "test-task",
		Description: "Test description",
		Body:        []ast.Statement{},
	}, "", "test.drun")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	ctx := &MockExecutionContext{
		CurrentTask: "test-task",
		Output:      output,
	}

	if err := executor.ExecuteTask(domainTask, ctx); err != nil {
		t.Errorf("ExecuteTask() error = %v", err)
	}

	// Verify dry-run output
	outputStr := output.String()
	if !bytes.Contains([]byte(outputStr), []byte("[DRY RUN]")) {
		t.Error("Expected [DRY RUN] in output")
	}
	if !bytes.Contains([]byte(outputStr), []byte("Test description")) {
		t.Error("Expected description in dry-run output")
	}
}

func TestExecutor_ExecuteSetupHooks(t *testing.T) {
	output := &bytes.Buffer{}
	mockExec := &MockStatementExecutor{}
	executor := NewExecutor(output, false, mockExec)

	hookMgr := hooks.NewManager()
	hookMgr.RegisterSetupHooks([]statement.Statement{
		&statement.Action{ActionType: "info", Message: "setup hook"},
	})

	ctx := &MockExecutionContext{
		CurrentTask: "test",
		Output:      output,
	}

	if err := executor.ExecuteSetupHooks(hookMgr, ctx); err != nil {
		t.Errorf("ExecuteSetupHooks() error = %v", err)
	}

	if len(mockExec.ExecutedDomainStatements) != 1 {
		t.Errorf("Expected 1 hook to be executed, got %d", len(mockExec.ExecutedDomainStatements))
	}
}

func TestExecutor_ExecuteSetupHooksFailFast(t *testing.T) {
	output := &bytes.Buffer{}
	mockExec := &MockStatementExecutor{ShouldFail: true}
	executor := NewExecutor(output, false, mockExec)

	hookMgr := hooks.NewManager()
	hookMgr.RegisterSetupHooks([]statement.Statement{
		&statement.Action{ActionType: "info", Message: "setup hook"},
	})

	ctx := &MockExecutionContext{
		CurrentTask: "test",
		Output:      output,
	}

	// Setup hooks should fail fast
	if err := executor.ExecuteSetupHooks(hookMgr, ctx); err == nil {
		t.Error("Expected error for failing setup hook, got nil")
	}
}

func TestExecutor_ExecuteTeardownHooksBestEffort(t *testing.T) {
	output := &bytes.Buffer{}
	mockExec := &MockStatementExecutor{ShouldFail: true}
	executor := NewExecutor(output, false, mockExec)

	hookMgr := hooks.NewManager()
	hookMgr.RegisterTeardownHooks([]statement.Statement{
		&statement.Action{ActionType: "info", Message: "teardown hook"},
	})

	ctx := &MockExecutionContext{
		CurrentTask: "test",
		Output:      output,
	}

	// Teardown hooks should NOT fail fast (best-effort)
	if err := executor.ExecuteTeardownHooks(hookMgr, ctx); err != nil {
		t.Errorf("ExecuteTeardownHooks() should not error on hook failure, got: %v", err)
	}

	// Should have warning in output
	if !bytes.Contains(output.Bytes(), []byte("⚠️")) {
		t.Error("Expected warning emoji in output for failed teardown hook")
	}
}
