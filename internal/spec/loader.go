package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// Loader handles loading and validating drun specifications
type Loader struct {
	baseDir string
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

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var spec model.Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", filePath, err)
	}

	// Set defaults
	l.setDefaults(&spec)

	// Validate the spec
	if err := l.validate(&spec); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", filePath, err)
	}

	// Process includes if any
	if len(spec.Include) > 0 {
		if err := l.processIncludes(&spec, filepath.Dir(filePath)); err != nil {
			return nil, fmt.Errorf("failed to process includes: %w", err)
		}
	}

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
	if spec.Recipes == nil || len(spec.Recipes) == 0 {
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

// processIncludes processes include directives (placeholder for now)
func (l *Loader) processIncludes(spec *model.Spec, baseDir string) error {
	// TODO: Implement include processing with glob support
	// For MVP, we'll skip this feature
	return nil
}
