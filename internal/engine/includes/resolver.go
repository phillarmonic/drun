package includes

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/cache"
	"github.com/phillarmonic/drun/internal/remote"
)

// Resolver handles file inclusion, both local and remote
type Resolver struct {
	cacheManager   *cache.Manager
	githubFetcher  remote.Fetcher
	httpsFetcher   remote.Fetcher
	drunhubFetcher remote.Fetcher
	verbose        bool
	output         io.Writer
	tempFiles      []string // Track temp files for cleanup
	parseFunc      ParseFunc
}

// ParseFunc is a function type for parsing drun files
type ParseFunc func(input, filename string) (*ast.Program, error)

// ProjectContext provides access to project-level configuration
type ProjectContext interface {
	GetIncludedFiles() map[string]bool
	GetIncludedSnippets() map[string]*ast.SnippetStatement
	GetIncludedTemplates() map[string]*ast.TaskTemplateStatement
	GetIncludedTasks() map[string]*ast.TaskStatement
	GetIncludedSettings() map[string]string
	GetIncludedParams() map[string]*ast.ProjectParameterStatement
}

// NewResolver creates a new include resolver
func NewResolver(
	cacheManager *cache.Manager,
	githubFetcher, httpsFetcher, drunhubFetcher remote.Fetcher,
	verbose bool,
	output io.Writer,
	parseFunc ParseFunc,
) *Resolver {
	return &Resolver{
		cacheManager:   cacheManager,
		githubFetcher:  githubFetcher,
		httpsFetcher:   httpsFetcher,
		drunhubFetcher: drunhubFetcher,
		verbose:        verbose,
		output:         output,
		tempFiles:      []string{},
		parseFunc:      parseFunc,
	}
}

// ProcessInclude loads and merges an included file into the project context
func (r *Resolver) ProcessInclude(ctx ProjectContext, include *ast.IncludeStatement, currentFile string) {
	// Resolve the include path relative to the current file
	includePath, err := r.resolveIncludePath(include.Path, currentFile)
	if err != nil {
		if r.verbose {
			_, _ = fmt.Fprintf(r.output, "⚠️  Failed to resolve include path %s: %v\n", include.Path, err)
		}
		return
	}

	// Check for circular includes
	if ctx.GetIncludedFiles()[includePath] {
		if r.verbose {
			_, _ = fmt.Fprintf(r.output, "⚠️  Circular include detected: %s (skipping)\n", includePath)
		}
		return
	}

	// Mark this file as included
	ctx.GetIncludedFiles()[includePath] = true

	// Load and parse the included file
	content, err := os.ReadFile(includePath)
	if err != nil {
		if r.verbose {
			_, _ = fmt.Fprintf(r.output, "⚠️  Failed to read included file %s: %v\n", includePath, err)
		}
		return
	}

	// Parse the included file
	program, err := r.parseFunc(string(content), includePath)
	if err != nil {
		if r.verbose {
			_, _ = fmt.Fprintf(r.output, "⚠️  Failed to parse included file %s: %v\n", includePath, err)
		}
		return
	}

	// Extract the namespace from the included project
	if program.Project == nil {
		if r.verbose {
			_, _ = fmt.Fprintf(r.output, "⚠️  Included file %s has no project declaration (skipping)\n", includePath)
		}
		return
	}

	// Use custom namespace if provided via "as" clause, otherwise use project name
	namespace := include.Namespace
	if namespace == "" {
		namespace = program.Project.Name
	}

	// Determine what to include based on selectors
	includeAll := len(include.Selectors) == 0
	includeSnippets := includeAll
	includeTemplates := includeAll
	includeTasks := includeAll

	for _, selector := range include.Selectors {
		switch selector {
		case "snippets":
			includeSnippets = true
		case "templates":
			includeTemplates = true
		case "tasks":
			includeTasks = true
		}
	}

	// Merge settings, parameters, and snippets from the included project
	if program.Project != nil {
		for _, setting := range program.Project.Settings {
			switch s := setting.(type) {
			case *ast.SetStatement:
				// Namespace project settings
				namespacedKey := namespace + "." + s.Key
				if s.Value != nil {
					ctx.GetIncludedSettings()[namespacedKey] = s.Value.String()
					if r.verbose {
						_, _ = fmt.Fprintf(r.output, "  ✓ Loaded setting: %s\n", namespacedKey)
					}
				}
			case *ast.ProjectParameterStatement:
				// Namespace project parameters
				namespacedName := namespace + "." + s.Name
				ctx.GetIncludedParams()[namespacedName] = s
				if r.verbose {
					_, _ = fmt.Fprintf(r.output, "  ✓ Loaded parameter: %s\n", namespacedName)
				}
			case *ast.SnippetStatement:
				// Namespace snippets (only if includeSnippets is true)
				if includeSnippets {
					namespacedName := namespace + "." + s.Name
					ctx.GetIncludedSnippets()[namespacedName] = s
					if r.verbose {
						_, _ = fmt.Fprintf(r.output, "  ✓ Loaded snippet: %s\n", namespacedName)
					}
				}
			}
		}
	}

	// Merge templates
	if includeTemplates {
		for _, template := range program.Templates {
			namespacedName := namespace + "." + template.Name
			ctx.GetIncludedTemplates()[namespacedName] = template
			if r.verbose {
				_, _ = fmt.Fprintf(r.output, "  ✓ Loaded template: %s\n", namespacedName)
			}
		}
	}

	// Merge tasks
	if includeTasks {
		for _, task := range program.Tasks {
			namespacedName := namespace + "." + task.Name
			ctx.GetIncludedTasks()[namespacedName] = task
			if r.verbose {
				_, _ = fmt.Fprintf(r.output, "  ✓ Loaded task: %s\n", namespacedName)
			}
		}
	}

	if r.verbose {
		_, _ = fmt.Fprintf(r.output, "✓ Included %s as namespace '%s'\n", include.Path, namespace)
	}
}

