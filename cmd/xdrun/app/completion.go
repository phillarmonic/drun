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

	// User-defined tasks come first
	for _, task := range tasks {
		completions = append(completions, task.Name+"\t[task] "+task.Description)
	}

	// Built-in cmd:* commands come after user tasks
	completions = append(completions, builtinCmdCompletions(cmd)...)

	// KeepOrder ensures zsh/fish respect the order we provide (tasks before builtins)
	return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// builtinCmdCompletions returns the cmd:* subcommand completions in a consistent order.
// These are appended after user tasks so user-defined names appear first in the list.
func builtinCmdCompletions(cmd *cobra.Command) []string {
	var completions []string
	for _, sub := range cmd.Root().Commands() {
		if len(sub.Name()) > 4 && sub.Name()[:4] == "cmd:" {
			completions = append(completions, sub.Name()+"\t"+sub.Short)
		}
	}
	return completions
}
