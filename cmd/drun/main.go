package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

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
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "drun [recipe] [positionals...] [flags...]",
	Short: "A YAML-based task runner with first-class positional arguments",
	Long: `drun is a task runner that uses YAML configuration files to define recipes.
It supports positional arguments, dependencies, templating, and cross-platform execution.`,
	RunE: runDrun,
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "file", "f", "", "Configuration file (default: drun.yml)")
	rootCmd.Flags().BoolVarP(&listRecipes, "list", "l", false, "List available recipes")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be executed without running")
	rootCmd.Flags().BoolVar(&explain, "explain", false, "Show rendered scripts and environment")
	rootCmd.Flags().IntVarP(&jobs, "jobs", "j", 1, "Number of parallel jobs")
	rootCmd.Flags().StringVar(&shellType, "shell", "", "Override shell type (linux/darwin/windows)")
	rootCmd.Flags().StringArrayVar(&setVars, "set", []string{}, "Set variables (KEY=VALUE)")
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "Initialize a new drun.yml configuration file")
}

func runDrun(cmd *cobra.Command, args []string) error {
	// Handle --init flag
	if initConfig {
		return initializeConfig(configFile)
	}

	// Load configuration
	loader := spec.NewLoader(".")
	specData, err := loader.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Handle --list flag
	if listRecipes {
		return listAllRecipes(specData)
	}

	// Determine target recipe
	var target string
	var positionalArgs []string
	var flags map[string]any

	if len(args) == 0 {
		// No arguments - try to find a default recipe or list recipes
		if defaultRecipe := findDefaultRecipe(specData); defaultRecipe != "" {
			target = defaultRecipe
		} else {
			return listAllRecipes(specData)
		}
	} else {
		target = args[0]
		positionalArgs = args[1:]
		flags = make(map[string]any)
	}

	// Check if recipe exists
	recipe, exists := specData.Recipes[target]
	if !exists {
		return fmt.Errorf("recipe '%s' not found", target)
	}

	// Parse positional arguments
	positionals, err := parsePositionals(recipe.Positionals, positionalArgs)
	if err != nil {
		return fmt.Errorf("invalid positional arguments: %w", err)
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
	dagBuilder := dag.NewBuilder(specData)
	taskRunner := runner.NewRunner(shellSelector, templateEngine, os.Stdout)

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

func parsePositionals(posArgs []model.PositionalArg, args []string) (map[string]any, error) {
	result := make(map[string]any)

	for i, posArg := range posArgs {
		if i >= len(args) {
			if posArg.Required {
				return nil, fmt.Errorf("required positional argument '%s' not provided", posArg.Name)
			}
			if posArg.Default != "" {
				result[posArg.Name] = posArg.Default
			}
			continue
		}

		value := args[i]

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
			result[posArg.Name] = args[i:]
			break
		} else {
			result[posArg.Name] = value
		}
	}

	// Check for excess arguments
	if len(args) > len(posArgs) && (len(posArgs) == 0 || !posArgs[len(posArgs)-1].Variadic) {
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
