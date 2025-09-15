package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/cache"
	"github.com/phillarmonic/drun/internal/dag"
	"github.com/phillarmonic/drun/internal/model"
	"github.com/phillarmonic/drun/internal/runner"
	"github.com/phillarmonic/drun/internal/shell"
	"github.com/phillarmonic/drun/internal/spec"
	"github.com/phillarmonic/drun/internal/tmpl"
	"github.com/spf13/cobra"
)

var (
	configFile  string
	listRecipes bool
	dryRun      bool
	explain     bool
	jobs        int
	shellType   string
	setVars     []string
	initConfig  bool
	showVersion bool
	noCache     bool
	updateSelf  bool
)

// Version information (set at build time)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "drun [recipe] [positionals...] [--name=value...] [flags...]",
	Short: "A YAML-based task runner with first-class positional arguments",
	Long: `drun is a task runner that uses YAML configuration files to define recipes.
It supports positional arguments (both positional and named), dependencies, templating, and cross-platform execution.

Named arguments can be specified as:
  --name=value  or  name=value`,
	RunE: runDrun,
	// Disable unknown command errors
	SilenceErrors: true,
	SilenceUsage:  true,
	// Allow any arguments to be passed through
	DisableFlagParsing: false,
}

// TODO: Cache subcommands temporarily disabled due to command resolution conflicts

func init() {
	rootCmd.Flags().StringVarP(&configFile, "file", "f", "", "Configuration file (default: drun.yml)")
	rootCmd.Flags().BoolVarP(&listRecipes, "list", "l", false, "List available recipes")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be executed without running")
	rootCmd.Flags().BoolVar(&explain, "explain", false, "Show rendered scripts and environment")
	rootCmd.Flags().IntVarP(&jobs, "jobs", "j", 1, "Number of parallel jobs")
	rootCmd.Flags().StringVar(&shellType, "shell", "", "Override shell type (linux/darwin/windows)")
	rootCmd.Flags().StringArrayVar(&setVars, "set", []string{}, "Set variables (KEY=VALUE)")
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "Initialize a new drun.yml configuration file")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	rootCmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable caching and force execution")
	rootCmd.Flags().BoolVar(&updateSelf, "update", false, "Update drun to the latest version")

	// TODO: Add cache subcommand later (causes command resolution issues)
	// cacheCmd.AddCommand(cacheClearCmd)
	// cacheCmd.AddCommand(cacheStatsCmd)
	// rootCmd.AddCommand(cacheCmd)

	// Set up unknown command handling
	rootCmd.SetArgs(os.Args[1:])
	rootCmd.FParseErrWhitelist.UnknownFlags = true
}

// filterGlobalFlags removes global flags from the argument list, leaving recipe and recipe-specific flags
func filterGlobalFlags(args []string) []string {
	var filtered []string
	globalFlags := map[string]bool{
		"--file": true, "-f": true,
		"--list": true, "-l": true,
		"--dry-run": true,
		"--explain": true,
		"--jobs":    true, "-j": true,
		"--shell":   true,
		"--set":     true,
		"--init":    true,
		"--version": true, "-v": true,
		"--no-cache": true,
		"--update":   true,
		"--help":     true, "-h": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if it's a global flag
		if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
			flagName := arg
			if strings.Contains(arg, "=") {
				flagName = strings.SplitN(arg, "=", 2)[0]
			}

			if globalFlags[flagName] {
				// Skip this flag
				if !strings.Contains(arg, "=") && i+1 < len(args) {
					// Also skip the next argument if it's the flag value
					nextArg := args[i+1]
					if !strings.HasPrefix(nextArg, "-") {
						i++ // Skip the value
					}
				}
				continue
			}
		}

		// Keep this argument
		filtered = append(filtered, arg)
	}

	return filtered
}

