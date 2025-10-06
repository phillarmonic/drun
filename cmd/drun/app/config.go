package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Domain: Configuration Management
// This file contains logic for file discovery, workspace config, and initialization

// DefaultFilename is the default filename for v2 drun files
const DefaultFilename = ".drun/spec.drun"

// WorkspaceConfig represents the workspace configuration
type WorkspaceConfig struct {
	DefaultTaskFile string            `yaml:"defaultTaskFile"`
	ParallelJobs    int               `yaml:"parallelJobs"`
	Shell           string            `yaml:"shell"`
	Variables       map[string]string `yaml:"variables"`
	Defaults        map[string]string `yaml:"defaults"`
}

// findConfigFile finds the drun configuration file to use
func FindConfigFile(filename string) (string, error) {
	if filename != "" {
		// User specified a file
		if _, err := os.Stat(filename); err != nil {
			return "", fmt.Errorf("specified file '%s' not found", filename)
		}
		return filename, nil
	}

	// Check workspace configuration first
	if workspaceFile := getWorkspaceDefaultFile(); workspaceFile != "" {
		if _, err := os.Stat(workspaceFile); err == nil {
			return workspaceFile, nil
		} else {
			return "", fmt.Errorf("workspace default file '%s' not found", workspaceFile)
		}
	}

	// Try default file locations in order
	defaultLocations := []string{
		".drun/spec.drun",
		".drun",
		"spec.drun",
		"ops/drun/spec.drun",
		"ops/spec.drun",
	}

	for _, location := range defaultLocations {
		if fileInfo, err := os.Stat(location); err == nil {
			// Skip if it's a directory - we only want files
			if !fileInfo.IsDir() {
				return location, nil
			}
		}
	}

	return "", fmt.Errorf("no drun task file found - expected one of: %v\nUse --file to specify location or run 'drun --init' to create one", defaultLocations)
}

// getWorkspaceDefaultFile checks for workspace configuration and returns default file
func getWorkspaceDefaultFile() string {
	workspaceConfigPath := ".drun/.drun_workspace.yml"
	if _, err := os.Stat(workspaceConfigPath); err != nil {
		return ""
	}

	// Read and parse workspace configuration
	data, err := os.ReadFile(workspaceConfigPath)
	if err != nil {
		return ""
	}

	var config WorkspaceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return ""
	}

	// Return the default task file if specified
	if config.DefaultTaskFile != "" {
		return config.DefaultTaskFile
	}

	return ""
}

// saveWorkspaceConfig saves a workspace configuration
func saveWorkspaceConfig(config WorkspaceConfig) error {
	workspaceConfigPath := ".drun/.drun_workspace.yml"

	// Create .drun directory if it doesn't exist
	if err := os.MkdirAll(".drun", 0755); err != nil {
		return fmt.Errorf("failed to create .drun directory: %w", err)
	}

	// Marshal configuration to YAML
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(workspaceConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write workspace config: %w", err)
	}

	return nil
}

// loadWorkspaceConfig loads the workspace configuration
func loadWorkspaceConfig() (*WorkspaceConfig, error) {
	workspaceConfigPath := ".drun/.drun_workspace.yml"
	if _, err := os.Stat(workspaceConfigPath); err != nil {
		// Return default config if file doesn't exist
		return &WorkspaceConfig{
			ParallelJobs: 4,
			Shell:        "/bin/bash",
			Variables:    make(map[string]string),
			Defaults:     make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(workspaceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace config: %w", err)
	}

	var config WorkspaceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse workspace config: %w", err)
	}

	// Set defaults if not specified
	if config.ParallelJobs == 0 {
		config.ParallelJobs = 4
	}
	if config.Shell == "" {
		config.Shell = "/bin/bash"
	}
	if config.Variables == nil {
		config.Variables = make(map[string]string)
	}
	if config.Defaults == nil {
		config.Defaults = make(map[string]string)
	}

	return &config, nil
}

// initializeConfig creates a new drun configuration file
func InitializeConfig(filename string, saveAsDefault bool) error {
	// Determine the target filename
	targetFile := ".drun/spec.drun"
	if filename != "" {
		targetFile = filename
	}

	// Check if file already exists
	if _, err := os.Stat(targetFile); err == nil {
		return fmt.Errorf("task file '%s' already exists", targetFile)
	}

	// Check if the directory needs to be created
	targetDir := filepath.Dir(targetFile)
	if targetDir != "." && targetDir != "" {
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			// Create the directory
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory '%s': %w", targetDir, err)
			}
			fmt.Printf("📁 Created directory: %s\n", targetDir)
		}
	}

	// Generate starter configuration
	config := generateStarterConfig()

	// Write the file
	if err := os.WriteFile(targetFile, []byte(config), 0600); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	fmt.Printf("✅ Created %s\n", targetFile)

	// Save as workspace default if requested or if using custom filename
	if saveAsDefault || (filename != "" && filename != ".drun/spec.drun") {
		if err := saveCustomFileAsDefault(targetFile); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save as workspace default: %v\n", err)
		} else {
			fmt.Printf("💾 Saved '%s' as workspace default\n", targetFile)
		}
	}

	fmt.Println("🚀 Get started with: xdrun --list")
	return nil
}

