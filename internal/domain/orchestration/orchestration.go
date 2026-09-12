package orchestration

import (
	"sync"
	"time"
)

// Orchestration represents an orchestration group
type Orchestration struct {
	LastFailure         time.Time
	Scale               map[string]int
	ContainerManagement *ContainerManagement
	Recovery            *RecoveryConfig
	Discovery           *DiscoveryConfig
	Metrics             *MetricsConfig
	UpdateStrategy      UpdateStrategy

	// Runtime state
	Status              OrchestrationStatus
	Name                string
	PostTask            string
	PreTask             string
	Description         string
	Strategy            OrchestrationStrategy
	Services            []string
	MakefileOrder       []string
	CloneOrder          []string
	StartupTimeout      time.Duration
	MaxUnavailable      int
	MakefileTimeout     time.Duration
	RecoveryTimeout     time.Duration
	FailureThreshold    int
	ShutdownTimeout     time.Duration
	HealthCheckInterval time.Duration
	CloneTimeout        time.Duration
	UpdateTimeout       time.Duration
	FailureCount        int
	mu                  sync.RWMutex
	StopOnFailure       bool
	CircuitBreaker      bool
	CircuitOpen         bool
}

// RegistryName returns the name this orchestration is registered under.
func (o *Orchestration) RegistryName() string { return o.Name }

// OrchestrationStatus represents the status of an orchestration group
type OrchestrationStatus string

const (
	OrchestrationStatusUnknown  OrchestrationStatus = "unknown"
	OrchestrationStatusStarting OrchestrationStatus = "starting"
	OrchestrationStatusRunning  OrchestrationStatus = "running"
	OrchestrationStatusHealthy  OrchestrationStatus = "healthy"
	OrchestrationStatusDegraded OrchestrationStatus = "degraded"
	OrchestrationStatusStopping OrchestrationStatus = "stopping"
	OrchestrationStatusStopped  OrchestrationStatus = "stopped"
	OrchestrationStatusFailed   OrchestrationStatus = "failed"
)

// OrchestrationStrategy represents the strategy for starting services
type OrchestrationStrategy string

const (
	StrategySequential      OrchestrationStrategy = "sequential"
	StrategyParallel        OrchestrationStrategy = "parallel"
	StrategyDependencyBased OrchestrationStrategy = "dependency-based"
)

// UpdateStrategy represents the strategy for updating services
type UpdateStrategy string

const (
	UpdateStrategyRolling   UpdateStrategy = "rolling"
	UpdateStrategyRecreate  UpdateStrategy = "recreate"
	UpdateStrategyBlueGreen UpdateStrategy = "blue-green"
)

// ContainerManagement represents container management options
type ContainerManagement struct {
	PullPolicy             string
	HealthCheckTimeout     time.Duration
	ForceRecreateOnStart   bool
	ForceRecreateOnRestart bool
	BuildBeforeStart       bool
	WaitForHealth          bool
}

// RecoveryConfig represents recovery configuration
type RecoveryConfig struct {
	FallbackAction     RecoveryAction
	MaxRetries         int
	RetryInterval      time.Duration
	ExponentialBackoff bool
}

// RecoveryAction represents the action to take on failure
type RecoveryAction string

const (
	RecoveryActionRestart RecoveryAction = "restart"
	RecoveryActionStop    RecoveryAction = "stop"
	RecoveryActionIgnore  RecoveryAction = "ignore"
)

// DiscoveryConfig represents service discovery configuration
type DiscoveryConfig struct {
	Type          string // consul, etcd, kubernetes, dns
	Endpoint      string
	Namespace     string
	DNSServer     string
	SearchDomains []string
	TTLCheck      bool
	CacheTimeout  time.Duration
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	Labels   map[string]string
	Endpoint string
	Interval time.Duration
	Enabled  bool
}

// IsHealthy returns true if all services are healthy
func (o *Orchestration) IsHealthy() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.Status == OrchestrationStatusHealthy
}

// IsRunning returns true if the orchestration is running
func (o *Orchestration) IsRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.Status == OrchestrationStatusRunning || o.Status == OrchestrationStatusHealthy || o.Status == OrchestrationStatusDegraded
}

// IsFailed returns true if the orchestration has failed
func (o *Orchestration) IsFailed() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.Status == OrchestrationStatusFailed
}

// MarkHealthy marks the orchestration as healthy
func (o *Orchestration) MarkHealthy() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Status = OrchestrationStatusHealthy
	o.FailureCount = 0
	o.CircuitOpen = false
}

// MarkDegraded marks the orchestration as degraded
func (o *Orchestration) MarkDegraded() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Status = OrchestrationStatusDegraded
}

// MarkFailed marks the orchestration as failed
func (o *Orchestration) MarkFailed() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Status = OrchestrationStatusFailed
	o.FailureCount++
	o.LastFailure = time.Now()

	// Check if we should open the circuit breaker
	if o.CircuitBreaker && o.FailureCount >= o.FailureThreshold {
		o.CircuitOpen = true
	}
}

// MarkStarting marks the orchestration as starting
func (o *Orchestration) MarkStarting() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Status = OrchestrationStatusStarting
}

// MarkStopped marks the orchestration as stopped
func (o *Orchestration) MarkStopped() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Status = OrchestrationStatusStopped
}

// ShouldRecover returns true if the orchestration should attempt recovery
func (o *Orchestration) ShouldRecover() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.CircuitOpen {
		return true
	}

	// Check if enough time has passed for recovery
	if time.Since(o.LastFailure) >= o.RecoveryTimeout {
		return true
	}

	return false
}

// ResetCircuitBreaker resets the circuit breaker
func (o *Orchestration) ResetCircuitBreaker() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.CircuitOpen = false
	o.FailureCount = 0
}
