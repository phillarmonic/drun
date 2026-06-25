package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/engine"
	"github.com/phillarmonic/drun/v2/internal/errors"
	"github.com/phillarmonic/drun/v2/internal/platform"
	"github.com/phillarmonic/drun/v2/internal/secrets"
)

// Domain: Task Execution
// This file contains logic for loading and running drun tasks

// ExecuteTask executes a drun task with the given parameters
func ExecuteTask(
	configFile string,
	listTasks bool,
	dryRun bool,
	verbose bool,
	taskModeOverride string,
	allowUndefinedVars bool,
	noDrunCache bool,
	args []string,
) error {
	taskModeOverride, err := normalizeRuntimeTaskMode(taskModeOverride)
	if err != nil {
		return err
	}

	// Determine the config file to use
	actualConfigFile, err := FindConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("no drun task file found: %w\n\nTo get started:\n  drun --init          # Create .drun/spec.drun", err)
	}

	// Verbose: Show we're starting
	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "📂  Loading: %s\n", actualConfigFile)
	}

	// Read the drun file
	// #nosec G304 -- task execution intentionally reads the discovered drun task file.
	content, err := os.ReadFile(actualConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read drun file '%s': %w", actualConfigFile, err)
	}

	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "🔍  Parsing drun file...\n")
	}

	// Parse the drun file
	program, err := engine.ParseStringWithFilename(string(content), actualConfigFile)
	if err != nil {
		// Check if it's an enhanced error list
		if errorList, ok := err.(*errors.ParseErrorList); ok {
			fmt.Fprint(os.Stderr, errorList.FormatErrors())
			os.Exit(1)
		}
		return fmt.Errorf("failed to parse drun file '%s': %w", actualConfigFile, err)
	}

	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "✅  Parsed successfully\n")
	}

	// Initialize secrets manager using platform-appropriate backend
	secretsMgr, err := secrets.NewManager()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to initialize secrets manager: %v\n", err)
		secretsMgr = nil
	}

	// Create engine with secrets support
	eng := engine.NewEngineWithOptions(
		engine.WithOutput(os.Stdout),
		engine.WithDryRun(dryRun),
		engine.WithVerbose(verbose),
		engine.WithTaskModeOverride(taskModeOverride),
		engine.WithSecretsManager(secretsMgr),
	)
	eng.SetAllowUndefinedVars(allowUndefinedVars)

	if verbose {
		if noDrunCache {
			_, _ = fmt.Fprintf(os.Stdout, "💾 Remote include caching: disabled\n")
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "💾 Remote include caching: enabled (1m expiration)\n")
		}
	}

	// Initialize cache for remote includes
	if err := eng.SetCacheEnabled(!noDrunCache); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to initialize remote include cache: %v\n", err)
	}

	// Ensure cleanup of temporary files
	defer eng.Cleanup()

	// Handle --list flag
	if listTasks {
		return ListAllTasks(eng, program)
	}

	// Determine target task and parse parameters
	var target string
	var params map[string]string

	if len(args) == 0 {
		// No arguments - try to find a default task or list tasks
		if defaultTask := FindDefaultTask(program); defaultTask != "" {
			target = defaultTask
		} else {
			return ListAllTasks(eng, program)
		}
		params = make(map[string]string)
	} else {
		// Resolve partial task name to full task name
		partialName := args[0]
		resolvedName, err := ResolvePartialTaskName(partialName, program)
		if err != nil {
			return fmt.Errorf("%w\n\nRun 'xdrun --list' to see all available tasks", err)
		}
		target = resolvedName

		// Show which task was resolved if it's a partial match
		if verbose && partialName != resolvedName {
			_, _ = fmt.Fprintf(os.Stdout, "🎯 Resolved '%s' → '%s'\n", partialName, resolvedName)
		}

		params = ParseTaskParameters(args[1:])
	}

	// Execute the task with parameters
	err = eng.ExecuteWithParamsAndFile(program, target, params, actualConfigFile)
	if err != nil {
		// Check if it's a parameter validation error
		if paramErr, ok := err.(*errors.ParameterValidationError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", paramErr.Message)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: execution failed: %v\n", err)
		os.Exit(1)
	}

	return nil
}

// ListAllTasks lists all available tasks
func ListAllTasks(eng *engine.Engine, program *ast.Program) error {
	fmt.Println("Available tasks:")

	tasks := eng.ListTasks(program)
	if len(tasks) == 0 {
		fmt.Println("  (no tasks defined)")
		return nil
	}

	for _, task := range tasks {
		platformSuffix := ""
		if len(task.Platforms) > 0 {
			platformSuffix = " [" + platform.FormatList(task.Platforms) + "]"
		}
		fmt.Printf("  %-20s  %s\n", task.Name+platformSuffix, task.Description)
	}

	return nil
}

// FindDefaultTask finds a default task to run
func FindDefaultTask(program *ast.Program) string {
	// Look for common default task names. We restrict this to safe informational
	// tasks so orchestrations do not launch unless explicitly requested.
	defaultNames := []string{"default", "help"}

	for _, defaultName := range defaultNames {
		for _, task := range program.Tasks {
			if task.Name == defaultName {
				return task.Name
			}
		}
	}

	return ""
}

// ParseTaskParameters parses task parameters from command line arguments
// Supports format: param1=value1 param2=value2
func ParseTaskParameters(args []string) map[string]string {
	params := make(map[string]string)

	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}
	}

	return params
}
