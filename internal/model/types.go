package model

import (
	"time"
)

// Spec represents the complete drun configuration
type Spec struct {
	Version  string                 `yaml:"version"`
	Shell    map[string]ShellConfig `yaml:"shell,omitempty"`
	Env      map[string]string      `yaml:"env,omitempty"`
	Vars     map[string]any         `yaml:"vars,omitempty"`
	Defaults Defaults               `yaml:"defaults,omitempty"`
	Snippets map[string]string      `yaml:"snippets,omitempty"`
	Include  []string               `yaml:"include,omitempty"`
	Recipes  map[string]Recipe      `yaml:"recipes"`
	Cache    CacheConfig            `yaml:"cache,omitempty"`
}

// ShellConfig defines shell configuration per OS
type ShellConfig struct {
	Cmd  string   `yaml:"cmd"`
	Args []string `yaml:"args"`
}

// Defaults contains global default settings
type Defaults struct {
	WorkingDir string        `yaml:"working_dir,omitempty"`
	Shell      string        `yaml:"shell,omitempty"`
	ExportEnv  bool          `yaml:"export_env,omitempty"`
	Timeout    time.Duration `yaml:"timeout,omitempty"`
	InheritEnv bool          `yaml:"inherit_env,omitempty"`
	Strict     bool          `yaml:"strict,omitempty"`
}

// Recipe represents a single task/recipe
type Recipe struct {
	Help         string            `yaml:"help,omitempty"`
	Positionals  []PositionalArg   `yaml:"positionals,omitempty"`
	Flags        map[string]Flag   `yaml:"flags,omitempty"`
	Env          map[string]string `yaml:"env,omitempty"`
	Deps         []string          `yaml:"deps,omitempty"`
	ParallelDeps bool              `yaml:"parallel_deps,omitempty"`
	Run          Step              `yaml:"run"`
	WorkingDir   string            `yaml:"working_dir,omitempty"`
	Shell        string            `yaml:"shell,omitempty"`
	Timeout      time.Duration     `yaml:"timeout,omitempty"`
	IgnoreError  bool              `yaml:"ignore_error,omitempty"`
	Aliases      []string          `yaml:"aliases,omitempty"`
	Matrix       map[string][]any  `yaml:"matrix,omitempty"`
	CacheKey     string            `yaml:"cache_key,omitempty"`
}

// PositionalArg defines a positional argument
type PositionalArg struct {
	Name     string   `yaml:"name"`
	Required bool     `yaml:"required,omitempty"`
	OneOf    []string `yaml:"one_of,omitempty"`
	Pattern  string   `yaml:"pattern,omitempty"`
	Default  string   `yaml:"default,omitempty"`
	Variadic bool     `yaml:"variadic,omitempty"`
}

// Flag defines a command-line flag
type Flag struct {
	Type    string `yaml:"type"` // string, int, bool, string[]
	Default any    `yaml:"default,omitempty"`
	Help    string `yaml:"help,omitempty"`
}

// Step represents the run commands (can be string or []string)
type Step struct {
	Lines []string
}

// CacheConfig defines caching behavior
type CacheConfig struct {
	Path string   `yaml:"path,omitempty"`
	Keys []string `yaml:"keys,omitempty"`
}

// ExecutionContext contains the runtime context for template rendering
type ExecutionContext struct {
	Vars        map[string]any
	Env         map[string]string
	Flags       map[string]any
	Positionals map[string]any
	OS          string
	Arch        string
	Hostname    string
}

// PlanNode represents a single execution unit in the DAG
type PlanNode struct {
	ID        string
	Recipe    *Recipe
	Context   *ExecutionContext
	Step      Step
	DependsOn []string
}

// ExecutionPlan represents the complete execution plan
type ExecutionPlan struct {
	Nodes []PlanNode
	Edges [][2]int // [from_index, to_index]
}
