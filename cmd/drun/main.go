package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/debug"
	"github.com/phillarmonic/drun/internal/engine"
	"github.com/phillarmonic/drun/internal/errors"
	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
	"github.com/phillarmonic/figlet/figletlib"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configFile         string
	listTasks          bool
	dryRun             bool
	verbose            bool
	showVersion        bool
	initConfig         bool
	saveAsDefault      bool
	setWorkspace       string
	selfUpdate         bool
	allowUndefinedVars bool
	noDrunCache        bool // Disable remote include caching
	// Debug flags
	debugMode   bool
	debugTokens bool
	debugAST    bool
	debugJSON   bool
	debugErrors bool
	debugFull   bool
	debugInput  string
)

// WorkspaceConfig represents the workspace configuration
type WorkspaceConfig struct {
	DefaultTaskFile string            `yaml:"defaultTaskFile"`
	ParallelJobs    int               `yaml:"parallelJobs"`
	Shell           string            `yaml:"shell"`
	Variables       map[string]string `yaml:"variables"`
	Defaults        map[string]string `yaml:"defaults"`
}

// Version information (set at build time)
var (
	version = "2.0.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

// Default filename for v2 drun files
var DefaultFilename = ".drun/spec.drun"

// completeTaskNames provides autocompletion for task names
func completeTaskNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Try to find and parse the drun file
	actualConfigFile, err := findConfigFile(configFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	content, err := os.ReadFile(actualConfigFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	program, err := engine.ParseStringWithFilename(string(content), actualConfigFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Create engine to get task list
	eng := engine.NewEngine(os.Stdout)
	tasks := eng.ListTasks(program)

	var completions []string
	for _, task := range tasks {
		completions = append(completions, task.Name+"\t[task] "+task.Description)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "[xdrun CLI cmd] Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(xdrun completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ xdrun completion bash > /etc/bash_completion.d/xdrun
  # macOS:
  $ xdrun completion bash > $(brew --prefix)/etc/bash_completion.d/xdrun

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ xdrun completion zsh > "${fpath[1]}/_xdrun"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ xdrun completion fish | source

  # To load completions for each session, execute once:
  $ xdrun completion fish > ~/.config/fish/completions/xdrun.fish

PowerShell:

  PS> xdrun completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> xdrun completion powershell > xdrun.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			_ = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			_ = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			_ = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "xdrun [task] [args...]",
	Short: "[xdrun CLI cmd] Execute drun automation language",
	Long: `xdrun is the CLI interpreter for the drun automation language.

drun uses semantic, English-like syntax to define automation tasks.
It supports natural language commands, smart detection, and direct execution without compilation.

Examples:
  xdrun hello                    # Run the 'hello' task from a .drun file
  xdrun build --env=production   # Run 'build' task with environment
  xdrun --list                   # List all available tasks
  xdrun --init                   # Create a new .drun file
  xdrun --debug --tokens         # Debug lexer tokens
  xdrun --debug --ast            # Debug AST structure
  xdrun --debug --full           # Full debug output`,
	RunE: runDrun,
	// Don't treat unknown arguments as errors
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: completeTaskNames,
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "file", "f", "", "[xdrun CLI cmd] Task file (default: .drun/spec.drun or workspace configured file)")
	rootCmd.Flags().BoolVarP(&listTasks, "list", "l", false, "[xdrun CLI cmd] List available tasks")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "[xdrun CLI cmd] Show what would be executed without running")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "[xdrun CLI cmd] Show detailed execution information")
	rootCmd.Flags().BoolVar(&noDrunCache, "no-drun-cache", false, "[xdrun CLI cmd] Disable remote include caching (always fetch)")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "[xdrun CLI cmd] Show version information")
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "[xdrun CLI cmd] Initialize a new .drun task file")
	rootCmd.Flags().BoolVar(&saveAsDefault, "save-as-default", false, "[xdrun CLI cmd] Save custom file name as workspace default (use with --init)")
	rootCmd.Flags().StringVar(&setWorkspace, "set-workspace", "", "[xdrun CLI cmd] Set workspace default task file location")
	rootCmd.Flags().BoolVar(&selfUpdate, "self-update", false, "[xdrun CLI cmd] Check for updates and update xdrun to the latest version")
	rootCmd.Flags().BoolVar(&allowUndefinedVars, "allow-undefined-variables", false, "[xdrun CLI cmd] Allow undefined variables in interpolation (default: strict mode)")

	// Debug flags
	rootCmd.Flags().BoolVar(&debugMode, "debug", false, "[xdrun CLI cmd] Enable debug mode - shows tokens, AST, and parse information")
	rootCmd.Flags().BoolVar(&debugTokens, "debug-tokens", false, "[xdrun CLI cmd] Show lexer tokens (requires --debug)")
	rootCmd.Flags().BoolVar(&debugAST, "debug-ast", false, "[xdrun CLI cmd] Show AST structure (requires --debug)")
	rootCmd.Flags().BoolVar(&debugJSON, "debug-json", false, "[xdrun CLI cmd] Show AST as JSON (requires --debug)")
	rootCmd.Flags().BoolVar(&debugErrors, "debug-errors", false, "[xdrun CLI cmd] Show parse errors only (requires --debug)")
	rootCmd.Flags().BoolVar(&debugFull, "debug-full", false, "[xdrun CLI cmd] Show full debug output (requires --debug)")
	rootCmd.Flags().StringVar(&debugInput, "debug-input", "", "[xdrun CLI cmd] Debug input string directly instead of file (requires --debug)")

	// Add completion commands
	rootCmd.AddCommand(completionCmd)
}

