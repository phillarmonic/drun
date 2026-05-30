package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// OrchestrationActionStatement represents orchestration actions in task bodies
// Examples: orchestrate "group" start, orchestrate "group" stop
type OrchestrationActionStatement struct {
	Token          lexer.Token
	GroupName      string
	Action         string // start, stop, restart, health_check, status, logs, etc.
	Options        map[string]string
	ServiceFilters []string // optional: specific services to act on
}

func (oas *OrchestrationActionStatement) statementNode() {}
func (oas *OrchestrationActionStatement) String() string {
	out := fmt.Sprintf("orchestrate \"%s\" %s", oas.GroupName, oas.Action)
	if len(oas.ServiceFilters) > 0 {
		out += fmt.Sprintf(" services %v", oas.ServiceFilters)
	}
	for key, value := range oas.Options {
		out += fmt.Sprintf(" %s \"%s\"", key, value)
	}
	return out
}

// ServiceStatement represents a microservice definition
type ServiceStatement struct {
	Token        lexer.Token
	Name         string
	Path         string
	Description  string
	Dependencies []string
	Repository   *RepositoryConfig
	HealthCheck  *HealthCheckConfig
	Build        *BuildConfig
	Compose      *ComposeConfig
	Environment  map[string]string
	EnvFile      *EnvFileConfig
	Networks     map[string]*DockerNetworkConfig
	PreTask      string
	PostTask     string
}

func (ss *ServiceStatement) statementNode()      {}
func (ss *ServiceStatement) projectSettingNode() {}
func (ss *ServiceStatement) String() string {
	var out strings.Builder
	fmt.Fprintf(&out, "service \"%s\" in \"%s\"", ss.Name, ss.Path)
	if ss.Description != "" {
		fmt.Fprintf(&out, " means \"%s\"", ss.Description)
	}
	out.WriteString(":\n")

	if len(ss.Dependencies) > 0 {
		fmt.Fprintf(&out, "    depends on %v\n", ss.Dependencies)
	}

	if ss.Repository != nil {
		fmt.Fprintf(&out, "    %s\n", ss.Repository.String())
	}

	if ss.HealthCheck != nil {
		fmt.Fprintf(&out, "    %s\n", ss.HealthCheck.String())
	}

	if ss.Build != nil {
		fmt.Fprintf(&out, "    %s\n", ss.Build.String())
	}

	if ss.Compose != nil {
		fmt.Fprintf(&out, "    %s\n", ss.Compose.String())
	}

	if len(ss.Environment) > 0 {
		out.WriteString("    environment:\n")
		for k, v := range ss.Environment {
			fmt.Fprintf(&out, "        %s \"%s\"\n", k, v)
		}
	}

	if ss.EnvFile != nil {
		fmt.Fprintf(&out, "    %s\n", ss.EnvFile.String())
	}

	if ss.PreTask != "" {
		fmt.Fprintf(&out, "    pre_task \"%s\"\n", ss.PreTask)
	}

	if ss.PostTask != "" {
		fmt.Fprintf(&out, "    post_task \"%s\"\n", ss.PostTask)
	}

	return out.String()
}

// RepositoryConfig represents Git repository configuration
type RepositoryConfig struct {
	URL           string
	Branch        string
	Tag           string
	SSHKey        string
	Clone         bool
	UpdateOnStart bool
}

func (rc *RepositoryConfig) String() string {
	var out strings.Builder
	out.WriteString("repository:\n")
	fmt.Fprintf(&out, "        url \"%s\"\n", rc.URL)
	if rc.Branch != "" {
		fmt.Fprintf(&out, "        branch \"%s\"\n", rc.Branch)
	}
	if rc.Tag != "" {
		fmt.Fprintf(&out, "        tag \"%s\"\n", rc.Tag)
	}
	if rc.SSHKey != "" {
		fmt.Fprintf(&out, "        ssh_key \"%s\"\n", rc.SSHKey)
	}
	if !rc.Clone {
		fmt.Fprintf(&out, "        clone %v\n", rc.Clone)
	}
	fmt.Fprintf(&out, "        update_on_start %v\n", rc.UpdateOnStart)
	return out.String()
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Type        string // http, tcp, docker, dns, custom
	Endpoint    string // for http/tcp
	Domain      string // for dns
	Container   string // for docker
	Command     string // for custom
	Timeout     string
	Interval    string
	Retries     int
	Condition   string            // for http (status code)
	RecordType  string            // for dns (A, AAAA, etc)
	ExpectedIP  string            // for dns
	ExpectedIPs []string          // for dns with load balancer
	Headers     map[string]string // for http
	WorkingDir  string            // for custom
	StartPeriod string            // wait before first check
}

