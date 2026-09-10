package orchestration

import (
	"fmt"
	"sync"
)

// RegistryEntry is implemented by the types a Registry stores. The accessor is
// separate from each type's Name field because Go constraints cannot require a
// struct field.
type RegistryEntry interface {
	RegistryName() string
}

// Registry maintains a named set of entries under a read/write lock.
//
// Construct one with NewRegistry; the zero value is not usable.
type Registry[T RegistryEntry] struct {
	entries map[string]T
	kind    string
	mu      sync.RWMutex
}

// NewRegistry creates an empty registry. The kind names the entries in error
// messages, for example "service" or "orchestration".
func NewRegistry[T RegistryEntry](kind string) *Registry[T] {
	return &Registry[T]{
		entries: make(map[string]T),
		kind:    kind,
	}
}

// Register adds an entry, rejecting a duplicate name.
func (r *Registry[T]) Register(entry T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := entry.RegistryName()
	if _, exists := r.entries[name]; exists {
		return fmt.Errorf("%s %s already registered", r.kind, name)
	}

	r.entries[name] = entry
	return nil
}

// Get retrieves an entry by name.
func (r *Registry[T]) Get(name string) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	if !exists {
		var zero T
		return zero, fmt.Errorf("%s %s not found", r.kind, name)
	}

	return entry, nil
}

// GetAll returns all registered entries.
func (r *Registry[T]) GetAll() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]T, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}

	return entries
}

// Update replaces an existing entry, rejecting an unknown name.
func (r *Registry[T]) Update(entry T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := entry.RegistryName()
	if _, exists := r.entries[name]; !exists {
		return fmt.Errorf("%s %s not found", r.kind, name)
	}

	r.entries[name] = entry
	return nil
}

// Delete removes an entry, rejecting an unknown name.
func (r *Registry[T]) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[name]; !exists {
		return fmt.Errorf("%s %s not found", r.kind, name)
	}

	delete(r.entries, name)
	return nil
}

// Exists reports whether an entry with the given name is registered.
func (r *Registry[T]) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.entries[name]
	return exists
}

// ServiceRegistry maintains the registry of all services
type ServiceRegistry = Registry[*Service]

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return NewRegistry[*Service]("service")
}

// OrchestrationRegistry maintains the registry of all orchestrations
type OrchestrationRegistry = Registry[*Orchestration]

// NewOrchestrationRegistry creates a new orchestration registry
func NewOrchestrationRegistry() *OrchestrationRegistry {
	return NewRegistry[*Orchestration]("orchestration")
}