func runDrun(cmd *cobra.Command, args []string) error {
	// Get raw arguments to handle recipe-specific flags
	rawArgs := os.Args[1:]

	// Filter out global flags that Cobra has already processed
	filteredArgs := filterGlobalFlags(rawArgs)
	// Handle --version flag
	if showVersion {
		return showVersionInfo()
	}

	// Handle --update flag
	if updateSelf {
		return performSelfUpdate()
	}

	// Handle --init flag
	if initConfig {
		return initializeConfig(configFile)
	}

	// Load configuration
	loader := spec.NewLoader(".")
	specData, err := loader.Load(configFile)
	if err != nil {
		return enhanceConfigError(err, configFile)
	}

	// Handle --list flag
	if listRecipes {
		return listAllRecipes(specData)
	}

	// Determine target recipe using filtered args (which include recipe-specific flags)
	var target string
	var recipeArgs []string
	var flags map[string]any

	if len(filteredArgs) == 0 {
		// No arguments - try to find a default recipe or list recipes
		if defaultRecipe := findDefaultRecipe(specData); defaultRecipe != "" {
			target = defaultRecipe
		} else {
			return listAllRecipes(specData)
		}
	} else {
		target = filteredArgs[0]
		recipeArgs = filteredArgs[1:]
		flags = make(map[string]any)
	}

	// Check if recipe exists
	recipe, exists := specData.Recipes[target]
	if !exists {
		return recipeNotFoundError(target, specData.Recipes)
	}

	// Parse recipe-specific flags and positional arguments
	positionals, recipeFlags, err := parseRecipeArgs(recipe, recipeArgs)
	if err != nil {
		return enhanceRecipeArgsError(err, target, recipe)
	}

	// Merge recipe flags into flags map
	for k, v := range recipeFlags {
		flags[k] = v
	}

	// Parse set variables
	setVarsMap, err := parseSetVars(setVars)
	if err != nil {
		return fmt.Errorf("invalid --set variables: %w", err)
	}

	// Build execution context
	ctx := buildExecutionContext(specData, positionals, flags, setVarsMap)

	// Override OS if shell type is specified
	if shellType != "" {
		ctx.OS = shellType
	}

	// Create components
	shellSelector := shell.NewSelector(specData.Shell)
	templateEngine := tmpl.NewEngine(specData.Snippets)

	// Set up caching
	cacheDir := ".drun/cache"
	if specData.Cache.Path != "" {
		cacheDir = specData.Cache.Path
	}
	// Make cache directory absolute
	if !filepath.IsAbs(cacheDir) {
		cacheDir = filepath.Join(".", cacheDir)
	}
	cacheManager := cache.NewManager(cacheDir, templateEngine, noCache)

	dagBuilder := dag.NewBuilder(specData)
	taskRunner := runner.NewRunner(shellSelector, templateEngine, cacheManager, os.Stdout)

	// Set runner modes
	taskRunner.SetDryRun(dryRun)
	taskRunner.SetExplain(explain)

	// Render environment variables with template engine
	if err := renderEnvironment(ctx, templateEngine); err != nil {
		return fmt.Errorf("failed to render environment variables: %w", err)
	}

	// Build execution plan
	plan, err := dagBuilder.Build(target, ctx)
	if err != nil {
		return fmt.Errorf("failed to build execution plan: %w", err)
	}

	// Execute plan
	return taskRunner.Execute(plan, jobs)
}

func listAllRecipes(specData *model.Spec) error {
	fmt.Println("Available recipes:")

	for name, recipe := range specData.Recipes {
		help := recipe.Help
		if help == "" {
			help = "No description"
		}
		fmt.Printf("  %-20s %s\n", name, help)

		// Show aliases if any
		if len(recipe.Aliases) > 0 {
			fmt.Printf("  %-20s (aliases: %s)\n", "", strings.Join(recipe.Aliases, ", "))
		}
	}

	return nil
}

func findDefaultRecipe(specData *model.Spec) string {
	// Look for common default recipe names
	defaultNames := []string{"default", "help", "list"}

	for _, name := range defaultNames {
		if _, exists := specData.Recipes[name]; exists {
			return name
		}
	}

	return ""
}

