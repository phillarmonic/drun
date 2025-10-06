package task

import (
	"fmt"
	"sync"
)

// Registry manages task registration and lookup
type Registry struct {
	mu              sync.RWMutex
	tasks           map[string]*Task // task name -> task
	namespacedTasks map[string]*Task // namespace.name -> task
	taskOrder       []string         // preserve insertion order
}

// NewRegistry creates a new task registry
func NewRegistry() *Registry {
	return &Registry{
		tasks:           make(map[string]*Task),
		namespacedTasks: make(map[string]*Task),
		taskOrder:       make([]string, 0),
	}
}

// Register registers a task
func (r *Registry) Register(task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate task
	if err := task.Validate(); err != nil {
		return err
	}

	// Check for duplicates
	if _, exists := r.tasks[task.Name]; exists {
		return fmt.Errorf("task '%s' already registered", task.Name)
	}

	// Register by name
	r.tasks[task.Name] = task
	r.taskOrder = append(r.taskOrder, task.Name) // preserve order

	// Register by full name if namespaced
	if task.Namespace != "" {
		fullName := task.FullName()
		r.namespacedTasks[fullName] = task
	}

	return nil
}

// Get retrieves a task by name
func (r *Registry) Get(name string) (*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try direct lookup first
	if task, exists := r.tasks[name]; exists {
		return task, nil
	}

	// Try namespaced lookup
	if task, exists := r.namespacedTasks[name]; exists {
		return task, nil
	}

	return nil, fmt.Errorf("task '%s' not found", name)
}

// Exists checks if a task exists
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, direct := r.tasks[name]
	_, namespaced := r.namespacedTasks[name]
	return direct || namespaced
}

// List returns all registered tasks in insertion order
func (r *Registry) List() []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*Task, 0, len(r.taskOrder))
	for _, name := range r.taskOrder {
		if task, exists := r.tasks[name]; exists {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// ListByNamespace returns tasks in a specific namespace
func (r *Registry) ListByNamespace(namespace string) []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*Task
	for _, task := range r.tasks {
		if task.Namespace == namespace {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// Clear clears all registered tasks
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks = make(map[string]*Task)
	r.namespacedTasks = make(map[string]*Task)
	r.taskOrder = make([]string, 0)
}

// Count returns the number of registered tasks
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tasks)
}
