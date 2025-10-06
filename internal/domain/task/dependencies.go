package task

import (
	"fmt"
)

// DependencyResolver resolves task dependencies
type DependencyResolver struct {
	registry *Registry
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(registry *Registry) *DependencyResolver {
	return &DependencyResolver{
		registry: registry,
	}
}

// Resolve resolves dependencies for a task
// Returns tasks in execution order
func (dr *DependencyResolver) Resolve(taskName string) ([]*Task, error) {
	// Get the task
	task, err := dr.registry.Get(taskName)
	if err != nil {
		return nil, err
	}

	// Check for circular dependencies
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	if err := dr.checkCircular(task, visited, inStack); err != nil {
		return nil, err
	}

	// Topological sort
	sorted := make([]*Task, 0)
	visitedSort := make(map[string]bool)

	if err := dr.topologicalSort(task, visitedSort, &sorted); err != nil {
		return nil, err
	}

	return sorted, nil
}

// checkCircular checks for circular dependencies
func (dr *DependencyResolver) checkCircular(task *Task, visited, inStack map[string]bool) error {
	visited[task.Name] = true
	inStack[task.Name] = true

	for _, dep := range task.Dependencies {
		if !visited[dep.Name] {
			depTask, err := dr.registry.Get(dep.Name)
			if err != nil {
				return &TaskError{
					Task:    task.Name,
					Message: fmt.Sprintf("dependency '%s' not found", dep.Name),
					Cause:   err,
				}
			}

			if err := dr.checkCircular(depTask, visited, inStack); err != nil {
				return err
			}
		} else if inStack[dep.Name] {
			return &TaskError{
				Task:    task.Name,
				Message: fmt.Sprintf("circular dependency detected: %s -> %s", task.Name, dep.Name),
			}
		}
	}

	inStack[task.Name] = false
	return nil
}

// topologicalSort performs topological sort on dependencies
func (dr *DependencyResolver) topologicalSort(task *Task, visited map[string]bool, sorted *[]*Task) error {
	visited[task.Name] = true

	for _, dep := range task.Dependencies {
		if !visited[dep.Name] {
			depTask, err := dr.registry.Get(dep.Name)
			if err != nil {
				return err
			}

			if err := dr.topologicalSort(depTask, visited, sorted); err != nil {
				return err
			}
		}
	}

	*sorted = append(*sorted, task)
	return nil
}

// GetParallelGroups groups dependencies that can run in parallel
func (dr *DependencyResolver) GetParallelGroups(task *Task) ([][]Dependency, error) {
	var groups [][]Dependency
	var currentGroup []Dependency

	for _, dep := range task.Dependencies {
		// Verify dependency exists
		if !dr.registry.Exists(dep.Name) {
			return nil, &TaskError{
				Task:    task.Name,
				Message: fmt.Sprintf("dependency '%s' not found", dep.Name),
			}
		}

		if dep.Parallel && !dep.Sequential {
			// Add to current parallel group
			currentGroup = append(currentGroup, dep)
		} else {
			// Start new group if there's a current group
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = nil
			}
			// Add sequential dependency as its own group
			groups = append(groups, []Dependency{dep})
		}
	}

	// Add remaining parallel group
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups, nil
}
