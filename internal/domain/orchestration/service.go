package orchestration

import (
	"time"
)

// Service represents a microservice definition
type Service struct {
	Name         string
	Path         string
	Description  string
	Dependencies []string
	Repository   *Repository
	HealthCheck  *HealthCheck
	Build        *BuildConfig
	Compose      *ComposeConfig
	Environment  map[string]string
	EnvFile      *EnvFileConfig
	PreTask      string
	PostTask     string

	// Runtime state
	Status          ServiceStatus
	Container       string
	LastHealthCheck time.Time
	HealthyCount    int
	UnhealthyCount  int
}

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
	URL            string
	Branch         string
	Tag            string
	SSHKey         string
	CloneIfMissing bool
	UpdateOnStart  bool
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Type        string // http, tcp, docker, dns, custom
	Endpoint    string // for http/tcp
	Domain      string // for dns
	Container   string // for docker
	Command     string // for custom
	Timeout     time.Duration
	Interval    time.Duration
	Retries     int
	Condition   string            // for http (status code)
	RecordType  string            // for dns (A, AAAA, etc)
	ExpectedIP  string            // for dns
	ExpectedIPs []string          // for dns with load balancer
	Headers     map[string]string // for http
	WorkingDir  string            // for custom
	StartPeriod time.Duration     // wait before first check
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Required         bool
	Command          string
	Makefile         string
	MakeTarget       string
	MakeArgs         []string
	PreMakeCommands  []string
	PostMakeCommands []string
	WorkingDirectory string
	MakefileTimeout  time.Duration
	ParallelJobs     int
	Verbose          bool
	RetryOnFailure   bool
	MaxRetries       int
	RetryDelay       time.Duration
	FallbackCommand  string
}

// ComposeConfig represents Docker Compose configuration
type ComposeConfig struct {
	File    string
	Project string
	Options *ComposeOptions
}

// ComposeOptions represents Docker Compose command options
type ComposeOptions struct {
	ForceRecreate bool
	NoDeps        bool
	Build         bool
	Pull          string // always, missing, never
	Timeout       time.Duration
	Scale         string
	Wait          bool
	WaitTimeout   time.Duration
	Detach        bool
	RemoveOrphans bool
	RestartPolicy string
	MemoryLimit   string
	CPULimit      string
}

// EnvFileConfig represents environment file configuration
type EnvFileConfig struct {
	Required bool
	Task     string // Task to call before service start
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
