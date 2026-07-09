package engine

import (
	"io"
	"os"

	"github.com/phillarmonic/drun/v2/internal/cache"
	"github.com/phillarmonic/drun/v2/internal/domain/parameter"
	"github.com/phillarmonic/drun/v2/internal/domain/task"
	"github.com/phillarmonic/drun/v2/internal/provisioning"
)

// EngineOptions configures the engine with optional dependencies
// Note: Most dependencies use concrete types for simplicity.
// For testing, use mocks/fakes at the task/executor level instead.
type EngineOptions struct {
	// Output writer (defaults to os.Stdout)
	Output io.Writer

	// Task registry (defaults to new registry)
	TaskRegistry *task.Registry

	// Parameter validator (defaults to new validator)
	ParamValidator *parameter.Validator

	// Dependency resolver (defaults to new resolver)
	DepResolver *task.DependencyResolver

	// Cache manager (defaults to nil, created on demand)
	CacheManager *cache.Manager

	// DryRun mode
	DryRun bool

	// Verbose mode
	Verbose bool

	// Runtime task mode override for the invocation
	TaskModeOverride string

	// Allow runtime provisioning to change an installed tool's version.
	AllowToolVersionChanges bool

	// User-level fallback provisioning catalogs loaded from ~/.drun/config.yml.
	UserProvisioningSources []string

	// Embedded built-in provisioning catalogs shipped with drun.
	EmbeddedProvisioningSources []provisioning.EmbeddedSource

	// Secrets manager
	SecretsManager SecretsManager
}

// Option is a functional option for configuring the Engine
type Option func(*EngineOptions)

// WithOutput sets the output writer
func WithOutput(w io.Writer) Option {
	return func(o *EngineOptions) {
		o.Output = w
	}
}

// WithTaskRegistry sets the task registry
func WithTaskRegistry(reg *task.Registry) Option {
	return func(o *EngineOptions) {
		o.TaskRegistry = reg
	}
}

// WithParamValidator sets the parameter validator
func WithParamValidator(v *parameter.Validator) Option {
	return func(o *EngineOptions) {
		o.ParamValidator = v
	}
}

// WithCacheManager sets the cache manager
func WithCacheManager(cm *cache.Manager) Option {
	return func(o *EngineOptions) {
		o.CacheManager = cm
	}
}

// WithDryRun sets dry-run mode
func WithDryRun(dryRun bool) Option {
	return func(o *EngineOptions) {
		o.DryRun = dryRun
	}
}

// WithVerbose sets verbose mode
func WithVerbose(verbose bool) Option {
	return func(o *EngineOptions) {
		o.Verbose = verbose
	}
}

// WithTaskModeOverride sets a runtime task mode override for this invocation.
func WithTaskModeOverride(mode string) Option {
	return func(o *EngineOptions) {
		o.TaskModeOverride = mode
	}
}

// WithAllowToolVersionChanges allows runtime provisioning to upgrade or
// downgrade installed tools when a versioned requirement opts into provisioning.
func WithAllowToolVersionChanges(allow bool) Option {
	return func(o *EngineOptions) {
		o.AllowToolVersionChanges = allow
	}
}

// WithUserProvisioningSources sets user-level fallback provisioning catalogs.
func WithUserProvisioningSources(sources []string) Option {
	return func(o *EngineOptions) {
		o.UserProvisioningSources = append([]string(nil), sources...)
	}
}

// WithEmbeddedProvisioningSources sets built-in provisioning catalogs.
func WithEmbeddedProvisioningSources(sources []provisioning.EmbeddedSource) Option {
	return func(o *EngineOptions) {
		o.EmbeddedProvisioningSources = append([]provisioning.EmbeddedSource(nil), sources...)
	}
}

// WithSecretsManager sets the secrets manager
func WithSecretsManager(sm SecretsManager) Option {
	return func(o *EngineOptions) {
		o.SecretsManager = sm
	}
}

// applyDefaults applies default values to unset options
func (opts *EngineOptions) applyDefaults() {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	if opts.TaskRegistry == nil {
		opts.TaskRegistry = task.NewRegistry()
	}

	if opts.ParamValidator == nil {
		opts.ParamValidator = parameter.NewValidator()
	}

	if opts.DepResolver == nil {
		opts.DepResolver = task.NewDependencyResolver(opts.TaskRegistry)
	}

	// Note: CacheManager defaults to nil and is created on demand in the engine
}
