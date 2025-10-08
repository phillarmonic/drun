package planner

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/task"
)

func TestPlanner_Plan(t *testing.T) {
	// Create task registry
	registry := task.NewRegistry()

	// Create test tasks
	task1, err := task.NewTask(&ast.TaskStatement{
		Name: "task1",
		Body: []ast.Statement{},
	}, "", "test.drun")
	if err != nil {
		t.Fatalf("Failed to create task1: %v", err)
	}

	task2, err := task.NewTask(&ast.TaskStatement{
		Name: "task2",
		Body: []ast.Statement{},
		Dependencies: []ast.DependencyGroup{
			{
				Dependencies: []ast.DependencyItem{
					{Name: "task1"},
				},
			},
		},
	}, "", "test.drun")
	if err != nil {
		t.Fatalf("Failed to create task2: %v", err)
	}

	// Register tasks
	if err := registry.Register(task1); err != nil {
		t.Fatalf("Failed to register task1: %v", err)
	}
	if err := registry.Register(task2); err != nil {
		t.Fatalf("Failed to register task2: %v", err)
	}

	// Create planner
	depResolver := task.NewDependencyResolver(registry)
	planner := NewPlanner(registry, depResolver)

	// Create mock program
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "task1", Body: []ast.Statement{}},
			{Name: "task2", Body: []ast.Statement{}},
		},
	}

	// Plan execution
	plan, err := planner.Plan("task2", program)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	// Verify execution order (task1 should come before task2)
	if len(plan.ExecutionOrder) != 2 {
		t.Errorf("ExecutionOrder length = %v, want 2", len(plan.ExecutionOrder))
	}
	if plan.ExecutionOrder[0] != "task1" {
		t.Errorf("ExecutionOrder[0] = %v, want task1", plan.ExecutionOrder[0])
	}
	if plan.ExecutionOrder[1] != "task2" {
		t.Errorf("ExecutionOrder[1] = %v, want task2", plan.ExecutionOrder[1])
	}

	// Verify domain tasks are available
	domainTask, err := plan.GetDomainTask("task1")
	if err != nil {
		t.Errorf("GetDomainTask(task1) error = %v", err)
	}
	if domainTask.Name != "task1" {
		t.Errorf("DomainTask.Name = %v, want task1", domainTask.Name)
	}

	// Verify AST tasks are available
	astTask, err := plan.GetASTTask("task1")
	if err != nil {
		t.Errorf("GetASTTask(task1) error = %v", err)
	}
	if astTask.Name != "task1" {
		t.Errorf("ASTTask.Name = %v, want task1", astTask.Name)
	}
}

func TestPlanner_PlanMissingTask(t *testing.T) {
	registry := task.NewRegistry()
	depResolver := task.NewDependencyResolver(registry)
	planner := NewPlanner(registry, depResolver)

	program := &ast.Program{
		Tasks: []*ast.TaskStatement{},
	}

	_, err := planner.Plan("missing", program)
	if err == nil {
		t.Error("Expected error for missing task, got nil")
	}
}
