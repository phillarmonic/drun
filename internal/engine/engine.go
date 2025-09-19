package engine

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/builtins"
	"github.com/phillarmonic/drun/internal/detection"
	"github.com/phillarmonic/drun/internal/errors"
	"github.com/phillarmonic/drun/internal/fileops"
	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parallel"
	"github.com/phillarmonic/drun/internal/parser"
	"github.com/phillarmonic/drun/internal/shell"
	"github.com/phillarmonic/drun/internal/types"
)

// Engine executes drun v2 programs directly
type Engine struct {
	output  io.Writer
	dryRun  bool
	verbose bool
}

// ExecutionContext holds parameter values and other runtime context
type ExecutionContext struct {
	Parameters map[string]*types.Value // parameter name -> typed value
	Variables  map[string]string       // captured variables from shell commands
	Project    *ProjectContext         // project-level settings and hooks
}

// ProjectContext holds project-level configuration
type ProjectContext struct {
	Name         string                              // project name
	Version      string                              // project version
	Settings     map[string]string                   // project settings (set key to value)
	BeforeHooks  []ast.Statement                     // before any task hooks
	AfterHooks   []ast.Statement                     // after any task hooks
	ShellConfigs map[string]*ast.PlatformShellConfig // platform-specific shell configurations
}

// NewEngine creates a new v2 execution engine
func NewEngine(output io.Writer) *Engine {
	if output == nil {
		output = os.Stdout
	}
	return &Engine{
		output: output,
		dryRun: false,
	}
}

// SetDryRun enables or disables dry run mode
func (e *Engine) SetDryRun(dryRun bool) {
	e.dryRun = dryRun
}

// SetVerbose enables or disables verbose mode
func (e *Engine) SetVerbose(verbose bool) {
	e.verbose = verbose
}

// Execute runs a v2 program with no parameters
func (e *Engine) Execute(program *ast.Program, taskName string) error {
	return e.ExecuteWithParams(program, taskName, map[string]string{})
}

// ExecuteWithParams runs a v2 program with the given parameters
func (e *Engine) ExecuteWithParams(program *ast.Program, taskName string, params map[string]string) error {
	if program == nil {
		return fmt.Errorf("program is nil")
	}

	// Create dependency resolver
	resolver := NewDependencyResolver(program.Tasks)

	// Validate all dependencies first
	if err := resolver.ValidateAllDependencies(); err != nil {
		return fmt.Errorf("dependency validation failed: %v", err)
	}

	// Resolve execution order
	executionOrder, err := resolver.ResolveDependencies(taskName)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %v", err)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Execution order: %v\n", executionOrder)
	}

	// Create execution context with parameters
	ctx := &ExecutionContext{
		Parameters: make(map[string]*types.Value),
		Variables:  make(map[string]string),
		Project:    e.createProjectContext(program.Project),
	}

	// Execute all tasks in dependency order
	for _, currentTaskName := range executionOrder {
		// Find the task
		var currentTask *ast.TaskStatement
		for _, task := range program.Tasks {
			if task.Name == currentTaskName {
				currentTask = task
				break
			}
		}

		if currentTask == nil {
			return fmt.Errorf("task '%s' not found during execution", currentTaskName)
		}

		// Set up parameters for this specific task
		if err := e.setupTaskParameters(currentTask, params, ctx); err != nil {
			return err
		}

		// Execute before hooks only for the target task
		if currentTaskName == taskName && ctx.Project != nil {
			for _, hook := range ctx.Project.BeforeHooks {
				if err := e.executeStatement(hook, ctx); err != nil {
					return fmt.Errorf("before hook failed: %v", err)
				}
			}
		}

		// Execute the task
		if err := e.executeTask(currentTask, ctx); err != nil {
			return fmt.Errorf("task '%s' failed: %v", currentTaskName, err)
		}

		// Execute after hooks only for the target task
		if currentTaskName == taskName && ctx.Project != nil {
			for _, hook := range ctx.Project.AfterHooks {
				if hookErr := e.executeStatement(hook, ctx); hookErr != nil {
					_, _ = fmt.Fprintf(e.output, "⚠️  After hook failed: %v\n", hookErr)
				}
			}
		}
	}

	return nil
}

// setupTaskParameters sets up parameters for a specific task
func (e *Engine) setupTaskParameters(task *ast.TaskStatement, params map[string]string, ctx *ExecutionContext) error {
	// Set up parameters with defaults and validation
	for _, param := range task.Parameters {
		var rawValue string
		var hasValue bool

		if providedValue, exists := params[param.Name]; exists {
			rawValue = providedValue
			hasValue = true
		} else if !param.Required {
			// For optional parameters (given/accepts), use default value (including empty string)
			rawValue = param.DefaultValue
			hasValue = true
		} else if param.Required {
			return errors.NewParameterValidationError(fmt.Sprintf("required parameter '%s' not provided", param.Name))
		}

		// Create typed value if we have a value
		if hasValue {
			// Determine parameter type
			paramType, err := types.ParseParameterType(param.DataType)
			if err != nil {
				// Fall back to type inference if parsing fails
				paramType = types.InferType(rawValue)
			}

			// Create typed value
			typedValue, err := types.NewValue(paramType, rawValue)
			if err != nil {
				return errors.NewParameterValidationError(fmt.Sprintf("parameter '%s': invalid %s value '%s': %v",
					param.Name, paramType, rawValue, err))
			}

			// Validate basic constraints (list constraints)
			if err := typedValue.ValidateConstraints(param.Constraints); err != nil {
				return errors.NewParameterValidationError(fmt.Sprintf("parameter '%s': %v", param.Name, err))
			}

			// Validate advanced constraints
			if err := typedValue.ValidateAdvancedConstraints(param.MinValue, param.MaxValue, param.Pattern, param.PatternMacro, param.EmailFormat); err != nil {
				return errors.NewParameterValidationError(fmt.Sprintf("parameter '%s': %v", param.Name, err))
			}

			ctx.Parameters[param.Name] = typedValue
		}
	}

	return nil
}

// createProjectContext creates a project context from the project statement
func (e *Engine) createProjectContext(project *ast.ProjectStatement) *ProjectContext {
	if project == nil {
		return nil
	}

	ctx := &ProjectContext{
		Name:         project.Name,
		Version:      project.Version,
		Settings:     make(map[string]string),
		BeforeHooks:  []ast.Statement{},
		AfterHooks:   []ast.Statement{},
		ShellConfigs: make(map[string]*ast.PlatformShellConfig),
	}

	// Process project settings
	for _, setting := range project.Settings {
		switch s := setting.(type) {
		case *ast.SetStatement:
			ctx.Settings[s.Key] = s.Value
		case *ast.LifecycleHook:
			switch s.Type {
			case "before":
				ctx.BeforeHooks = append(ctx.BeforeHooks, s.Body...)
			case "after":
				ctx.AfterHooks = append(ctx.AfterHooks, s.Body...)
			}
		case *ast.ShellConfigStatement:
			// Store shell configurations for each platform
			for platform, config := range s.Platforms {
				ctx.ShellConfigs[platform] = config
			}
		}
	}

	return ctx
}

// getPlatformShellConfig returns the shell configuration for the current platform
func (e *Engine) getPlatformShellConfig(ctx *ExecutionContext) *shell.Options {
	opts := shell.DefaultOptions()

	if ctx.Project == nil || len(ctx.Project.ShellConfigs) == 0 {
		return opts
	}

	// Determine current platform
	platform := runtime.GOOS

	// Get platform-specific configuration
	config, exists := ctx.Project.ShellConfigs[platform]
	if !exists {
		return opts
	}

	// Apply platform configuration
	if config.Executable != "" {
		opts.Shell = config.Executable
	}

	// Add startup arguments to environment or handle them appropriately
	// Note: The shell package currently doesn't support startup args directly,
	// so we'll store them in environment for now
	if len(config.Args) > 0 {
		if opts.Environment == nil {
			opts.Environment = make(map[string]string)
		}
		// Store args as a space-separated string for now
		opts.Environment["DRUN_SHELL_ARGS"] = strings.Join(config.Args, " ")
	}

	// Apply environment variables
	if len(config.Environment) > 0 {
		if opts.Environment == nil {
			opts.Environment = make(map[string]string)
		}
		for key, value := range config.Environment {
			opts.Environment[key] = value
		}
	}

	return opts
}

// executeTask executes a single task with the given context
func (e *Engine) executeTask(task *ast.TaskStatement, ctx *ExecutionContext) error {
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute task: %s\n", task.Name)
		if task.Description != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Description: %s\n", task.Description)
		}
		// Process statements in dry run mode
		for _, stmt := range task.Body {
			if err := e.executeStatement(stmt, ctx); err != nil {
				return err
			}
		}
		return nil
	}

	// Execute each statement in the task body
	for _, stmt := range task.Body {
		if err := e.executeStatement(stmt, ctx); err != nil {
			return err
		}
	}

	return nil
}

