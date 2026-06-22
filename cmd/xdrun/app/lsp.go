package app

import (
	"os"

	"github.com/phillarmonic/drun/v2/internal/lsp"
	"github.com/spf13/cobra"
)

func (a *App) createLSPCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cmd:lsp",
		Short: "Start a simple Language Server Protocol server over stdio",
		Long: `Start a simple Language Server Protocol server over stdio.

This command is intended for editor integrations. It currently supports:
- initialize / shutdown / exit
- full text document sync
- parser-backed diagnostics
- simple keyword and task-name completions

Example:
  xdrun cmd:lsp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			server := lsp.NewServer(os.Stdin, os.Stdout)
			return server.Run()
		},
		Args: cobra.NoArgs,
	}
}
