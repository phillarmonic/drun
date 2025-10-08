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
	plan, err := planner.Plan("task2", program, nil)
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

	// Verify task plans are available
	taskPlan, err := plan.GetTask("task1")
	if err != nil {
		t.Errorf("GetTask(task1) error = %v", err)
	}
	if taskPlan.Name != "task1" {
		t.Errorf("TaskPlan.Name = %v, want task1", taskPlan.Name)
	}
}

func TestPlanner_PlanMissingTask(t *testing.T) {
	registry := task.NewRegistry()
	depResolver := task.NewDependencyResolver(registry)
	planner := NewPlanner(registry, depResolver)

	program := &ast.Program{
		Tasks: []*ast.TaskStatement{},
	}

	_, err := planner.Plan("missing", program, nil)
	if err == nil {
		t.Error("Expected error for missing task, got nil")
	}
}

func TestExecutionPlan_ToJSON(t *testing.T) {
	plan := &ExecutionPlan{
		TargetTask:     "test",
		ExecutionOrder: []string{"dep1", "test"},
		ProjectName:    "test-project",
		ProjectVersion: "1.0",
		Namespaces:     map[string]bool{"ns1": true, "ns2": true},
		Tasks: map[string]*TaskPlan{
			"test": {Name: "test"},
			"dep1": {Name: "dep1"},
		},
		Hooks: &HookPlan{},
	}

	json, err := plan.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	if json == "" {
		t.Error("Expected non-empty JSON output")
	}

	// Verify it contains expected fields
	if !contains(json, "test-project") {
		t.Error("Expected JSON to contain project name")
	}
	if !contains(json, "\"TaskCount\": 2") {
		t.Error("Expected JSON to contain task count")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && len(s) >= len(substr) &&
		(s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