func (hc *HealthCheckConfig) String() string {
	var out strings.Builder
	out.WriteString("health check:\n")
	fmt.Fprintf(&out, "        type \"%s\"\n", hc.Type)

	switch hc.Type {
	case "http":
		fmt.Fprintf(&out, "        endpoint \"%s\"\n", hc.Endpoint)
		if hc.Condition != "" {
			fmt.Fprintf(&out, "        condition \"%s\"\n", hc.Condition)
		}
		if len(hc.Headers) > 0 {
			out.WriteString("        headers:\n")
			for k, v := range hc.Headers {
				fmt.Fprintf(&out, "            %s \"%s\"\n", k, v)
			}
		}
	case "tcp":
		fmt.Fprintf(&out, "        endpoint \"%s\"\n", hc.Endpoint)
	case "docker":
		fmt.Fprintf(&out, "        container \"%s\"\n", hc.Container)
	case "dns":
		fmt.Fprintf(&out, "        domain \"%s\"\n", hc.Domain)
		if hc.RecordType != "" {
			fmt.Fprintf(&out, "        record_type \"%s\"\n", hc.RecordType)
		}
		if hc.ExpectedIP != "" {
			fmt.Fprintf(&out, "        expected_ip \"%s\"\n", hc.ExpectedIP)
		}
	case "custom":
		fmt.Fprintf(&out, "        command \"%s\"\n", hc.Command)
		if hc.WorkingDir != "" {
			fmt.Fprintf(&out, "        working_directory \"%s\"\n", hc.WorkingDir)
		}
	}

	fmt.Fprintf(&out, "        timeout \"%s\"\n", hc.Timeout)
	fmt.Fprintf(&out, "        interval \"%s\"\n", hc.Interval)
	fmt.Fprintf(&out, "        retries %d\n", hc.Retries)

	if hc.StartPeriod != "" {
		fmt.Fprintf(&out, "        start_period \"%s\"\n", hc.StartPeriod)
	}

	return out.String()
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
	MakefileTimeout  string
	ParallelJobs     int
	Verbose          bool
	RetryOnFailure   bool
	MaxRetries       int
	RetryDelay       string
	FallbackCommand  string
	AllocateTTY      bool
}

func (bc *BuildConfig) String() string {
	var out strings.Builder
	out.WriteString("build:\n")
	fmt.Fprintf(&out, "        required %v\n", bc.Required)

	if bc.Command != "" {
		fmt.Fprintf(&out, "        command \"%s\"\n", bc.Command)
	}

	if bc.Makefile != "" {
		fmt.Fprintf(&out, "        makefile \"%s\"\n", bc.Makefile)
		if bc.MakeTarget != "" {
			fmt.Fprintf(&out, "        make_target \"%s\"\n", bc.MakeTarget)
		}
		if len(bc.MakeArgs) > 0 {
			fmt.Fprintf(&out, "        make_args %v\n", bc.MakeArgs)
		}
	}

	return out.String()
}

// ComposeConfig represents Docker Compose configuration
type ComposeConfig struct {
	File    string
	Project string
	Options *ComposeOptions
}

func (cc *ComposeConfig) String() string {
	var out strings.Builder
	out.WriteString("compose:\n")

	if cc.File != "" {
		fmt.Fprintf(&out, "        file \"%s\"\n", cc.File)
	}

	if cc.Project != "" {
		fmt.Fprintf(&out, "        project \"%s\"\n", cc.Project)
	}

	if cc.Options != nil {
		fmt.Fprintf(&out, "        %s\n", cc.Options.String())
	}

	return out.String()
}

// ComposeOptions represents Docker Compose command options
type ComposeOptions struct {
	ForceRecreate bool
	NoDeps        bool
	Build         bool
	Pull          string // always, missing, never
	Timeout       string
	Scale         string
	Wait          bool
	WaitTimeout   string
	Detach        bool
	RemoveOrphans bool
	RestartPolicy string
	MemoryLimit   string
	CPULimit      string
}

func (co *ComposeOptions) String() string {
	var out strings.Builder
	out.WriteString("options:\n")

	fmt.Fprintf(&out, "            force_recreate %v\n", co.ForceRecreate)
	fmt.Fprintf(&out, "            no_deps %v\n", co.NoDeps)
	fmt.Fprintf(&out, "            build %v\n", co.Build)

	if co.Pull != "" {
		fmt.Fprintf(&out, "            pull \"%s\"\n", co.Pull)
	}
	if co.Timeout != "" {
		fmt.Fprintf(&out, "            timeout \"%s\"\n", co.Timeout)
	}
	if co.Scale != "" {
		fmt.Fprintf(&out, "            scale \"%s\"\n", co.Scale)
	}

	fmt.Fprintf(&out, "            wait %v\n", co.Wait)

	if co.WaitTimeout != "" {
		fmt.Fprintf(&out, "            wait_timeout \"%s\"\n", co.WaitTimeout)
	}

	return out.String()
}

// EnvFileConfig represents environment file configuration
type EnvFileConfig struct {
	Required bool
	Task     string // Task to call before service start
}