// parseRecipeArgs parses both positional arguments and recipe-specific flags
func parseRecipeArgs(recipe model.Recipe, args []string) (map[string]any, map[string]any, error) {
	flags := make(map[string]any)
	namedArgs := make(map[string]string) // Store named positional arguments

	// Apply default values for flags first
	for flagName, flagDef := range recipe.Flags {
		if flagDef.Default != nil {
			flags[flagName] = flagDef.Default
		} else {
			// Set zero values based on type
			switch flagDef.Type {
			case "bool":
				flags[flagName] = false
			case "int":
				flags[flagName] = 0
			case "string":
				flags[flagName] = ""
			case "string[]":
				flags[flagName] = []string{}
			default:
				flags[flagName] = ""
			}
		}
	}

	var positionalArgs []string

	// Parse arguments, separating flags, named args, and positionals
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if it's a flag (starts with --)
		if strings.HasPrefix(arg, "--") {
			flagName := strings.TrimPrefix(arg, "--")

			// Handle --flag=value format
			if strings.Contains(flagName, "=") {
				parts := strings.SplitN(flagName, "=", 2)
				flagName = parts[0]
				flagValue := parts[1]

				// Check if this is a named positional argument
				if isPositionalArgName(recipe.Positionals, flagName) {
					namedArgs[flagName] = flagValue
					continue
				}

				if err := setRecipeFlag(flags, recipe.Flags, flagName, flagValue); err != nil {
					return nil, nil, err
				}
				continue
			}

			// Check if this is a named positional argument
			if isPositionalArgName(recipe.Positionals, flagName) {
				// For named positional args, we need a value
				if i+1 >= len(args) {
					return nil, nil, fmt.Errorf("named argument --%s requires a value", flagName)
				}
				i++ // Move to the value
				namedArgs[flagName] = args[i]
				continue
			}

			// Check if this flag is defined for the recipe
			flagDef, exists := recipe.Flags[flagName]
			if !exists {
				return nil, nil, fmt.Errorf("unknown flag: --%s", flagName)
			}

			// Handle boolean flags
			if flagDef.Type == "bool" {
				flags[flagName] = true
				continue
			}

			// For non-boolean flags, we need a value
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag --%s requires a value", flagName)
			}

			i++ // Move to the value
			flagValue := args[i]

			if err := setRecipeFlag(flags, recipe.Flags, flagName, flagValue); err != nil {
				return nil, nil, err
			}
		} else if strings.Contains(arg, "=") && !strings.HasPrefix(arg, "-") {
			// Check if it's a named argument in key=value format
			parts := strings.SplitN(arg, "=", 2)
			argName := parts[0]
			argValue := parts[1]

			if isPositionalArgName(recipe.Positionals, argName) {
				namedArgs[argName] = argValue
				continue
			}

			// If not a named positional arg, treat as regular positional
			positionalArgs = append(positionalArgs, arg)
		} else {
			// It's a positional argument
			positionalArgs = append(positionalArgs, arg)
		}
	}

	// Parse positional arguments (including named ones)
	parsedPositionals, err := parsePositionalsWithNamed(recipe.Positionals, positionalArgs, namedArgs)
	if err != nil {
		return nil, nil, err
	}

	return parsedPositionals, flags, nil
}

// setRecipeFlag sets a flag value with proper type conversion
func setRecipeFlag(flags map[string]any, flagDefs map[string]model.Flag, flagName, value string) error {
	flagDef, exists := flagDefs[flagName]
	if !exists {
		return fmt.Errorf("unknown flag: --%s", flagName)
	}

	switch flagDef.Type {
	case "string":
		flags[flagName] = value
	case "int":
		intVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("flag --%s requires an integer value, got: %s", flagName, value)
		}
		flags[flagName] = intVal
	case "bool":
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("flag --%s requires a boolean value (true/false), got: %s", flagName, value)
		}
		flags[flagName] = boolVal
	case "string[]":
		// For string arrays, we append to existing values
		if existing, ok := flags[flagName].([]string); ok {
			flags[flagName] = append(existing, value)
		} else {
			flags[flagName] = []string{value}
		}
	default:
		return fmt.Errorf("unsupported flag type: %s for flag --%s", flagDef.Type, flagName)
	}

	return nil
}

