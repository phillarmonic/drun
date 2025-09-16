package spec

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/phillarmonic/drun/internal/model"
	"gopkg.in/yaml.v3"
)

// DefaultFilenames are the default config file names to look for
var DefaultFilenames = []string{
	"drun.yml",
	"drun.yaml",
	".drun.yml",
	".drun.yaml",
	".drun/drun.yml",
	".drun/drun.yaml",
	"ops.drun.yml",
	"ops.drun.yaml",
}

// CacheEntry represents a cached spec with metadata
type CacheEntry struct {
	Spec     *model.Spec
	ModTime  time.Time
	FileHash string
}

// Loader handles loading and validating drun specifications
type Loader struct {
	baseDir     string
	cache       sync.Map // Cache specs by file path
	fileCache   sync.Map // Cache file contents by path+modtime
	remoteCache sync.Map // Cache remote includes by URL
	cacheDir    string   // Directory for remote include cache
}

// NewLoader creates a new spec loader
func NewLoader(baseDir string) *Loader {
	// Set up cache directory
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".drun", "cache", "includes")

	return &Loader{
		baseDir:  baseDir,
		cacheDir: cacheDir,
	}
}

// Load loads a drun specification from a file
func (l *Loader) Load(filename string) (*model.Spec, error) {
	var filePath string

	if filename == "" {
		// Try default filenames
		found := false
		for _, defaultName := range DefaultFilenames {
			candidate := filepath.Join(l.baseDir, defaultName)
			if _, err := os.Stat(candidate); err == nil {
				filePath = candidate
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("no drun configuration file found (tried: %s)", strings.Join(DefaultFilenames, ", "))
		}
	} else {
		if filepath.IsAbs(filename) {
			filePath = filename
		} else {
			filePath = filepath.Join(l.baseDir, filename)
		}
	}

	// Check if we have a cached version
	if cached, valid := l.getCachedSpec(filePath); valid {
		return cached, nil
	}

	data, err := l.readFileWithCache(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var spec model.Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", filePath, err)
	}

	// Store the main spec content before processing includes
	mainSpec := spec

	// Process includes first, before validation
	if len(spec.Include) > 0 {
		if err := l.processIncludes(&spec, filepath.Dir(filePath)); err != nil {
			return nil, fmt.Errorf("failed to process includes: %w", err)
		}

		// Re-merge main spec over included content (main overrides includes)
		l.mergeSpecs(&spec, &mainSpec)
	}

	// Set defaults after includes are processed
	l.setDefaults(&spec)

	// Validate the spec after includes and defaults
	if err := l.validate(&spec); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", filePath, err)
	}

	// Cache the successfully loaded and validated spec
	l.cacheSpec(filePath, &spec)

	return &spec, nil
}

// setDefaults sets reasonable defaults for the spec
func (l *Loader) setDefaults(spec *model.Spec) {
	if spec.Version == "" {
		spec.Version = "0.1"
	}

	// Set default shell configurations
	if spec.Shell == nil {
		spec.Shell = make(map[string]model.ShellConfig)
	}

	if _, exists := spec.Shell["linux"]; !exists {
		spec.Shell["linux"] = model.ShellConfig{
			Cmd:  "/bin/sh",
			Args: []string{"-ceu"},
		}
	}

	if _, exists := spec.Shell["darwin"]; !exists {
		spec.Shell["darwin"] = model.ShellConfig{
			Cmd:  "/bin/zsh",
			Args: []string{"-ceu"},
		}
	}

	if _, exists := spec.Shell["windows"]; !exists {
		spec.Shell["windows"] = model.ShellConfig{
			Cmd:  "pwsh",
			Args: []string{"-NoLogo", "-Command"},
		}
	}

	// Set global defaults
	if spec.Defaults.WorkingDir == "" {
		spec.Defaults.WorkingDir = "."
	}
	if spec.Defaults.Shell == "" {
		spec.Defaults.Shell = "auto"
	}
	if spec.Defaults.Timeout == 0 {
		spec.Defaults.Timeout = 2 * time.Hour
	}
	spec.Defaults.ExportEnv = true
	spec.Defaults.InheritEnv = true
	spec.Defaults.Strict = true

	// Set defaults for each recipe
	for name, recipe := range spec.Recipes {
		if recipe.WorkingDir == "" {
			recipe.WorkingDir = spec.Defaults.WorkingDir
		}
		if recipe.Shell == "" {
			recipe.Shell = spec.Defaults.Shell
		}
		if recipe.Timeout == 0 {
			recipe.Timeout = spec.Defaults.Timeout
		}
		spec.Recipes[name] = recipe
	}
}

