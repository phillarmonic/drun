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

// handleSelfUpdate handles the --self-update flag
func handleSelfUpdate() error {
	fmt.Println("🔄 Checking for drun updates...")

	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Check for latest version
	latestVersion, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Compare versions
	currentVersion := normalizeVersion(version)
	if currentVersion == latestVersion {
		fmt.Printf("✅ You're already running the latest version: %s\n", version)
		return nil
	}

	fmt.Printf("📦 New version available: %s (current: %s)\n", latestVersion, version)

	// Ask for user confirmation
	if !askForConfirmation("Do you want to update now?") {
		fmt.Println("Update cancelled.")
		return nil
	}

	// Create backup
	backupPath, err := createBackup(currentExe)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Printf("💾 Created backup at: %s\n", backupPath)

	// Download and install new version
	if err := downloadAndInstall(latestVersion, currentExe); err != nil {
		// Restore backup on failure
		fmt.Printf("❌ Update failed: %v\n", err)
		fmt.Println("🔄 Restoring backup...")
		if restoreErr := restoreBackup(backupPath, currentExe); restoreErr != nil {
			return fmt.Errorf("update failed and backup restoration failed: %v (original error: %w)", restoreErr, err)
		}
		fmt.Println("✅ Backup restored successfully")
		return err
	}

	fmt.Printf("🎉 Successfully updated to version %s!\n", latestVersion)
	fmt.Printf("💾 Backup available at: %s\n", backupPath)

	return nil
}

// getLatestVersion fetches the latest version from GitHub
func getLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://api.github.com/repos/phillarmonic/drun/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch release information: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release information: %w", err)
	}

	return release.TagName, nil
}

// normalizeVersion removes 'v' prefix and '-dev' suffix for comparison
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimSuffix(v, "-dev")
	return v
}

// askForConfirmation asks the user for yes/no confirmation
func askForConfirmation(question string) bool {
	fmt.Printf("%s (y/N): ", question)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

// createBackup creates a backup of the current executable
func createBackup(currentExe string) (string, error) {
	// Create backup directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	backupDir := filepath.Join(homeDir, ".drun")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create timestamped backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("xdrun_%s_backup_%s", normalizeVersion(version), timestamp)
	if runtime.GOOS == "windows" {
		backupFilename += ".exe"
	}

	backupPath := filepath.Join(backupDir, backupFilename)

	// Copy current executable to backup location
	if err := copyFile(currentExe, backupPath); err != nil {
		return "", fmt.Errorf("failed to copy executable: %w", err)
	}

	// Make backup executable
	if err := os.Chmod(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make backup executable: %w", err)
	}

	// Clean up old backups (keep last 5)
	cleanupOldBackups(backupDir)

	return backupPath, nil
}

// downloadAndInstall downloads and installs the new version
func downloadAndInstall(version, targetPath string) error {
	// Determine platform and architecture
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go arch to release arch
	var arch string
	switch goarch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return fmt.Errorf("unsupported architecture: %s", goarch)
	}

	// Construct download URL
	binaryName := fmt.Sprintf("xdrun-%s-%s", goos, arch)
	if goos == "windows" {
		binaryName += ".exe"
	}

	downloadURL := fmt.Sprintf("https://github.com/phillarmonic/drun/releases/download/%s/%s", version, binaryName)

	fmt.Printf("📥 Downloading %s...\n", binaryName)

	// Download the binary
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "drun-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(tempFile.Name()); removeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file: %v\n", removeErr)
		}
	}()
	defer func() {
		if closeErr := tempFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close temporary file: %v\n", closeErr)
		}
	}()

	// Copy downloaded content to temp file
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write downloaded binary: %w", err)
	}

	// Make temp file executable
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Verify the binary works
	fmt.Println("🔍 Verifying downloaded binary...")
	cmd := exec.Command(tempFile.Name(), "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("downloaded binary failed verification: %w", err)
	}

	// Install the binary (may require elevated permissions)
	fmt.Println("📦 Installing new version...")
	if err := installBinary(tempFile.Name(), targetPath); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	return nil
}

// installBinary installs the binary, handling permissions as needed
func installBinary(sourcePath, targetPath string) error {
	// Try direct copy first
	if err := copyFile(sourcePath, targetPath); err == nil {
		return nil
	}

	// If direct copy failed, try with elevated permissions
	fmt.Println("🔐 Requesting elevated permissions...")

	switch runtime.GOOS {
	case "darwin", "linux":
		// Use sudo on Unix-like systems
		cmd := exec.Command("sudo", "cp", sourcePath, targetPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case "windows":
		// On Windows, we need to use PowerShell with elevation
		// This is more complex and might require the user to run as administrator
		return copyFile(sourcePath, targetPath)

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close source file: %v\n", closeErr)
		}
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := destFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close destination file: %v\n", closeErr)
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// restoreBackup restores from backup
func restoreBackup(backupPath, targetPath string) error {
	return copyFile(backupPath, targetPath)
}

// cleanupOldBackups removes old backup files, keeping only the last 5
func cleanupOldBackups(backupDir string) {
	files, err := filepath.Glob(filepath.Join(backupDir, "xdrun*backup*"))
	if err != nil {
		return
	}

	if len(files) <= 5 {
		return
	}

	// Sort files by modification time (newest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var fileInfos []fileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			path:    file,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.Before(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove old files (keep first 5)
	for i := 5; i < len(fileInfos); i++ {
		if removeErr := os.Remove(fileInfos[i].path); removeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove old backup %s: %v\n", fileInfos[i].path, removeErr)
		}
	}
}
