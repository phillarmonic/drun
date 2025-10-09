package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/engine"
	"github.com/phillarmonic/drun/internal/errors"
	"github.com/phillarmonic/drun/internal/secrets"
)

// Domain: Task Execution
// This file contains logic for loading and running drun tasks

// ExecuteTask executes a drun task with the given parameters
func ExecuteTask(
	configFile string,
	listTasks bool,
	dryRun bool,
	verbose bool,
	allowUndefinedVars bool,
	noDrunCache bool,
	args []string,
) error {
	// Determine the config file to use
	actualConfigFile, err := FindConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("no drun task file found: %w\n\nTo get started:\n  drun --init          # Create .drun/spec.drun", err)
	}

	// Verbose: Show we're starting
	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "📂 Loading: %s\n", actualConfigFile)
	}

	// Read the drun file
	content, err := os.ReadFile(actualConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read drun file '%s': %w", actualConfigFile, err)
	}

	if verbose {
		_, _ = fmt.Fprintf(os.Stdout, "🔍 Parsing drun file...\n")
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
		_, _ = fmt.Fprintf(os.Stdout, "✅ Parsed successfully\n")
	}

	// Initialize secrets manager - use fallback backend for now
	// TODO: Fix platform backends to handle all characters in namespace names
	secretsMgr, err := secrets.NewManager(secrets.WithFallback())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: Failed to initialize secrets manager: %v\n", err)
		secretsMgr = nil
	}

	// Create engine with secrets support
	eng := engine.NewEngineWithOptions(
		engine.WithOutput(os.Stdout),
		engine.WithDryRun(dryRun),
		engine.WithVerbose(verbose),
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
		target = args[0]
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
		fmt.Printf("  %-20s  %s\n", task.Name, task.Description)
	}

	return nil
}

// FindDefaultTask finds a default task to run
func FindDefaultTask(program *ast.Program) string {
	// Look for common default task names
	defaultNames := []string{"default", "help", "start", "main"}

	for _, task := range program.Tasks {
		for _, defaultName := range defaultNames {
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
