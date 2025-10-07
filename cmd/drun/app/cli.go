package app

import (
	"os"

	"github.com/spf13/cobra"
)

// Domain: CLI Application Structure
// This file contains the main CLI application setup with Cobra commands and flags

// App represents the CLI application
type App struct {
	version string
	commit  string
	date    string

	rootCmd *cobra.Command

	// Flags
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
	noDrunCache        bool

	// Debug flags
	debugMode   bool
	debugTokens bool
	debugAST    bool
	debugJSON   bool
	debugErrors bool
	debugFull   bool
	debugDomain bool
	debugInput  string
}

// NewApp creates a new CLI application
func NewApp(version, commit, date string) *App {
	app := &App{
		version: version,
		commit:  commit,
		date:    date,
	}

	app.rootCmd = &cobra.Command{
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
  xdrun --debug --full           # Full debug output

Built-in Commands:
  Use the 'cmd:' prefix for built-in commands to avoid conflicts with tasks:
  xdrun cmd:completion bash      # Generate shell completion
  xdrun cmd:from makefile        # Convert Makefile to drun`,
		RunE:              app.run,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: CompleteTaskNames,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true, // Disable default 'completion' command
		},
	}

	app.setupFlags()
	app.setupCommands()

	return app
}

// Execute runs the CLI application
func (a *App) Execute() error {
	return a.rootCmd.Execute()
}

// setupFlags sets up all command-line flags
func (a *App) setupFlags() {
	flags := a.rootCmd.Flags()

	flags.StringVarP(&a.configFile, "file", "f", "", "[xdrun CLI cmd] Task file (default: .drun/spec.drun or workspace configured file)")
	flags.BoolVarP(&a.listTasks, "list", "l", false, "[xdrun CLI cmd] List available tasks")
	flags.BoolVar(&a.dryRun, "dry-run", false, "[xdrun CLI cmd] Show what would be executed without running")
	flags.BoolVarP(&a.verbose, "verbose", "v", false, "[xdrun CLI cmd] Show detailed execution information")
	flags.BoolVar(&a.noDrunCache, "no-drun-cache", false, "[xdrun CLI cmd] Disable remote include caching (always fetch)")
	flags.BoolVar(&a.showVersion, "version", false, "[xdrun CLI cmd] Show version information")
	flags.BoolVar(&a.initConfig, "init", false, "[xdrun CLI cmd] Initialize a new .drun task file")
	flags.BoolVar(&a.saveAsDefault, "save-as-default", false, "[xdrun CLI cmd] Save custom file name as workspace default (use with --init)")
	flags.StringVar(&a.setWorkspace, "set-workspace", "", "[xdrun CLI cmd] Set workspace default task file location")
	flags.BoolVar(&a.selfUpdate, "self-update", false, "[xdrun CLI cmd] Check for updates and update xdrun to the latest version")
	flags.BoolVar(&a.allowUndefinedVars, "allow-undefined-variables", false, "[xdrun CLI cmd] Allow undefined variables in interpolation (default: strict mode)")

	// Debug flags
	flags.BoolVar(&a.debugMode, "debug", false, "[xdrun CLI cmd] Enable debug mode - shows tokens, AST, and parse information")
	flags.BoolVar(&a.debugTokens, "debug-tokens", false, "[xdrun CLI cmd] Show lexer tokens (requires --debug)")
	flags.BoolVar(&a.debugAST, "debug-ast", false, "[xdrun CLI cmd] Show AST structure (requires --debug)")
	flags.BoolVar(&a.debugJSON, "debug-json", false, "[xdrun CLI cmd] Show AST as JSON (requires --debug)")
	flags.BoolVar(&a.debugErrors, "debug-errors", false, "[xdrun CLI cmd] Show parse errors only (requires --debug)")
	flags.BoolVar(&a.debugFull, "debug-full", false, "[xdrun CLI cmd] Show full debug output (requires --debug)")
	flags.BoolVar(&a.debugDomain, "debug-domain", false, "[xdrun CLI cmd] Show domain layer information (task registry, dependencies)")
	flags.StringVar(&a.debugInput, "debug-input", "", "[xdrun CLI cmd] Debug input string directly instead of file (requires --debug)")
}

// setupCommands sets up subcommands
func (a *App) setupCommands() {
	a.rootCmd.AddCommand(a.createCompletionCommand())
	a.rootCmd.AddCommand(a.createConvertCommand())
}

// run is the main command handler
func (a *App) run(cmd *cobra.Command, args []string) error {
	// Handle --version flag
	if a.showVersion {
		return ShowVersion(a.version, a.commit, a.date)
	}

	// Handle --self-update flag
	if a.selfUpdate {
		return HandleSelfUpdate(a.version)
	}

	// Handle --init flag
	if a.initConfig {
		return InitializeConfig(a.configFile, a.saveAsDefault)
	}

	// Handle --set-workspace flag
	if a.setWorkspace != "" {
		return SetWorkspaceDefault(a.setWorkspace)
	}

	// Handle debug mode
	if a.debugMode {
		return HandleDebugMode(
			a.configFile,
			a.debugInput,
			a.debugFull,
			a.debugTokens,
			a.debugAST,
			a.debugJSON,
			a.debugErrors,
			a.debugDomain,
		)
	}

	// Normal execution - run task
	return ExecuteTask(
		a.configFile,
		a.listTasks,
		a.dryRun,
		a.verbose,
		a.allowUndefinedVars,
		a.noDrunCache,
		args,
	)
}

// createCompletionCommand creates the cmd:completion subcommand
func (a *App) createCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cmd:completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `Generate shell completion script for xdrun.

Note: The 'cmd:' prefix is reserved for built-in commands to avoid conflicts with user tasks.

To load completions:

Bash:

  $ source <(xdrun cmd:completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ xdrun cmd:completion bash > /etc/bash_completion.d/xdrun
  # macOS:
  $ xdrun cmd:completion bash > $(brew --prefix)/etc/bash_completion.d/xdrun

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ xdrun cmd:completion zsh > "${fpath[1]}/_xdrun"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ xdrun cmd:completion fish | source

  # To load completions for each session, execute once:
  $ xdrun cmd:completion fish > ~/.config/fish/completions/xdrun.fish

PowerShell:

  PS> xdrun cmd:completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> xdrun cmd:completion powershell > xdrun.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				_ = a.rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				_ = a.rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				_ = a.rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				_ = a.rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}
