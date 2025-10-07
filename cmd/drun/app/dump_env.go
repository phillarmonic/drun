package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phillarmonic/drun/internal/envloader"
	"github.com/spf13/cobra"
)

// createDumpEnvCommand creates the cmd:dump-env subcommand
func (a *App) createDumpEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmd:dump-env",
		Short: "Dump all environment variables available to drun commands",
		Long: `Dump all environment variables that would be available to drun commands.

This command evaluates and displays all environment variables from:
  • Host OS environment variables
  • .env files (hierarchical loading)
  • Environment-specific .env files

The output shows the final merged environment that drun commands would see,
including which .env files were loaded and any loading errors.

Examples:
  xdrun cmd:dump-env                    # Show all environment variables
  xdrun cmd:dump-env --env=production   # Load production .env files
  xdrun cmd:dump-env --format=json      # Output in JSON format
  xdrun cmd:dump-env --filter=API_      # Show only variables starting with API_

Note: The 'cmd:' prefix is reserved for built-in commands to avoid conflicts with user tasks.
`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.handleDumpEnv(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().String("env", "", "Environment name (e.g., production, staging)")
	cmd.Flags().String("format", "table", "Output format: table, json, env, shell")
	cmd.Flags().String("filter", "", "Filter variables by prefix (e.g., API_)")
	cmd.Flags().Bool("show-files", false, "Show .env file loading information")
	cmd.Flags().Bool("show-sources", false, "Show which file each variable came from (JSON format only)")
	cmd.Flags().Bool("debug", false, "Show detailed environment loading debug information")

	return cmd
}

// handleDumpEnv handles the dump-env command execution
func (a *App) handleDumpEnv(cmd *cobra.Command, args []string) error {
	// Get flags
	env := cmd.Flag("env").Value.String()
	format := cmd.Flag("format").Value.String()
	filter := cmd.Flag("filter").Value.String()
	showFiles := cmd.Flag("show-files").Value.String() == "true"
	showSources := cmd.Flag("show-sources").Value.String() == "true"

	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create environment loader (disable debug by default)
	debug := cmd.Flag("debug").Value.String() == "true"
	loader := envloader.NewLoader(workingDir, env, debug, os.Stdout)

	// Load environment variables
	result, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Apply filter if specified
	filteredEnv := result.FinalEnv
	if filter != "" {
		filteredEnv = make(map[string]string)
		for key, value := range result.FinalEnv {
			if strings.HasPrefix(key, filter) {
				filteredEnv[key] = value
			}
		}
	}

	// Output based on format
	switch format {
	case "json":
		return a.outputJSON(filteredEnv, result, showFiles, showSources)
	case "env":
		return a.outputEnvFormat(filteredEnv)
	case "shell":
		return a.outputShellFormat(filteredEnv)
	default:
		return a.outputTableFormat(filteredEnv, result, showFiles, showSources)
	}
}

// outputJSON outputs environment variables in JSON format
func (a *App) outputJSON(env map[string]string, result *envloader.LoadResult, showFiles, showSources bool) error {
	fmt.Println("{")
	fmt.Println("  \"environment\": \"" + result.Environment + "\",")
	fmt.Println("  \"host_env_included\": " + fmt.Sprintf("%t", result.HostEnvIncluded) + ",")

	if showFiles {
		fmt.Println("  \"env_files\": [")
		for i, file := range result.Files {
			fmt.Printf("    {\n")
			fmt.Printf("      \"path\": %q,\n", file.Path)
			fmt.Printf("      \"exists\": %t,\n", file.Exists)
			fmt.Printf("      \"loaded\": %t", file.Loaded)
			if file.Error != nil {
				fmt.Printf(",\n      \"error\": %q", file.Error.Error())
			}
			if showSources && len(file.Vars) > 0 {
				fmt.Printf(",\n      \"variables\": {")
				keys := make([]string, 0, len(file.Vars))
				for k := range file.Vars {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for j, key := range keys {
					if j > 0 {
						fmt.Print(",")
					}
					fmt.Printf("\n        %q: %q", key, file.Vars[key])
				}
				fmt.Print("\n      }")
			}
			fmt.Print("\n    }")
			if i < len(result.Files)-1 {
				fmt.Print(",")
			}
			fmt.Println()
		}
		fmt.Println("  ],")
	}

	fmt.Println("  \"variables\": {")
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Printf("\n    %q: %q", key, env[key])
	}
	fmt.Println("\n  }")
	fmt.Println("}")
	return nil
}

// outputEnvFormat outputs environment variables in .env format
func (a *App) outputEnvFormat(env map[string]string) error {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		// Escape quotes and newlines in values
		value := strings.ReplaceAll(env[key], `"`, `\"`)
		value = strings.ReplaceAll(value, "\n", "\\n")
		value = strings.ReplaceAll(value, "\r", "\\r")
		fmt.Printf("%s=\"%s\"\n", key, value)
	}
	return nil
}

// outputShellFormat outputs environment variables as shell export statements
func (a *App) outputShellFormat(env map[string]string) error {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		// Escape quotes and newlines in values
		value := strings.ReplaceAll(env[key], `"`, `\"`)
		value = strings.ReplaceAll(value, "\n", "\\n")
		value = strings.ReplaceAll(value, "\r", "\\r")
		fmt.Printf("export %s=\"%s\"\n", key, value)
	}
	return nil
}

// outputTableFormat outputs environment variables in a table format
func (a *App) outputTableFormat(env map[string]string, result *envloader.LoadResult, showFiles, showSources bool) error {
	// Show environment info
	fmt.Printf("Environment: %s\n", result.Environment)
	fmt.Printf("Working Directory: %s\n", result.Files[0].Path[:strings.LastIndex(result.Files[0].Path, string(filepath.Separator))])
	fmt.Printf("Host Environment: %s\n", map[bool]string{true: "Included", false: "Not included"}[result.HostEnvIncluded])
	fmt.Println()

	// Show .env files info if requested
	if showFiles {
		fmt.Println("Environment Files:")
		for _, file := range result.Files {
			status := "Not found"
			if file.Exists {
				if file.Loaded {
					status = fmt.Sprintf("Loaded (%d variables)", len(file.Vars))
				} else {
					status = fmt.Sprintf("Failed (%v)", file.Error)
				}
			}
			fmt.Printf("  %s: %s\n", file.Path, status)
		}
		fmt.Println()
	}

	// Show variables
	fmt.Printf("Environment Variables (%d total):\n", len(env))
	fmt.Println()

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := env[key]

		// Mask sensitive values
		maskedValue := maskSensitiveValue(key, value)

		// Truncate very long values
		if len(maskedValue) > 100 {
			maskedValue = maskedValue[:97] + "..."
		}

		fmt.Printf("  %s=%s\n", key, maskedValue)
	}

	return nil
}

// maskSensitiveValue masks sensitive environment variable values
func maskSensitiveValue(key, value string) string {
	keyLower := strings.ToLower(key)

	// List of sensitive key patterns
	sensitivePatterns := []string{
		"password", "pass", "pwd",
		"secret", "key", "token",
		"auth", "credential", "cred",
		"api_key", "apikey",
		"private", "priv",
		"session", "cookie",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(keyLower, pattern) {
			if len(value) <= 4 {
				return "***"
			}
			return value[:2] + "***" + value[len(value)-2:]
		}
	}

	return value
}
