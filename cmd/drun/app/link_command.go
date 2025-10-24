package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

// createLinkCommand registers cmd:link for directory associations.
func (a *App) createLinkCommand() *cobra.Command {
	var taskFile string

	cmd := &cobra.Command{
		Use:   "cmd:link <directory[,directory...]> [more directories...]",
		Short: "Link directories to a drun task file",
		Long: `Link one or more directories to a drun task file so xdrun commands
can be executed from any of the linked locations.

Examples:
  xdrun cmd:link services/api,services/web              # Link directories using the current task file
  xdrun cmd:link ../app --file ../ops/drun/spec.drun    # Link directory to the specified task file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := LinkDirectories(args, taskFile); err != nil {
				return err
			}
			fmt.Println("✅ Link configuration updated")
			return nil
		},
	}

	cmd.Flags().StringVar(&taskFile, "file", "", "Task file to link (default: detected from current directory)")

	return cmd
}

// createUnlinkCommand registers cmd:unlink.
func (a *App) createUnlinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cmd:unlink <directory[,directory...]> [more directories...]",
		Short: "Unlink directories from their associated task file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := UnlinkDirectories(args); err != nil {
				return err
			}
			return nil
		},
	}
}

// createUnlinkAllCommand registers cmd:unlink-all.
func (a *App) createUnlinkAllCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cmd:unlink-all",
		Short: "Remove all directory links",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UnlinkAllDirectories()
		},
	}
}