// validate validates the spec for correctness
func (l *Loader) validate(spec *model.Spec) error {
	if len(spec.Recipes) == 0 {
		return fmt.Errorf("no recipes defined")
	}

	// Validate each recipe
	for name, recipe := range spec.Recipes {
		if err := l.validateRecipe(name, &recipe, spec); err != nil {
			return fmt.Errorf("recipe '%s': %w", name, err)
		}
	}

	return nil
}

// validateRecipe validates a single recipe
func (l *Loader) validateRecipe(name string, recipe *model.Recipe, spec *model.Spec) error {
	if recipe.Run.IsEmpty() && len(recipe.Deps) == 0 {
		return fmt.Errorf("recipe must have either a 'run' section or dependencies")
	}

	// Validate dependencies exist
	for _, dep := range recipe.Deps {
		if _, exists := spec.Recipes[dep]; !exists {
			return fmt.Errorf("dependency '%s' not found", dep)
		}
	}

	// Validate positional arguments
	hasVariadic := false
	for i, pos := range recipe.Positionals {
		if pos.Name == "" {
			return fmt.Errorf("positional argument %d must have a name", i)
		}
		if hasVariadic {
			return fmt.Errorf("variadic positional argument must be last")
		}
		if pos.Variadic {
			hasVariadic = true
		}
	}

	// Validate flags
	for flagName, flag := range recipe.Flags {
		if flag.Type == "" {
			return fmt.Errorf("flag '%s' must specify a type", flagName)
		}
		validTypes := map[string]bool{
			"string": true, "int": true, "bool": true, "string[]": true,
		}
		if !validTypes[flag.Type] {
			return fmt.Errorf("flag '%s' has invalid type '%s' (must be: string, int, bool, string[])", flagName, flag.Type)
		}
	}

	return nil
}

// processIncludes processes include directives with glob support
func (l *Loader) processIncludes(spec *model.Spec, baseDir string) error {
	for _, includePattern := range spec.Include {
		if err := l.processIncludePattern(spec, baseDir, includePattern); err != nil {
			return fmt.Errorf("failed to process include pattern '%s': %w", includePattern, err)
		}
	}
	return nil
}

func (l *Loader) processIncludePattern(spec *model.Spec, baseDir, pattern string) error {
	// Check if this is a remote URL
	if l.isRemoteURL(pattern) {
		return l.processRemoteInclude(spec, pattern)
	}

	// Handle local files
	// Handle relative paths
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(baseDir, pattern)
	}

	// Use filepath.Glob to expand the pattern
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Process each matched file
	for _, matchedFile := range matches {
		if err := l.mergeIncludedFile(spec, matchedFile); err != nil {
			return fmt.Errorf("failed to merge file '%s': %w", matchedFile, err)
		}
	}

	return nil
}

func (l *Loader) mergeIncludedFile(spec *model.Spec, filename string) error {
	// Read the included file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read included file: %w", err)
	}

	// Parse the included spec
	var includedSpec model.Spec
	if err := yaml.Unmarshal(data, &includedSpec); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Merge the specs (later values override earlier ones)
	l.mergeSpecs(spec, &includedSpec)

	return nil
}

