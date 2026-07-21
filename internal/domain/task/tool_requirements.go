package task

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

// ResolveInheritedToolRequirements flattens every requires-tools task reference
// in registered tasks so execution only sees concrete tool requirements.
func ResolveInheritedToolRequirements(registry *Registry) error {
	resolver := toolRequirementResolver{
		registry: registry,
		resolved: make(map[string][]statement.ToolRequirement),
		stack:    make(map[string]bool),
	}

	for _, task := range registry.List() {
		if _, err := resolver.resolveTask(task); err != nil {
			return err
		}
	}

	return nil
}

// ResolveInheritedProjectToolRequirements flattens project-level requires-tools
// references against the registered task set.
func ResolveInheritedProjectToolRequirements(registry *Registry, refs []string) ([]statement.ToolRequirement, error) {
	resolver := toolRequirementResolver{
		registry: registry,
		resolved: make(map[string][]statement.ToolRequirement),
		stack:    make(map[string]bool),
	}

	var tools []statement.ToolRequirement
	for _, ref := range refs {
		refTask, err := registry.Get(ref)
		if err != nil {
			return nil, fmt.Errorf("requires tools from task %q not found: %w", ref, err)
		}

		inherited, err := resolver.resolveTask(refTask)
		if err != nil {
			return nil, err
		}
		tools = append(tools, inherited...)
	}

	return tools, nil
}

type toolRequirementResolver struct {
	registry *Registry
	resolved map[string][]statement.ToolRequirement
	stack    map[string]bool
}

func (r *toolRequirementResolver) resolveTask(task *Task) ([]statement.ToolRequirement, error) {
	key := toolRequirementTaskKey(task)
	if tools, ok := r.resolved[key]; ok {
		return tools, nil
	}
	if r.stack[key] {
		return nil, &TaskError{
			Task:    task.FullName(),
			Message: fmt.Sprintf("circular requires-tools inheritance detected at task %q", task.FullName()),
		}
	}

	r.stack[key] = true
	defer delete(r.stack, key)

	var taskTools []statement.ToolRequirement
	for _, stmt := range task.Body {
		requiresTools, ok := stmt.(*statement.RequiresTools)
		if !ok {
			continue
		}

		flattened := append([]statement.ToolRequirement(nil), requiresTools.Tools...)
		for _, ref := range requiresTools.TaskRefs {
			refTask, err := r.registry.Get(ref)
			if err != nil {
				return nil, &TaskError{
					Task:    task.FullName(),
					Message: fmt.Sprintf("requires tools from task %q not found", ref),
					Cause:   err,
				}
			}

			inherited, err := r.resolveTask(refTask)
			if err != nil {
				return nil, err
			}
			flattened = append(flattened, inherited...)
		}

		requiresTools.Tools = flattened
		requiresTools.TaskRefs = nil
		taskTools = append(taskTools, flattened...)
	}

	r.resolved[key] = taskTools
	return taskTools, nil
}

func toolRequirementTaskKey(task *Task) string {
	return task.FullName() + "\x00" + task.PlatformLabel()
}