// executeStatement executes a single statement (action, parameter, conditional, etc.)
func (e *Engine) executeStatement(stmt ast.Statement, ctx *ExecutionContext) error {
	switch s := stmt.(type) {
	case *ast.ActionStatement:
		return e.executeAction(s, ctx)
	case *ast.ShellStatement:
		return e.executeShell(s, ctx)
	case *ast.FileStatement:
		return e.executeFile(s, ctx)
	case *ast.TryStatement:
		return e.executeTry(s, ctx)
	case *ast.ThrowStatement:
		return e.executeThrow(s, ctx)
	case *ast.DockerStatement:
		return e.executeDocker(s, ctx)
	case *ast.GitStatement:
		return e.executeGit(s, ctx)
	case *ast.HTTPStatement:
		return e.executeHTTP(s, ctx)
	case *ast.NetworkStatement:
		return e.executeNetwork(s, ctx)
	case *ast.DetectionStatement:
		return e.executeDetection(s, ctx)
	case *ast.BreakStatement:
		return e.executeBreak(s, ctx)
	case *ast.ContinueStatement:
		return e.executeContinue(s, ctx)
	case *ast.VariableStatement:
		return e.executeVariable(s, ctx)
	case *ast.ParameterStatement:
		// Parameters are handled during task setup, not execution
		return nil
	case *ast.ConditionalStatement:
		return e.executeConditional(s, ctx)
	case *ast.LoopStatement:
		return e.executeLoop(s, ctx)
	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// executeAction executes a single action statement
func (e *Engine) executeAction(action *ast.ActionStatement, ctx *ExecutionContext) error {
	// Interpolate variables in the message
	interpolatedMessage := e.interpolateVariables(action.Message, ctx)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] %s: %s\n", action.Action, interpolatedMessage)
		return nil
	}

	// Map actions to output with appropriate formatting and emojis
	switch action.Action {
	case "info":
		_, _ = fmt.Fprintf(e.output, "ℹ️  %s\n", interpolatedMessage)
	case "step":
		_, _ = fmt.Fprintf(e.output, "🚀 %s\n", interpolatedMessage)
	case "warn":
		_, _ = fmt.Fprintf(e.output, "⚠️  %s\n", interpolatedMessage)
	case "error":
		_, _ = fmt.Fprintf(e.output, "❌ %s\n", interpolatedMessage)
	case "success":
		_, _ = fmt.Fprintf(e.output, "✅ %s\n", interpolatedMessage)
	case "fail":
		_, _ = fmt.Fprintf(e.output, "💥 %s\n", interpolatedMessage)
		return fmt.Errorf("task failed: %s", interpolatedMessage)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}

	return nil
}

// executeShell executes a shell command statement
func (e *Engine) executeShell(shellStmt *ast.ShellStatement, ctx *ExecutionContext) error {
	if shellStmt.IsMultiline {
		return e.executeMultilineShell(shellStmt, ctx)
	}
	return e.executeSingleLineShell(shellStmt, ctx)
}

// executeSingleLineShell executes a single-line shell command
func (e *Engine) executeSingleLineShell(shellStmt *ast.ShellStatement, ctx *ExecutionContext) error {
	// Interpolate variables in the command
	interpolatedCommand := e.interpolateVariables(shellStmt.Command, ctx)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute shell command: %s\n", interpolatedCommand)
		if shellStmt.CaptureVar != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture output as: %s\n", shellStmt.CaptureVar)
		}
		return nil
	}

	// Configure shell options based on the action type and platform configuration
	opts := e.getPlatformShellConfig(ctx)
	opts.CaptureOutput = true
	opts.StreamOutput = shellStmt.StreamOutput
	opts.Output = e.output

	// Show what we're about to execute (verbose mode only)
	if e.verbose {
		switch shellStmt.Action {
		case "run":
			_, _ = fmt.Fprintf(e.output, "🏃 Running: %s\n", interpolatedCommand)
		case "exec":
			_, _ = fmt.Fprintf(e.output, "⚡ Executing: %s\n", interpolatedCommand)
		case "shell":
			_, _ = fmt.Fprintf(e.output, "🐚 Shell: %s\n", interpolatedCommand)
		case "capture":
			_, _ = fmt.Fprintf(e.output, "📥 Capturing: %s\n", interpolatedCommand)
		}
	}

	// Execute the command
	result, err := shell.Execute(interpolatedCommand, opts)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌ Command failed: %v\n", err)
		return err
	}

	// Handle capture
	if shellStmt.CaptureVar != "" && shellStmt.Action == "capture" {
		ctx.Variables[shellStmt.CaptureVar] = result.Stdout
		_, _ = fmt.Fprintf(e.output, "📦 Captured output in variable '%s'\n", shellStmt.CaptureVar)
	}

	// Show execution summary
	if result.Success {
		_, _ = fmt.Fprintf(e.output, "✅ Command completed successfully (exit code: %d, duration: %v)\n",
			result.ExitCode, result.Duration)
	} else {
		_, _ = fmt.Fprintf(e.output, "⚠️  Command completed with exit code: %d (duration: %v)\n",
			result.ExitCode, result.Duration)
	}

	return nil
}

// executeMultilineShell executes multiline shell commands as a single shell session
func (e *Engine) executeMultilineShell(shellStmt *ast.ShellStatement, ctx *ExecutionContext) error {
	// Interpolate variables in all commands
	var interpolatedCommands []string
	for _, cmd := range shellStmt.Commands {
		interpolatedCmd := e.interpolateVariables(cmd, ctx)
		interpolatedCommands = append(interpolatedCommands, interpolatedCmd)
	}

	// Join commands with newlines to create a single script
	script := strings.Join(interpolatedCommands, "\n")

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute multiline shell commands:\n")
		for i, cmd := range interpolatedCommands {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN]   %d: %s\n", i+1, cmd)
		}
		if shellStmt.CaptureVar != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture output as: %s\n", shellStmt.CaptureVar)
		}
		return nil
	}

	// Configure shell options based on the action type and platform configuration
	opts := e.getPlatformShellConfig(ctx)
	opts.CaptureOutput = true
	opts.StreamOutput = shellStmt.StreamOutput
	opts.Output = e.output

	// Show what we're about to execute (verbose mode only)
	if e.verbose {
		switch shellStmt.Action {
		case "run":
			_, _ = fmt.Fprintf(e.output, "🏃 Running multiline commands (%d lines):\n", len(interpolatedCommands))
		case "exec":
			_, _ = fmt.Fprintf(e.output, "⚡ Executing multiline commands (%d lines):\n", len(interpolatedCommands))
		case "shell":
			_, _ = fmt.Fprintf(e.output, "🐚 Shell multiline commands (%d lines):\n", len(interpolatedCommands))
		case "capture":
			_, _ = fmt.Fprintf(e.output, "📥 Capturing multiline commands (%d lines):\n", len(interpolatedCommands))
		}

		// Show each command with line numbers
		for i, cmd := range interpolatedCommands {
			_, _ = fmt.Fprintf(e.output, "  %d: %s\n", i+1, cmd)
		}
	}

	// Execute the script as a single shell session
	result, err := shell.Execute(script, opts)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌ Multiline command failed: %v\n", err)
		return err
	}

	// Handle capture
	if shellStmt.CaptureVar != "" && shellStmt.Action == "capture" {
		ctx.Variables[shellStmt.CaptureVar] = result.Stdout
		_, _ = fmt.Fprintf(e.output, "📦 Captured output in variable '%s'\n", shellStmt.CaptureVar)
	}

	// Show execution summary
	if result.Success {
		_, _ = fmt.Fprintf(e.output, "✅ Multiline commands completed successfully (exit code: %d, duration: %v)\n",
			result.ExitCode, result.Duration)
	} else {
		_, _ = fmt.Fprintf(e.output, "⚠️  Multiline commands completed with exit code: %d (duration: %v)\n",
			result.ExitCode, result.Duration)
	}

	return nil
}

// executeFile executes a file operation statement
func (e *Engine) executeFile(fileStmt *ast.FileStatement, ctx *ExecutionContext) error {
	// Interpolate variables in paths and content
	target := e.interpolateVariables(fileStmt.Target, ctx)
	source := e.interpolateVariables(fileStmt.Source, ctx)
	content := e.interpolateVariables(fileStmt.Content, ctx)

	// Create file operation
	op := &fileops.FileOperation{
		Type:    fileStmt.Action,
		Target:  target,
		Source:  source,
		Content: content,
		IsDir:   fileStmt.IsDir,
	}

	if e.dryRun {
		result, err := op.Execute(true) // dry run
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "❌ File operation failed: %v\n", err)
			return err
		}
		_, _ = fmt.Fprintf(e.output, "📁 %s\n", result.Message)
		if fileStmt.CaptureVar != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture file content in variable '%s'\n", fileStmt.CaptureVar)
		}
		return nil
	}

	// Handle special actions that need preprocessing
	switch fileStmt.Action {
	case "backup":
		if target == "" {
			// Generate default backup name with timestamp
			timestamp := time.Now().Format("2006-01-02-15-04-05")
			target = source + ".backup-" + timestamp
		}
		op.Target = target
		op.Type = "copy" // Backup is essentially a copy operation
	case "check_exists":
		// Check if file exists
		if e.fileExists(target) {
			_, _ = fmt.Fprintf(e.output, "✅ File exists: %s\n", target)
		} else {
			_, _ = fmt.Fprintf(e.output, "❌ File does not exist: %s\n", target)
		}
		return nil
	case "get_size":
		// Get file size
		size, err := e.getFileSize(target)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "❌ Failed to get file size: %v\n", err)
			return err
		}
		_, _ = fmt.Fprintf(e.output, "📏 File size: %s (%d bytes)\n", target, size)
		return nil
	}

	// Show what we're about to do
	switch fileStmt.Action {
	case "create":
		if fileStmt.IsDir {
			_, _ = fmt.Fprintf(e.output, "📁 Creating directory: %s\n", target)
		} else {
			_, _ = fmt.Fprintf(e.output, "📄 Creating file: %s\n", target)
		}
	case "copy":
		_, _ = fmt.Fprintf(e.output, "📋 Copying: %s → %s\n", source, target)
	case "move":
		_, _ = fmt.Fprintf(e.output, "🚚 Moving: %s → %s\n", source, target)
	case "delete":
		if fileStmt.IsDir {
			_, _ = fmt.Fprintf(e.output, "🗑️  Deleting directory: %s\n", target)
		} else {
			_, _ = fmt.Fprintf(e.output, "🗑️  Deleting file: %s\n", target)
		}
	case "read":
		_, _ = fmt.Fprintf(e.output, "📖 Reading file: %s\n", target)
	case "write":
		_, _ = fmt.Fprintf(e.output, "✏️  Writing to file: %s\n", target)
	case "append":
		_, _ = fmt.Fprintf(e.output, "➕ Appending to file: %s\n", target)
	case "backup":
		_, _ = fmt.Fprintf(e.output, "💾 Backing up: %s → %s\n", source, target)
	}

	// Execute the file operation
	result, err := op.Execute(false)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌ File operation failed: %v\n", err)
		return err
	}

	// Handle capture for read operations
	if fileStmt.CaptureVar != "" && fileStmt.Action == "read" {
		ctx.Variables[fileStmt.CaptureVar] = result.Content
		_, _ = fmt.Fprintf(e.output, "📦 Captured file content in variable '%s' (%d bytes)\n",
			fileStmt.CaptureVar, len(result.Content))
	}

	// Show success message
	if result.Success {
		_, _ = fmt.Fprintf(e.output, "✅ %s\n", result.Message)
	} else {
		_, _ = fmt.Fprintf(e.output, "⚠️  %s\n", result.Message)
	}

	return nil
}

