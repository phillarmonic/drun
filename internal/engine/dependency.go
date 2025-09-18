package engine

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
)

// DependencyResolver handles task dependency resolution and execution ordering
type DependencyResolver struct {
	tasks map[string]*ast.TaskStatement // task name -> task
	graph map[string][]string           // task name -> list of dependencies
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(tasks []*ast.TaskStatement) *DependencyResolver {
	resolver := &DependencyResolver{
		tasks: make(map[string]*ast.TaskStatement),
		graph: make(map[string][]string),
	}

	// Build task map and dependency graph
	for _, task := range tasks {
		resolver.tasks[task.Name] = task
		resolver.graph[task.Name] = []string{}

		// Extract dependencies from all dependency groups
		for _, depGroup := range task.Dependencies {
			for _, dep := range depGroup.Dependencies {
				resolver.graph[task.Name] = append(resolver.graph[task.Name], dep.Name)
			}
		}
	}

	return resolver
}

// ResolveDependencies calculates the execution order for a target task
func (dr *DependencyResolver) ResolveDependencies(targetTask string) ([]string, error) {
	// Check if target task exists
	if _, exists := dr.tasks[targetTask]; !exists {
		return nil, fmt.Errorf("task '%s' not found", targetTask)
	}

	// Use a different approach: collect all tasks needed, then sort them
	needed := make(map[string]bool)
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	// First, collect all tasks that need to be executed
	var collectTasks func(string) error
	collectTasks = func(taskName string) error {
		if recursionStack[taskName] {
			return fmt.Errorf("circular dependency detected involving task '%s'", taskName)
		}

		if visited[taskName] {
			return nil
		}

		visited[taskName] = true
		recursionStack[taskName] = true
		needed[taskName] = true

		// Visit all dependencies
		for _, dep := range dr.graph[taskName] {
			if _, exists := dr.tasks[dep]; !exists {
				return fmt.Errorf("dependency task '%s' not found (required by '%s')", dep, taskName)
			}
			if err := collectTasks(dep); err != nil {
				return err
			}
		}

		recursionStack[taskName] = false
		return nil
	}

	// Collect all needed tasks
	if err := collectTasks(targetTask); err != nil {
		return nil, err
	}

	// Now perform topological sort on needed tasks
	visited = make(map[string]bool)
	var result []string

	var topSort func(string) error
	topSort = func(taskName string) error {
		if visited[taskName] {
			return nil
		}

		visited[taskName] = true

		// Visit dependencies first
		for _, dep := range dr.graph[taskName] {
			if needed[dep] { // only visit if it's needed
				if err := topSort(dep); err != nil {
					return err
				}
			}
		}

		// Add current task after its dependencies
		result = append(result, taskName)
		return nil
	}

	// Sort all needed tasks
	for taskName := range needed {
		if err := topSort(taskName); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// GetDependencyGroups returns the dependency groups for a task
func (dr *DependencyResolver) GetDependencyGroups(taskName string) []ast.DependencyGroup {
	if task, exists := dr.tasks[taskName]; exists {
		return task.Dependencies
	}
	return []ast.DependencyGroup{}
}

// GetDirectDependencies returns the immediate dependencies of a task
func (dr *DependencyResolver) GetDirectDependencies(taskName string) []string {
	if deps, exists := dr.graph[taskName]; exists {
		// Return a copy to avoid modification
		result := make([]string, len(deps))
		copy(result, deps)
		return result
	}
	return []string{}
}

// ValidateAllDependencies validates all task dependencies in the program
func (dr *DependencyResolver) ValidateAllDependencies() error {
	// Check that all referenced dependencies exist
	for taskName, deps := range dr.graph {
		for _, dep := range deps {
			if _, exists := dr.tasks[dep]; !exists {
				return fmt.Errorf("task '%s' depends on non-existent task '%s'", taskName, dep)
			}
		}
	}

	// Check for circular dependencies by trying to resolve each task
	for taskName := range dr.tasks {
		if _, err := dr.ResolveDependencies(taskName); err != nil {
			return err
		}
	}

	return nil
}

// GetExecutionPlan creates a detailed execution plan for a task
type ExecutionPlan struct {
	TaskName     string          // the target task
	Dependencies []ExecutionStep // ordered list of dependency steps
	TotalTasks   int             // total number of tasks to execute
}

type ExecutionStep struct {
	TaskName   string   // task to execute
	Type       string   // "dependency" or "target"
	Sequential bool     // whether this step must be sequential
	Parallel   []string // tasks that can run in parallel (if any)
}

// CreateExecutionPlan creates a detailed execution plan for a task
func (dr *DependencyResolver) CreateExecutionPlan(targetTask string) (*ExecutionPlan, error) {
	// Get execution order
	order, err := dr.ResolveDependencies(targetTask)
	if err != nil {
		return nil, err
	}

	plan := &ExecutionPlan{
		TaskName:     targetTask,
		Dependencies: []ExecutionStep{},
		TotalTasks:   len(order),
	}

	// Create execution steps
	for _, taskName := range order {
		stepType := "dependency"
		if taskName == targetTask {
			stepType = "target"
		}

		step := ExecutionStep{
			TaskName:   taskName,
			Type:       stepType,
			Sequential: true, // default to sequential
			Parallel:   []string{},
		}

		// Analyze dependency groups to determine if parallel execution is possible
		task := dr.tasks[taskName]
		for _, depGroup := range task.Dependencies {
			if !depGroup.Sequential && len(depGroup.Dependencies) > 1 {
				// This is a parallel dependency group
				step.Sequential = false
				for _, dep := range depGroup.Dependencies {
					step.Parallel = append(step.Parallel, dep.Name)
				}
			}
		}

		plan.Dependencies = append(plan.Dependencies, step)
	}

	return plan, nil
}

// PrintExecutionPlan prints a human-readable execution plan
func (plan *ExecutionPlan) String() string {
	result := fmt.Sprintf("Execution Plan for '%s' (%d tasks):\n", plan.TaskName, plan.TotalTasks)

	for i, step := range plan.Dependencies {
		prefix := fmt.Sprintf("%d. ", i+1)
		if step.Type == "target" {
			result += fmt.Sprintf("%s🎯 %s (target task)\n", prefix, step.TaskName)
		} else {
			if step.Sequential {
				result += fmt.Sprintf("%s📋 %s (dependency)\n", prefix, step.TaskName)
			} else {
				result += fmt.Sprintf("%s⚡ %s (parallel dependencies: %v)\n", prefix, step.TaskName, step.Parallel)
			}
		}
	}

	return result
}