func (l *Loader) mergeSpecs(base *model.Spec, included *model.Spec) {
	// Merge environment variables
	if base.Env == nil {
		base.Env = make(map[string]string)
	}
	for k, v := range included.Env {
		base.Env[k] = v
	}

	// Merge variables
	if base.Vars == nil {
		base.Vars = make(map[string]any)
	}
	for k, v := range included.Vars {
		base.Vars[k] = v
	}

	// Merge snippets
	if base.Snippets == nil {
		base.Snippets = make(map[string]string)
	}
	for k, v := range included.Snippets {
		base.Snippets[k] = v
	}

	// Merge recipes
	if base.Recipes == nil {
		base.Recipes = make(map[string]model.Recipe)
	}
	for k, v := range included.Recipes {
		base.Recipes[k] = v
	}

	// Merge shell configurations
	if base.Shell == nil {
		base.Shell = make(map[string]model.ShellConfig)
	}
	for k, v := range included.Shell {
		base.Shell[k] = v
	}

	// Override defaults if specified in included file
	if included.Defaults.WorkingDir != "" {
		base.Defaults.WorkingDir = included.Defaults.WorkingDir
	}
	if included.Defaults.Shell != "" {
		base.Defaults.Shell = included.Defaults.Shell
	}
	if included.Defaults.Timeout != 0 {
		base.Defaults.Timeout = included.Defaults.Timeout
	}
	// Note: boolean fields need special handling since false is a valid value
	if included.Defaults.ExportEnv != base.Defaults.ExportEnv {
		base.Defaults.ExportEnv = included.Defaults.ExportEnv
	}
	if included.Defaults.InheritEnv != base.Defaults.InheritEnv {
		base.Defaults.InheritEnv = included.Defaults.InheritEnv
	}
	if included.Defaults.Strict != base.Defaults.Strict {
		base.Defaults.Strict = included.Defaults.Strict
	}

	// Merge cache configuration
	if included.Cache.Path != "" {
		base.Cache.Path = included.Cache.Path
	}
	if len(included.Cache.Keys) > 0 {
		base.Cache.Keys = append(base.Cache.Keys, included.Cache.Keys...)
	}

	// Merge secrets
	if base.Secrets == nil {
		base.Secrets = make(map[string]model.Secret)
	}
	for k, v := range included.Secrets {
		base.Secrets[k] = v
	}
}

// getCachedSpec retrieves a cached spec if it's still valid
func (l *Loader) getCachedSpec(filePath string) (*model.Spec, bool) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, false
	}

	// Check cache
	if cached, ok := l.cache.Load(filePath); ok {
		entry := cached.(*CacheEntry)
		// Check if file hasn't been modified
		if entry.ModTime.Equal(info.ModTime()) {
			return entry.Spec, true
		}
		// File was modified, remove from cache
		l.cache.Delete(filePath)
	}

	return nil, false
}

// readFileWithCache reads a file with caching based on modification time
func (l *Loader) readFileWithCache(filePath string) ([]byte, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Create cache key with path and modification time
	cacheKey := fmt.Sprintf("%s:%d", filePath, info.ModTime().Unix())

	// Check file content cache
	if cached, ok := l.fileCache.Load(cacheKey); ok {
		return cached.([]byte), nil
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Cache the file content
	l.fileCache.Store(cacheKey, data)

	// Clean up old cache entries for this file
	l.cleanupFileCache(filePath, cacheKey)

	return data, nil
}

// ResolveSecrets resolves secrets from their sources
func (l *Loader) ResolveSecrets(secrets map[string]model.Secret) (map[string]string, error) {
	resolved := make(map[string]string)

	for name, secret := range secrets {
		value, err := l.resolveSecret(name, secret)
		if err != nil {
			if secret.Required {
				return nil, fmt.Errorf("failed to resolve required secret '%s': %w", name, err)
			}
			// Optional secret - skip if not available
			continue
		}
		resolved[name] = value
	}

	return resolved, nil
}

// resolveSecret resolves a single secret from its source
func (l *Loader) resolveSecret(name string, secret model.Secret) (string, error) {
	// Parse the source URL
	sourceURL, err := url.Parse(secret.Source)
	if err != nil {
		return "", fmt.Errorf("invalid secret source format: %w", err)
	}

	switch sourceURL.Scheme {
	case "env":
		// Environment variable source: env://VAR_NAME
		envVar := sourceURL.Host
		if envVar == "" {
			envVar = strings.TrimPrefix(sourceURL.Path, "/")
		}
		value := os.Getenv(envVar)
		if value == "" {
			return "", fmt.Errorf("environment variable '%s' not set", envVar)
		}
		return value, nil

	case "file":
		// File source: file://path/to/secret
		filePath := sourceURL.Path
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(l.baseDir, filePath)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read secret file: %w", err)
		}
		return strings.TrimSpace(string(data)), nil

	case "vault":
		// HashiCorp Vault source: vault://path/to/secret
		// This would require Vault client implementation
		return "", fmt.Errorf("vault:// secrets not yet implemented")

	default:
		return "", fmt.Errorf("unsupported secret source scheme: %s", sourceURL.Scheme)
	}
}