// saveCustomFileAsDefault saves a custom file name as the workspace default
func saveCustomFileAsDefault(filename string) error {
	// Load existing workspace config or create new one
	config, err := loadWorkspaceConfig()
	if err != nil {
		config = &WorkspaceConfig{
			ParallelJobs: 4,
			Shell:        "/bin/bash",
			Variables:    make(map[string]string),
			Defaults:     make(map[string]string),
		}
	}

	// Set the default task file
	config.DefaultTaskFile = filename

	// Save the updated configuration
	return saveWorkspaceConfig(*config)
}

// SetWorkspaceDefault sets the workspace default task file
func SetWorkspaceDefault(filename string) error {
	// Check if the specified file exists
	if _, err := os.Stat(filename); err != nil {
		return fmt.Errorf("specified file '%s' not found", filename)
	}

	// Load existing workspace config or create new one
	config, err := loadWorkspaceConfig()
	if err != nil {
		config = &WorkspaceConfig{
			ParallelJobs: 4,
			Shell:        "/bin/bash",
			Variables:    make(map[string]string),
			Defaults:     make(map[string]string),
		}
	}

	// Set the default task file
	config.DefaultTaskFile = filename

	// Save the updated configuration
	if err := saveWorkspaceConfig(*config); err != nil {
		return fmt.Errorf("failed to save workspace configuration: %w", err)
	}

	fmt.Printf("✅ Set workspace default task file to: %s\n", filename)
	fmt.Printf("💾 Saved to .drun/.drun_workspace.yml\n")
	return nil
}

// generateStarterConfig creates a starter drun v2 configuration
func generateStarterConfig() string {
	return `# drun (do-run) CLI is a fast, semantic task runner with 
# its own powerful automation language. Effortless tasks, serious speed.
# Learn more at https://github.com/phillarmonic/drun

version: 2.0

project "my-app" version "1.0":
	/* Cross-platform shell configuration with sensible defaults
	 These are all default values, you can remove them if you don't intend to change it. */

	shell config:
		darwin:
			executable: "/bin/zsh"
			args:
				- "-l"
				- "-i"
			environment:
				TERM: "xterm-256color"
				SHELL_SESSION_HISTORY: "0"
		
		linux:
			executable: "/bin/bash"
			args:
				- "--login"
				- "--interactive"
			environment:
				TERM: "xterm-256color"
				HISTCONTROL: "ignoredups"
		
		windows:
			executable: "powershell.exe"
			args:
				- "-NoProfile"
				- "-ExecutionPolicy"
				- "Bypass"
			environment:
				PSModulePath: ""

task "default" means "Welcome to drun v2":
	echo "Starting up..."
	info "Welcome to drun v2! 🚀"
	step "This is your starter task file"
	success "Ready to build amazing automation!"

task "hello" means "Say hello":
	info "Hello from the semantic task runner!"

task "build" means "Build the project":
	step "Building project..."
	info "Add your build commands here"
	success "Build completed!"

task "test" means "Run tests":
	step "Running tests..."
	info "Add your test commands here"
	success "All tests passed!"

task "deploy" means "Deploy application":
	given $environment defaults to "development"
	step "Deploying application to {$environment}..."
	warn "Make sure you're deploying to the right environment!"
	info "Add your deployment commands here"
	success "Deployment to {$environment} completed!"
`
}