// resolveIncludePath resolves the include path relative to the current file
func (r *Resolver) resolveIncludePath(includePath, currentFile string) (string, error) {
	// Check if remote URL
	if remote.IsRemoteURL(includePath) {
		return r.fetchRemoteInclude(includePath)
	}

	// If absolute path, use as-is
	if filepath.IsAbs(includePath) {
		return includePath, nil
	}

	// Get the directory of the current file
	currentDir := filepath.Dir(currentFile)

	// Try relative to current file first
	resolvedPath := filepath.Join(currentDir, includePath)
	if _, err := os.Stat(resolvedPath); err == nil {
		// Make it absolute
		absPath, _ := filepath.Abs(resolvedPath)
		return absPath, nil
	}

	// Try relative to workspace root (current working directory)
	if cwd, err := os.Getwd(); err == nil {
		resolvedPath = filepath.Join(cwd, includePath)
		if _, err := os.Stat(resolvedPath); err == nil {
			absPath, _ := filepath.Abs(resolvedPath)
			return absPath, nil
		}
	}

	// Fall back to the original path (will likely fail when reading)
	return includePath, nil
}

// fetchRemoteInclude fetches a remote include and returns the path to a temp file
func (r *Resolver) fetchRemoteInclude(url string) (string, error) {
	protocol, path, ref, err := remote.ParseRemoteURL(url)
	if err != nil {
		return "", err
	}

	// Generate cache key
	cacheKey := cache.GenerateKey(url, ref)

	// Check cache (if enabled)
	if r.cacheManager != nil {
		if content, hit, err := r.cacheManager.Get(cacheKey); err == nil && hit {
			if r.verbose {
				_, _ = fmt.Fprintf(r.output, "  ✓ Cache hit for %s\n", url)
			}
			return r.writeTempFile(content, url)
		}
	}

	// Fetch from remote
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var fetcher remote.Fetcher
	switch protocol {
	case "github":
		fetcher = r.githubFetcher
	case "https":
		fetcher = r.httpsFetcher
	case "drunhub":
		fetcher = r.drunhubFetcher
	default:
		return "", fmt.Errorf("unsupported protocol: %s", protocol)
	}

	if r.verbose {
		_, _ = fmt.Fprintf(r.output, "🌐 Fetching remote include: %s\n", url)
		if protocol == "github" && ref == "" {
			_, _ = fmt.Fprintf(r.output, "  ✓ Detecting default branch...\n")
		}
	}

	content, err := fetcher.Fetch(ctx, path, ref)
	if err != nil {
		// Try stale cache as fallback
		if r.cacheManager != nil {
			if stale, ok := r.cacheManager.GetStale(cacheKey); ok {
				if r.verbose {
					_, _ = fmt.Fprintf(r.output, "  ⚠️  Network error, using stale cache\n")
				}
				return r.writeTempFile(stale, url)
			}
		}
		return "", fmt.Errorf("failed to fetch %s: %w (no cache available)", url, err)
	}

	if r.verbose {
		_, _ = fmt.Fprintf(r.output, "  ✓ Downloaded %.1f KB\n", float64(len(content))/1024)
	}

	// Store in cache
	if r.cacheManager != nil {
		if err := r.cacheManager.Set(cacheKey, content, 1*time.Minute); err != nil {
			// Log but don't fail
			if r.verbose {
				_, _ = fmt.Fprintf(r.output, "  ⚠️  Failed to cache: %v\n", err)
			}
		} else if r.verbose {
			_, _ = fmt.Fprintf(r.output, "  ✓ Cached with 1m expiration\n")
		}
	}

	return r.writeTempFile(content, url)
}

// writeTempFile writes content to a temporary file and tracks it for cleanup
func (r *Resolver) writeTempFile(content []byte, sourceURL string) (string, error) {
	// Create temp file with .drun extension
	tmpFile, err := os.CreateTemp("", "drun-remote-*.drun")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = tmpFile.Close() }()

	// Write content
	if _, err := tmpFile.Write(content); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// Track for cleanup
	r.tempFiles = append(r.tempFiles, tmpFile.Name())

	// Return absolute path
	return tmpFile.Name(), nil
}

// Cleanup removes all temporary files created during include resolution
func (r *Resolver) Cleanup() {
	for _, f := range r.tempFiles {
		_ = os.Remove(f)
	}
	r.tempFiles = nil
}

// GetTempFiles returns the list of temporary files created
func (r *Resolver) GetTempFiles() []string {
	return r.tempFiles
}