// executeTry executes a try/catch/finally statement
func (e *Engine) executeTry(tryStmt *ast.TryStatement, ctx *ExecutionContext) error {
	var tryError error
	var finallyError error

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute try block\n")

		// Execute try body in dry run
		for _, stmt := range tryStmt.TryBody {
			if err := e.executeStatement(stmt, ctx); err != nil {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would catch error: %v\n", err)
				break
			}
		}

		if len(tryStmt.CatchClauses) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute catch blocks if needed\n")
		}

		if len(tryStmt.FinallyBody) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute finally block\n")
		}

		return nil
	}

	// Execute try block
	_, _ = fmt.Fprintf(e.output, "🔄 Executing try block\n")
	for _, stmt := range tryStmt.TryBody {
		if err := e.executeStatement(stmt, ctx); err != nil {
			tryError = err
			_, _ = fmt.Fprintf(e.output, "⚠️  Error in try block: %v\n", err)
			break
		}
	}

	// Execute catch blocks if there was an error
	if tryError != nil {
		handled := false
		for _, catchClause := range tryStmt.CatchClauses {
			if e.shouldHandleError(tryError, catchClause) {
				_, _ = fmt.Fprintf(e.output, "🔧 Handling error with catch block\n")

				// Set error variable if specified
				if catchClause.ErrorVar != "" {
					ctx.Variables[catchClause.ErrorVar] = tryError.Error()
					_, _ = fmt.Fprintf(e.output, "📦 Captured error in variable '%s'\n", catchClause.ErrorVar)
				}

				// Execute catch body
				for _, stmt := range catchClause.Body {
					if err := e.executeStatement(stmt, ctx); err != nil {
						// Error in catch block - this becomes the new error
						tryError = err
						break
					}
				}

				handled = true
				break
			}
		}

		if !handled {
			_, _ = fmt.Fprintf(e.output, "❌ Unhandled error: %v\n", tryError)
		} else {
			_, _ = fmt.Fprintf(e.output, "✅ Error handled successfully\n")
			tryError = nil // Error was handled
		}
	} else {
		_, _ = fmt.Fprintf(e.output, "✅ Try block completed successfully\n")
	}

	// Always execute finally block
	if len(tryStmt.FinallyBody) > 0 {
		_, _ = fmt.Fprintf(e.output, "🔄 Executing finally block\n")
		for _, stmt := range tryStmt.FinallyBody {
			if err := e.executeStatement(stmt, ctx); err != nil {
				finallyError = err
				_, _ = fmt.Fprintf(e.output, "⚠️  Error in finally block: %v\n", err)
				break
			}
		}

		if finallyError == nil {
			_, _ = fmt.Fprintf(e.output, "✅ Finally block completed successfully\n")
		}
	}

	// Return the most relevant error
	if finallyError != nil {
		return finallyError // Finally errors take precedence
	}
	return tryError // Original error (if not handled)
}

// executeThrow executes throw, rethrow, and ignore statements
func (e *Engine) executeThrow(throwStmt *ast.ThrowStatement, ctx *ExecutionContext) error {
	if e.dryRun {
		switch throwStmt.Action {
		case "throw":
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would throw error: %s\n", throwStmt.Message)
		case "rethrow":
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would rethrow current error\n")
		case "ignore":
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would ignore current error\n")
		}
		return nil
	}

	switch throwStmt.Action {
	case "throw":
		message := e.interpolateVariables(throwStmt.Message, ctx)
		_, _ = fmt.Fprintf(e.output, "💥 Throwing error: %s\n", message)
		return fmt.Errorf("thrown error: %s", message)
	case "rethrow":
		_, _ = fmt.Fprintf(e.output, "🔄 Rethrowing current error\n")
		// In a real implementation, we'd need to track the current error context
		return fmt.Errorf("rethrown error")
	case "ignore":
		_, _ = fmt.Fprintf(e.output, "🤐 Ignoring current error\n")
		return nil // Ignore effectively suppresses the error
	default:
		return fmt.Errorf("unknown throw action: %s", throwStmt.Action)
	}
}

