package task

import (
	"fmt"
	"strings"
	"sync"

	"github.com/phillarmonic/drun/v2/internal/platform"
)

// Registry manages task registration and lookup
type Registry struct {
	mu              sync.RWMutex
	tasks           map[string][]*Task // task name -> variants
	namespacedTasks map[string][]*Task // namespace.name -> variants
	taskOrder       []*Task            // preserve insertion order
	currentPlatform string
}

// NewRegistry creates a new task registry
func NewRegistry() *Registry {
	return &Registry{
		tasks:           make(map[string][]*Task),
		namespacedTasks: make(map[string][]*Task),
		taskOrder:       make([]*Task, 0),
		currentPlatform: platform.Current(),
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
	family := r.tasks[task.Name]
	if err := validateTaskVariantFamily(task.Name, family, task); err != nil {
		return err
	}

	var fullName string
	if task.Namespace != "" {
		fullName = task.FullName()
		family := r.namespacedTasks[fullName]
		if err := validateTaskVariantFamily(fullName, family, task); err != nil {
			return err
		}
	}

	// Register by name
	r.tasks[task.Name] = append(r.tasks[task.Name], task)
	r.taskOrder = append(r.taskOrder, task)

	// Register by full name if namespaced
	if fullName != "" {
		r.namespacedTasks[fullName] = append(r.namespacedTasks[fullName], task)
	}

	return nil
}

// RegisterNamespaced registers a task only by its fully qualified namespace.
func (r *Registry) RegisterNamespaced(task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task.Namespace == "" {
		return fmt.Errorf("namespaced task %q must have a namespace", task.Name)
	}
	if err := task.Validate(); err != nil {
		return err
	}

	fullName := task.FullName()
	family := r.namespacedTasks[fullName]
	if err := validateTaskVariantFamily(fullName, family, task); err != nil {
		return err
	}

	r.namespacedTasks[fullName] = append(r.namespacedTasks[fullName], task)
	r.taskOrder = append(r.taskOrder, task)

	return nil
}

// Get retrieves a task by name
func (r *Registry) Get(name string) (*Task, error) {
	return r.GetForPlatform(name, r.currentPlatform)
}

func (r *Registry) GetForPlatform(name, targetPlatform string) (*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try direct lookup first
	if tasks, exists := r.tasks[name]; exists {
		return resolveTaskVariant(name, tasks, targetPlatform)
	}

	// Try namespaced lookup
	if tasks, exists := r.namespacedTasks[name]; exists {
		return resolveTaskVariant(name, tasks, targetPlatform)
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

	return append([]*Task(nil), r.taskOrder...)
}

// ListByNamespace returns tasks in a specific namespace
func (r *Registry) ListByNamespace(namespace string) []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*Task
	for _, task := range r.taskOrder {
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

	r.tasks = make(map[string][]*Task)
	r.namespacedTasks = make(map[string][]*Task)
	r.taskOrder = make([]*Task, 0)
}

// Count returns the number of registered tasks
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.taskOrder)
}

func (r *Registry) SetCurrentPlatform(targetPlatform string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if normalized, err := platform.Normalize(targetPlatform); err == nil {
		r.currentPlatform = normalized
		return
	}
	r.currentPlatform = targetPlatform
}

func validateTaskVariantFamily(name string, family []*Task, candidate *Task) error {
	if len(family) == 0 {
		return nil
	}

	hasFallback := len(candidate.Platforms) == 0
	for _, existing := range family {
		if len(existing.Platforms) == 0 {
			if hasFallback {
				return fmt.Errorf("task %q may only declare one unannotated fallback variant", name)
			}
			continue
		}
		if hasFallback {
			continue
		}

		for _, existingPlatform := range existing.Platforms {
			for _, candidatePlatform := range candidate.Platforms {
				if existingPlatform == candidatePlatform {
					return fmt.Errorf("task %q has overlapping platform variants for %s", name, candidatePlatform)
				}
			}
		}
	}

	return nil
}

func resolveTaskVariant(name string, family []*Task, targetPlatform string) (*Task, error) {
	var fallback *Task

	for _, candidate := range family {
		if len(candidate.Platforms) == 0 {
			if fallback == nil {
				fallback = candidate
			}
			continue
		}
		for _, allowed := range candidate.Platforms {
			if allowed == targetPlatform {
				return candidate, nil
			}
		}
	}

	if fallback != nil {
		return fallback, nil
	}

	available := make([]string, 0, len(family))
	for _, candidate := range family {
		label := candidate.PlatformLabel()
		if label != "" {
			available = append(available, label)
		}
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("task %q not found", name)
	}
	return nil, fmt.Errorf("task %q has no variant for platform %s; available variants: %s", name, targetPlatform, joinUnique(available))
}

func joinUnique(values []string) string {
	seen := make(map[string]struct{}, len(values))
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		filtered = append(filtered, value)
	}
	return strings.Join(filtered, "; ")
}
