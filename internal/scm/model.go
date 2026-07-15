// Package scm contains operation-neutral SCM registry and Git repository contracts.
package scm

import (
	"context"
	"fmt"
	"time"

	"github.com/phillarmonic/drun/v2/internal/ast"
)

type SCMRegistry struct {
	Technologies map[string]*TechnologyRegistry
}

type TechnologyRegistry struct {
	Name string
	Git  *GitRegistry
}

type GitRegistry struct {
	Providers map[string]map[string]*GitSource
	Sources   map[string]*GitSource
}

type GitSource struct {
	Alias       string
	Provider    string
	Default     string
	Metadata    string
	Access      map[string]*GitAccessProfile
	VersionTags *GitVersionTagContract
}

type GitAccessProfile struct {
	Method         string
	URL            string
	Repository     string
	Host           string
	Authentication string
	Key            string
	Path           string
}

type GitProviderCapabilities struct {
	RemoteRefs     bool
	ObjectMetadata bool
	ProviderAPI    bool
}

type GitRef struct {
	Name       string
	TargetHash string
	Date       time.Time
}

// GitProviderAdapter opens a repository without coupling source registration to an operation.
type GitProviderAdapter interface {
	Capabilities() GitProviderCapabilities
	Open(context.Context, *GitSource, *GitAccessProfile, bool) (GitRepositorySession, error)
}

// GitRepositorySession exposes primitives shared by version queries and future Git operations.
type GitRepositorySession interface {
	Tags(context.Context, bool) ([]GitRef, error)
	Close() error
}

func RegistryFromAST(node *ast.SCMRegistryStatement) (*SCMRegistry, error) {
	registry := &SCMRegistry{Technologies: make(map[string]*TechnologyRegistry)}
	if node == nil {
		return registry, nil
	}
	for name, technology := range node.Technologies {
		if name != "git" {
			return nil, fmt.Errorf("SCM technology %q is not supported", name)
		}
		gitRegistry := &GitRegistry{
			Providers: make(map[string]map[string]*GitSource),
			Sources:   make(map[string]*GitSource),
		}
		for providerName, provider := range technology.Providers {
			gitRegistry.Providers[providerName] = make(map[string]*GitSource)
			for alias, source := range provider.Sources {
				if _, exists := gitRegistry.Sources[alias]; exists {
					return nil, fmt.Errorf("Git source alias %q is registered more than once", alias)
				}
				typed, err := gitSourceFromAST(source)
				if err != nil {
					return nil, fmt.Errorf("Git source %q: %w", alias, err)
				}
				gitRegistry.Sources[alias] = typed
				gitRegistry.Providers[providerName][alias] = typed
			}
		}
		registry.Technologies[name] = &TechnologyRegistry{Name: name, Git: gitRegistry}
	}
	return registry, nil
}

func gitSourceFromAST(source *ast.SCMSource) (*GitSource, error) {
	typed := &GitSource{
		Alias:    source.Alias,
		Provider: source.Provider,
		Default:  source.Default,
		Metadata: source.Metadata,
		Access:   make(map[string]*GitAccessProfile),
	}
	for method, profile := range source.Access {
		typed.Access[method] = &GitAccessProfile{
			Method: profile.Method, URL: profile.URL, Repository: profile.Repository,
			Host: profile.Host, Authentication: profile.Authentication, Key: profile.Key,
			Path: profile.Path,
		}
	}
	if source.VersionTags != nil {
		contract, err := NewGitVersionTagContract(source.VersionTags.Preset, source.VersionTags.Formats, source.VersionTags.Pattern, DefaultTagFormatMacroRegistry())
		if err != nil {
			return nil, err
		}
		typed.VersionTags = contract
	}
	return typed, nil
}

func (r *SCMRegistry) Git() *GitRegistry {
	if r == nil || r.Technologies["git"] == nil {
		return nil
	}
	return r.Technologies["git"].Git
}

func (r *GitRegistry) Source(alias string) (*GitSource, error) {
	if r == nil {
		return nil, fmt.Errorf("project does not declare scm → git sources")
	}
	source, ok := r.Sources[alias]
	if !ok {
		return nil, fmt.Errorf("Git source %q is not registered", alias)
	}
	return source, nil
}

func (s *GitSource) AccessProfile(method string) (*GitAccessProfile, error) {
	if method == "" {
		method = s.Default
	}
	profile, ok := s.Access[method]
	if !ok {
		return nil, fmt.Errorf("Git source %q does not declare access method %q", s.Alias, method)
	}
	return profile, nil
}