// executeDocker executes Docker operations
func (e *Engine) executeDocker(dockerStmt *ast.DockerStatement, ctx *ExecutionContext) error {
	// Interpolate variables in Docker statement
	operation := dockerStmt.Operation
	resource := dockerStmt.Resource
	name := e.interpolateVariables(dockerStmt.Name, ctx)

	// Interpolate options
	options := make(map[string]string)
	for key, value := range dockerStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun {
		return e.buildDockerCommand(operation, resource, name, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch operation {
	case "build":
		_, _ = fmt.Fprintf(e.output, "🔨 Building Docker image")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "push":
		_, _ = fmt.Fprintf(e.output, "📤 Pushing Docker image")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		if registry, exists := options["to"]; exists {
			_, _ = fmt.Fprintf(e.output, " to %s", registry)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "pull":
		_, _ = fmt.Fprintf(e.output, "📥 Pulling Docker image")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "run":
		_, _ = fmt.Fprintf(e.output, "🚀 Running Docker container")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		if port, exists := options["port"]; exists {
			_, _ = fmt.Fprintf(e.output, " on port %s", port)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "stop":
		_, _ = fmt.Fprintf(e.output, "🛑 Stopping Docker container")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "remove":
		_, _ = fmt.Fprintf(e.output, "🗑️  Removing Docker %s", resource)
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "compose":
		command := options["command"]
		switch command {
		case "up":
			_, _ = fmt.Fprintf(e.output, "🚀 Starting Docker Compose services\n")
		case "down":
			_, _ = fmt.Fprintf(e.output, "🛑 Stopping Docker Compose services\n")
		case "build":
			_, _ = fmt.Fprintf(e.output, "🔨 Building Docker Compose services\n")
		default:
			_, _ = fmt.Fprintf(e.output, "🐳 Running Docker Compose: %s\n", command)
		}
	case "scale":
		if resource == "compose" {
			replicas := options["replicas"]
			_, _ = fmt.Fprintf(e.output, "📊 Scaling Docker Compose service")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, " %s", name)
			}
			if replicas != "" {
				_, _ = fmt.Fprintf(e.output, " to %s replicas", replicas)
			}
			_, _ = fmt.Fprintf(e.output, "\n")
		}
	default:
		_, _ = fmt.Fprintf(e.output, "🐳 Running Docker %s", operation)
		if resource != "" {
			_, _ = fmt.Fprintf(e.output, " %s", resource)
		}
		if name != "" {
			_, _ = fmt.Fprintf(e.output, " %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	// Build and execute the actual command
	return e.buildDockerCommand(operation, resource, name, options, false)
}

// buildDockerCommand builds and optionally executes the Docker command
func (e *Engine) buildDockerCommand(operation, resource, name string, options map[string]string, dryRun bool) error {
	var dockerCmd []string
	dockerCmd = append(dockerCmd, "docker")

	// Handle Docker Compose separately
	if operation == "compose" {
		dockerCmd = append(dockerCmd, "compose")
		if command, exists := options["command"]; exists {
			dockerCmd = append(dockerCmd, command)
		}
	} else if operation == "scale" && resource == "compose" {
		// Handle "docker compose scale service_name replicas"
		dockerCmd = append(dockerCmd, "compose", "scale")
		if name != "" {
			if replicas, exists := options["replicas"]; exists {
				dockerCmd = append(dockerCmd, fmt.Sprintf("%s=%s", name, replicas))
			}
		}
	} else {
		// Regular Docker commands
		dockerCmd = append(dockerCmd, operation)
		if resource != "" {
			dockerCmd = append(dockerCmd, resource)
		}
		if name != "" {
			dockerCmd = append(dockerCmd, name)
		}

		// Add options in a logical order
		if from, exists := options["from"]; exists {
			if operation == "build" {
				dockerCmd = append(dockerCmd, "--file", from)
			} else {
				dockerCmd = append(dockerCmd, from)
			}
		}
		if to, exists := options["to"]; exists {
			dockerCmd = append(dockerCmd, to)
		}
		if as, exists := options["as"]; exists {
			dockerCmd = append(dockerCmd, as)
		}
		if port, exists := options["port"]; exists {
			if operation == "run" {
				dockerCmd = append(dockerCmd, "-p", fmt.Sprintf("%s:%s", port, port))
			}
		}
	}

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute Docker command: %s\n", strings.Join(dockerCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(dockerCmd, " "))
	}

	// For now, we'll simulate the command execution
	// In a real implementation, you would use exec.Command to run the Docker command
	// cmd := exec.Command(dockerCmd[0], dockerCmd[1:]...)
	// return cmd.Run()

	return nil
}

// executeGit executes Git operations
func (e *Engine) executeGit(gitStmt *ast.GitStatement, ctx *ExecutionContext) error {
	// Interpolate variables in Git statement
	operation := gitStmt.Operation
	resource := gitStmt.Resource
	name := e.interpolateVariables(gitStmt.Name, ctx)

	// Interpolate options
	options := make(map[string]string)
	for key, value := range gitStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun {
		return e.buildGitCommand(operation, resource, name, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch operation {
	case "create":
		switch resource {
		case "branch":
			_, _ = fmt.Fprintf(e.output, "🌿 Creating Git branch")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, ": %s", name)
			}
		case "tag":
			_, _ = fmt.Fprintf(e.output, "🏷️  Creating Git tag")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, ": %s", name)
			}
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "checkout":
		_, _ = fmt.Fprintf(e.output, "🔀 Checking out Git branch")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "merge":
		_, _ = fmt.Fprintf(e.output, "🔀 Merging Git branch")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "commit":
		_, _ = fmt.Fprintf(e.output, "💾 Committing Git changes")
		if message, exists := options["message"]; exists {
			_, _ = fmt.Fprintf(e.output, ": %s", message)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "push":
		if resource == "tag" {
			_, _ = fmt.Fprintf(e.output, "📤 Pushing Git tag")
			if name != "" {
				_, _ = fmt.Fprintf(e.output, ": %s", name)
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "📤 Pushing Git changes")
			if remote, exists := options["remote"]; exists {
				_, _ = fmt.Fprintf(e.output, " to %s", remote)
			}
			if branch, exists := options["branch"]; exists {
				_, _ = fmt.Fprintf(e.output, "/%s", branch)
			}
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "clone":
		_, _ = fmt.Fprintf(e.output, "📥 Cloning Git repository")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "init":
		_, _ = fmt.Fprintf(e.output, "🆕 Initializing Git repository\n")
	case "add":
		_, _ = fmt.Fprintf(e.output, "➕ Adding files to Git")
		if name != "" {
			_, _ = fmt.Fprintf(e.output, ": %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	case "status":
		_, _ = fmt.Fprintf(e.output, "📊 Checking Git status\n")
	case "show":
		if resource == "branch" {
			_, _ = fmt.Fprintf(e.output, "🌿 Showing current Git branch\n")
		} else {
			_, _ = fmt.Fprintf(e.output, "📖 Showing Git information\n")
		}
	default:
		_, _ = fmt.Fprintf(e.output, "🔗 Running Git %s", operation)
		if resource != "" {
			_, _ = fmt.Fprintf(e.output, " %s", resource)
		}
		if name != "" {
			_, _ = fmt.Fprintf(e.output, " %s", name)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
	}

	// Build and execute the actual command
	return e.buildGitCommand(operation, resource, name, options, false)
}

// buildGitCommand builds and displays the git command
func (e *Engine) buildGitCommand(operation, resource, name string, options map[string]string, dryRun bool) error {
	var gitCmd []string
	gitCmd = append(gitCmd, "git")

	switch operation {
	case "create":
		switch resource {
		case "branch":
			// git checkout -b branch_name
			gitCmd = append(gitCmd, "checkout", "-b")
			if name != "" {
				gitCmd = append(gitCmd, name)
			}
		case "tag":
			// git tag tag_name
			gitCmd = append(gitCmd, "tag")
			if name != "" {
				gitCmd = append(gitCmd, name)
			}
		}

	case "checkout":
		// git checkout branch_name
		gitCmd = append(gitCmd, "checkout")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}

	case "merge":
		// git merge branch_name
		gitCmd = append(gitCmd, "merge")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}

	case "clone":
		// git clone repository "url" to "dir"
		gitCmd = append(gitCmd, "clone")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}
		if to, exists := options["to"]; exists {
			gitCmd = append(gitCmd, to)
		}

	case "init":
		// git init repository in "dir"
		gitCmd = append(gitCmd, "init")
		if in, exists := options["in"]; exists {
			gitCmd = append(gitCmd, in)
		}

	case "add":
		// git add files "pattern"
		gitCmd = append(gitCmd, "add")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}

	case "commit":
		// git commit changes with message "msg"
		// git commit all changes with message "msg"
		gitCmd = append(gitCmd, "commit")
		if all, exists := options["all"]; exists && all == "true" {
			gitCmd = append(gitCmd, "-a")
		}
		if message, exists := options["message"]; exists {
			gitCmd = append(gitCmd, "-m", fmt.Sprintf("\"%s\"", message))
		}

	case "push":
		// git push to remote "origin" branch "main"
		// git push tag "v1.0.0" to remote "origin"
		gitCmd = append(gitCmd, "push")
		if resource == "tag" && name != "" {
			gitCmd = append(gitCmd, "origin", name)
		} else {
			if remote, exists := options["remote"]; exists {
				gitCmd = append(gitCmd, remote)
			}
			if branch, exists := options["branch"]; exists {
				gitCmd = append(gitCmd, branch)
			}
		}

	case "pull":
		// git pull from remote "origin" branch "main"
		gitCmd = append(gitCmd, "pull")
		if from, exists := options["from"]; exists {
			gitCmd = append(gitCmd, from)
		}
		if remote, exists := options["remote"]; exists {
			gitCmd = append(gitCmd, remote)
		}
		if branch, exists := options["branch"]; exists {
			gitCmd = append(gitCmd, branch)
		}

	case "fetch":
		// git fetch from remote "origin"
		gitCmd = append(gitCmd, "fetch")
		if from, exists := options["from"]; exists {
			gitCmd = append(gitCmd, from)
		}
		if remote, exists := options["remote"]; exists {
			gitCmd = append(gitCmd, remote)
		}

	case "status":
		// git status
		gitCmd = append(gitCmd, "status")

	case "log":
		// git log --oneline
		gitCmd = append(gitCmd, "log", "--oneline")

	case "show":
		// git show current branch
		// git show current commit
		if current, exists := options["current"]; exists && current == "true" {
			switch resource {
			case "branch":
				gitCmd = append(gitCmd, "branch", "--show-current")
			case "commit":
				gitCmd = append(gitCmd, "rev-parse", "HEAD")
			}
		} else {
			gitCmd = append(gitCmd, "show")
		}

	default:
		gitCmd = append(gitCmd, operation)
		if resource != "" {
			gitCmd = append(gitCmd, resource)
		}
		if name != "" {
			gitCmd = append(gitCmd, name)
		}
	}

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute Git command: %s\n", strings.Join(gitCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(gitCmd, " "))
	}

	// For now, we'll simulate the command execution
	// In a real implementation, you would use exec.Command to run the git command
	// cmd := exec.Command(gitCmd[0], gitCmd[1:]...)
	// return cmd.Run()

	return nil
}

// executeHTTP executes HTTP operations
func (e *Engine) executeHTTP(httpStmt *ast.HTTPStatement, ctx *ExecutionContext) error {
	// Interpolate variables in HTTP statement
	method := httpStmt.Method
	url := e.interpolateVariables(httpStmt.URL, ctx)
	body := e.interpolateVariables(httpStmt.Body, ctx)

	// Interpolate headers
	headers := make(map[string]string)
	for key, value := range httpStmt.Headers {
		headers[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate auth
	auth := make(map[string]string)
	for key, value := range httpStmt.Auth {
		auth[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate options
	options := make(map[string]string)
	for key, value := range httpStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun {
		return e.buildHTTPCommand(method, url, body, headers, auth, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch method {
	case "GET":
		_, _ = fmt.Fprintf(e.output, "📥 GET request to: %s\n", url)
	case "POST":
		_, _ = fmt.Fprintf(e.output, "📤 POST request to: %s\n", url)
	case "PUT":
		_, _ = fmt.Fprintf(e.output, "🔄 PUT request to: %s\n", url)
	case "PATCH":
		_, _ = fmt.Fprintf(e.output, "🔧 PATCH request to: %s\n", url)
	case "DELETE":
		_, _ = fmt.Fprintf(e.output, "🗑️  DELETE request to: %s\n", url)
	case "HEAD":
		_, _ = fmt.Fprintf(e.output, "🔍 HEAD request to: %s\n", url)
	default:
		_, _ = fmt.Fprintf(e.output, "🌐 %s request to: %s\n", method, url)
	}

	// Handle special HTTP operations
	if downloadPath, exists := options["download"]; exists {
		_, _ = fmt.Fprintf(e.output, "💾 Downloading to: %s\n", downloadPath)
	}

	if uploadPath, exists := options["upload"]; exists {
		_, _ = fmt.Fprintf(e.output, "📤 Uploading from: %s\n", uploadPath)
	}

	// Build and execute the actual HTTP request
	return e.buildHTTPCommand(method, url, body, headers, auth, options, false)
}

// buildHTTPCommand builds and displays the HTTP request details
func (e *Engine) buildHTTPCommand(method, url, body string, headers, auth, options map[string]string, dryRun bool) error {
	var httpCmd []string
	httpCmd = append(httpCmd, "curl", "-X", method)

	// Add headers
	for key, value := range headers {
		httpCmd = append(httpCmd, "-H", fmt.Sprintf("\"%s: %s\"", key, value))
	}

	// Add authentication
	for authType, value := range auth {
		switch authType {
		case "bearer":
			httpCmd = append(httpCmd, "-H", fmt.Sprintf("\"Authorization: Bearer %s\"", value))
		case "basic":
			httpCmd = append(httpCmd, "--user", value)
		case "token":
			httpCmd = append(httpCmd, "-H", fmt.Sprintf("\"Authorization: Token %s\"", value))
		}
	}

	// Handle special operations
	if downloadPath, exists := options["download"]; exists {
		httpCmd = append(httpCmd, "-o", downloadPath)
	}

	if uploadPath, exists := options["upload"]; exists {
		httpCmd = append(httpCmd, "-T", uploadPath)
	}

	// Add body
	if body != "" {
		httpCmd = append(httpCmd, "-d", body)
	}

	// Add advanced options
	if timeout, exists := options["timeout"]; exists {
		httpCmd = append(httpCmd, "--max-time", timeout)
	}
	if retry, exists := options["retry"]; exists {
		httpCmd = append(httpCmd, "--retry", retry)
	}
	if followRedirects, exists := options["follow_redirects"]; exists && followRedirects == "true" {
		httpCmd = append(httpCmd, "-L")
	}
	if insecure, exists := options["insecure"]; exists && insecure == "true" {
		httpCmd = append(httpCmd, "-k")
	}
	if verbose, exists := options["verbose"]; exists && verbose == "true" {
		httpCmd = append(httpCmd, "-v")
	}
	if silent, exists := options["silent"]; exists && silent == "true" {
		httpCmd = append(httpCmd, "-s")
	}

	// Add URL last
	httpCmd = append(httpCmd, url)

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute HTTP command: %s\n", strings.Join(httpCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(httpCmd, " "))
	}

	// For now, we'll simulate the HTTP request execution
	// In a real implementation, you would use exec.Command to run the curl command
	// or use Go's http.Client for more advanced features
	// cmd := exec.Command(httpCmd[0], httpCmd[1:]...)
	// return cmd.Run()

	return nil
}

// executeNetwork executes network operations (health checks, port testing, ping)
func (e *Engine) executeNetwork(networkStmt *ast.NetworkStatement, ctx *ExecutionContext) error {
	// Interpolate variables in network statement
	target := e.interpolateVariables(networkStmt.Target, ctx)
	port := e.interpolateVariables(networkStmt.Port, ctx)
	condition := e.interpolateVariables(networkStmt.Condition, ctx)

	// Interpolate options
	options := make(map[string]string)
	for key, value := range networkStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun {
		return e.buildNetworkCommand(networkStmt.Action, target, port, condition, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch networkStmt.Action {
	case "health_check":
		_, _ = fmt.Fprintf(e.output, "🏥 Health check: %s\n", target)
	case "wait_for_service":
		_, _ = fmt.Fprintf(e.output, "⏳ Waiting for service: %s\n", target)
	case "port_check":
		if port != "" {
			_, _ = fmt.Fprintf(e.output, "🔌 Port check: %s:%s\n", target, port)
		} else {
			_, _ = fmt.Fprintf(e.output, "🔌 Connection test: %s\n", target)
		}
	case "ping":
		_, _ = fmt.Fprintf(e.output, "🏓 Ping: %s\n", target)
	default:
		_, _ = fmt.Fprintf(e.output, "🌐 Network operation: %s on %s\n", networkStmt.Action, target)
	}

	// Build and execute the actual network command
	return e.buildNetworkCommand(networkStmt.Action, target, port, condition, options, false)
}

// buildNetworkCommand builds and executes network commands
func (e *Engine) buildNetworkCommand(action, target, port, condition string, options map[string]string, dryRun bool) error {
	var networkCmd []string

	switch action {
	case "health_check":
		// Use curl for health checks with status code validation
		networkCmd = append(networkCmd, "curl", "-f", "-s", "-S")

		// Add timeout if specified
		if timeout, exists := options["timeout"]; exists {
			networkCmd = append(networkCmd, "--max-time", timeout)
		} else {
			networkCmd = append(networkCmd, "--max-time", "10") // Default 10s timeout
		}

		// Add retry if specified
		if retry, exists := options["retry"]; exists {
			networkCmd = append(networkCmd, "--retry", retry)
		}

		// Add condition checking
		if condition != "" {
			if condition == "200" || strings.HasPrefix(condition, "20") {
				networkCmd = append(networkCmd, "-w", "%{http_code}")
			}
		}

		networkCmd = append(networkCmd, target)

	case "wait_for_service":
		// Create a retry loop for service waiting
		timeout := "60" // Default 60s timeout
		if t, exists := options["timeout"]; exists {
			timeout = t
		}

		retryInterval := "2" // Default 2s retry interval
		if r, exists := options["retry"]; exists {
			retryInterval = r
		}

		// Build a shell script for waiting
		script := fmt.Sprintf(`
timeout=%s
interval=%s
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if curl -f -s -S --max-time 5 "%s" > /dev/null 2>&1; then
    echo "Service is ready"
    exit 0
  fi
  sleep $interval
  elapsed=$((elapsed + interval))
  echo "Waiting for service... ($elapsed/${timeout}s)"
done
echo "Timeout waiting for service"
exit 1`, timeout, retryInterval, target)

		networkCmd = []string{"sh", "-c", script}

	case "port_check":
		// Use netcat for port checking
		networkCmd = append(networkCmd, "nc", "-z")

		// Add timeout if specified
		if timeout, exists := options["timeout"]; exists {
			networkCmd = append(networkCmd, "-w", timeout)
		} else {
			networkCmd = append(networkCmd, "-w", "5") // Default 5s timeout
		}

		networkCmd = append(networkCmd, target, port)

	case "ping":
		// Use ping command
		networkCmd = append(networkCmd, "ping", "-c", "1")

		// Add timeout if specified (ping uses different timeout format)
		if timeout, exists := options["timeout"]; exists {
			networkCmd = append(networkCmd, "-W", timeout)
		}

		networkCmd = append(networkCmd, target)

	default:
		return fmt.Errorf("unknown network action: %s", action)
	}

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute network command: %s\n", strings.Join(networkCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(networkCmd, " "))
	}

	// For now, we'll simulate the network command execution
	// In a real implementation, you would use exec.Command to run the network command
	// cmd := exec.Command(networkCmd[0], networkCmd[1:]...)
	// return cmd.Run()

	return nil
}

// executeDetection executes smart detection operations
func (e *Engine) executeDetection(detectionStmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	detector := detection.NewDetector()

	switch detectionStmt.Type {
	case "detect":
		return e.executeDetectOperation(detector, detectionStmt, ctx)
	case "detect_available":
		return e.executeDetectAvailable(detector, detectionStmt, ctx)
	case "if_available":
		return e.executeIfAvailable(detector, detectionStmt, ctx)
	case "if_version":
		return e.executeIfVersion(detector, detectionStmt, ctx)
	case "when_environment":
		return e.executeWhenEnvironment(detector, detectionStmt, ctx)
	default:
		return fmt.Errorf("unknown detection type: %s", detectionStmt.Type)
	}
}

// executeDetectOperation executes detect operations
func (e *Engine) executeDetectOperation(detector *detection.Detector, stmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	switch stmt.Target {
	case "project":
		if stmt.Condition == "type" {
			types := detector.DetectProjectType()
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would detect project types: %v\n", types)
			} else {
				_, _ = fmt.Fprintf(e.output, "🔍 Detected project types: %v\n", types)
			}
		}
	default:
		// Detect tool
		if stmt.Condition == "version" {
			version := detector.GetToolVersion(stmt.Target)
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would detect %s version: %s\n", stmt.Target, version)
			} else {
				_, _ = fmt.Fprintf(e.output, "🔍 Detected %s version: %s\n", stmt.Target, version)
			}
		} else {
			available := detector.IsToolAvailable(stmt.Target)
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s is available: %t\n", stmt.Target, available)
			} else {
				_, _ = fmt.Fprintf(e.output, "🔍 %s available: %t\n", stmt.Target, available)
			}
		}
	}

	return nil
}

// executeIfAvailable executes "if tool is available" conditions
func (e *Engine) executeIfAvailable(detector *detection.Detector, stmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	available := detector.IsToolAvailable(stmt.Target)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s is available: %t\n", stmt.Target, available)
		if available {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute if-available body for %s\n", stmt.Target)
			for _, bodyStmt := range stmt.Body {
				if err := e.executeStatement(bodyStmt, ctx); err != nil {
					return err
				}
			}
		} else if len(stmt.ElseBody) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute else body for %s\n", stmt.Target)
			for _, elseStmt := range stmt.ElseBody {
				if err := e.executeStatement(elseStmt, ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "🔍 Checking if %s is available: %t\n", stmt.Target, available)

	if available {
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	} else if len(stmt.ElseBody) > 0 {
		for _, elseStmt := range stmt.ElseBody {
			if err := e.executeStatement(elseStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeIfVersion executes "if tool version comparison" conditions
func (e *Engine) executeIfVersion(detector *detection.Detector, stmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	version := detector.GetToolVersion(stmt.Target)
	targetVersion := e.interpolateVariables(stmt.Value, ctx)

	matches := detector.CompareVersion(version, stmt.Condition, targetVersion)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s version %s %s %s: %t (current: %s)\n",
			stmt.Target, version, stmt.Condition, targetVersion, matches, version)
		if matches {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute if-version body for %s\n", stmt.Target)
			for _, bodyStmt := range stmt.Body {
				if err := e.executeStatement(bodyStmt, ctx); err != nil {
					return err
				}
			}
		} else if len(stmt.ElseBody) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute else body for %s\n", stmt.Target)
			for _, elseStmt := range stmt.ElseBody {
				if err := e.executeStatement(elseStmt, ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "🔍 Checking %s version %s %s %s: %t (current: %s)\n",
		stmt.Target, version, stmt.Condition, targetVersion, matches, version)

	if matches {
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	} else if len(stmt.ElseBody) > 0 {
		for _, elseStmt := range stmt.ElseBody {
			if err := e.executeStatement(elseStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeWhenEnvironment executes "when in environment" conditions
func (e *Engine) executeWhenEnvironment(detector *detection.Detector, stmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	currentEnv := detector.DetectEnvironment()
	matches := currentEnv == stmt.Target

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if in %s environment: %t (current: %s)\n",
			stmt.Target, matches, currentEnv)
		if matches {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute when-environment body\n")
			for _, bodyStmt := range stmt.Body {
				if err := e.executeStatement(bodyStmt, ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "🔍 Checking if in %s environment: %t (current: %s)\n",
		stmt.Target, matches, currentEnv)

	if matches {
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeDetectAvailable executes "detect available" operations with tool alternatives
func (e *Engine) executeDetectAvailable(detector *detection.Detector, stmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	// Build list of tools to try (primary + alternatives)
	toolsToTry := []string{stmt.Target}
	toolsToTry = append(toolsToTry, stmt.Alternatives...)

	var workingTool string
	var found bool

	// Try each tool variant until we find one that works
	for _, tool := range toolsToTry {
		if detector.IsToolAvailable(tool) {
			workingTool = tool
			found = true
			break
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would detect available tool from: %v\n", toolsToTry)
		if found {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would find: %s\n", workingTool)
			if stmt.CaptureVar != "" {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture as %s: %s\n", stmt.CaptureVar, workingTool)
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would find: none available\n")
		}
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Detecting available tool from: %v\n", toolsToTry)
	}

	if found {
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅ Found: %s\n", workingTool)
		}

		// Capture the working tool variant in a variable if specified
		if stmt.CaptureVar != "" {
			ctx.Variables[stmt.CaptureVar] = workingTool
			if e.verbose {
				_, _ = fmt.Fprintf(e.output, "📝 Captured as %s: %s\n", stmt.CaptureVar, workingTool)
			}
		}
	} else {
		_, _ = fmt.Fprintf(e.output, "❌ None of the tools are available: %v\n", toolsToTry)
	}

	return nil
}

// BreakError represents a break statement execution
type BreakError struct {
	Condition string
}

func (e BreakError) Error() string {
	if e.Condition != "" {
		return "break when " + e.Condition
	}
	return "break"
}

// ContinueError represents a continue statement execution
type ContinueError struct {
	Condition string
}

func (e ContinueError) Error() string {
	if e.Condition != "" {
		return "continue if " + e.Condition
	}
	return "continue"
}

// executeBreak executes break statements
func (e *Engine) executeBreak(breakStmt *ast.BreakStatement, ctx *ExecutionContext) error {
	condition := e.interpolateVariables(breakStmt.Condition, ctx)

	if e.dryRun {
		if condition != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would break when: %s\n", condition)
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would break\n")
		}
		return BreakError{Condition: condition}
	}

	if condition != "" {
		// Evaluate the condition
		if e.evaluateSimpleCondition(condition, ctx) {
			_, _ = fmt.Fprintf(e.output, "🔄 Breaking loop (condition: %s)\n", condition)
			return BreakError{Condition: condition}
		}
		// Condition not met, don't break
		return nil
	} else {
		_, _ = fmt.Fprintf(e.output, "🔄 Breaking loop\n")
		return BreakError{Condition: condition}
	}
}

// executeContinue executes continue statements
func (e *Engine) executeContinue(continueStmt *ast.ContinueStatement, ctx *ExecutionContext) error {
	condition := e.interpolateVariables(continueStmt.Condition, ctx)

	if e.dryRun {
		if condition != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would continue if: %s\n", condition)
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would continue\n")
		}
		return ContinueError{Condition: condition}
	}

	if condition != "" {
		// Evaluate the condition
		if e.evaluateSimpleCondition(condition, ctx) {
			_, _ = fmt.Fprintf(e.output, "🔄 Continuing loop (condition: %s)\n", condition)
			return ContinueError{Condition: condition}
		}
		// Condition not met, don't continue
		return nil
	} else {
		_, _ = fmt.Fprintf(e.output, "🔄 Continuing loop\n")
		return ContinueError{Condition: condition}
	}
}

// evaluateSimpleCondition evaluates simple conditions like "item == 'test'"
func (e *Engine) evaluateSimpleCondition(condition string, ctx *ExecutionContext) bool {
	// This is a simplified implementation
	// In a real implementation, you would parse and evaluate the condition properly
	// For now, we'll just return true to demonstrate the flow
	return true
}

// executeVariable executes variable operation statements
func (e *Engine) executeVariable(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	switch varStmt.Operation {
	case "let":
		return e.executeLetStatement(varStmt, ctx)
	case "set":
		return e.executeSetStatement(varStmt, ctx)
	case "transform":
		return e.executeTransformStatement(varStmt, ctx)
	default:
		return fmt.Errorf("unknown variable operation: %s", varStmt.Operation)
	}
}

// executeLetStatement executes "let variable = value" statements
func (e *Engine) executeLetStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	value := e.interpolateVariables(varStmt.Value, ctx)

	// Store the variable in the context even in dry run for interpolation
	ctx.Variables[varStmt.Variable] = value

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set variable %s = %s\n", varStmt.Variable, value)
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "📝 Set variable %s = %s\n", varStmt.Variable, value)

	return nil
}

// executeSetStatement executes "set variable to value" statements
func (e *Engine) executeSetStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	value := e.interpolateVariables(varStmt.Value, ctx)

	// Store the variable in the context even in dry run for interpolation
	ctx.Variables[varStmt.Variable] = value

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set variable %s to %s\n", varStmt.Variable, value)
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "📝 Set variable %s to %s\n", varStmt.Variable, value)

	return nil
}

// executeTransformStatement executes "transform variable with function args" statements
func (e *Engine) executeTransformStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	// Get the current value of the variable
	currentValue, exists := ctx.Variables[varStmt.Variable]
	if !exists {
		return fmt.Errorf("variable '%s' not found", varStmt.Variable)
	}

	// Apply the transformation function
	newValue, err := e.applyTransformation(currentValue, varStmt.Function, varStmt.Arguments, ctx)
	if err != nil {
		return fmt.Errorf("transformation failed: %v", err)
	}

	// Update the variable with the transformed value even in dry run for interpolation
	ctx.Variables[varStmt.Variable] = newValue

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would transform variable %s with %s: %s -> %s\n",
			varStmt.Variable, varStmt.Function, currentValue, newValue)
		return nil
	}
	_, _ = fmt.Fprintf(e.output, "🔄 Transformed variable %s with %s: %s -> %s\n",
		varStmt.Variable, varStmt.Function, currentValue, newValue)

	return nil
}

// applyTransformation applies a transformation function to a value
func (e *Engine) applyTransformation(value, function string, args []string, ctx *ExecutionContext) (string, error) {
	// Interpolate arguments
	interpolatedArgs := make([]string, len(args))
	for i, arg := range args {
		interpolatedArgs[i] = e.interpolateVariables(arg, ctx)
	}

	switch function {
	case "uppercase":
		return strings.ToUpper(value), nil
	case "lowercase":
		return strings.ToLower(value), nil
	case "trim":
		return strings.TrimSpace(value), nil
	case "concat":
		if len(interpolatedArgs) > 0 {
			return value + interpolatedArgs[0], nil
		}
		return value, nil
	case "split":
		if len(interpolatedArgs) > 0 {
			parts := strings.Split(value, interpolatedArgs[0])
			return strings.Join(parts, "\n"), nil // Return as newline-separated for display
		}
		return value, nil
	case "replace":
		if len(interpolatedArgs) >= 2 {
			return strings.ReplaceAll(value, interpolatedArgs[0], interpolatedArgs[1]), nil
		}
		return value, nil
	case "join":
		if len(interpolatedArgs) > 0 {
			// Assume value is a newline-separated list
			parts := strings.Split(value, "\n")
			return strings.Join(parts, interpolatedArgs[0]), nil
		}
		return value, nil
	case "length":
		return fmt.Sprintf("%d", len(value)), nil
	case "slice":
		if len(interpolatedArgs) >= 2 {
			start, err1 := strconv.Atoi(interpolatedArgs[0])
			end, err2 := strconv.Atoi(interpolatedArgs[1])
			if err1 == nil && err2 == nil && start >= 0 && end <= len(value) && start <= end {
				return value[start:end], nil
			}
		}
		return value, nil
	default:
		return "", fmt.Errorf("unknown transformation function: %s", function)
	}
}

// shouldHandleError determines if a catch clause should handle the given error
func (e *Engine) shouldHandleError(err error, catchClause ast.CatchClause) bool {
	// If no specific error type is specified, catch all errors
	if catchClause.ErrorType == "" {
		return true
	}

	// Simple error type matching based on error message content
	// In a more sophisticated implementation, we'd have typed errors
	errorMsg := strings.ToLower(err.Error())
	errorType := strings.ToLower(catchClause.ErrorType)

	switch errorType {
	case "filenotfounderror", "filenotfound":
		return strings.Contains(errorMsg, "no such file") ||
			strings.Contains(errorMsg, "not found") ||
			strings.Contains(errorMsg, "does not exist")
	case "shellerror", "commanderror":
		return strings.Contains(errorMsg, "command") ||
			strings.Contains(errorMsg, "shell") ||
			strings.Contains(errorMsg, "exit")
	case "permissionerror", "permission":
		return strings.Contains(errorMsg, "permission") ||
			strings.Contains(errorMsg, "access denied")
	default:
		// For custom error types, do a simple string match
		return strings.Contains(errorMsg, errorType)
	}
}

// ListTasks returns a list of available tasks in the program
func (e *Engine) ListTasks(program *ast.Program) []TaskInfo {
	var tasks []TaskInfo
	for _, task := range program.Tasks {
		info := TaskInfo{
			Name:        task.Name,
			Description: task.Description,
		}
		if info.Description == "" {
			info.Description = "No description"
		}
		tasks = append(tasks, info)
	}
	return tasks
}

// TaskInfo represents information about a task
type TaskInfo struct {
	Name        string
	Description string
}

// ExecuteString is a convenience function that parses and executes v2 source code
func ExecuteString(input string, taskName string, output io.Writer) error {
	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		return fmt.Errorf("parse errors: %s", strings.Join(parser.Errors(), "; "))
	}

	engine := NewEngine(output)
	return engine.Execute(program, taskName)
}

// ParseString is a convenience function that parses v2 source code
func ParseString(input string) (*ast.Program, error) {
	return ParseStringWithFilename(input, "<input>")
}

// ParseStringWithFilename parses v2 source code with filename for better error reporting
func ParseStringWithFilename(input, filename string) (*ast.Program, error) {
	lexer := lexer.NewLexer(input)
	parser := parser.NewParserWithSource(lexer, filename, input)
	program := parser.ParseProgram()

	// Check for enhanced errors first
	if parser.ErrorList() != nil && parser.ErrorList().HasErrors() {
		return nil, parser.ErrorList()
	}

	// Fallback to legacy errors
	if len(parser.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors: %s", strings.Join(parser.Errors(), "; "))
	}

	return program, nil
}

// interpolateVariables replaces {variable} placeholders with actual values
func (e *Engine) interpolateVariables(message string, ctx *ExecutionContext) string {
	// Use regex to find {variable} patterns
	re := regexp.MustCompile(`\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(message, func(match string) string {
		// Extract content (remove { and })
		content := match[1 : len(match)-1]

		// Try to resolve simple variables first (most common case)
		if result, found := e.resolveSimpleVariableDirectly(content, ctx); found {
			return result
		}

		// Fall back to complex expression resolution
		if result := e.resolveExpression(content, ctx); result != "" {
			return result
		}

		// If nothing worked, return the original placeholder
		return match
	})
}

// resolveSimpleVariableDirectly handles simple variable resolution with proper empty string support
func (e *Engine) resolveSimpleVariableDirectly(variable string, ctx *ExecutionContext) (string, bool) {
	if ctx == nil {
		return "", false
	}

	// Handle variables with $ prefix (most common case for interpolation)
	if strings.HasPrefix(variable, "$") {
		varName := variable[1:] // Remove the $ prefix

		// Check parameters (stored without $ prefix)
		if value, exists := ctx.Parameters[varName]; exists {
			return value.AsString(), true
		}

		// Check captured variables (stored without $ prefix)
		if value, exists := ctx.Variables[varName]; exists {
			return value, true
		}
	} else {
		// Check parameters (bare identifiers)
		if value, exists := ctx.Parameters[variable]; exists {
			return value.AsString(), true
		}
		// Check captured variables (bare identifiers)
		if value, exists := ctx.Variables[variable]; exists {
			return value, true
		}
	}

	return "", false
}

// resolveExpression resolves various types of expressions
func (e *Engine) resolveExpression(expr string, ctx *ExecutionContext) string {
	// 1. Check for variable operations (e.g., "$version without prefix 'v'")
	if chain, err := e.parseVariableOperations(expr); err == nil && chain != nil {
		// Get the base variable value
		baseValue := e.resolveSimpleVariable(chain.Variable, ctx)
		if baseValue != "" {
			// Apply operations chain
			if result, err := e.applyVariableOperations(baseValue, chain, ctx); err == nil {
				return result
			}
		}
	}

	// 2. Check if it's a simple builtin function call (no arguments)
	if builtins.IsBuiltin(expr) {
		if result, err := builtins.CallBuiltin(expr); err == nil {
			return result
		}
	}

	// 3. Check for function calls with quoted string arguments
	// Pattern: "function('arg')" or "function(\"arg\")" or "function('arg1', 'arg2')"
	quotedArgRe := regexp.MustCompile(`^([^(]+)\((.+)\)$`)
	if matches := quotedArgRe.FindStringSubmatch(expr); len(matches) == 3 {
		funcName := strings.TrimSpace(matches[1])
		argsStr := matches[2]

		// Parse arguments - handle both single and multiple quoted arguments
		args := e.parseQuotedArguments(argsStr)

		if builtins.IsBuiltin(funcName) && len(args) > 0 {
			if result, err := builtins.CallBuiltin(funcName, args...); err == nil {
				return result
			}
		}
	}

	// 4. Check for function calls with parameter arguments
	// Pattern: "function(param)" where param is a parameter name
	paramArgRe := regexp.MustCompile(`^([^(]+)\(([^)]+)\)$`)
	if matches := paramArgRe.FindStringSubmatch(expr); len(matches) == 3 {
		funcName := strings.TrimSpace(matches[1])
		paramName := strings.TrimSpace(matches[2])

		// Resolve the parameter first
		if ctx != nil {
			if paramValue, exists := ctx.Parameters[paramName]; exists {
				if builtins.IsBuiltin(funcName) {
					if result, err := builtins.CallBuiltin(funcName, paramValue.AsString()); err == nil {
						return result
					}
				}
			}
		}
	}

	// 5. Check for $globals.key syntax for project settings
	if strings.HasPrefix(expr, "$globals.") {
		if ctx != nil && ctx.Project != nil {
			key := expr[9:] // Remove "$globals." prefix
			if value, exists := ctx.Project.Settings[key]; exists {
				return value
			}
			// Check special project variables
			if key == "version" && ctx.Project.Version != "" {
				return ctx.Project.Version
			}
			if key == "project" && ctx.Project.Name != "" {
				return ctx.Project.Name
			}
		}
		return ""
	}

	// 6. Check for simple parameter lookup (fallback for complex expressions)
	if ctx != nil {
		// Check for variables with $ prefix first (parameters and task-scoped variables)
		if strings.HasPrefix(expr, "$") {
			varName := expr[1:] // Remove the $ prefix

			// Check parameters (stored without $ prefix)
			if value, exists := ctx.Parameters[varName]; exists {
				return value.AsString()
			}

			// Check captured variables (stored without $ prefix)
			if value, exists := ctx.Variables[varName]; exists {
				return value
			}
		} else {
			// Check parameters (bare identifiers)
			if value, exists := ctx.Parameters[expr]; exists {
				return value.AsString()
			}
			// Check captured variables (bare identifiers)
			if value, exists := ctx.Variables[expr]; exists {
				return value
			}
		}
	}

	return ""
}

// resolveSimpleVariable resolves a simple variable without operations
func (e *Engine) resolveSimpleVariable(variable string, ctx *ExecutionContext) string {
	// Handle $globals.key syntax
	if strings.HasPrefix(variable, "$globals.") {
		if ctx != nil && ctx.Project != nil {
			key := variable[9:] // Remove "$globals." prefix
			if value, exists := ctx.Project.Settings[key]; exists {
				return value
			}
			// Check special project variables
			if key == "version" && ctx.Project.Version != "" {
				return ctx.Project.Version
			}
			if key == "project" && ctx.Project.Name != "" {
				return ctx.Project.Name
			}
		}
		return ""
	}

	// Handle regular variables
	if ctx != nil {
		// Check for variables with $ prefix first (parameters and task-scoped variables)
		if strings.HasPrefix(variable, "$") {
			varName := variable[1:] // Remove the $ prefix

			// Check parameters (stored without $ prefix)
			if value, exists := ctx.Parameters[varName]; exists {
				return value.AsString()
			}

			// Check captured variables (stored without $ prefix)
			if value, exists := ctx.Variables[varName]; exists {
				return value
			}
		} else {
			// Check parameters (bare identifiers)
			if value, exists := ctx.Parameters[variable]; exists {
				return value.AsString()
			}
			// Check captured variables (bare identifiers)
			if value, exists := ctx.Variables[variable]; exists {
				return value
			}
		}
	}

	return ""
}

// parseQuotedArguments parses comma-separated quoted arguments
func (e *Engine) parseQuotedArguments(argsStr string) []string {
	var args []string

	// Simple regex to match quoted strings
	quotedRe := regexp.MustCompile(`['"]([^'"]*?)['"]`)
	matches := quotedRe.FindAllStringSubmatch(argsStr, -1)

	for _, match := range matches {
		if len(match) > 1 {
			args = append(args, match[1])
		}
	}

	return args
}

// executeConditional executes conditional statements (when, if/else)
func (e *Engine) executeConditional(stmt *ast.ConditionalStatement, ctx *ExecutionContext) error {
	// Evaluate the condition
	conditionResult := e.evaluateCondition(stmt.Condition, ctx)

	if conditionResult {
		// Execute the main body
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	} else if len(stmt.ElseBody) > 0 {
		// Execute the else body if condition is false
		for _, elseStmt := range stmt.ElseBody {
			if err := e.executeStatement(elseStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeLoop executes loop statements (for each)
func (e *Engine) executeLoop(stmt *ast.LoopStatement, ctx *ExecutionContext) error {
	// If Type is not set, default to "each"
	loopType := stmt.Type
	if loopType == "" {
		loopType = "each"
	}

	switch loopType {
	case "range":
		return e.executeRangeLoop(stmt, ctx)
	case "line":
		return e.executeLineLoop(stmt, ctx)
	case "match":
		return e.executeMatchLoop(stmt, ctx)
	default: // "each"
		return e.executeEachLoop(stmt, ctx)
	}
}

// executeSequentialLoop executes loop items sequentially
func (e *Engine) executeSequentialLoop(stmt *ast.LoopStatement, items []string, ctx *ExecutionContext) error {
	_, _ = fmt.Fprintf(e.output, "🔄 Executing %d items sequentially\n", len(items))

	for i, item := range items {
		_, _ = fmt.Fprintf(e.output, "📋 Processing item %d/%d: %s\n", i+1, len(items), item)

		// Create a new context with the loop variable
		loopCtx := e.createLoopContext(ctx, stmt.Variable, item)

		// Execute the loop body
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, loopCtx); err != nil {
				// Check for break/continue control flow
				if breakErr, ok := err.(BreakError); ok {
					_, _ = fmt.Fprintf(e.output, "🔄 Breaking loop: %s\n", breakErr.Error())
					return nil // Break out of the entire loop
				}
				if continueErr, ok := err.(ContinueError); ok {
					_, _ = fmt.Fprintf(e.output, "🔄 Continuing loop: %s\n", continueErr.Error())
					break // Break out of the body execution, continue to next item
				}
				return fmt.Errorf("error processing item '%s': %v", item, err)
			}
		}
	}

	_, _ = fmt.Fprintf(e.output, "✅ Sequential loop completed: %d items processed\n", len(items))
	return nil
}

// executeParallelLoop executes loop items in parallel
func (e *Engine) executeParallelLoop(stmt *ast.LoopStatement, items []string, ctx *ExecutionContext) error {
	// Determine parallel execution settings
	maxWorkers := stmt.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 5 // reasonable default
	}

	failFast := stmt.FailFast

	// Create parallel executor
	executor := parallel.NewParallelExecutor(maxWorkers, failFast, e.output, e.dryRun)

	// Define the execution function for each item
	executeItem := func(body []ast.Statement, variables map[string]string) error {
		// Create a new context for this parallel execution
		loopCtx := &ExecutionContext{
			Parameters: make(map[string]*types.Value),
			Variables:  make(map[string]string),
			Project:    ctx.Project, // inherit project context
		}

		// Copy existing parameters and variables
		for k, v := range ctx.Parameters {
			loopCtx.Parameters[k] = v
		}
		for k, v := range ctx.Variables {
			loopCtx.Variables[k] = v
		}

		// Add the variables from the parallel executor
		for k, v := range variables {
			loopCtx.Variables[k] = v
			// Also add as a typed parameter for compatibility
			if itemValue, err := types.NewValue(types.StringType, v); err == nil {
				loopCtx.Parameters[k] = itemValue
			}
		}

		// Execute the loop body
		for _, bodyStmt := range body {
			if err := e.executeStatement(bodyStmt, loopCtx); err != nil {
				return err
			}
		}

		return nil
	}

	// Execute in parallel
	results, err := executor.ExecuteLoop(items, stmt.Variable, stmt.Body, executeItem)

	// Report results
	if err != nil {
		// Count successful executions
		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}

		_, _ = fmt.Fprintf(e.output, "⚠️  Parallel loop completed with errors: %d/%d successful\n",
			successCount, len(items))
		return err
	}

	return nil
}

// executeRangeLoop executes range loops
func (e *Engine) executeRangeLoop(stmt *ast.LoopStatement, ctx *ExecutionContext) error {
	start := e.interpolateVariables(stmt.RangeStart, ctx)
	end := e.interpolateVariables(stmt.RangeEnd, ctx)
	step := "1"
	if stmt.RangeStep != "" {
		step = e.interpolateVariables(stmt.RangeStep, ctx)
	}

	// Convert to integers (simplified implementation)
	startInt := 0
	endInt := 10
	stepInt := 1

	// In a real implementation, you would parse these properly
	// For now, we'll create a simple range
	var items []string
	for i := startInt; i <= endInt; i += stepInt {
		items = append(items, fmt.Sprintf("%d", i))
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute range loop from %s to %s step %s (%d items)\n", start, end, step, len(items))
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "🔄 Executing range loop from %s to %s step %s (%d items)\n", start, end, step, len(items))

	// Apply filter if present
	if stmt.Filter != nil {
		items = e.applyFilter(items, stmt.Filter, ctx)
	}

	// Execute loop
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, items, ctx)
	}
	return e.executeSequentialLoop(stmt, items, ctx)
}

// executeLineLoop executes line-by-line file processing loops
func (e *Engine) executeLineLoop(stmt *ast.LoopStatement, ctx *ExecutionContext) error {
	filename := e.interpolateVariables(stmt.Iterable, ctx)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would read lines from file: %s\n", filename)
		return nil
	}

	// In a real implementation, you would read the file
	// For now, we'll simulate with some sample lines
	lines := []string{"line1", "line2", "line3"}

	_, _ = fmt.Fprintf(e.output, "📄 Reading lines from file: %s (%d lines)\n", filename, len(lines))

	// Apply filter if present
	if stmt.Filter != nil {
		lines = e.applyFilter(lines, stmt.Filter, ctx)
	}

	// Execute loop
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, lines, ctx)
	}
	return e.executeSequentialLoop(stmt, lines, ctx)
}

// executeMatchLoop executes pattern matching loops
func (e *Engine) executeMatchLoop(stmt *ast.LoopStatement, ctx *ExecutionContext) error {
	pattern := e.interpolateVariables(stmt.Iterable, ctx)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would find matches for pattern: %s\n", pattern)
		return nil
	}

	// In a real implementation, you would use regex to find matches
	// For now, we'll simulate with some sample matches
	matches := []string{"match1", "match2"}

	_, _ = fmt.Fprintf(e.output, "🔍 Finding matches for pattern: %s (%d matches)\n", pattern, len(matches))

	// Apply filter if present
	if stmt.Filter != nil {
		matches = e.applyFilter(matches, stmt.Filter, ctx)
	}

	// Execute loop
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, matches, ctx)
	}
	return e.executeSequentialLoop(stmt, matches, ctx)
}

// executeEachLoop executes traditional each loops
func (e *Engine) executeEachLoop(stmt *ast.LoopStatement, ctx *ExecutionContext) error {
	// Resolve the iterable (could be a parameter, variable, file list, etc.)
	var iterableStr string

	// First check if it's a variable (starts with $)
	if strings.HasPrefix(stmt.Iterable, "$") {
		// Try both with and without $ prefix to handle different storage methods
		if value, exists := ctx.Variables[stmt.Iterable]; exists {
			iterableStr = value
		} else if value, exists := ctx.Variables[stmt.Iterable[1:]]; exists {
			iterableStr = value
		} else {
			return fmt.Errorf("variable '%s' not found", stmt.Iterable)
		}
	} else {
		// Check parameters
		iterableValue, exists := ctx.Parameters[stmt.Iterable]
		if !exists {
			return fmt.Errorf("iterable '%s' not found in parameters", stmt.Iterable)
		}
		iterableStr = iterableValue.AsString()
	}

	// Split by space to get items (for our variable operations system)
	iterableStr = strings.TrimSpace(iterableStr)
	if iterableStr == "" {
		_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
		return nil
	}

	items := strings.Fields(iterableStr) // Use Fields to split by any whitespace

	if len(items) == 0 {
		_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
		return nil
	}

	// Apply filter if present
	if stmt.Filter != nil {
		items = e.applyFilter(items, stmt.Filter, ctx)
	}

	// Check if this should run in parallel
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, items, ctx)
	}

	// Sequential execution
	return e.executeSequentialLoop(stmt, items, ctx)
}

// applyFilter applies filter conditions to a list of items
func (e *Engine) applyFilter(items []string, filter *ast.FilterExpression, ctx *ExecutionContext) []string {
	var filtered []string

	filterValue := e.interpolateVariables(filter.Value, ctx)

	for _, item := range items {
		match := false

		switch filter.Operator {
		case "contains":
			match = strings.Contains(item, filterValue)
		case "starts", "starts with":
			match = strings.HasPrefix(item, filterValue)
		case "ends", "ends with":
			match = strings.HasSuffix(item, filterValue)
		case "matches":
			// In a real implementation, you would use regex
			match = strings.Contains(item, filterValue)
		case "==":
			match = item == filterValue
		case "!=":
			match = item != filterValue
		default:
			// For other operators, just include the item
			match = true
		}

		if match {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) != len(items) {
		_, _ = fmt.Fprintf(e.output, "🔍 Filter applied: %d items match condition '%s %s %s'\n",
			len(filtered), filter.Variable, filter.Operator, filterValue)
	}

	return filtered
}

// createLoopContext creates a new execution context for a loop iteration
func (e *Engine) createLoopContext(ctx *ExecutionContext, variable, value string) *ExecutionContext {
	loopCtx := &ExecutionContext{
		Parameters: make(map[string]*types.Value),
		Variables:  make(map[string]string),
		Project:    ctx.Project, // inherit project context
	}

	// Copy existing parameters and variables
	for k, v := range ctx.Parameters {
		loopCtx.Parameters[k] = v
	}
	for k, v := range ctx.Variables {
		loopCtx.Variables[k] = v
	}

	// Set the loop variable as a string type
	itemValue, _ := types.NewValue(types.StringType, value)
	loopCtx.Parameters[variable] = itemValue
	loopCtx.Variables[variable] = value

	return loopCtx
}

// evaluateCondition evaluates condition expressions
func (e *Engine) evaluateCondition(condition string, ctx *ExecutionContext) bool {
	// Simple condition evaluation
	// For now, we'll handle basic patterns like "variable is value"

	// Handle "variable is value" pattern
	if strings.Contains(condition, " is ") {
		parts := strings.SplitN(condition, " is ", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// Try to get the value of the left side from parameters
			if value, exists := ctx.Parameters[left]; exists {
				return value.AsString() == right
			}

			// If not found in parameters, compare as strings
			return left == right
		}
	}

	// Interpolate variables in the condition for other cases
	interpolatedCondition := e.interpolateVariables(condition, ctx)

	// Handle boolean values directly
	switch strings.ToLower(strings.TrimSpace(interpolatedCondition)) {
	case "true":
		return true
	case "false":
		return false
	}

	// Default: treat non-empty strings as true
	return strings.TrimSpace(interpolatedCondition) != ""
}

// fileExists checks if a file exists
func (e *Engine) fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// getFileSize returns the size of a file in bytes
func (e *Engine) getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
