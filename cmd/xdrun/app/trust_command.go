package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) createTrustCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cmd:trust [directory]",
		Short: "Trust the current (or specified) directory for security-sensitive operations",
		Long: `Marks a directory as trusted so that security-sensitive statements such as
"open url" are allowed to execute without prompting.

Trust is checked against the directory of the task file (not the working
directory). Parent directories also satisfy the check: trusting /home/user/repos
covers all repositories underneath it.

The trusted directory list is stored in ~/.drun/trusted.yml.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			if err := TrustDir(dir); err != nil {
				return err
			}
			fmt.Printf("Trusted: %s\n", dir)
			return nil
		},
	}
}

func (a *App) createUntrustCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cmd:untrust [directory]",
		Short: "Remove trust for the current (or specified) directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			if err := UntrustDir(dir); err != nil {
				return err
			}
			fmt.Printf("Untrusted: %s\n", dir)
			return nil
		},
	}
}
