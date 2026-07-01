package app

import (
	"bytes"
	"fmt"
	"os"
	"strings"

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
	taskMode           string
	showVersion        bool
	initConfig         bool
	initMinimalConfig  bool
	initFromTemplate   string
	initTemplateName   string
	templatesRepo      string
	listTemplates      bool
	saveAsDefault      bool
	setWorkspace       string
	selfUpdate         bool
	allowUndefinedVars bool
	noDrunCache        bool

	// Debug flags
	debugMode          bool
	debugTokens        bool
	debugAST           bool
	debugJSON          bool
	debugErrors        bool
	debugFull          bool
	debugDomain        bool
	debugInput         string
	debugPlan          bool
	debugExportGraph   string
	debugExportMermaid string
	debugExportJSON    string
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
  xdrun --list-templates --templates-repo ../drun-templates
                                 # List available init templates from a local template repo
  xdrun --init                   # Create a new .drun file
  xdrun --init --template go-cli --templates-repo ../drun-templates
                                 # Create a new .drun file from a local template repo
  xdrun --init --from-template github:owner/repo/templates.yaml@main --template go-cli
                                 # Create a new .drun file from a specific template manifest
  xdrun --init-minimal           # Create a minimal .drun file
  xdrun --debug --tokens         # Debug lexer tokens
  xdrun --debug --ast            # Debug AST structure
  xdrun --debug --full           # Full debug output

Built-in Commands:
  Use the 'cmd:' prefix for built-in commands to avoid conflicts with tasks:
  xdrun cmd:completion bash      # Generate shell completion
  xdrun cmd:from makefile        # Convert Makefile to drun
  xdrun cmd:dump-env             # Dump all environment variables
  xdrun cmd:link services/api    # Link directories to this task file
  xdrun cmd:lsp                  # Start the Drun language server over stdio
  xdrun cmd:skill install basics # Install project AI guidance for drun/xdrun
  xdrun cmd:secret add key       # Manage secrets (add, remove, list)
  xdrun cmd:hook install         # Install git hooks for git policies`,
		RunE:              app.run,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: CompleteTaskNames,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true, // Disable default 'completion' command
		},
	}

	app.setupFlags()
	app.setupCommands()
	// Initialize the default help command explicitly so we can hide it.
	// This hides cobra's built-in 'help' subcommand from completion if possible, though Cobra
	// may still forcefully include it. We accept 'help' at the top since user tasks
	// successfully appear before all 'cmd:*' subcommands.
	app.rootCmd.InitDefaultHelpCmd()
	for _, c := range app.rootCmd.Commands() {
		if c.Name() == "help" {
			c.Hidden = true
			break
		}
	}

	return app
}

// Execute runs the CLI application
func (a *App) Execute() error {
	// Intercept autocomplete to reorder 'help' to the bottom
	if len(os.Args) > 1 && (os.Args[1] == "__complete" || os.Args[1] == "__completeNoDesc") {
		var buf bytes.Buffer
		a.rootCmd.SetOut(&buf)

		err := a.rootCmd.Execute()

		lines := strings.Split(buf.String(), "\n")
		var helpLine string
		var otherLines []string

		for _, line := range lines {
			if strings.HasPrefix(line, "help\t") || line == "help" {
				helpLine = line
			} else {
				otherLines = append(otherLines, line)
			}
		}

		if helpLine != "" {
			insertIdx := len(otherLines)
			// Find the directive line (e.g., ":4") which must remain at the very end
			for i := len(otherLines) - 1; i >= 0; i-- {
				if otherLines[i] != "" && strings.HasPrefix(otherLines[i], ":") {
					insertIdx = i
					break
				}
			}

			if insertIdx < len(otherLines) {
				otherLines = append(otherLines[:insertIdx], append([]string{helpLine}, otherLines[insertIdx:]...)...)
			} else {
				otherLines = append(otherLines, helpLine)
			}
		}

		fmt.Print(strings.Join(otherLines, "\n"))
		return err
	}

	return a.rootCmd.Execute()
}

