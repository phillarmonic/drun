package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// OrchestrationActionStatement represents orchestration actions in task bodies
// Examples: orchestrate "group" start, orchestrate "group" stop
type OrchestrationActionStatement struct {
	Options        map[string]string
	GroupName      string
	Action         string   // start, stop, restart, health_check, status, logs, etc.
	ServiceFilters []string // optional: specific services to act on
	Token          lexer.Token
}

func (oas *OrchestrationActionStatement) statementNode() {}
func (oas *OrchestrationActionStatement) String() string {
	out := fmt.Sprintf("orchestrate \"%s\" %s", oas.GroupName, oas.Action)
	if len(oas.ServiceFilters) > 0 {
		out += fmt.Sprintf(" services %v", oas.ServiceFilters)
	}
	var outSb26 strings.Builder
	for key, value := range oas.Options {
		fmt.Fprintf(&outSb26, " %s \"%s\"", key, value)
	}
	out += outSb26.String()
	return out
}

// ServiceStatement represents a microservice definition
type ServiceStatement struct {
	Environment  map[string]string
	HealthCheck  *HealthCheckConfig
	Build        *BuildConfig
	Networks     map[string]*DockerNetworkConfig
	EnvFile      *EnvFileConfig
	Repository   *RepositoryConfig
	Compose      *ComposeConfig
	PreTask      string
	Name         string
	Description  string
	Path         string
	PostTask     string
	Dependencies []string
	Token        lexer.Token
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
	Headers     map[string]string // for http
	Interval    string
	Condition   string // for http (status code)
	Container   string // for docker
	Command     string // for custom
	Timeout     string
	Type        string   // http, tcp, docker, dns, custom
	StartPeriod string   // wait before first check
	Domain      string   // for dns
	RecordType  string   // for dns (A, AAAA, etc)
	ExpectedIP  string   // for dns
	WorkingDir  string   // for custom
	Endpoint    string   // for http/tcp
	ExpectedIPs []string // for dns with load balancer
	Retries     int
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
	WorkingDirectory string
	Command          string
	Makefile         string
	MakeTarget       string
	FallbackCommand  string
	RetryDelay       string
	MakefileTimeout  string
	MakeArgs         []string
	PostMakeCommands []string
	PreMakeCommands  []string
	ParallelJobs     int
	MaxRetries       int
	Verbose          bool
	RetryOnFailure   bool
	Required         bool
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
	Options *ComposeOptions
	File    string
	Project string
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
	WaitTimeout   string
	CPULimit      string
	MemoryLimit   string
	Pull          string // always, missing, never
	Timeout       string
	Scale         string
	RestartPolicy string
	Wait          bool
	Detach        bool
	RemoveOrphans bool
	ForceRecreate bool
	Build         bool
	NoDeps        bool
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
	Task     string // Task to call before service start
	Required bool
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
	ContainerManagement *ContainerManagement
	Scale               map[string]int
	Metrics             *MetricsConfig
	Discovery           *DiscoveryConfig
	Recovery            *RecoveryConfig
	HealthCheckInterval string
	Description         string
	GitSSHKey           string // Optional: default SSH key for git repository operations
	StartupTimeout      string
	ShutdownTimeout     string
	PreTask             string
	PostTask            string
	UpdateTimeout       string
	RecoveryTimeout     string
	UpdateStrategy      string
	MakefileTimeout     string
	Name                string
	CloneTimeout        string
	Strategy            string // sequential, parallel, dependency-based
	Services            []string
	CloneOrder          []string
	MakefileOrder       []string
	DNSChecks           []string // Optional: domains to check DNS resolution for (warns if not resolvable)
	Token               lexer.Token
	MaxUnavailable      int
	FailureThreshold    int
	CircuitBreaker      bool
	StopOnFailure       bool
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
	PullPolicy             string
	HealthCheckTimeout     string
	ForceRecreateOnStart   bool
	ForceRecreateOnRestart bool
	BuildBeforeStart       bool
	WaitForHealth          bool
}

// RecoveryConfig represents recovery configuration
type RecoveryConfig struct {
	RetryInterval      string
	FallbackAction     string // restart, stop, ignore
	MaxRetries         int
	ExponentialBackoff bool
}

// DiscoveryConfig represents service discovery configuration
type DiscoveryConfig struct {
	Type          string // consul, etcd, kubernetes, dns
	Endpoint      string
	Namespace     string
	DNSServer     string
	CacheTimeout  string
	SearchDomains []string
	TTLCheck      bool
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	Labels   map[string]string
	Endpoint string
	Interval string
	Enabled  bool
}

// OrchestrateActionStatement represents actions on orchestration groups
type OrchestrateActionStatement struct {
	Options   map[string]string
	GroupName string
	Action    string // start, stop, restart, health_check, build, status, etc.
	Service   string // specific service name (optional)
	Token     lexer.Token
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
	Options       map[string]string
	Name          string
	Driver        string
	Token         lexer.Token
	External      bool
	Required      bool
	AutoProvision bool // defaults to false
}
