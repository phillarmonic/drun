package app

import (
	"os"

	"github.com/phillarmonic/drun/internal/engine"
	"github.com/spf13/cobra"
)

// Domain: Shell Completion
// This file contains logic for shell completion

// CompleteTaskNames provides autocompletion for task names
func CompleteTaskNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get config file from flag
	configFile, _ := cmd.Flags().GetString("file")

	// Try to find and parse the drun file
	actualConfigFile, err := FindConfigFile(configFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	content, err := os.ReadFile(actualConfigFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	program, err := engine.ParseStringWithFilename(string(content), actualConfigFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Create engine to get task list
	eng := engine.NewEngine(os.Stdout)
	tasks := eng.ListTasks(program)

	var completions []string
	for _, task := range tasks {
		completions = append(completions, task.Name+"\t[task] "+task.Description)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