// cleanupFileCache removes old cache entries for a file
func (l *Loader) cleanupFileCache(filePath, currentKey string) {
	// Remove old entries for this file (different modification times)
	l.fileCache.Range(func(key, value any) bool {
		keyStr := key.(string)
		if strings.HasPrefix(keyStr, filePath+":") && keyStr != currentKey {
			l.fileCache.Delete(key)
		}
		return true
	})
}

// cacheSpec stores a spec in the cache
func (l *Loader) cacheSpec(filePath string, spec *model.Spec) {
	info, err := os.Stat(filePath)
	if err != nil {
		return // Don't cache if we can't get file info
	}

	// Create file hash for additional validation
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	entry := &CacheEntry{
		Spec:     spec,
		ModTime:  info.ModTime(),
		FileHash: hash,
	}

	l.cache.Store(filePath, entry)
}

// isRemoteURL checks if a pattern is a remote URL
func (l *Loader) isRemoteURL(pattern string) bool {
	// Check for common URL schemes
	return strings.HasPrefix(pattern, "http://") ||
		strings.HasPrefix(pattern, "https://") ||
		strings.HasPrefix(pattern, "git+") ||
		strings.Contains(pattern, "://")
}

// processRemoteInclude processes a remote include URL
func (l *Loader) processRemoteInclude(spec *model.Spec, includeURL string) error {
	// Parse the URL to determine the type and parameters
	parsedURL, err := l.parseRemoteInclude(includeURL)
	if err != nil {
		return fmt.Errorf("failed to parse remote include URL: %w", err)
	}

	// Fetch the remote content
	data, err := l.fetchRemoteInclude(parsedURL)
	if err != nil {
		return fmt.Errorf("failed to fetch remote include: %w", err)
	}

	// Parse and merge the remote spec
	var includedSpec model.Spec
	if err := yaml.Unmarshal(data, &includedSpec); err != nil {
		return fmt.Errorf("failed to parse remote YAML: %w", err)
	}

	// Merge the specs
	l.mergeSpecs(spec, &includedSpec)

	return nil
}

// RemoteInclude represents a parsed remote include
type RemoteInclude struct {
	Type     string // "http", "git"
	URL      string
	Ref      string // branch, tag, or commit for git
	Path     string // path within the repository
	CacheKey string // unique cache identifier
}

// parseRemoteInclude parses a remote include URL into components
func (l *Loader) parseRemoteInclude(includeURL string) (*RemoteInclude, error) {
	// Handle different URL formats:
	// 1. HTTP/HTTPS: https://raw.githubusercontent.com/org/repo/main/drun.yml
	// 2. Git with path: git+https://github.com/org/repo.git@main:path/to/file.yml
	// 3. Git shorthand: git+https://github.com/org/repo@v1.0.0:drun.yml

	if strings.HasPrefix(includeURL, "git+") {
		return l.parseGitInclude(includeURL)
	} else if strings.HasPrefix(includeURL, "http://") || strings.HasPrefix(includeURL, "https://") {
		return l.parseHTTPInclude(includeURL)
	}

	return nil, fmt.Errorf("unsupported remote include format: %s", includeURL)
}

// parseHTTPInclude parses HTTP/HTTPS URLs
func (l *Loader) parseHTTPInclude(includeURL string) (*RemoteInclude, error) {
	_, err := url.Parse(includeURL)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP URL: %w", err)
	}

	// Create cache key from URL
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(includeURL)))
	cacheKey := fmt.Sprintf("http_%s", hash[:16])

	return &RemoteInclude{
		Type:     "http",
		URL:      includeURL,
		CacheKey: cacheKey,
	}, nil
}

