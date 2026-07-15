package scm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ValueExpander func(string) (string, error)

type GitSourceResolver struct {
	Registry *GitRegistry
	Adapters map[string]GitProviderAdapter
	BaseDir  string
	Expand   ValueExpander
}

type ResolvedGitSource struct {
	Source  *GitSource
	Access  *GitAccessProfile
	Adapter GitProviderAdapter
}

func AdapterKey(provider, method string) string {
	return provider + ":" + method
}

func (r *GitSourceResolver) Resolve(alias, method string) (*ResolvedGitSource, error) {
	source, err := r.Registry.Source(alias)
	if err != nil {
		return nil, err
	}
	profile, err := source.AccessProfile(method)
	if err != nil {
		return nil, err
	}
	resolvedSource := *source
	resolvedSource.Access = make(map[string]*GitAccessProfile, len(source.Access))
	resolvedProfile := *profile
	fields := []struct {
		value       string
		destination *string
	}{
		{profile.URL, &resolvedProfile.URL}, {profile.Repository, &resolvedProfile.Repository},
		{profile.Host, &resolvedProfile.Host}, {profile.Key, &resolvedProfile.Key},
		{profile.Path, &resolvedProfile.Path},
	}
	for _, field := range fields {
		value, destination := field.value, field.destination
		expanded, expandErr := r.expand(value)
		if expandErr != nil {
			return nil, fmt.Errorf("Git source %q %s: %w", alias, profile.Method, expandErr)
		}
		*destination = expanded
	}
	if resolvedProfile.Key != "" {
		resolvedProfile.Key, err = expandPath(resolvedProfile.Key, r.BaseDir)
		if err != nil {
			return nil, fmt.Errorf("Git source %q SSH key path: %w", alias, err)
		}
	}
	if resolvedProfile.Path != "" {
		resolvedProfile.Path, err = expandPath(resolvedProfile.Path, r.BaseDir)
		if err != nil {
			return nil, fmt.Errorf("Git source %q filesystem path: %w", alias, err)
		}
	}
	resolvedSource.Access[resolvedProfile.Method] = &resolvedProfile
	adapter := r.Adapters[AdapterKey(source.Provider, profile.Method)]
	if adapter == nil {
		adapter = r.Adapters[AdapterKey("*", profile.Method)]
	}
	if adapter == nil {
		return nil, fmt.Errorf("no Git adapter is available for %s %s access", source.Provider, profile.Method)
	}
	return &ResolvedGitSource{Source: &resolvedSource, Access: &resolvedProfile, Adapter: adapter}, nil
}

func (r *GitSourceResolver) Open(ctx context.Context, alias, method string, allowFetch bool) (GitRepositorySession, error) {
	resolved, err := r.Resolve(alias, method)
	if err != nil {
		return nil, err
	}
	return resolved.Adapter.Open(ctx, resolved.Source, resolved.Access, allowFetch)
}

func (r *GitSourceResolver) expand(value string) (string, error) {
	if value == "" || r.Expand == nil {
		return value, nil
	}
	return r.Expand(value)
}

func expandPath(path, baseDir string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	if !filepath.IsAbs(path) && baseDir != "" {
		path = filepath.Join(baseDir, path)
	}
	return filepath.Clean(path), nil
}