// setupFlags sets up all command-line flags
func (a *App) setupFlags() {
	flags := a.rootCmd.Flags()

	flags.StringVarP(&a.configFile, "file", "f", "", "[xdrun CLI cmd] Task file (default: .drun/spec.drun or workspace configured file)")
	flags.BoolVarP(&a.listTasks, "list", "l", false, "[xdrun CLI cmd] List available tasks")
	flags.BoolVar(&a.dryRun, "dry-run", false, "[xdrun CLI cmd] Show what would be executed without running")
	flags.BoolVarP(&a.verbose, "verbose", "v", false, "[xdrun CLI cmd] Show detailed execution information")
	flags.StringVar(&a.taskMode, "task-mode", "", "[xdrun CLI cmd] Override task execution mode for this run (supported: ci, normal)")
	flags.BoolVar(&a.noDrunCache, "no-drun-cache", false, "[xdrun CLI cmd] Disable remote include caching (always fetch)")
	flags.BoolVar(&a.showVersion, "version", false, "[xdrun CLI cmd] Show version information")
	flags.BoolVar(&a.initConfig, "init", false, "[xdrun CLI cmd] Initialize a new .drun task file")
	flags.BoolVar(&a.initMinimalConfig, "init-minimal", false, "[xdrun CLI cmd] Initialize a new minimal .drun task file")
	flags.StringVar(&a.initFromTemplate, "from-template", "", "[xdrun CLI cmd] Initialize from a specific template manifest (github:/drunhub:/https:// or local path)")
	flags.StringVar(&a.initTemplateName, "template", "", "[xdrun CLI cmd] Template entry name to use with --from-template or --templates-repo")
	flags.StringVar(&a.templatesRepo, "templates-repo", "", "[xdrun CLI cmd] Local template repository root containing templates.yaml")
	flags.BoolVar(&a.listTemplates, "list-templates", false, "[xdrun CLI cmd] List available init templates from a manifest, local template repo, or configured catalog")
	flags.BoolVar(&a.saveAsDefault, "save-as-default", false, "[xdrun CLI cmd] Save custom file name as workspace default (use with --init or --init-minimal)")
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
	flags.BoolVar(&a.debugPlan, "debug-plan", false, "[xdrun CLI cmd] Show execution plan (requires --debug-domain)")
	flags.StringVar(&a.debugExportGraph, "debug-export-graph", "", "[xdrun CLI cmd] Export execution plan as Graphviz DOT file (e.g., 'plan' creates plan-<task>.dot)")
	flags.StringVar(&a.debugExportMermaid, "debug-export-mermaid", "", "[xdrun CLI cmd] Export execution plan as Mermaid diagram (e.g., 'plan' creates plan-<task>.mmd)")
	flags.StringVar(&a.debugExportJSON, "debug-export-json", "", "[xdrun CLI cmd] Export execution plan as JSON (e.g., 'plan' creates plan-<task>.json)")
}

// setupCommands sets up subcommands
func (a *App) setupCommands() {
	// All cmd:* subcommands are registered as Hidden so cobra does not prepend them
	// to the completion list ahead of user-defined tasks. They remain fully functional
	// when invoked directly; our ValidArgsFunction returns them after user tasks.
	cmds := []*cobra.Command{
		a.createCompletionCommand(),
		a.createConvertCommand(),
		a.createDumpEnvCommand(),
		a.createStatelessCommand(),
		a.createLinkCommand(),
		a.createUnlinkCommand(),
		a.createUnlinkAllCommand(),
		a.createLSPCommand(),
		a.createSkillCommand(),
		a.createSecretsCommand(),
		a.createHookCommand(),
	}
	for _, cmd := range cmds {
		cmd.Hidden = true
		a.rootCmd.AddCommand(cmd)
	}
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

	if a.listTemplates {
		if a.initConfig || a.initMinimalConfig {
			return fmt.Errorf("--list-templates cannot be combined with --init or --init-minimal")
		}
		if a.initTemplateName != "" {
			return fmt.Errorf("--template cannot be combined with --list-templates")
		}
		return ListInitTemplates(a.initFromTemplate, a.templatesRepo)
	}

	// Handle --init flag
	if a.initConfig {
		return InitializeConfig(a.configFile, a.saveAsDefault, false, a.initFromTemplate, a.initTemplateName, a.templatesRepo)
	}

	if a.initMinimalConfig {
		return InitializeConfig(a.configFile, a.saveAsDefault, true, a.initFromTemplate, a.initTemplateName, a.templatesRepo)
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
			DebugOptions{
				ShowPlan:       a.debugPlan,
				ExportGraphviz: a.debugExportGraph,
				ExportMermaid:  a.debugExportMermaid,
				ExportJSON:     a.debugExportJSON,
			},
		)
	}

	// Normal execution - run task
	return ExecuteTask(
		a.configFile,
		a.listTasks,
		a.dryRun,
		a.verbose,
		a.taskMode,
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

// createStatelessCommand creates the cmd:stateless subcommand
func (a *App) createStatelessCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmd:stateless",
		Short: "Manage stateless drun directories (configs stored in home directory)",
		Long: `Manage stateless drun directories.

When a directory is marked as stateless, drun will store its configuration
in your home directory (~/.drun/stateless/) instead of in the repository.
This is useful for repositories where you can't commit drun configs.

Note: The 'cmd:' prefix is reserved for built-in commands to avoid conflicts with user tasks.`,
	}

	// Add subcommand
	addCmd := &cobra.Command{
		Use:   "add [directory]",
		Short: "Mark a directory as stateless",
		Long: `Mark a directory as stateless.

The directory's drun configuration will be stored in your home directory
instead of in the repository itself.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			createTemplate, _ := cmd.Flags().GetBool("create")
			return AddStatelessDirectory(dir, createTemplate)
		},
	}
	addCmd.Flags().BoolP("create", "c", false, "Create a template configuration file")

	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove [directory]",
		Short: "Remove stateless marking from a directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			deleteConfig, _ := cmd.Flags().GetBool("delete")
			return RemoveStatelessDirectory(dir, deleteConfig)
		},
	}
	removeCmd.Flags().BoolP("delete", "d", false, "Also delete the configuration file")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all stateless directories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ListStatelessDirectories()
		},
	}

	// Info subcommand
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show stateless status of current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ShowStatelessInfo()
		},
	}

	cmd.AddCommand(addCmd)
	cmd.AddCommand(removeCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(infoCmd)

	return cmd
}
