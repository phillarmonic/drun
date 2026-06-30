package app

import (
	"os"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/engine"
	"github.com/phillarmonic/drun/v2/internal/platform"
	"github.com/spf13/cobra"
)

var defaultBuiltinCmdNames = map[string]struct{}{
	"cmd:from":       {},
	"cmd:link":       {},
	"cmd:secret":     {},
	"cmd:skill":      {},
	"cmd:stateless":  {},
	"cmd:unlink":     {},
	"cmd:unlink-all": {},
}

// Domain: Shell Completion
// This file contains logic for shell completion

// CompleteTaskNames provides autocompletion for task names
func CompleteTaskNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	builtins := builtinCmdCompletions(cmd, toComplete, shouldIncludeBuiltinByDefault)
	if isCmdNamespacePrefix(toComplete) {
		return builtinCmdCompletions(cmd, toComplete, includeAllBuiltins), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	}

	// Get config file from flag
	configFile, _ := cmd.Flags().GetString("file")

	// Try to find and parse the drun file
	actualConfigFile, err := FindConfigFile(configFile)
	if err != nil {
		return builtins, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	}

	// #nosec G304 -- completion intentionally parses the discovered drun task file.
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
	type completionMeta struct {
		description string
		platforms   []string
	}
	families := make(map[string]*completionMeta, len(tasks))

	for _, task := range tasks {
		meta, exists := families[task.Name]
		if !exists {
			families[task.Name] = &completionMeta{
				description: task.Description,
				platforms:   append([]string(nil), task.Platforms...),
			}
			continue
		}
		meta.platforms = append(meta.platforms, task.Platforms...)
		if meta.description == "" || meta.description == "No description" {
			meta.description = task.Description
		}
	}

	for _, task := range tasks {
		meta, exists := families[task.Name]
		if !exists {
			continue
		}
		description := "[task] " + meta.description
		if len(meta.platforms) > 0 {
			description += " [" + platform.FormatList(uniquePlatforms(meta.platforms)) + "]"
		}
		completions = append(completions, task.Name+"\t"+description)
		delete(families, task.Name)
	}

	// Built-in cmd:* commands come after user tasks
	completions = append(completions, builtins...)

	// KeepOrder ensures zsh/fish respect the order we provide (tasks before builtins)
	return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// builtinCmdCompletions returns the cmd:* subcommand completions in a consistent order.
// These are appended after user tasks so user-defined names appear first in the list.
func builtinCmdCompletions(cmd *cobra.Command, toComplete string, include func(name string) bool) []string {
	var completions []string
	for _, sub := range cmd.Root().Commands() {
		if len(sub.Name()) > 4 && sub.Name()[:4] == "cmd:" {
			if toComplete != "" && !strings.HasPrefix(sub.Name(), toComplete) {
				continue
			}
			if !include(sub.Name()) {
				continue
			}
			completions = append(completions, sub.Name()+"\t"+sub.Short)
		}
	}
	return completions
}

func isCmdNamespacePrefix(toComplete string) bool {
	if toComplete == "" {
		return false
	}

	return strings.HasPrefix("cmd:", toComplete)
}

func shouldIncludeBuiltinByDefault(name string) bool {
	_, ok := defaultBuiltinCmdNames[name]
	return ok
}

func includeAllBuiltins(string) bool {
	return true
}

func uniquePlatforms(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