func (efc *EnvFileConfig) String() string {
	var out strings.Builder
	out.WriteString("env_file:\n")
	fmt.Fprintf(&out, "        required %v\n", efc.Required)
	if efc.Task != "" {
		fmt.Fprintf(&out, "        task \"%s\"\n", efc.Task)
	}
	return out.String()
}

// OrchestrateStatement represents an orchestration group
type OrchestrateStatement struct {
	Token               lexer.Token
	Name                string
	Description         string
	Services            []string
	Strategy            string // sequential, parallel, dependency-based
	CircuitBreaker      bool
	StopOnFailure       bool
	HealthCheckInterval string
	StartupTimeout      string
	ShutdownTimeout     string
	PreTask             string
	PostTask            string
	FailureThreshold    int
	RecoveryTimeout     string
	MakefileOrder       []string
	MakefileTimeout     string
	CloneOrder          []string
	CloneTimeout        string
	ContainerManagement *ContainerManagement
	Recovery            *RecoveryConfig
	Discovery           *DiscoveryConfig
	Metrics             *MetricsConfig
	Scale               map[string]int
	UpdateStrategy      string
	MaxUnavailable      int
	UpdateTimeout       string
	GitSSHKey           string   // Optional: default SSH key for git repository operations
	DNSChecks           []string // Optional: domains to check DNS resolution for (warns if not resolvable)
}

func (os *OrchestrateStatement) statementNode()      {}
func (os *OrchestrateStatement) projectSettingNode() {}
func (os *OrchestrateStatement) String() string {
	var out strings.Builder
	fmt.Fprintf(&out, "orchestrate \"%s\"", os.Name)
	if os.Description != "" {
		fmt.Fprintf(&out, " means \"%s\"", os.Description)
	}
	out.WriteString(":\n")

	fmt.Fprintf(&out, "    services %v\n", os.Services)
	fmt.Fprintf(&out, "    strategy \"%s\"\n", os.Strategy)
	fmt.Fprintf(&out, "    circuit_breaker %v\n", os.CircuitBreaker)
	fmt.Fprintf(&out, "    stop_on_failure %v\n", os.StopOnFailure)

	if os.HealthCheckInterval != "" {
		fmt.Fprintf(&out, "    health_check_interval \"%s\"\n", os.HealthCheckInterval)
	}
	if os.StartupTimeout != "" {
		fmt.Fprintf(&out, "    startup_timeout \"%s\"\n", os.StartupTimeout)
	}
	if os.ShutdownTimeout != "" {
		fmt.Fprintf(&out, "    shutdown_timeout \"%s\"\n", os.ShutdownTimeout)
	}

	if os.PreTask != "" {
		fmt.Fprintf(&out, "    pre_task \"%s\"\n", os.PreTask)
	}
	if os.PostTask != "" {
		fmt.Fprintf(&out, "    post_task \"%s\"\n", os.PostTask)
	}

	return out.String()
}

// ContainerManagement represents container management options
type ContainerManagement struct {
	ForceRecreateOnStart   bool
	ForceRecreateOnRestart bool
	BuildBeforeStart       bool
	PullPolicy             string
	WaitForHealth          bool
	HealthCheckTimeout     string
}

// RecoveryConfig represents recovery configuration
type RecoveryConfig struct {
	MaxRetries         int
	RetryInterval      string
	ExponentialBackoff bool
	FallbackAction     string // restart, stop, ignore
}

// DiscoveryConfig represents service discovery configuration
type DiscoveryConfig struct {
	Type          string // consul, etcd, kubernetes, dns
	Endpoint      string
	Namespace     string
	DNSServer     string
	SearchDomains []string
	TTLCheck      bool
	CacheTimeout  string
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	Enabled  bool
	Endpoint string
	Interval string
	Labels   map[string]string
}

// OrchestrateActionStatement represents actions on orchestration groups
type OrchestrateActionStatement struct {
	Token     lexer.Token
	GroupName string
	Action    string // start, stop, restart, health_check, build, status, etc.
	Service   string // specific service name (optional)
	Options   map[string]string
}

func (oas *OrchestrateActionStatement) statementNode() {}
func (oas *OrchestrateActionStatement) String() string {
	var out strings.Builder
	fmt.Fprintf(&out, "orchestrate \"%s\" %s", oas.GroupName, oas.Action)

	if oas.Service != "" {
		fmt.Fprintf(&out, " service \"%s\"", oas.Service)
	}

	if len(oas.Options) > 0 {
		out.WriteString(" with:")
		for k, v := range oas.Options {
			fmt.Fprintf(&out, "\n        %s \"%s\"", k, v)
		}
	}

	return out.String()
}

// DockerNetworkConfig represents Docker network configuration
type DockerNetworkConfig struct {
	Token         lexer.Token
	Name          string
	External      bool
	Required      bool
	AutoProvision bool // defaults to false
	Driver        string
	Options       map[string]string
}
