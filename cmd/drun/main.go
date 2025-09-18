package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/engine"
	"github.com/spf13/cobra"
)

var (
	configFile  string
	listTasks   bool
	dryRun      bool
	showVersion bool
	initConfig  bool
)

// Version information (set at build time)
var (
	version = "2.0.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

// Default filenames for v2 drun files
var DefaultFilenames = []string{
	".drun/default.drun",
	".ops/drun/spec.drun",
	"ops.drun",
	"spec.drun",
	"drun.drun",
	".drun.drun",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "drun [task] [args...]",
	Short: "A semantic task runner with natural language syntax",
	Long: `drun v2 is a task runner that uses semantic, English-like syntax to define automation tasks.
It supports natural language commands, smart detection, and direct execution without compilation.

Examples:
  drun hello                    # Run the 'hello' task
  drun build --env=production   # Run 'build' task with environment
  drun --list                   # List all available tasks
  drun --init                   # Create a new drun file`,
	RunE: runDrun,
	// Don't treat unknown arguments as errors
	Args: cobra.ArbitraryArgs,
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "file", "f", "", "Configuration file (default: auto-detect .drun files)")
	rootCmd.Flags().BoolVarP(&listTasks, "list", "l", false, "List available tasks")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be executed without running")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "Initialize a new drun configuration file")
}

func runDrun(cmd *cobra.Command, args []string) error {
	// Handle --version flag
	if showVersion {
		return showVersionInfo()
	}

	// Handle --init flag
	if initConfig {
		return initializeConfig(configFile)
	}

	// Determine the config file to use
	actualConfigFile, err := findConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("no drun configuration file found: %w\n\nTo get started:\n  drun --init          # Create a starter configuration", err)
	}

	// Read the drun file
	content, err := os.ReadFile(actualConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read drun file '%s': %w", actualConfigFile, err)
	}

	// Parse the drun file
	program, err := engine.ParseString(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse drun file '%s': %w", actualConfigFile, err)
	}

	// Create engine
	eng := engine.NewEngine(os.Stdout)
	eng.SetDryRun(dryRun)

	// Handle --list flag
	if listTasks {
		return listAllTasks(eng, program)
	}

	// Determine target task and parse parameters
	var target string
	var params map[string]string

	if len(args) == 0 {
		// No arguments - try to find a default task or list tasks
		if defaultTask := findDefaultTask(program); defaultTask != "" {
			target = defaultTask
		} else {
			return listAllTasks(eng, program)
		}
		params = make(map[string]string)
	} else {
		target = args[0]
		params = parseTaskParameters(args[1:])
	}

	// Execute the task with parameters
	return eng.ExecuteWithParams(program, target, params)
}

// findConfigFile finds the drun configuration file to use
func findConfigFile(filename string) (string, error) {
	if filename != "" {
		// User specified a file
		if _, err := os.Stat(filename); err != nil {
			return "", fmt.Errorf("specified file '%s' not found", filename)
		}
		return filename, nil
	}

	// Auto-detect configuration file
	for _, defaultName := range DefaultFilenames {
		if _, err := os.Stat(defaultName); err == nil {
			return defaultName, nil
		}
	}

	return "", fmt.Errorf("no drun configuration file found in current directory")
}

// listAllTasks lists all available tasks
func listAllTasks(eng *engine.Engine, program *ast.Program) error {
	fmt.Println("Available tasks:")

	tasks := eng.ListTasks(program)
	if len(tasks) == 0 {
		fmt.Println("  (no tasks defined)")
		return nil
	}

	for _, task := range tasks {
		fmt.Printf("  %-20s %s\n", task.Name, task.Description)
	}

	return nil
}

// findDefaultTask finds a default task to run
func findDefaultTask(program *ast.Program) string {
	// Look for common default task names
	defaultNames := []string{"default", "help", "start", "main"}

	for _, task := range program.Tasks {
		for _, defaultName := range defaultNames {
			if task.Name == defaultName {
				return task.Name
			}
		}
	}

	return ""
}

// parseTaskParameters parses task parameters from command line arguments
// Supports format: param1=value1 param2=value2
func parseTaskParameters(args []string) map[string]string {
	params := make(map[string]string)

	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}
	}

	return params
}

// showVersionInfo displays version information
func showVersionInfo() error {
	fmt.Printf("drun version %s\n", version)
	if commit != "unknown" {
		fmt.Printf("commit: %s\n", commit)
	}
	if date != "unknown" {
		fmt.Printf("built: %s\n", date)
	}
	return nil
}

// initializeConfig creates a new drun configuration file
func initializeConfig(filename string) error {
	// Determine the target filename
	targetFile := "spec.drun"
	if filename != "" {
		targetFile = filename
	}

	// Check if file already exists
	if _, err := os.Stat(targetFile); err == nil {
		return fmt.Errorf("configuration file '%s' already exists", targetFile)
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
	if err := os.WriteFile(targetFile, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("✅ Created %s\n", targetFile)
	fmt.Println("🚀 Get started with: drun --list")
	return nil
}

// generateStarterConfig creates a starter drun v2 configuration
func generateStarterConfig() string {
	return `version: 2.0

task "default":
  info "Welcome to drun v2! 🚀"
  step "This is your starter configuration"
  success "Ready to build amazing automation!"

task "hello":
  info "Hello from the semantic task runner!"

task "build":
  step "Building project..."
  info "Add your build commands here"
  success "Build completed!"

task "test":
  step "Running tests..."
  info "Add your test commands here"
  success "All tests passed!"

task "deploy":
  step "Deploying application..."
  warn "Make sure you're deploying to the right environment!"
  info "Add your deployment commands here"
  success "Deployment completed!"
`
}