// enhanceRecipeArgsError provides better error messages for recipe argument parsing
func enhanceRecipeArgsError(err error, recipeName string, recipe model.Recipe) error {
	errStr := err.Error()

	// Check if it's a flag-related error
	if strings.Contains(errStr, "unknown flag") {
		msg := fmt.Sprintf("Error: %s\n\n", err.Error())
		msg += fmt.Sprintf("Available flags for recipe '%s':\n", recipeName)

		if len(recipe.Flags) == 0 {
			msg += "  (no flags defined)\n"
		} else {
			for flagName, flagDef := range recipe.Flags {
				msg += fmt.Sprintf("  --%s", flagName)
				if flagDef.Type != "bool" {
					msg += fmt.Sprintf(" <%s>", flagDef.Type)
				}
				if flagDef.Help != "" {
					msg += fmt.Sprintf(" - %s", flagDef.Help)
				}
				if flagDef.Default != nil {
					msg += fmt.Sprintf(" (default: %v)", flagDef.Default)
				}
				msg += "\n"
			}
		}

		msg += "\nUsage:\n"
		msg += fmt.Sprintf("  drun %s", recipeName)

		// Show positionals
		for _, pos := range recipe.Positionals {
			if pos.Required {
				msg += fmt.Sprintf(" <%s>", pos.Name)
			} else {
				msg += fmt.Sprintf(" [%s]", pos.Name)
			}
		}

		// Show flags
		for flagName, flagDef := range recipe.Flags {
			if flagDef.Type == "bool" {
				msg += fmt.Sprintf(" [--%s]", flagName)
			} else {
				msg += fmt.Sprintf(" [--%s <%s>]", flagName, flagDef.Type)
			}
		}

		// Show named argument syntax if there are positionals
		if len(recipe.Positionals) > 0 {
			msg += "\n\nNamed argument syntax:\n"
			msg += fmt.Sprintf("  drun %s", recipeName)
			for _, pos := range recipe.Positionals {
				if pos.Required {
					msg += fmt.Sprintf(" --%s=<%s>", pos.Name, pos.Name)
				} else {
					msg += fmt.Sprintf(" [--%s=<%s>]", pos.Name, pos.Name)
				}
			}
			msg += "\n  or:\n"
			msg += fmt.Sprintf("  drun %s", recipeName)
			for _, pos := range recipe.Positionals {
				if pos.Required {
					msg += fmt.Sprintf(" %s=<%s>", pos.Name, pos.Name)
				} else {
					msg += fmt.Sprintf(" [%s=<%s>]", pos.Name, pos.Name)
				}
			}
		}

		return fmt.Errorf("%s", msg)
	}

	// For other errors, use the existing positional error enhancement
	return enhancePositionalError(err, recipeName, recipe.Positionals, recipe.Help)
}

// isPositionalArgName checks if a name corresponds to a positional argument
func isPositionalArgName(positionals []model.PositionalArg, name string) bool {
	for _, pos := range positionals {
		if pos.Name == name {
			return true
		}
	}
	return false
}

