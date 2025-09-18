package engine

import (
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/v2/ast"
)

func TestDependencyResolver_SimpleDependency(t *testing.T) {
	// build -> deploy
	tasks := []*ast.TaskStatement{
		{
			Name:         "build",
			Dependencies: []ast.DependencyGroup{},
		},
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"},
					},
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	order, err := resolver.ResolveDependencies("deploy")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	expected := []string{"build", "deploy"}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d tasks, got %d", len(expected), len(order))
	}

	for i, task := range expected {
		if order[i] != task {
			t.Errorf("Expected task %d to be %s, got %s", i, task, order[i])
		}
	}
}

func TestDependencyResolver_ChainedDependencies(t *testing.T) {
	// install -> build -> test -> deploy
	tasks := []*ast.TaskStatement{
		{
			Name:         "install",
			Dependencies: []ast.DependencyGroup{},
		},
		{
			Name: "build",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "install"},
					},
				},
			},
		},
		{
			Name: "test",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"},
					},
				},
			},
		},
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "test"},
					},
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	order, err := resolver.ResolveDependencies("deploy")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	expected := []string{"install", "build", "test", "deploy"}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d tasks, got %d", len(expected), len(order))
	}

	for i, task := range expected {
		if order[i] != task {
			t.Errorf("Expected task %d to be %s, got %s", i, task, order[i])
		}
	}
}

func TestDependencyResolver_ParallelDependencies(t *testing.T) {
	// lint, test -> deploy
	tasks := []*ast.TaskStatement{
		{
			Name:         "lint",
			Dependencies: []ast.DependencyGroup{},
		},
		{
			Name:         "test",
			Dependencies: []ast.DependencyGroup{},
		},
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "lint"},
						{Name: "test"},
					},
					Sequential: false, // parallel
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	order, err := resolver.ResolveDependencies("deploy")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	// Both lint and test should come before deploy
	if len(order) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(order))
	}

	if order[2] != "deploy" {
		t.Errorf("Expected deploy to be last, got %s", order[2])
	}

	// lint and test can be in any order (they're parallel)
	deps := []string{order[0], order[1]}
	if !containsString(deps, "lint") || !containsString(deps, "test") {
		t.Errorf("Expected lint and test as dependencies, got %v", deps)
	}
}

func TestDependencyResolver_CircularDependency(t *testing.T) {
	// build -> test -> build (circular)
	tasks := []*ast.TaskStatement{
		{
			Name: "build",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "test"},
					},
				},
			},
		},
		{
			Name: "test",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"},
					},
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	_, err := resolver.ResolveDependencies("build")
	if err == nil {
		t.Fatalf("Expected circular dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

func TestDependencyResolver_MissingDependency(t *testing.T) {
	// deploy depends on non-existent "build" task
	tasks := []*ast.TaskStatement{
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"}, // doesn't exist
					},
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	_, err := resolver.ResolveDependencies("deploy")
	if err == nil {
		t.Fatalf("Expected missing dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected missing dependency error, got: %v", err)
	}
}

func TestDependencyResolver_ComplexDependencies(t *testing.T) {
	// Complex dependency graph:
	// install -> build -> test
	//         -> lint -> security_scan
	//                 -> deploy
	tasks := []*ast.TaskStatement{
		{
			Name:         "install",
			Dependencies: []ast.DependencyGroup{},
		},
		{
			Name: "build",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "install"},
					},
				},
			},
		},
		{
			Name: "test",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"},
					},
				},
			},
		},
		{
			Name: "lint",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "install"},
					},
				},
			},
		},
		{
			Name: "security_scan",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "lint"},
					},
				},
			},
		},
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "test"},
						{Name: "security_scan"},
					},
					Sequential: false, // parallel
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	order, err := resolver.ResolveDependencies("deploy")
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	// Verify execution order constraints
	installIdx := indexOf(order, "install")
	buildIdx := indexOf(order, "build")
	testIdx := indexOf(order, "test")
	lintIdx := indexOf(order, "lint")
	securityIdx := indexOf(order, "security_scan")
	deployIdx := indexOf(order, "deploy")

	// install must come before build and lint
	if installIdx >= buildIdx || installIdx >= lintIdx {
		t.Errorf("install should come before build and lint")
	}

	// build must come before test
	if buildIdx >= testIdx {
		t.Errorf("build should come before test")
	}

	// lint must come before security_scan
	if lintIdx >= securityIdx {
		t.Errorf("lint should come before security_scan")
	}

	// test and security_scan must come before deploy
	if testIdx >= deployIdx || securityIdx >= deployIdx {
		t.Errorf("test and security_scan should come before deploy")
	}

	// deploy should be last
	if deployIdx != len(order)-1 {
		t.Errorf("deploy should be the last task")
	}
}

func TestDependencyResolver_ExecutionPlan(t *testing.T) {
	tasks := []*ast.TaskStatement{
		{
			Name:         "build",
			Dependencies: []ast.DependencyGroup{},
		},
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"},
					},
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	plan, err := resolver.CreateExecutionPlan("deploy")
	if err != nil {
		t.Fatalf("CreateExecutionPlan failed: %v", err)
	}

	if plan.TaskName != "deploy" {
		t.Errorf("Expected plan task name 'deploy', got %s", plan.TaskName)
	}

	if plan.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", plan.TotalTasks)
	}

	if len(plan.Dependencies) != 2 {
		t.Fatalf("Expected 2 execution steps, got %d", len(plan.Dependencies))
	}

	// First step should be build (dependency)
	if plan.Dependencies[0].TaskName != "build" || plan.Dependencies[0].Type != "dependency" {
		t.Errorf("First step should be build dependency")
	}

	// Second step should be deploy (target)
	if plan.Dependencies[1].TaskName != "deploy" || plan.Dependencies[1].Type != "target" {
		t.Errorf("Second step should be deploy target")
	}
}

func TestDependencyResolver_ValidationSuccess(t *testing.T) {
	tasks := []*ast.TaskStatement{
		{
			Name:         "build",
			Dependencies: []ast.DependencyGroup{},
		},
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"},
					},
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	err := resolver.ValidateAllDependencies()
	if err != nil {
		t.Errorf("ValidateAllDependencies should succeed, got: %v", err)
	}
}

func TestDependencyResolver_ValidationFailure(t *testing.T) {
	tasks := []*ast.TaskStatement{
		{
			Name: "deploy",
			Dependencies: []ast.DependencyGroup{
				{
					Dependencies: []ast.DependencyItem{
						{Name: "build"}, // doesn't exist
					},
				},
			},
		},
	}

	resolver := NewDependencyResolver(tasks)
	err := resolver.ValidateAllDependencies()
	if err == nil {
		t.Errorf("ValidateAllDependencies should fail for missing dependency")
	}
}

// Helper functions
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
