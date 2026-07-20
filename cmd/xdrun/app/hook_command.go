package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/engine"
	"github.com/phillarmonic/drun/v2/internal/gitpolicy"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
	"github.com/spf13/cobra"
)

// Domain: Git Hook Management
// This file contains CLI commands for managing git hooks that enforce drun's git policy.

func (a *App) createHookCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmd:hook",
		Short: "Manage git hooks for drun policies",
		Long: `Install, remove, and run git hooks that enforce the project's git policy.

Git policies are defined in the project's .drun/spec.drun file under the "git policy:" block.

Note: The 'cmd:' prefix is reserved for built-in commands to avoid conflicts with user tasks.`,
	}

	cmd.AddCommand(createHookInstallCommand(a))
	cmd.AddCommand(createHookUninstallCommand())
	cmd.AddCommand(createHookListCommand())
	cmd.AddCommand(createHookRunCommand(a))

	return cmd
}

var supportedHooks = []string{"commit-msg", "pre-push"}

func createHookInstallCommand(a *App) *cobra.Command {
	return &cobra.Command{
		Use:   "install [hook-name]",
		Short: "Install drun-managed git hooks",
		Long: `Install git hooks that enforce the project's git policy.
If no hook-name is specified, all supported hooks (commit-msg, pre-push) are installed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hooksToInstall := supportedHooks
			if len(args) == 1 {
				hooksToInstall = []string{args[0]}
			}

			gitDir := filepath.Join(".git", "hooks")
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				return fmt.Errorf("not a git repository (or no .git/hooks directory)")
			}

			for _, h := range hooksToInstall {
				err := installHook(gitDir, h)
				if err != nil {
					return fmt.Errorf("failed to install hook %s: %w", h, err)
				}
				fmt.Printf("✅ Installed hook: %s\n", h)
			}

			return nil
		},
	}
}

func createHookUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [hook-name]",
		Short: "Remove drun-managed git hooks",
		Long: `Remove git hooks that were installed by drun.
If no hook-name is specified, all drun-managed hooks are removed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hooksToRemove := supportedHooks
			if len(args) == 1 {
				hooksToRemove = []string{args[0]}
			}

			gitDir := filepath.Join(".git", "hooks")
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				return fmt.Errorf("not a git repository (or no .git/hooks directory)")
			}

			for _, h := range hooksToRemove {
				err := uninstallHook(gitDir, h)
				if err != nil {
					return fmt.Errorf("failed to uninstall hook %s: %w", h, err)
				}
			}

			return nil
		},
	}
}

func createHookListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed git hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			gitDir := filepath.Join(".git", "hooks")
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				return fmt.Errorf("not a git repository")
			}

			fmt.Println("Git Hooks Status:")
			for _, h := range supportedHooks {
				status := "Not installed"
				path := filepath.Join(gitDir, h)
				if content, err := os.ReadFile(path); err == nil {
					if strings.Contains(string(content), "# managed by drun") {
						status = "Installed (managed by drun)"
					} else {
						status = "Installed (custom)"
					}
				}
				fmt.Printf("  %-15s %s\n", h, status)
			}

			return nil
		},
	}
}

func createHookRunCommand(a *App) *cobra.Command {
	return &cobra.Command{
		Use:    "run <hook-name> [args...]",
		Short:  "Run a git hook validation manually",
		Hidden: true, // Typically invoked by the git hook script itself
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hookName := args[0]
			hookArgs := args[1:]

			// Parse the project config to get the git policy
			specPath := ".drun/spec.drun"
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				return fmt.Errorf("no %s found. Cannot run git policy hooks", specPath)
			}

			content, err := os.ReadFile(specPath)
			if err != nil {
				return fmt.Errorf("failed to read spec file: %w", err)
			}

			l := lexer.NewLexer(string(content))
			p := parser.NewParser(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				return fmt.Errorf("failed to parse spec file: %v", p.Errors())
			}

			eng := engine.NewEngine(os.Stdout)

			// Extract project settings
			ctx, err := eng.BuildProjectContext(program.Project, specPath)
			if err != nil {
				return fmt.Errorf("failed to load project context: %w", err)
			}

			if ctx.GitPolicy == nil {
				return fmt.Errorf("no git policy configured in project")
			}

			policyStmt := ctx.GitPolicy
			policy := &gitpolicy.Policy{
				DefaultBranches:      policyStmt.DefaultBranches,
				ProtectedBranches:    policyStmt.ProtectedBranches,
				BranchPattern:        policyStmt.BranchPattern,
				BranchTypes:          policyStmt.BranchTypes,
				CommitPattern:        policyStmt.CommitPattern,
				ExtractIdentifier:    policyStmt.ExtractIdentifier,
				CommitMinLength:      policyStmt.CommitMinLength,
				CommitBans:           policyStmt.CommitBans,
				EnforceSignedCommits: policyStmt.EnforceSignedCommits,
			}

			switch hookName {
			case "commit-msg":
				if len(hookArgs) < 1 {
					return fmt.Errorf("commit-msg hook requires the commit message file path as argument")
				}
				msgFile := hookArgs[0]
				msgBytes, err := os.ReadFile(msgFile)
				if err != nil {
					return fmt.Errorf("failed to read commit message file: %w", err)
				}

				// Validate branch name first
				branchName, err := eng.RunGitCommandOutput("rev-parse", "--abbrev-ref", "HEAD")
				if err != nil {
					return fmt.Errorf("failed to get current branch: %w", err)
				}
				branchName = strings.TrimSpace(branchName)

				if err := policy.ValidateBranchName(branchName); err != nil {
					fmt.Printf("❌ Invalid branch name: %v\n", err)
					return err
				}

				// Then validate commit message
				if err := policy.ValidateCommitMessage(string(msgBytes), branchName); err != nil {
					fmt.Printf("❌ Invalid commit message: %v\n", err)
					return err
				}

				fmt.Println("✅ Git policy checks passed.")
				return nil

			case "pre-push":
				if policy.EnforceSignedCommits {
					// Validate signed commits
					out, err := eng.RunGitCommandOutput("log", "-1", "--format=%G?")
					if err != nil {
						return fmt.Errorf("failed to check commit signature: %w", err)
					}
					sigStatus := strings.TrimSpace(out)
					if sigStatus != "G" && sigStatus != "U" {
						fmt.Printf("❌ Commit is not properly signed (signature status: '%s')\n", sigStatus)
						return fmt.Errorf("unsigned commit")
					}
					fmt.Println("✅ Signed commits check passed.")
				}
				return nil

			default:
				return fmt.Errorf("unsupported hook name: %s", hookName)
			}
		},
	}
}

func installHook(gitDir, hookName string) error {
	path := filepath.Join(gitDir, hookName)

	// Check if exists and not managed by drun
	if content, err := os.ReadFile(path); err == nil {
		if !strings.Contains(string(content), "# managed by drun") {
			return fmt.Errorf("custom hook already exists at %s", path)
		}
	}

	script := fmt.Sprintf(`#!/bin/sh
# managed by drun

xdrun cmd:hook run %s "$@"
`, hookName)

	return os.WriteFile(path, []byte(script), 0755)
}

func uninstallHook(gitDir, hookName string) error {
	path := filepath.Join(gitDir, hookName)

	if content, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(content), "# managed by drun") {
			err = os.Remove(path)
			if err != nil {
				return err
			}
			fmt.Printf("✅ Uninstalled hook: %s\n", hookName)
		} else {
			fmt.Printf("⚠️ Skipped %s (not managed by drun)\n", hookName)
		}
	}
	return nil
}