// parsePositionalsWithNamed parses positional arguments, supporting both positional and named syntax
func parsePositionalsWithNamed(posArgs []model.PositionalArg, args []string, namedArgs map[string]string) (map[string]any, error) {
	result := make(map[string]any)
	usedPositional := make([]bool, len(posArgs)) // Track which positional slots are used

	// First, process named arguments
	for name, value := range namedArgs {
		// Find the positional argument definition
		var posArg *model.PositionalArg
		var index int
		for i, pa := range posArgs {
			if pa.Name == name {
				posArg = &pa
				index = i
				break
			}
		}

		if posArg == nil {
			return nil, fmt.Errorf("unknown positional argument: %s", name)
		}

		// Validate one_of constraint
		if len(posArg.OneOf) > 0 {
			valid := false
			for _, allowed := range posArg.OneOf {
				if value == allowed {
					valid = true
					break
				}
			}
			if !valid {
				return nil, fmt.Errorf("positional argument '%s' must be one of: %v", posArg.Name, posArg.OneOf)
			}
		}

		if posArg.Variadic {
			// For variadic named args, split by comma or treat as single value
			result[posArg.Name] = []string{value}
		} else {
			result[posArg.Name] = value
		}

		usedPositional[index] = true
	}

	// Then, process regular positional arguments
	argIndex := 0
	for i, posArg := range posArgs {
		if usedPositional[i] {
			// This position was filled by a named argument, skip
			continue
		}

		if argIndex >= len(args) {
			if posArg.Required {
				return nil, fmt.Errorf("required positional argument '%s' not provided", posArg.Name)
			}
			if posArg.Default != "" {
				result[posArg.Name] = posArg.Default
			}
			continue
		}

		value := args[argIndex]

		// Validate one_of constraint
		if len(posArg.OneOf) > 0 {
			valid := false
			for _, allowed := range posArg.OneOf {
				if value == allowed {
					valid = true
					break
				}
			}
			if !valid {
				return nil, fmt.Errorf("positional argument '%s' must be one of: %v", posArg.Name, posArg.OneOf)
			}
		}

		if posArg.Variadic {
			// Collect remaining arguments
			remainingArgs := args[argIndex:]
			if existing, exists := result[posArg.Name]; exists {
				// Merge with named args if any
				if existingSlice, ok := existing.([]string); ok {
					result[posArg.Name] = append(existingSlice, remainingArgs...)
				} else {
					result[posArg.Name] = remainingArgs
				}
			} else {
				result[posArg.Name] = remainingArgs
			}
			argIndex = len(args) // Consume all remaining args
			break
		} else {
			result[posArg.Name] = value
			argIndex++
		}
	}

	// Check for excess arguments
	if argIndex < len(args) {
		return nil, fmt.Errorf("too many positional arguments provided")
	}

	return result, nil
}

func parseSetVars(setVars []string) (map[string]any, error) {
	result := make(map[string]any)

	for _, setVar := range setVars {
		parts := strings.SplitN(setVar, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format for --set flag: %s (expected KEY=VALUE)", setVar)
		}
		result[parts[0]] = parts[1]
	}

	return result, nil
}

func buildExecutionContext(specData *model.Spec, positionals, flags, setVars map[string]any) *model.ExecutionContext {
	ctx := &model.ExecutionContext{
		Vars:        make(map[string]any),
		Env:         make(map[string]string),
		Flags:       flags,
		Positionals: positionals,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
	}

	// Set hostname
	if hostname, err := os.Hostname(); err == nil {
		ctx.Hostname = hostname
	}

	// Add spec vars
	for k, v := range specData.Vars {
		ctx.Vars[k] = v
	}

	// Add set vars (override spec vars)
	for k, v := range setVars {
		ctx.Vars[k] = v
	}

	// Add spec env
	for k, v := range specData.Env {
		ctx.Env[k] = v
	}

	return ctx
}

func recipeNotFoundError(target string, recipes map[string]model.Recipe) error {
	var suggestions []string

	// Find similar recipe names (simple string distance)
	for recipeName := range recipes {
		if levenshteinDistance(target, recipeName) <= 2 {
			suggestions = append(suggestions, recipeName)
		}
	}

	// If no close matches, suggest some common ones
	if len(suggestions) == 0 {
		commonNames := []string{"build", "test", "dev", "start", "deploy", "clean"}
		for _, common := range commonNames {
			if _, exists := recipes[common]; exists {
				suggestions = append(suggestions, common)
			}
		}
	}

	msg := fmt.Sprintf("recipe '%s' not found", target)

	if len(suggestions) > 0 {
		msg += "\n\nDid you mean one of these?"
		for _, suggestion := range suggestions {
			if recipe, exists := recipes[suggestion]; exists && recipe.Help != "" {
				msg += fmt.Sprintf("\n  %s - %s", suggestion, recipe.Help)
			} else {
				msg += fmt.Sprintf("\n  %s", suggestion)
			}
		}
	}

	msg += "\n\nRun 'drun --list' to see all available recipes."

	return fmt.Errorf("%s", msg)
}

