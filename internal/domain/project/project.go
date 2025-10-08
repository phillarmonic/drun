package project

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Project represents a domain project entity
type Project struct {
	Name          string
	Version       string
	Settings      map[string]string
	ShellConfigs  map[string]*ShellConfig
	SetupHooks    []Hook
	TeardownHooks []Hook
	BeforeHooks   []Hook
	AfterHooks    []Hook
}

// NewProject creates a new project from AST
func NewProject(stmt *ast.ProjectStatement) (*Project, error) {
	project := &Project{
		Name:         stmt.Name,
		Version:      stmt.Version,
		Settings:     make(map[string]string),
		ShellConfigs: make(map[string]*ShellConfig),
	}

	// Process settings
	for _, setting := range stmt.Settings {
		switch s := setting.(type) {
		case *ast.SetStatement:
			if s.Value != nil {
				project.Settings[s.Key] = s.Value.String()
			}

		case *ast.ShellConfigStatement:
			for platform, config := range s.Platforms {
				project.ShellConfigs[platform] = &ShellConfig{
					Executable:  config.Executable,
					Args:        config.Args,
					Environment: config.Environment,
				}
			}

		case *ast.LifecycleHook:
			// Convert hook body from AST to domain
			body, err := statement.FromASTList(s.Body)
			if err != nil {
				return nil, fmt.Errorf("converting %s hook body: %w", s.Type, err)
			}

			hook := Hook{
				Type:  s.Type,
				Scope: s.Scope,
				Body:  body,
			}

			switch s.Type {
			case "setup":
				project.SetupHooks = append(project.SetupHooks, hook)
			case "teardown":
				project.TeardownHooks = append(project.TeardownHooks, hook)
			case "before":
				project.BeforeHooks = append(project.BeforeHooks, hook)
			case "after":
				project.AfterHooks = append(project.AfterHooks, hook)
			}
		}
	}

	return project, nil
}

// GetSetting gets a project setting
func (p *Project) GetSetting(key string) (string, bool) {
	value, exists := p.Settings[key]
	return value, exists
}

// GetShellConfig gets shell config for platform
func (p *Project) GetShellConfig(platform string) (*ShellConfig, bool) {
	config, exists := p.ShellConfigs[platform]
	return config, exists
}

// HasHooks checks if project has any hooks
func (p *Project) HasHooks() bool {
	return len(p.SetupHooks) > 0 ||
		len(p.TeardownHooks) > 0 ||
		len(p.BeforeHooks) > 0 ||
		len(p.AfterHooks) > 0
}

// ShellConfig represents shell configuration for a platform
type ShellConfig struct {
	Executable  string
	Args        []string
	Environment map[string]string
}

// Hook represents a lifecycle hook
type Hook struct {
	Type  string // "before", "after", "setup", "teardown"
	Scope string // "any", "drun"
	Body  []statement.Statement
}
