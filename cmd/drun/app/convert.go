package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/drun/internal/make2drun"
	"github.com/spf13/cobra"
)

// createConvertCommand creates the cmd:from subcommand with converters
func (a *App) createConvertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cmd:from",
		Short: "Convert other build tools to drun format",
		Long: `Convert from other build tools and task runners to drun format.

Supported formats:
  • makefile - Convert Makefiles to drun tasks

Examples:
  xdrun cmd:from makefile                              # Convert ./Makefile to Makefile.drun
  xdrun cmd:from makefile -i myproject.mk -o tasks.drun

Note: The 'cmd:' prefix is reserved for built-in commands to avoid conflicts with user tasks.
`,
	}

	cmd.AddCommand(createMakefileConvertCommand())
	return cmd
}

// createMakefileConvertCommand creates the makefile converter subcommand
func createMakefileConvertCommand() *cobra.Command {
	var (
		inputFile  string
		outputFile string
	)

	cmd := &cobra.Command{
		Use:   "makefile [flags]",
		Short: "Convert Makefile to drun format",
		Long: `Convert a Makefile to drun v2 format.

This command parses a Makefile and generates an equivalent drun task file.
It handles:
  • Targets → drun tasks
  • Dependencies → 'depends on' declarations
  • Variables → drun variables with interpolation
  • Comments → task descriptions
  • Shell commands → appropriate drun actions
  • .PHONY targets
  • @ prefix (silent) → preserved as appropriate drun actions
  • - prefix (ignore errors) → wrapped in try/ignore blocks

Examples:
  xdrun cmd:from makefile                              # Convert ./Makefile to Makefile.drun
  xdrun cmd:from makefile -i myproject.mk -o tasks.drun
  xdrun cmd:from makefile --input Makefile --output build.drun
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return convertMakefile(inputFile, outputFile)
		},
	}

	cmd.Flags().StringVarP(&inputFile, "input", "i", "Makefile", "Path to input Makefile")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path to output .drun file (default: <input>.drun)")

	return cmd
}

// convertMakefile handles the Makefile to drun conversion
func convertMakefile(inputFile, outputFile string) error {
	// Determine output file
	output := outputFile
	if output == "" {
		// Generate output filename from input
		base := filepath.Base(inputFile)
		if base == "Makefile" || base == "makefile" {
			output = "Makefile.drun"
		} else {
			output = strings.TrimSuffix(base, filepath.Ext(base)) + ".drun"
		}
	}

	// Parse the Makefile
	fmt.Printf("📖 Reading Makefile: %s\n", inputFile)
	makefile, err := make2drun.ParseMakefile(inputFile)
	if err != nil {
		return fmt.Errorf("error parsing Makefile: %w", err)
	}

	fmt.Printf("✅ Found %d targets and %d variables\n", len(makefile.Targets), len(makefile.Variables))

	// Generate drun file
	fmt.Printf("🔄 Converting to drun v2 syntax...\n")
	drunContent := make2drun.GenerateDrun(makefile)

	// Write output file
	fmt.Printf("💾 Writing to: %s\n", output)
	err = os.WriteFile(output, []byte(drunContent), 0644)
	if err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	fmt.Printf("🎉 Successfully converted Makefile to drun!\n")
	fmt.Printf("\nYou can now run your tasks with:\n")
	fmt.Printf("  xdrun -f %s <task-name>\n", output)

	return nil
}