// Simple Levenshtein distance implementation
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func enhancePositionalError(err error, recipeName string, positionals []model.PositionalArg, help string) error {
	msg := fmt.Sprintf("Error in recipe '%s': %v", recipeName, err)

	if help != "" {
		msg += fmt.Sprintf("\n\nUsage: %s", help)
	}

	if len(positionals) > 0 {
		msg += "\n\nExpected arguments:"
		for i, pos := range positionals {
			required := ""
			if pos.Required {
				required = " (required)"
			}

			constraints := ""
			if len(pos.OneOf) > 0 {
				constraints = fmt.Sprintf(" [one of: %s]", strings.Join(pos.OneOf, ", "))
			}

			if pos.Variadic {
				msg += fmt.Sprintf("\n  %d. %s... %s%s", i+1, pos.Name, required, constraints)
			} else {
				msg += fmt.Sprintf("\n  %d. %s%s%s", i+1, pos.Name, required, constraints)
			}
		}
	}

	return fmt.Errorf("%s", msg)
}

func enhanceConfigError(err error, configFile string) error {
	errStr := err.Error()

	// Check if it's a "no config file found" error
	if strings.Contains(errStr, "no drun configuration file found") {
		msg := "No drun configuration file found.\n\n"
		msg += "To get started:\n"
		msg += "  drun --init          # Create a starter configuration\n"
		msg += "  drun --init -f FILE  # Create with custom filename\n\n"
		msg += "Or create one of these files manually:\n"
		for _, filename := range spec.DefaultFilenames {
			msg += fmt.Sprintf("  %s\n", filename)
		}
		return fmt.Errorf("%s", msg)
	}

	// Check if it's a YAML parsing error
	if strings.Contains(errStr, "failed to parse YAML") {
		msg := fmt.Sprintf("Configuration file has invalid YAML syntax: %v\n\n", err)
		msg += "Common YAML issues:\n"
		msg += "  - Incorrect indentation (use spaces, not tabs)\n"
		msg += "  - Missing quotes around strings with special characters\n"
		msg += "  - Unmatched brackets or quotes\n\n"
		msg += "Tip: Use a YAML validator or editor with YAML support"
		return fmt.Errorf("%s", msg)
	}

	// Check if it's a validation error
	if strings.Contains(errStr, "validation failed") {
		return fmt.Errorf("configuration validation failed: %v\n\nCheck your recipe definitions and ensure all required fields are present", err)
	}

	// Default enhanced error
	return fmt.Errorf("failed to load configuration: %v\n\nTry 'drun --init' to create a new configuration file", err)
}

func showVersionInfo() error {
	fmt.Printf("drun version %s\n", version)
	if commit != "unknown" {
		fmt.Printf("commit: %s\n", commit)
	}
	if date != "unknown" {
		fmt.Printf("built: %s\n", date)
	}
	fmt.Printf("go: %s\n", runtime.Version())
	fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	return nil
}