// parseGitInclude parses Git URLs with the format: git+https://github.com/org/repo.git@ref:path
func (l *Loader) parseGitInclude(includeURL string) (*RemoteInclude, error) {
	// Remove git+ prefix
	gitURL := strings.TrimPrefix(includeURL, "git+")

	// Parse ref and path: url@ref:path
	var repoURL, ref, path string

	// Split on @ to separate URL from ref:path
	atIndex := strings.LastIndex(gitURL, "@")
	if atIndex == -1 {
		// No ref specified, use default branch
		repoURL = gitURL
		ref = "main"
		path = "drun.yml" // default filename
	} else {
		repoURL = gitURL[:atIndex]
		refPath := gitURL[atIndex+1:]

		// Split ref:path
		colonIndex := strings.Index(refPath, ":")
		if colonIndex == -1 {
			// No path specified, just ref
			ref = refPath
			path = "drun.yml" // default filename
		} else {
			ref = refPath[:colonIndex]
			path = refPath[colonIndex+1:]
		}
	}

	// Create cache key
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(repoURL+ref+path)))
	cacheKey := fmt.Sprintf("git_%s", hash[:16])

	return &RemoteInclude{
		Type:     "git",
		URL:      repoURL,
		Ref:      ref,
		Path:     path,
		CacheKey: cacheKey,
	}, nil
}

// fetchRemoteInclude fetches content from a remote include
func (l *Loader) fetchRemoteInclude(remote *RemoteInclude) ([]byte, error) {
	// Check cache first
	if cached, ok := l.getCachedRemoteInclude(remote.CacheKey); ok {
		return cached, nil
	}

	var data []byte
	var err error

	switch remote.Type {
	case "http":
		data, err = l.fetchHTTPInclude(remote)
	case "git":
		data, err = l.fetchGitInclude(remote)
	default:
		return nil, fmt.Errorf("unsupported remote include type: %s", remote.Type)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result
	l.cacheRemoteInclude(remote.CacheKey, data)

	return data, nil
}

// fetchHTTPInclude fetches content via HTTP/HTTPS
func (l *Loader) fetchHTTPInclude(remote *RemoteInclude) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(remote.URL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Ignore close error
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response: %w", err)
	}

	return data, nil
}

// fetchGitInclude fetches content from a Git repository
func (l *Loader) fetchGitInclude(remote *RemoteInclude) ([]byte, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(l.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	repoDir := filepath.Join(l.cacheDir, remote.CacheKey)

	// Clone or update repository
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		// Clone repository
		if err := l.cloneGitRepo(remote.URL, repoDir); err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		// Update existing repository
		if err := l.updateGitRepo(repoDir); err != nil {
			return nil, fmt.Errorf("failed to update repository: %w", err)
		}
	}

	// Checkout the specified ref
	if err := l.checkoutGitRef(repoDir, remote.Ref); err != nil {
		return nil, fmt.Errorf("failed to checkout ref %s: %w", remote.Ref, err)
	}

	// Read the specified file
	filePath := filepath.Join(repoDir, remote.Path)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", remote.Path, err)
	}

	return data, nil
}

// cloneGitRepo clones a Git repository
func (l *Loader) cloneGitRepo(repoURL, destDir string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, destDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

// updateGitRepo updates a Git repository
func (l *Loader) updateGitRepo(repoDir string) error {
	cmd := exec.Command("git", "fetch", "--depth", "1")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}
	return nil
}

// checkoutGitRef checks out a specific Git reference
func (l *Loader) checkoutGitRef(repoDir, ref string) error {
	// First try to checkout as a branch
	cmd := exec.Command("git", "checkout", ref)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		// If that fails, try as a tag
		cmd = exec.Command("git", "checkout", "tags/"+ref)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout ref %s: %w", ref, err)
		}
	}
	return nil
}

// getCachedRemoteInclude retrieves cached remote include content
func (l *Loader) getCachedRemoteInclude(cacheKey string) ([]byte, bool) {
	if cached, ok := l.remoteCache.Load(cacheKey); ok {
		if data, ok := cached.([]byte); ok {
			return data, true
		}
	}
	return nil, false
}

// cacheRemoteInclude caches remote include content
func (l *Loader) cacheRemoteInclude(cacheKey string, data []byte) {
	l.remoteCache.Store(cacheKey, data)
}