func runDrun(cmd *cobra.Command, args []string) error {
	// Handle --version flag
	if showVersion {
		return showVersionInfo()
	}

	// Handle --self-update flag
	if selfUpdate {
		return handleSelfUpdate()
	}

	// Handle --init flag
	if initConfig {
		return initializeConfig(configFile)
	}

	// Handle --set-workspace flag
	if setWorkspace != "" {
		return setWorkspaceDefault(setWorkspace)
	}

	// Handle debug mode
	if debugMode {
		return handleDebugMode()
	}

	// Determine the config file to use
	actualConfigFile, err := findConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("no drun task file found: %w\n\nTo get started:\n  drun --init          # Create .drun/spec.drun", err)
	}

	// Verbose: Show we're starting
	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "📂 Loading: %s\n", actualConfigFile)
	}

	// Read the drun file
	content, err := os.ReadFile(actualConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read drun file '%s': %w", actualConfigFile, err)
	}

	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "🔍 Parsing drun file...\n")
	}

	// Parse the drun file
	program, err := engine.ParseStringWithFilename(string(content), actualConfigFile)
	if err != nil {
		// Check if it's an enhanced error list
		if errorList, ok := err.(*errors.ParseErrorList); ok {
			fmt.Fprint(os.Stderr, errorList.FormatErrors())
			os.Exit(1)
		}
		// Fallback to regular error
		return fmt.Errorf("failed to parse drun file '%s': %w", actualConfigFile, err)
	}

	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "✅ Parsed successfully\n")
	}

	// Create engine
	eng := engine.NewEngine(os.Stdout)
	eng.SetDryRun(dryRun)
	eng.SetVerbose(verbose)
	eng.SetAllowUndefinedVars(allowUndefinedVars)

	if verbose {
		if noDrunCache {
			_, _ = fmt.Fprintf(os.Stdout, "💾 Remote include caching: disabled\n")
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "💾 Remote include caching: enabled (1m expiration)\n")
		}
	}

	// Initialize cache for remote includes (respect --no-drun-cache flag)
	if err := eng.SetCacheEnabled(!noDrunCache); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize remote include cache: %v\n", err)
	}

	// Ensure cleanup of temporary files
	defer eng.Cleanup()

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
	err = eng.ExecuteWithParamsAndFile(program, target, params, actualConfigFile)
	if err != nil {
		// Check if it's a parameter validation error (don't show usage)
		if paramErr, ok := err.(*errors.ParameterValidationError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", paramErr.Message)
			os.Exit(1)
		}
		// For task execution errors, don't show usage - just print error and exit
		fmt.Fprintf(os.Stderr, "Error: execution failed: %v\n", err)
		os.Exit(1)
	}
	return nil
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

// listAllTasks lists all available tasks
func listAllTasks(eng *engine.Engine, program *ast.Program) error {
	fmt.Println("Available tasks:")

	tasks := eng.ListTasks(program)
	if len(tasks) == 0 {
		fmt.Println("  (no tasks defined)")
		return nil
	}

	for _, task := range tasks {
		fmt.Printf("  %-20s  %s\n", task.Name, task.Description)
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
	loader := figletlib.NewEmbededLoader()
	font, err := loader.GetFontByName("standard")
	startColor, _ := figletlib.ParseColor("#00FF95")
	endColor, _ := figletlib.ParseColor("#00C2FF")
	gradientConfig := figletlib.ColorConfig{
		Mode:       figletlib.ColorModeGradient,
		StartColor: startColor,
		EndColor:   endColor,
	}
	if err != nil {
		panic(err)
	}

	fmt.Println("")
	figletlib.PrintColoredMsg("dRun CLI", font, 80, font.Settings(), "left", gradientConfig)

	fmt.Println("drun (do-run) automation language")
	fmt.Println("xDrun (eXecute drun) CLI")
	fmt.Println()
	fmt.Println("Effortless tasks, serious speed.")
	fmt.Println("By Phillarmonic Software <https://github.com/phillarmonic/drun>")
	fmt.Println("")
	fmt.Printf("Version %s\n", version)
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

// setWorkspaceDefault sets the workspace default task file
func setWorkspaceDefault(filename string) error {
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

// handleDebugMode handles debug mode execution
func handleDebugMode() error {
	var content string

	// Get content from input string or file
	if debugInput != "" {
		content = debugInput
	} else {
		// Determine the config file to use
		actualConfigFile, err := findConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("no drun task file found for debugging: %w\n\nTo get started:\n  drun --init          # Create .drun/spec.drun", err)
		}

		// Read the drun file
		data, err := os.ReadFile(actualConfigFile)
		if err != nil {
			return fmt.Errorf("failed to read drun file '%s': %w", actualConfigFile, err)
		}
		content = string(data)
	}

	// Handle specific debug flags
	if debugFull {
		debug.DebugFull(content)
		return nil
	}

	// Handle individual debug flags
	hasSpecificFlag := debugTokens || debugAST || debugJSON || debugErrors

	if debugTokens {
		debug.DebugTokens(content)
	}

	if debugAST || debugJSON || debugErrors {
		// Parse without full debug output
		l := lexer.NewLexer(content)
		p := parser.NewParser(l)
		program := p.ParseProgram()
		parseErrors := p.Errors()

		if debugErrors {
			debug.DebugParseErrors(parseErrors)
		}

		if debugAST {
			debug.DebugAST(program)
		}

		if debugJSON {
			debug.DebugJSON(program)
		}
	}

	// If no specific debug flags were set, show full debug by default
	if !hasSpecificFlag {
		debug.DebugFull(content)
	}

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

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
}
