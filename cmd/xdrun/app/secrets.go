package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phillarmonic/drun/internal/secrets"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Domain: Secret Management Commands
// This file contains CLI commands for managing secrets (add, remove, list)

// createSecretsCommand creates the cmd:secret subcommand for managing secrets
func (a *App) createSecretsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmd:secret",
		Short: "Manage secrets (add, remove, list)",
		Long: `Manage secrets stored in the system keychain or encrypted storage.

Secrets can be stored in different namespaces:
- Project scope: Uses the current project name as namespace
- Global scope: Uses a shared "global" namespace
- Custom scope: Specify any namespace name

Note: The 'cmd:' prefix is reserved for built-in commands to avoid conflicts with user tasks.`,
	}

	// Add subcommands
	cmd.AddCommand(createSecretAddCommand())
	cmd.AddCommand(createSecretRemoveCommand())
	cmd.AddCommand(createSecretListCommand())
	cmd.AddCommand(createSecretListAllCommand())

	return cmd
}

// createSecretAddCommand creates the "add" subcommand
func createSecretAddCommand() *cobra.Command {
	var (
		namespace    string
		projectScope bool
		globalScope  bool
		value        string
		masked       bool
	)

	cmd := &cobra.Command{
		Use:   "add <key> [value]",
		Short: "Add or update a secret",
		Long: `Add or update a secret in the keychain.

By default, secrets are stored in the "default" namespace.
Use --project to store in project scope (uses current directory name).
Use --global to store in global scope.
Use --namespace to specify a custom namespace.

If no value is provided, you'll be prompted to enter it securely.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			// Get the secret value
			var secretValue string
			if len(args) == 2 {
				secretValue = args[1]
			} else if value != "" {
				secretValue = value
			} else {
				// Prompt for value
				var err error
				if masked {
					secretValue, err = promptForSecret(fmt.Sprintf("Enter value for '%s': ", key))
				} else {
					secretValue, err = promptForInput(fmt.Sprintf("Enter value for '%s': ", key))
				}
				if err != nil {
					return fmt.Errorf("failed to read value: %w", err)
				}
			}

			// Determine namespace
			ns, err := determineNamespace(namespace, projectScope, globalScope)
			if err != nil {
				return err
			}

			// Create secrets manager
			mgr, err := secrets.NewManager()
			if err != nil {
				return fmt.Errorf("failed to initialize secrets manager: %w", err)
			}

			// Store the secret
			if err := mgr.Set(ns, key, secretValue); err != nil {
				return fmt.Errorf("failed to store secret: %w", err)
			}

			fmt.Printf("✓ Secret '%s' stored in namespace '%s'\n", key, ns)
			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Custom namespace for the secret")
	cmd.Flags().BoolVarP(&projectScope, "project", "p", false, "Store in project scope (uses current directory name)")
	cmd.Flags().BoolVarP(&globalScope, "global", "g", false, "Store in global scope")
	cmd.Flags().StringVarP(&value, "value", "v", "", "Secret value (if not provided, will prompt)")
	cmd.Flags().BoolVarP(&masked, "masked", "m", false, "Hide input when prompting for value")

	return cmd
}

// createSecretRemoveCommand creates the "remove" subcommand
func createSecretRemoveCommand() *cobra.Command {
	var (
		namespace    string
		projectScope bool
		globalScope  bool
	)

	cmd := &cobra.Command{
		Use:     "remove <key>",
		Aliases: []string{"delete", "rm"},
		Short:   "Remove a secret",
		Long: `Remove a secret from the keychain.

By default, removes from the "default" namespace.
Use --project to remove from project scope (uses current directory name).
Use --global to remove from global scope.
Use --namespace to specify a custom namespace.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			// Determine namespace
			ns, err := determineNamespace(namespace, projectScope, globalScope)
			if err != nil {
				return err
			}

			// Create secrets manager
			mgr, err := secrets.NewManager()
			if err != nil {
				return fmt.Errorf("failed to initialize secrets manager: %w", err)
			}

			// Delete the secret
			if err := mgr.Delete(ns, key); err != nil {
				return fmt.Errorf("failed to delete secret: %w", err)
			}

			fmt.Printf("✓ Secret '%s' removed from namespace '%s'\n", key, ns)
			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Custom namespace for the secret")
	cmd.Flags().BoolVarP(&projectScope, "project", "p", false, "Remove from project scope (uses current directory name)")
	cmd.Flags().BoolVarP(&globalScope, "global", "g", false, "Remove from global scope")

	return cmd
}

// createSecretListCommand creates the "list" subcommand
func createSecretListCommand() *cobra.Command {
	var (
		namespace    string
		projectScope bool
		globalScope  bool
		showValues   bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List secrets in a namespace",
		Long: `List all secrets in a specific namespace.

By default, lists secrets from the "default" namespace.
Use --project to list from project scope (uses current directory name).
Use --global to list from global scope.
Use --namespace to specify a custom namespace.

Note: By default, only secret keys are shown, not values.
Use --show-values to display secret values (use with caution).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine namespace
			ns, err := determineNamespace(namespace, projectScope, globalScope)
			if err != nil {
				return err
			}

			// Create secrets manager
			mgr, err := secrets.NewManager()
			if err != nil {
				return fmt.Errorf("failed to initialize secrets manager: %w", err)
			}

			// List secrets
			keys, err := mgr.List(ns)
			if err != nil {
				return fmt.Errorf("failed to list secrets: %w", err)
			}

			if len(keys) == 0 {
				fmt.Printf("No secrets found in namespace '%s'\n", ns)
				return nil
			}

			// Sort keys for consistent output
			sort.Strings(keys)

			fmt.Printf("Secrets in namespace '%s':\n", ns)
			for _, key := range keys {
				if showValues {
					value, err := mgr.Get(ns, key)
					if err != nil {
						fmt.Printf("  - %s: <error: %v>\n", key, err)
					} else {
						fmt.Printf("  - %s: %s\n", key, value)
					}
				} else {
					fmt.Printf("  - %s\n", key)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Custom namespace to list secrets from")
	cmd.Flags().BoolVarP(&projectScope, "project", "p", false, "List from project scope (uses current directory name)")
	cmd.Flags().BoolVarP(&globalScope, "global", "g", false, "List from global scope")
	cmd.Flags().BoolVar(&showValues, "show-values", false, "Show secret values (use with caution)")

	return cmd
}

// createSecretListAllCommand creates the "list-all" subcommand
func createSecretListAllCommand() *cobra.Command {
	var showValues bool

	cmd := &cobra.Command{
		Use:   "list-all",
		Short: "List all secrets across all namespaces",
		Long: `List all secrets organized by namespace.

This command shows all namespaces and their secrets.

Note: By default, only secret keys are shown, not values.
Use --show-values to display secret values (use with caution).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create secrets manager
			mgr, err := secrets.NewManager()
			if err != nil {
				return fmt.Errorf("failed to initialize secrets manager: %w", err)
			}

			// List all namespaces
			namespaces, err := mgr.ListNamespaces()
			if err != nil {
				return fmt.Errorf("failed to list namespaces: %w", err)
			}

			if len(namespaces) == 0 {
				fmt.Println("No secrets found in any namespace")
				return nil
			}

			// Sort namespaces for consistent output
			sort.Strings(namespaces)

			fmt.Println("All secrets:")
			fmt.Println()

			for _, ns := range namespaces {
				keys, err := mgr.List(ns)
				if err != nil {
					fmt.Printf("Namespace '%s': <error: %v>\n", ns, err)
					continue
				}

				if len(keys) == 0 {
					continue
				}

				sort.Strings(keys)
				fmt.Printf("Namespace '%s' (%d secret%s):\n", ns, len(keys), pluralize(len(keys)))

				for _, key := range keys {
					if showValues {
						value, err := mgr.Get(ns, key)
						if err != nil {
							fmt.Printf("  - %s: <error: %v>\n", key, err)
						} else {
							fmt.Printf("  - %s: %s\n", key, value)
						}
					} else {
						fmt.Printf("  - %s\n", key)
					}
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showValues, "show-values", false, "Show secret values (use with caution)")

	return cmd
}

// Helper functions

// determineNamespace determines which namespace to use based on flags
func determineNamespace(namespace string, projectScope, globalScope bool) (string, error) {
	// Count how many options are set
	options := 0
	if namespace != "" {
		options++
	}
	if projectScope {
		options++
	}
	if globalScope {
		options++
	}

	// Only one option should be set
	if options > 1 {
		return "", fmt.Errorf("only one of --namespace, --project, or --global can be specified")
	}

	// Determine the namespace
	if namespace != "" {
		return namespace, nil
	}

	if projectScope {
		// Use current directory name as project name
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		return filepath.Base(cwd), nil
	}

	if globalScope {
		return "global", nil
	}

	// Default namespace
	return "default", nil
}

// promptForSecret prompts for a secret value (masked input)
func promptForSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line after password input
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// promptForInput prompts for a value (visible input)
func promptForInput(prompt string) (string, error) {
	fmt.Print(prompt)
	var value string
	_, err := fmt.Scanln(&value)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

// pluralize returns "s" if count is not 1, empty string otherwise
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
