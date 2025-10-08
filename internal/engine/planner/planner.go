package planner

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/task"
)

// ExecutionPlan represents a planned execution with resolved dependencies
type ExecutionPlan struct {
	TaskName       string
	ExecutionOrder []string                      // Tasks to execute in order
	DomainTasks    map[string]*task.Task         // Map of task name to domain task
	ASTTasks       map[string]*ast.TaskStatement // Map of task name to AST task (temporary)
}

// Planner orchestrates dependency resolution and produces execution plans
type Planner struct {
	taskRegistry *task.Registry
	depResolver  *task.DependencyResolver
}

// NewPlanner creates a new execution planner
func NewPlanner(taskRegistry *task.Registry, depResolver *task.DependencyResolver) *Planner {
	return &Planner{
		taskRegistry: taskRegistry,
		depResolver:  depResolver,
	}
}

// Plan creates an execution plan for the given task
func (p *Planner) Plan(taskName string, program *ast.Program) (*ExecutionPlan, error) {
	// Resolve dependencies using domain resolver
	domainTasks, err := p.depResolver.Resolve(taskName)
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Build execution order
	executionOrder := make([]string, len(domainTasks))
	domainTaskMap := make(map[string]*task.Task)
	for i, t := range domainTasks {
		executionOrder[i] = t.Name
		domainTaskMap[t.Name] = t
	}

	// Build AST task map (temporary - will be removed in Phase 4)
	astTaskMap := make(map[string]*ast.TaskStatement)
	for _, astTask := range program.Tasks {
		astTaskMap[astTask.Name] = astTask
	}

	return &ExecutionPlan{
		TaskName:       taskName,
		ExecutionOrder: executionOrder,
		DomainTasks:    domainTaskMap,
		ASTTasks:       astTaskMap,
	}, nil
}

// GetDomainTask retrieves a domain task from the plan
func (ep *ExecutionPlan) GetDomainTask(name string) (*task.Task, error) {
	t, ok := ep.DomainTasks[name]
	if !ok {
		return nil, fmt.Errorf("task '%s' not found in execution plan", name)
	}
	return t, nil
}

// GetASTTask retrieves an AST task from the plan (temporary)
func (ep *ExecutionPlan) GetASTTask(name string) (*ast.TaskStatement, error) {
	t, ok := ep.ASTTasks[name]
	if !ok {
		return nil, fmt.Errorf("AST task '%s' not found in execution plan", name)
	}
	return t, nil
}
