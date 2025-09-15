package spec

import (
	"crypto/sha256"
	"fmt"
	"os"
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
	baseDir   string
	cache     sync.Map // Cache specs by file path
	fileCache sync.Map // Cache file contents by path+modtime
}

// NewLoader creates a new spec loader
func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
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
