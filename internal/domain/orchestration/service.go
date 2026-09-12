package orchestration

import (
	"time"
)

// Service represents a microservice definition
type Service struct {
	LastHealthCheck time.Time
	Environment     map[string]string
	Networks        map[string]*DockerNetwork
	EnvFile         *EnvFileConfig
	Repository      *Repository
	HealthCheck     *HealthCheck
	Build           *BuildConfig
	Compose         *ComposeConfig
	Name            string
	Description     string
	PreTask         string
	PostTask        string
	Status          ServiceStatus
	Container       string // for docker
	Path            string
	Dependencies    []string
	HealthyCount    int
	UnhealthyCount  int
}

// RegistryName returns the name this service is registered under.
func (s *Service) RegistryName() string { return s.Name }

// ServiceStatus represents the current status of a service
type ServiceStatus string

const (
	ServiceStatusUnknown   ServiceStatus = "unknown"
	ServiceStatusStarting  ServiceStatus = "starting"
	ServiceStatusRunning   ServiceStatus = "running"
	ServiceStatusHealthy   ServiceStatus = "healthy"
	ServiceStatusUnhealthy ServiceStatus = "unhealthy"
	ServiceStatusStopping  ServiceStatus = "stopping"
	ServiceStatusStopped   ServiceStatus = "stopped"
	ServiceStatusFailed    ServiceStatus = "failed"
)

// Repository represents Git repository configuration
type Repository struct {
	URL           string
	Branch        string
	Tag           string
	SSHKey        string
	Clone         bool
	UpdateOnStart bool
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Headers    map[string]string // for http
	RecordType string            // for dns (A, AAAA, etc)
	ExpectedIP string            // for dns
	Container  string
	Command    string // for custom
	WorkingDir string // for custom
	Endpoint   string // for http/tcp
	Domain     string // for dns
	Condition  string // for http (status code)

	// Runtime state
	Type        string   // http, tcp, docker, dns, custom
	ExpectedIPs []string // for dns with load balancer
	Retries     int
	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration // wait before first check
}

// BuildConfig represents build configuration
type BuildConfig struct {
	FallbackCommand  string
	Command          string
	Makefile         string
	MakeTarget       string
	WorkingDirectory string
	MakeArgs         []string
	PreMakeCommands  []string
	PostMakeCommands []string
	MakefileTimeout  time.Duration
	ParallelJobs     int
	MaxRetries       int
	RetryDelay       time.Duration
	Verbose          bool
	RetryOnFailure   bool
	Required         bool
	AllocateTTY      bool
}

// ComposeConfig represents Docker Compose configuration
type ComposeConfig struct {
	Options *ComposeOptions
	File    string
	Project string
}

// ComposeOptions represents Docker Compose command options
type ComposeOptions struct {
	CPULimit      string
	MemoryLimit   string
	Pull          string // always, missing, never
	Scale         string
	RestartPolicy string
	WaitTimeout   time.Duration
	Timeout       time.Duration
	Wait          bool
	Detach        bool
	RemoveOrphans bool
	ForceRecreate bool
	Build         bool
	NoDeps        bool
}

// EnvFileConfig represents environment file configuration
type EnvFileConfig struct {
	Task     string // Task to call before service start
	Required bool
}

// IsHealthy returns true if the service is healthy
func (s *Service) IsHealthy() bool {
	return s.Status == ServiceStatusHealthy
}

// IsRunning returns true if the service is running
func (s *Service) IsRunning() bool {
	return s.Status == ServiceStatusRunning || s.Status == ServiceStatusHealthy
}

// IsFailed returns true if the service has failed
func (s *Service) IsFailed() bool {
	return s.Status == ServiceStatusFailed || s.Status == ServiceStatusUnhealthy
}

// MarkHealthy marks the service as healthy
func (s *Service) MarkHealthy() {
	s.Status = ServiceStatusHealthy
	s.HealthyCount++
	s.UnhealthyCount = 0
	s.LastHealthCheck = time.Now()
}

// MarkUnhealthy marks the service as unhealthy
func (s *Service) MarkUnhealthy() {
	s.Status = ServiceStatusUnhealthy
	s.UnhealthyCount++
	s.HealthyCount = 0
	s.LastHealthCheck = time.Now()
}

// MarkStarting marks the service as starting
func (s *Service) MarkStarting() {
	s.Status = ServiceStatusStarting
}

// MarkStopped marks the service as stopped
func (s *Service) MarkStopped() {
	s.Status = ServiceStatusStopped
}

// MarkFailed marks the service as failed
func (s *Service) MarkFailed() {
	s.Status = ServiceStatusFailed
}

// DockerNetwork represents Docker network configuration
type DockerNetwork struct {
	Options       map[string]string
	Name          string
	Driver        string
	External      bool
	Required      bool
	AutoProvision bool
}
