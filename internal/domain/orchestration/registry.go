package orchestration

import (
	"fmt"
	"sync"
)

// ServiceRegistry maintains the registry of all services
type ServiceRegistry struct {
	services map[string]*Service
	mu       sync.RWMutex
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*Service),
	}
}

// Register registers a service
func (r *ServiceRegistry) Register(service *Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[service.Name]; exists {
		return fmt.Errorf("service %s already registered", service.Name)
	}

	r.services[service.Name] = service
	return nil
}

// Get retrieves a service by name
func (r *ServiceRegistry) Get(name string) (*Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// GetAll returns all registered services
func (r *ServiceRegistry) GetAll() []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*Service, 0, len(r.services))
	for _, service := range r.services {
		services = append(services, service)
	}

	return services
}

// Update updates a service
func (r *ServiceRegistry) Update(service *Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[service.Name]; !exists {
		return fmt.Errorf("service %s not found", service.Name)
	}

	r.services[service.Name] = service
	return nil
}

// Delete removes a service from the registry
func (r *ServiceRegistry) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; !exists {
		return fmt.Errorf("service %s not found", name)
	}

	delete(r.services, name)
	return nil
}

// Exists checks if a service exists
func (r *ServiceRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.services[name]
	return exists
}

// OrchestrationRegistry maintains the registry of all orchestrations
type OrchestrationRegistry struct {
	orchestrations map[string]*Orchestration
	mu             sync.RWMutex
}

// NewOrchestrationRegistry creates a new orchestration registry
func NewOrchestrationRegistry() *OrchestrationRegistry {
	return &OrchestrationRegistry{
		orchestrations: make(map[string]*Orchestration),
	}
}

// Register registers an orchestration
func (r *OrchestrationRegistry) Register(orchestration *Orchestration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orchestrations[orchestration.Name]; exists {
		return fmt.Errorf("orchestration %s already registered", orchestration.Name)
	}

	r.orchestrations[orchestration.Name] = orchestration
	return nil
}

// Get retrieves an orchestration by name
func (r *OrchestrationRegistry) Get(name string) (*Orchestration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orchestration, exists := r.orchestrations[name]
	if !exists {
		return nil, fmt.Errorf("orchestration %s not found", name)
	}

	return orchestration, nil
}

// GetAll returns all registered orchestrations
func (r *OrchestrationRegistry) GetAll() []*Orchestration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orchestrations := make([]*Orchestration, 0, len(r.orchestrations))
	for _, orchestration := range r.orchestrations {
		orchestrations = append(orchestrations, orchestration)
	}

	return orchestrations
}

// Update updates an orchestration
func (r *OrchestrationRegistry) Update(orchestration *Orchestration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orchestrations[orchestration.Name]; !exists {
		return fmt.Errorf("orchestration %s not found", orchestration.Name)
	}

	r.orchestrations[orchestration.Name] = orchestration
	return nil
}

// Delete removes an orchestration from the registry
func (r *OrchestrationRegistry) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.orchestrations[name]; !exists {
		return fmt.Errorf("orchestration %s not found", name)
	}

	delete(r.orchestrations, name)
	return nil
}

// Exists checks if an orchestration exists
func (r *OrchestrationRegistry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.orchestrations[name]
	return exists
}