func initializeConfig(filename string) error {
	// Determine the target filename
	targetFile := "drun.yml"
	if filename != "" {
		targetFile = filename
	}

	// Check if file already exists
	if _, err := os.Stat(targetFile); err == nil {
		return fmt.Errorf("configuration file '%s' already exists", targetFile)
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

func generateStarterConfig() string {
	return `# drun configuration file
# Learn more: https://github.com/phillarmonic/drun

version: 0.1

# Shell configuration per OS (optional - these are the defaults)
shell:
  linux: { cmd: "/bin/sh", args: ["-ceu"] }
  darwin: { cmd: "/bin/zsh", args: ["-ceu"] }
  windows: { cmd: "pwsh", args: ["-NoLogo", "-Command"] }

# Environment variables available to all recipes
env:
  PROJECT_NAME: "my-project"
  BUILD_DATE: '{{ now "2006-01-02T15:04:05Z" }}'

# Variables for templating
vars:
  app_name: "myapp"
  version: "1.0.0"

# Global defaults (optional)
defaults:
  working_dir: "."
  shell: auto
  timeout: "10m"

# Reusable code snippets
snippets:
  setup_colors: |
    # ANSI color codes
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m' # No Color

recipes:
  # Default recipe (runs when no recipe specified)
  default:
    help: "Show available commands"
    run: |
      echo "🚀 {{ .PROJECT_NAME }} Task Runner"
      echo "=================================="
      echo ""
      echo "Available recipes:"
      echo "  build     - Build the project"
      echo "  test      - Run tests"
      echo "  clean     - Clean build artifacts"
      echo "  deploy    - Deploy to environment"
      echo ""
      echo "Usage: drun <recipe> [args...]"
      echo "       drun --list  (show all recipes)"

  # Simple build recipe
  build:
    help: "Build the project"
    run: |
      {{ snippet "setup_colors" }}
      echo -e "${GREEN}🏗️  Building {{ .app_name }}...${NC}"
      # Add your build commands here
      echo "Build completed successfully!"

  # Test recipe with optional pattern
  test:
    help: "Run tests (usage: test [pattern])"
    positionals:
      - name: pattern
        default: ""
    run: |
      {{ snippet "setup_colors" }}
      echo -e "${YELLOW}🧪 Running tests...${NC}"
      {{ if .pattern }}
      echo "Running tests matching: {{ .pattern }}"
      # Add filtered test command here
      {{ else }}
      echo "Running all tests..."
      # Add full test command here
      {{ end }}
      echo -e "${GREEN}✅ Tests passed!${NC}"

  # Clean recipe
  clean:
    help: "Clean build artifacts"
    run: |
      {{ snippet "setup_colors" }}
      echo -e "${YELLOW}🧹 Cleaning build artifacts...${NC}"
      # Add cleanup commands here (e.g., rm -rf build/ dist/)
      echo -e "${GREEN}✅ Clean completed!${NC}"

  # Deploy recipe with environment selection
  deploy:
    help: "Deploy to environment (usage: deploy <env>)"
    positionals:
      - name: environment
        required: true
        one_of: ["dev", "staging", "production"]
    deps: [build, test]
    run: |
      {{ snippet "setup_colors" }}
      {{ if eq .environment "production" }}
      echo -e "${RED}⚠️  PRODUCTION DEPLOYMENT${NC}"
      echo "Deploying {{ .app_name }} v{{ .version }} to production..."
      {{ else }}
      echo -e "${YELLOW}🚀 Deploying to {{ .environment }}...${NC}"
      {{ end }}
      # Add deployment commands here
      echo -e "${GREEN}✅ Deployment completed!${NC}"

  # Development server
  dev:
    help: "Start development server"
    run: |
      {{ snippet "setup_colors" }}
      echo -e "${GREEN}🚀 Starting development server...${NC}"
      echo "Project: {{ .PROJECT_NAME }}"
      echo "Version: {{ .version }}"
      echo "Build Date: {{ .BUILD_DATE }}"
      # Add dev server command here
      echo "Press Ctrl+C to stop"

  # Show environment info
  info:
    help: "Show project information"
    run: |
      {{ snippet "setup_colors" }}
      echo -e "${GREEN}📋 Project Information${NC}"
      echo "======================="
      echo "Name: {{ .PROJECT_NAME }}"
      echo "App: {{ .app_name }}"
      echo "Version: {{ .version }}"
      echo "Build Date: {{ .BUILD_DATE }}"
      echo "OS: {{ os }}"
      echo "Architecture: {{ arch }}"
      echo "Working Directory: $(pwd)"
`
}

func renderEnvironment(ctx *model.ExecutionContext, templateEngine *tmpl.Engine) error {
	// Render environment variables that contain templates
	for k, v := range ctx.Env {
		if strings.Contains(v, "{{") {
			rendered, err := templateEngine.Render(v, ctx)
			if err != nil {
				return fmt.Errorf("failed to render environment variable %s: %w", k, err)
			}
			ctx.Env[k] = rendered
		}
	}
	return nil
}

// GitHub release structures
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Assets  []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

const (
	githubRepo    = "phillarmonic/drun"
	githubAPIURL  = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	updateTimeout = 30 * time.Second
)

// performSelfUpdate handles the self-update process
func performSelfUpdate() error {
	fmt.Println("🔍 Checking for updates...")

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Fetch latest release info
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// Check if update is needed
	if !isUpdateNeeded(release.TagName) {
		fmt.Printf("✅ Already up to date (version %s)\n", version)
		return nil
	}

	fmt.Printf("📦 New version available: %s (current: %s)\n", release.TagName, version)

	// Find appropriate asset for current platform
	asset, err := findAssetForPlatform(release.Assets)
	if err != nil {
		return fmt.Errorf("failed to find binary for platform %s/%s: %w", runtime.GOOS, runtime.GOARCH, err)
	}

	fmt.Printf("📥 Found binary: %s (%.1f MB)\n", asset.Name, float64(asset.Size)/(1024*1024))

	// Ask for user confirmation
	if !confirmUpdate(release.TagName) {
		fmt.Println("❌ Update cancelled by user")
		return nil
	}

	// Download and replace binary
	if err := downloadAndReplace(asset, execPath); err != nil {
		return fmt.Errorf("failed to update binary: %w", err)
	}

	fmt.Printf("✅ Successfully updated to version %s!\n", release.TagName)
	fmt.Println("🚀 Run 'drun --version' to verify the update")
	return nil
}

// fetchLatestRelease gets the latest release information from GitHub
func fetchLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{Timeout: updateTimeout}

	resp, err := client.Get(githubAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s - this may be a development version or releases haven't been published yet", githubRepo)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// isUpdateNeeded compares current version with latest version
func isUpdateNeeded(latestVersion string) bool {
	// Remove 'v' prefix if present
	latest := strings.TrimPrefix(latestVersion, "v")
	current := strings.TrimPrefix(version, "v")

	// If current version is "dev", always update
	if current == "dev" {
		return true
	}

	// Simple string comparison - in production you might want semantic version comparison
	return latest != current
}

// findAssetForPlatform finds the appropriate binary asset for the current platform
func findAssetForPlatform(assets []GitHubAsset) (*GitHubAsset, error) {
	// Determine expected binary name based on platform
	var expectedName string
	if runtime.GOOS == "windows" {
		expectedName = fmt.Sprintf("drun-%s-%s.exe", runtime.GOOS, runtime.GOARCH)
	} else {
		expectedName = fmt.Sprintf("drun-%s-%s", runtime.GOOS, runtime.GOARCH)
	}

	// Find matching asset
	for _, asset := range assets {
		if asset.Name == expectedName {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("no binary found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
}

// confirmUpdate asks the user for confirmation before updating
func confirmUpdate(newVersion string) bool {
	fmt.Printf("\n⚠️  This will replace your current drun binary with version %s\n", newVersion)
	fmt.Print("Do you want to continue? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// downloadAndReplace downloads the new binary and replaces the current one
func downloadAndReplace(asset *GitHubAsset, execPath string) error {
	// Create backup of current binary
	backupPath := execPath + ".backup"
	if err := copyFile(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Println("💾 Created backup of current binary")

	// Download new binary to temporary file
	tempPath := execPath + ".tmp"
	if err := downloadFile(asset.BrowserDownloadURL, tempPath); err != nil {
		// Clean up backup on failure
		_ = os.Remove(backupPath)
		return fmt.Errorf("failed to download new binary: %w", err)
	}

	fmt.Println("📥 Downloaded new binary")

	// Make the new binary executable (Unix systems)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tempPath, 0755); err != nil {
			_ = os.Remove(tempPath)
			_ = os.Remove(backupPath)
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	// Replace current binary with new one
	if err := os.Rename(tempPath, execPath); err != nil {
		// Try to restore backup on failure
		_ = os.Rename(backupPath, execPath)
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Clean up backup file
	_ = os.Remove(backupPath)

	fmt.Println("🔄 Replaced binary successfully")
	return nil
}

// downloadFile downloads a file from URL to the specified path
func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: updateTimeout}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	_, err = io.Copy(out, resp.Body)
	return err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = destFile.Close()
	}()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
