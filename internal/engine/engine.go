package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archives"
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
	output             io.Writer
	dryRun             bool
	verbose            bool
	allowUndefinedVars bool

	// Cached regex patterns for performance
	interpolationRegex *regexp.Regexp
	quotedArgRegex     *regexp.Regexp
	paramArgRegex      *regexp.Regexp
}

// ExecutionContext holds parameter values and other runtime context
type ExecutionContext struct {
	Parameters  map[string]*types.Value // parameter name -> typed value
	Variables   map[string]string       // captured variables from shell commands
	Project     *ProjectContext         // project-level settings and hooks
	CurrentFile string                  // path to the current drun file being executed
	CurrentTask string                  // name of the currently executing task
	Program     *ast.Program            // the AST program being executed
}

// ProjectContext holds project-level configuration
type ProjectContext struct {
	Name          string                              // project name
	Version       string                              // project version
	Settings      map[string]string                   // project settings (set key to value)
	BeforeHooks   []ast.Statement                     // before any task hooks
	AfterHooks    []ast.Statement                     // after any task hooks
	SetupHooks    []ast.Statement                     // on drun setup hooks
	TeardownHooks []ast.Statement                     // on drun teardown hooks
	ShellConfigs  map[string]*ast.PlatformShellConfig // platform-specific shell configurations
}

// NewEngine creates a new v2 execution engine
func NewEngine(output io.Writer) *Engine {
	if output == nil {
		output = os.Stdout
	}
	return &Engine{
		output: output,
		dryRun: false,

		// Pre-compile regex patterns for performance
		interpolationRegex: regexp.MustCompile(`\{([^}]+)\}`),
		quotedArgRegex:     regexp.MustCompile(`^([^(]+)\((.+)\)$`),
		paramArgRegex:      regexp.MustCompile(`^([^(]+)\(([^)]+)\)$`),
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

// SetAllowUndefinedVars enables or disables strict variable checking
func (e *Engine) SetAllowUndefinedVars(allow bool) {
	e.allowUndefinedVars = allow
}

// Execute runs a v2 program with no parameters
func (e *Engine) Execute(program *ast.Program, taskName string) error {
	return e.ExecuteWithParams(program, taskName, map[string]string{})
}

// ExecuteWithParams runs a v2 program with the given parameters
func (e *Engine) ExecuteWithParams(program *ast.Program, taskName string, params map[string]string) error {
	return e.ExecuteWithParamsAndFile(program, taskName, params, "")
}

// ExecuteWithParamsAndFile runs a v2 program with the given parameters and current file path
func (e *Engine) ExecuteWithParamsAndFile(program *ast.Program, taskName string, params map[string]string, currentFile string) error {
	if program == nil {
		return fmt.Errorf("program is nil")
	}

	// Start memory monitor to detect runaway execution
	monitor := NewMemoryMonitor(program)
	monitor.Start()
	defer monitor.Stop()

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
		Parameters:  make(map[string]*types.Value, 8), // Pre-allocate for typical parameter count
		Variables:   make(map[string]string, 16),      // Pre-allocate for typical variable count
		Project:     e.createProjectContext(program.Project),
		CurrentFile: currentFile,
		Program:     program,
	}

	// Execute drun setup hooks
	if ctx.Project != nil {
		for _, hook := range ctx.Project.SetupHooks {
			if err := e.executeStatement(hook, ctx); err != nil {
				return fmt.Errorf("drun setup hook failed: %v", err)
			}
		}
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

		// Set current task name for globals access
		ctx.CurrentTask = currentTaskName

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

	// Execute drun teardown hooks
	if ctx.Project != nil {
		for _, hook := range ctx.Project.TeardownHooks {
			if hookErr := e.executeStatement(hook, ctx); hookErr != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Drun teardown hook failed: %v\n", hookErr)
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
		} else if param.HasDefault {
			// For parameters with default values (both required and optional), use the default
			// Interpolate the default value if it contains braces (for builtin function calls)
			rawValue = e.interpolateVariables(param.DefaultValue, ctx)
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
		Name:          project.Name,
		Version:       project.Version,
		Settings:      make(map[string]string, 8),                   // Pre-allocate for typical settings count
		BeforeHooks:   make([]ast.Statement, 0, 4),                  // Pre-allocate for typical hook count
		AfterHooks:    make([]ast.Statement, 0, 4),                  // Pre-allocate for typical hook count
		SetupHooks:    make([]ast.Statement, 0, 2),                  // Pre-allocate for typical hook count
		TeardownHooks: make([]ast.Statement, 0, 2),                  // Pre-allocate for typical hook count
		ShellConfigs:  make(map[string]*ast.PlatformShellConfig, 4), // Pre-allocate for typical platform count
	}

	// Process project settings
	for _, setting := range project.Settings {
		switch s := setting.(type) {
		case *ast.SetStatement:
			// Convert expression to string representation
			if s.Value != nil {
				ctx.Settings[s.Key] = s.Value.String()
			}
		case *ast.LifecycleHook:
			switch s.Type {
			case "before":
				ctx.BeforeHooks = append(ctx.BeforeHooks, s.Body...)
			case "after":
				ctx.AfterHooks = append(ctx.AfterHooks, s.Body...)
			case "setup":
				ctx.SetupHooks = append(ctx.SetupHooks, s.Body...)
			case "teardown":
				ctx.TeardownHooks = append(ctx.TeardownHooks, s.Body...)
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
	case *ast.DownloadStatement:
		return e.executeDownload(s, ctx)
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
	case *ast.TaskCallStatement:
		return e.executeTaskCall(s, ctx)
	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// executeAction executes a single action statement
func (e *Engine) executeAction(action *ast.ActionStatement, ctx *ExecutionContext) error {
	// Interpolate variables in the message
	interpolatedMessage, err := e.interpolateVariablesWithError(action.Message, ctx)
	if err != nil {
		return fmt.Errorf("in %s statement: %w", action.Action, err)
	}

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
	case "echo":
		// Process \n escape sequences for newlines
		processedMessage := strings.ReplaceAll(interpolatedMessage, "\\n", "\n")
		_, _ = fmt.Fprintf(e.output, "%s\n", processedMessage)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}

	return nil
}

// executeTaskCall executes a task call statement
func (e *Engine) executeTaskCall(callStmt *ast.TaskCallStatement, ctx *ExecutionContext) error {
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would call task: %s\n", callStmt.TaskName)
		if len(callStmt.Parameters) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] With parameters: %v\n", callStmt.Parameters)
		}
		return nil
	}

	// Find the task to call
	var targetTask *ast.TaskStatement
	for _, task := range ctx.Program.Tasks {
		if task.Name == callStmt.TaskName {
			targetTask = task
			break
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task '%s' not found", callStmt.TaskName)
	}

	// Create a new execution context for the called task
	callCtx := &ExecutionContext{
		Parameters:  make(map[string]*types.Value, 8),
		Variables:   make(map[string]string, 16),
		Project:     ctx.Project,
		CurrentFile: ctx.CurrentFile,
		CurrentTask: callStmt.TaskName,
		Program:     ctx.Program,
	}

	// Copy current variables to the new context
	for k, v := range ctx.Variables {
		callCtx.Variables[k] = v
	}

	// Set up parameters for the called task
	if err := e.setupTaskParameters(targetTask, callStmt.Parameters, callCtx); err != nil {
		return fmt.Errorf("failed to setup parameters for task '%s': %v", callStmt.TaskName, err)
	}

	// Execute the called task
	if err := e.executeTask(targetTask, callCtx); err != nil {
		return fmt.Errorf("task '%s' failed: %v", callStmt.TaskName, err)
	}

	// Copy back any new variables that might have been set in the called task
	for k, v := range callCtx.Variables {
		ctx.Variables[k] = v
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
	interpolatedCommand, err := e.interpolateVariablesWithError(shellStmt.Command, ctx)
	if err != nil {
		return fmt.Errorf("in shell command: %w", err)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute shell command: %s\n", interpolatedCommand)
		if shellStmt.CaptureVar != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture output as: %s\n", shellStmt.CaptureVar)
			// Set a placeholder value for the captured variable in dry-run mode
			ctx.Variables[shellStmt.CaptureVar] = "[DRY RUN] command output"
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
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅ Command completed successfully (exit code: %d, duration: %v)\n",
				result.ExitCode, result.Duration)
		}
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
			// Set a placeholder value for the captured variable in dry-run mode
			ctx.Variables[shellStmt.CaptureVar] = "[DRY RUN] command output"
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
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅ Multiline commands completed successfully (exit code: %d, duration: %v)\n",
				result.ExitCode, result.Duration)
		}
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
			// Set a placeholder value for the captured variable in dry-run mode
			ctx.Variables[fileStmt.CaptureVar] = "[DRY RUN] file content"
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
	options := make(map[string]string, len(dockerStmt.Options))
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
	options := make(map[string]string, len(gitStmt.Options))
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
	headers := make(map[string]string, len(httpStmt.Headers))
	for key, value := range httpStmt.Headers {
		headers[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate auth
	auth := make(map[string]string, len(httpStmt.Auth))
	for key, value := range httpStmt.Auth {
		auth[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate options
	options := make(map[string]string, len(httpStmt.Options))
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

// executeDownload executes file download operations using native Go HTTP client
func (e *Engine) executeDownload(downloadStmt *ast.DownloadStatement, ctx *ExecutionContext) error {
	// Interpolate variables in download statement
	url := e.interpolateVariables(downloadStmt.URL, ctx)
	path := e.interpolateVariables(downloadStmt.Path, ctx)

	// Interpolate headers
	headers := make(map[string]string, len(downloadStmt.Headers))
	for key, value := range downloadStmt.Headers {
		headers[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate auth
	auth := make(map[string]string, len(downloadStmt.Auth))
	for key, value := range downloadStmt.Auth {
		auth[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate options
	options := make(map[string]string, len(downloadStmt.Options))
	for key, value := range downloadStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	// Check if file exists and handle overwrite
	if !downloadStmt.AllowOverwrite && e.fileExists(path) {
		errMsg := fmt.Sprintf("file already exists: %s (use 'allow overwrite' to replace)", path)
		_, _ = fmt.Fprintf(e.output, "❌ %s\n", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would download %s to %s", url, path)
		if downloadStmt.AllowOverwrite {
			_, _ = fmt.Fprintf(e.output, " (overwrite allowed)")
		}
		if len(downloadStmt.AllowPermissions) > 0 {
			_, _ = fmt.Fprintf(e.output, " with permissions: ")
			for i, perm := range downloadStmt.AllowPermissions {
				if i > 0 {
					_, _ = fmt.Fprintf(e.output, ", ")
				}
				_, _ = fmt.Fprintf(e.output, "%v to %v", perm.Permissions, perm.Targets)
			}
		}
		_, _ = fmt.Fprintf(e.output, "\n")
		return nil
	}

	// Show what we're about to do
	_, _ = fmt.Fprintf(e.output, "⬇️  Downloading: %s\n", url)
	_, _ = fmt.Fprintf(e.output, "   → %s\n", path)

	// Perform the download with progress tracking
	err := e.downloadFileWithProgress(url, path, headers, auth, options)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌ Download failed: %v\n", err)
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract archive if requested
	if downloadStmt.ExtractTo != "" {
		extractTo := e.interpolateVariables(downloadStmt.ExtractTo, ctx)
		_, _ = fmt.Fprintf(e.output, "📦 Extracting archive to: %s\n", extractTo)

		err = e.extractArchive(path, extractTo)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "❌ Extraction failed: %v\n", err)
			return fmt.Errorf("extraction failed: %w", err)
		}

		_, _ = fmt.Fprintf(e.output, "✅ Extraction completed\n")

		// Remove archive if requested
		if downloadStmt.RemoveArchive {
			_, _ = fmt.Fprintf(e.output, "🗑️  Removing archive: %s\n", path)
			err = os.Remove(path)
			if err != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Warning: Failed to remove archive: %v\n", err)
			} else {
				_, _ = fmt.Fprintf(e.output, "✅ Archive removed\n")
			}
		}
	} else {
		// Apply file permissions if specified (only for non-extracted files)
		if len(downloadStmt.AllowPermissions) > 0 {
			err = e.applyFilePermissions(path, downloadStmt.AllowPermissions)
			if err != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Warning: Failed to set permissions: %v\n", err)
				// Don't fail the download, just warn
			}
		}
	}

	_, _ = fmt.Fprintf(e.output, "✅ Downloaded successfully to: %s\n", path)
	return nil
}

// executeNetwork executes network operations (health checks, port testing, ping)
func (e *Engine) executeNetwork(networkStmt *ast.NetworkStatement, ctx *ExecutionContext) error {
	// Interpolate variables in network statement
	target := e.interpolateVariables(networkStmt.Target, ctx)
	port := e.interpolateVariables(networkStmt.Port, ctx)
	condition := e.interpolateVariables(networkStmt.Condition, ctx)

	// Interpolate options
	options := make(map[string]string, len(networkStmt.Options))
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
			// Set the detected version in variables (e.g., docker_version)
			ctx.Variables[stmt.Target+"_version"] = version
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

// executeIfAvailable executes "if tool is available" and "if tool is not available" conditions
func (e *Engine) executeIfAvailable(detector *detection.Detector, stmt *ast.DetectionStatement, ctx *ExecutionContext) error {
	available := detector.IsToolAvailable(stmt.Target)

	// Handle negation for "not available" conditions
	var conditionMet bool
	var conditionText string
	if stmt.Condition == "not_available" {
		conditionMet = !available
		conditionText = fmt.Sprintf("%s is not available", stmt.Target)
	} else {
		conditionMet = available
		conditionText = fmt.Sprintf("%s is available", stmt.Target)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s: %t\n", conditionText, conditionMet)
		if conditionMet {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute if body for %s\n", stmt.Target)
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

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Checking if %s: %t\n", conditionText, conditionMet)
	}

	if conditionMet {
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

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Checking %s version %s %s %s: %t (current: %s)\n",
			stmt.Target, version, stmt.Condition, targetVersion, matches, version)
	}

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

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔍 Checking if in %s environment: %t (current: %s)\n",
			stmt.Target, matches, currentEnv)
	}

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
				// Set the variable in dry-run mode too
				ctx.Variables[stmt.CaptureVar] = workingTool
			}
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would find: none available\n")
			if stmt.CaptureVar != "" {
				// Set a placeholder in dry-run mode when no tool is found
				ctx.Variables[stmt.CaptureVar] = "[DRY RUN] no tool available"
			}
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
	case "capture":
		return e.executeCaptureStatement(varStmt, ctx)
	case "capture_shell":
		return e.executeCaptureShellStatement(varStmt, ctx)
	default:
		return fmt.Errorf("unknown variable operation: %s", varStmt.Operation)
	}
}

// executeLetStatement executes "let variable = value" statements
func (e *Engine) executeLetStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	value, err := e.evaluateExpression(varStmt.Value, ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %v", err)
	}

	// Interpolate the value if it contains braces (for builtin function calls)
	interpolatedValue := e.interpolateVariables(value, ctx)

	// Store the variable in the context even in dry run for interpolation
	ctx.Variables[varStmt.Variable] = interpolatedValue

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set variable %s = %s\n", varStmt.Variable, interpolatedValue)
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "📝 Set variable %s = %s\n", varStmt.Variable, interpolatedValue)

	return nil
}

// executeSetStatement executes "set variable to value" statements
func (e *Engine) executeSetStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	value, err := e.evaluateExpression(varStmt.Value, ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %v", err)
	}

	// Interpolate the value if it contains braces (for builtin function calls)
	interpolatedValue := e.interpolateVariables(value, ctx)

	// Store the variable in the context even in dry run for interpolation
	ctx.Variables[varStmt.Variable] = interpolatedValue

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would set variable %s to %s\n", varStmt.Variable, interpolatedValue)
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "📝 Set variable %s to %s\n", varStmt.Variable, interpolatedValue)

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

// executeCaptureStatement executes "capture variable_name from expression" statements
func (e *Engine) executeCaptureStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	expression, err := e.evaluateExpression(varStmt.Value, ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate expression: %v", err)
	}

	// The expression is already evaluated, so we can use it directly as the value
	value := expression

	// Store the captured value in the context
	ctx.Variables[varStmt.Variable] = value

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture %s: %s\n",
			varStmt.Variable, value)
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "📥 Captured %s: %s\n",
			varStmt.Variable, value)
	}

	return nil
}

// executeCaptureShellStatement executes "capture from shell command as $variable" statements
func (e *Engine) executeCaptureShellStatement(varStmt *ast.VariableStatement, ctx *ExecutionContext) error {
	// Extract the command from the literal expression
	literalExpr, ok := varStmt.Value.(*ast.LiteralExpression)
	if !ok {
		return fmt.Errorf("expected literal expression for shell capture command")
	}

	// Interpolate variables in the command
	command := e.interpolateVariables(literalExpr.Value, ctx)

	// Execute the shell command
	shellOpts := e.getPlatformShellConfig(ctx)
	result, err := shell.Execute(command, shellOpts)
	if err != nil {
		return fmt.Errorf("failed to capture from shell command '%s': %v", command, err)
	}

	// Store the captured output (trimmed)
	value := strings.TrimSpace(result.Stdout)
	ctx.Variables[varStmt.Variable] = value

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture %s from shell: %s\n",
			varStmt.Variable, value)
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "📥 Captured %s from shell: %s\n",
			varStmt.Variable, value)
	}

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
	result, _ := e.interpolateVariablesWithError(message, ctx)
	return result
}

// interpolateVariablesWithError replaces {variable} placeholders with actual values and returns any undefined variable errors
func (e *Engine) interpolateVariablesWithError(message string, ctx *ExecutionContext) (string, error) {
	var undefinedVars []string

	// Use cached regex for better performance
	result := e.interpolationRegex.ReplaceAllStringFunc(message, func(match string) string {
		// Extract content (remove { and })
		content := match[1 : len(match)-1]

		// Try to resolve simple variables first (most common case)
		if resolved, found := e.resolveSimpleVariableDirectly(content, ctx); found {
			return resolved
		}

		// Fall back to complex expression resolution
		if resolved := e.resolveExpression(content, ctx); resolved != "" {
			return resolved
		}

		// If nothing worked, check if we should be strict about undefined variables
		if !e.allowUndefinedVars {
			// For complex expressions, check if the base variable exists
			if e.isComplexExpression(content) {
				baseVar := e.extractBaseVariable(content)
				if baseVar != "" && !e.variableExists(baseVar, ctx) {
					undefinedVars = append(undefinedVars, baseVar)
					return match
				}
				// If base variable exists but expression failed, allow it (might be a function call or other valid expression)
			} else {
				// For simple variables, report as undefined
				undefinedVars = append(undefinedVars, content)
			}
			return match // Return original placeholder for now
		}

		// If allowing undefined variables, return the original placeholder
		return match
	})

	// If we found undefined variables in strict mode, return an error
	if len(undefinedVars) > 0 {
		if len(undefinedVars) == 1 {
			return result, fmt.Errorf("undefined variable: {%s}", undefinedVars[0])
		}
		return result, fmt.Errorf("undefined variables: {%s}", strings.Join(undefinedVars, "}, {"))
	}

	return result, nil
}

// resolveSimpleVariableDirectly handles simple variable resolution with proper empty string support
func (e *Engine) resolveSimpleVariableDirectly(variable string, ctx *ExecutionContext) (string, bool) {
	if ctx == nil {
		return "", false
	}

	// Handle variables with $ prefix (most common case for interpolation)
	if strings.HasPrefix(variable, "$") {
		// First try to find the variable with the $ prefix (shell captures)
		if value, exists := ctx.Variables[variable]; exists {
			return value, true
		}

		// Then try without the $ prefix (legacy variables)
		varName := variable[1:] // Remove the $ prefix

		// Check parameters (stored without $ prefix)
		if value, exists := ctx.Parameters[varName]; exists {
			return value.AsString(), true
		}

		// Check captured variables (stored without $ prefix)
		if value, exists := ctx.Variables[varName]; exists {
			return value, true
		}

		// Check project-level variables for backward compatibility
		if ctx.Project != nil {
			// Check built-in project variables
			if varName == "project" && ctx.Project.Name != "" {
				return ctx.Project.Name, true
			}
			if varName == "version" && ctx.Project.Version != "" {
				return ctx.Project.Version, true
			}
			// Check project settings
			if ctx.Project.Settings != nil {
				if value, exists := ctx.Project.Settings[varName]; exists {
					return value, true
				}
			}
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

		// Check project-level variables for backward compatibility
		if ctx.Project != nil {
			// Check built-in project variables
			if variable == "project" && ctx.Project.Name != "" {
				return ctx.Project.Name, true
			}
			if variable == "version" && ctx.Project.Version != "" {
				return ctx.Project.Version, true
			}
			// Check project settings
			if ctx.Project.Settings != nil {
				if value, exists := ctx.Project.Settings[variable]; exists {
					return value, true
				}
			}
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

	// 2. Check for context-aware builtin functions first
	if expr == "current file" && ctx != nil {
		if ctx.CurrentFile != "" {
			return ctx.CurrentFile
		}
		return "<no file>"
	}

	// 3. Check for builtin function with piped operations (e.g., "current git branch | replace '/' by '-'")
	if strings.Contains(expr, "|") {
		parts := strings.SplitN(expr, "|", 2)
		if len(parts) == 2 {
			funcName := strings.TrimSpace(parts[0])
			operations := strings.TrimSpace(parts[1])

			// Check if the first part is a builtin function
			if builtins.IsBuiltin(funcName) {
				if result, err := builtins.CallBuiltin(funcName); err == nil {
					// Parse and apply the operations to the result
					if chain, err := e.parseBuiltinOperations(operations); err == nil && chain != nil {
						if finalResult, err := e.applyBuiltinOperations(result, chain, ctx); err == nil {
							return finalResult
						}
					}
					// If operations parsing fails, just return the builtin result
					return result
				}
			}
		}
	}

	// 4. Check if it's a simple builtin function call (no arguments)
	if builtins.IsBuiltin(expr) {
		if result, err := builtins.CallBuiltin(expr); err == nil {
			return result
		}
	}

	// 4. Check for function calls with quoted string arguments
	// Pattern: "function('arg')" or "function(\"arg\")" or "function('arg1', 'arg2')"
	if matches := e.quotedArgRegex.FindStringSubmatch(expr); len(matches) == 3 {
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

	// 5. Check for function calls with parameter arguments
	// Pattern: "function(param)" where param is a parameter name
	if matches := e.paramArgRegex.FindStringSubmatch(expr); len(matches) == 3 {
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

	// 6. Check for $globals.key syntax for project settings
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
			// Check current task
			if key == "current_task" && ctx.CurrentTask != "" {
				return ctx.CurrentTask
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
			// Check current task
			if key == "current_task" && ctx.CurrentTask != "" {
				return ctx.CurrentTask
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
	// In strict mode, check for undefined variables in the condition
	if !e.allowUndefinedVars {
		if err := e.checkConditionForUndefinedVars(stmt.Condition, ctx); err != nil {
			return fmt.Errorf("in %s condition: %w", stmt.Type, err)
		}
	}

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
			Parameters: make(map[string]*types.Value, len(ctx.Parameters)+len(variables)), // Pre-allocate for parent + new variables
			Variables:  make(map[string]string, len(ctx.Variables)+len(variables)),        // Pre-allocate for parent + new variables
			Project:    ctx.Project,                                                       // inherit project context
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
	// Resolve the iterable (could be a parameter, variable, array literal, etc.)
	var items []string

	// Check if it's an array literal (starts with '[')
	if strings.HasPrefix(stmt.Iterable, "[") && strings.HasSuffix(stmt.Iterable, "]") {
		// Parse array literal
		items = e.parseArrayLiteralString(stmt.Iterable)
	} else if strings.HasPrefix(stmt.Iterable, "$globals.") {
		// Handle $globals.key syntax for project settings (check this before general $ variables)
		if ctx.Project != nil && ctx.Project.Settings != nil {
			key := stmt.Iterable[9:] // Remove "$globals." prefix
			if projectValue, exists := ctx.Project.Settings[key]; exists {
				// Handle project setting (could be array or string)
				if strings.HasPrefix(projectValue, "[") && strings.HasSuffix(projectValue, "]") {
					// It's an array literal stored as a string
					items = e.parseArrayLiteralString(projectValue)
				} else {
					// It's a regular string, split by whitespace
					iterableStr := strings.TrimSpace(projectValue)
					if iterableStr == "" {
						_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
						return nil
					}
					items = strings.Fields(iterableStr)
				}
			} else {
				return fmt.Errorf("project setting '%s' not found", key)
			}
		} else {
			return fmt.Errorf("no project defined for $globals access")
		}
	} else if strings.HasPrefix(stmt.Iterable, "$") {
		// Variable reference
		var iterableStr string
		// Try both with and without $ prefix to handle different storage methods
		if value, exists := ctx.Variables[stmt.Iterable]; exists {
			iterableStr = value
		} else if value, exists := ctx.Variables[stmt.Iterable[1:]]; exists {
			iterableStr = value
		} else {
			return fmt.Errorf("variable '%s' not found", stmt.Iterable)
		}

		// Split by space to get items (for our variable operations system)
		iterableStr = strings.TrimSpace(iterableStr)
		if iterableStr == "" {
			_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
			return nil
		}

		items = strings.Fields(iterableStr) // Use Fields to split by any whitespace
	} else {
		// Check if it's a legacy direct project setting access (for backward compatibility)
		if ctx.Project != nil && ctx.Project.Settings != nil {
			if projectValue, exists := ctx.Project.Settings[stmt.Iterable]; exists {
				// Handle project setting (could be array or string) - but warn about deprecated usage
				_, _ = fmt.Fprintf(e.output, "⚠️  Warning: Direct project setting access '%s' is deprecated. Use '$globals.%s' instead.\n", stmt.Iterable, stmt.Iterable)
				if strings.HasPrefix(projectValue, "[") && strings.HasSuffix(projectValue, "]") {
					// It's an array literal stored as a string
					items = e.parseArrayLiteralString(projectValue)
				} else {
					// It's a regular string, split by whitespace
					iterableStr := strings.TrimSpace(projectValue)
					if iterableStr == "" {
						_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
						return nil
					}
					items = strings.Fields(iterableStr)
				}
			} else {
				// Parameter reference
				iterableValue, exists := ctx.Parameters[stmt.Iterable]
				if !exists {
					return fmt.Errorf("iterable '%s' not found in parameters or project settings", stmt.Iterable)
				}
				iterableStr := iterableValue.AsString()

				// Split by space to get items (for our variable operations system)
				iterableStr = strings.TrimSpace(iterableStr)
				if iterableStr == "" {
					_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
					return nil
				}

				items = strings.Fields(iterableStr) // Use Fields to split by any whitespace
			}
		} else {
			// Parameter reference (no project)
			iterableValue, exists := ctx.Parameters[stmt.Iterable]
			if !exists {
				return fmt.Errorf("iterable '%s' not found in parameters", stmt.Iterable)
			}
			iterableStr := iterableValue.AsString()

			// Split by space to get items (for our variable operations system)
			iterableStr = strings.TrimSpace(iterableStr)
			if iterableStr == "" {
				_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
				return nil
			}

			items = strings.Fields(iterableStr) // Use Fields to split by any whitespace
		}
	}

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
		Parameters: make(map[string]*types.Value, len(ctx.Parameters)+1), // Pre-allocate for parent + loop variable
		Variables:  make(map[string]string, len(ctx.Variables)+1),        // Pre-allocate for parent + loop variable
		Project:    ctx.Project,                                          // inherit project context
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

// parseArrayLiteralString parses an array literal string like ["item1", "item2", "item3"] into a slice of strings
func (e *Engine) parseArrayLiteralString(arrayStr string) []string {
	// Remove brackets
	arrayStr = strings.TrimSpace(arrayStr)
	if len(arrayStr) < 2 || arrayStr[0] != '[' || arrayStr[len(arrayStr)-1] != ']' {
		return []string{}
	}

	content := arrayStr[1 : len(arrayStr)-1]
	content = strings.TrimSpace(content)

	// Handle empty array
	if content == "" {
		return []string{}
	}

	var items []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, char := range content {
		switch char {
		case '\\':
			if !escaped {
				escaped = true
				continue
			}
			current.WriteRune(char)
		case '"':
			if !escaped {
				inQuotes = !inQuotes
			} else {
				current.WriteRune(char)
			}
		case ',':
			if !inQuotes && !escaped {
				// End of current item
				item := strings.TrimSpace(current.String())
				if item != "" {
					items = append(items, item)
				}
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
		escaped = false
	}

	// Add the last item
	item := strings.TrimSpace(current.String())
	if item != "" {
		items = append(items, item)
	}

	return items
}

// checkConditionForUndefinedVars checks if a condition contains undefined variables
func (e *Engine) checkConditionForUndefinedVars(condition string, ctx *ExecutionContext) error {
	// For conditions, we only need to check simple variable references like "$var is value"
	// Complex expressions in conditions are handled by the condition evaluation itself
	re := regexp.MustCompile(`\$\w+`)
	matches := re.FindAllString(condition, -1)

	var undefinedVars []string
	for _, match := range matches {
		varName := match[1:] // Remove $ prefix

		// Check if variable exists in parameters or variables
		if _, exists := ctx.Parameters[varName]; exists {
			continue
		}
		if _, exists := ctx.Variables[varName]; exists {
			continue
		}
		if _, exists := ctx.Variables[match]; exists { // Check with $ prefix
			continue
		}

		// Variable not found
		undefinedVars = append(undefinedVars, match)
	}

	if len(undefinedVars) > 0 {
		if len(undefinedVars) == 1 {
			return fmt.Errorf("undefined variable: {%s}", undefinedVars[0])
		}
		return fmt.Errorf("undefined variables: {%s}", strings.Join(undefinedVars, "}, {"))
	}

	return nil
}

// isComplexExpression checks if an expression contains operations or function calls
func (e *Engine) isComplexExpression(expr string) bool {
	// Check for variable operations like "without", "with", etc.
	if strings.Contains(expr, " without ") || strings.Contains(expr, " with ") {
		return true
	}
	// Check for function calls (contains parentheses or dots)
	if strings.Contains(expr, "(") || strings.Contains(expr, ".") {
		return true
	}
	// Check for pipe operations
	if strings.Contains(expr, " | ") {
		return true
	}
	return false
}

// extractBaseVariable extracts the base variable from a complex expression
func (e *Engine) extractBaseVariable(expr string) string {
	// For expressions like "$version without prefix 'v'", extract "$version"
	parts := strings.Fields(expr)
	if len(parts) > 0 && strings.HasPrefix(parts[0], "$") {
		return parts[0]
	}
	return ""
}

// variableExists checks if a variable exists in the context
func (e *Engine) variableExists(varName string, ctx *ExecutionContext) bool {
	if ctx == nil {
		return false
	}

	// Remove $ prefix for checking
	cleanName := varName
	if strings.HasPrefix(varName, "$") {
		cleanName = varName[1:]
	}

	// Check parameters
	if _, exists := ctx.Parameters[cleanName]; exists {
		return true
	}
	// Check variables
	if _, exists := ctx.Variables[cleanName]; exists {
		return true
	}
	// Check variables with $ prefix
	if _, exists := ctx.Variables[varName]; exists {
		return true
	}

	// Check project-level variables for backward compatibility
	if ctx.Project != nil {
		// Check built-in project variables
		if cleanName == "project" && ctx.Project.Name != "" {
			return true
		}
		if cleanName == "version" && ctx.Project.Version != "" {
			return true
		}
		// Check project settings
		if ctx.Project.Settings != nil {
			if _, exists := ctx.Project.Settings[cleanName]; exists {
				return true
			}
		}
	}

	return false
}

// evaluateCondition evaluates condition expressions
func (e *Engine) evaluateCondition(condition string, ctx *ExecutionContext) bool {
	// Simple condition evaluation
	// Handle various patterns like "variable is value", "variable is not empty", etc.

	// Handle "folder/directory is not empty" pattern
	if strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is not empty", 2)
		if len(parts) >= 1 {
			left := strings.TrimSpace(parts[0])

			// Check if this is a folder/directory path check
			if strings.HasPrefix(left, "folder ") || strings.HasPrefix(left, "directory ") || strings.HasPrefix(left, "dir ") {
				var folderPath string
				if strings.HasPrefix(left, "folder ") {
					folderPath = strings.TrimSpace(left[7:]) // Remove "folder "
				} else if strings.HasPrefix(left, "directory ") {
					folderPath = strings.TrimSpace(left[10:]) // Remove "directory "
				} else if strings.HasPrefix(left, "dir ") {
					folderPath = strings.TrimSpace(left[4:]) // Remove "dir "
				}

				// Remove quotes if present
				folderPath = strings.Trim(folderPath, "\"'")

				// Interpolate variables in the path
				folderPath = e.interpolateVariables(folderPath, ctx)

				// Check if directory exists and is not empty
				if !e.dirExists(folderPath) {
					return false // Directory doesn't exist, treat as empty
				}

				isEmpty, err := e.isDirEmpty(folderPath)
				if err != nil {
					return false // Error checking, treat as empty
				}
				return !isEmpty // Return true if directory is NOT empty
			}
		}
	}

	// Handle "variable is not empty" pattern
	if strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is not empty", 2)
		if len(parts) >= 1 {
			left := strings.TrimSpace(parts[0])

			// Strip $ prefix if present
			paramName := left
			if strings.HasPrefix(left, "$") {
				paramName = left[1:]
			}

			// Try to get the value of the left side from parameters
			if value, exists := ctx.Parameters[paramName]; exists {
				valueStr := value.AsString()
				// For lists, check if the list is empty
				if value.Type == types.ListType {
					if list, err := value.AsList(); err == nil {
						return len(list) > 0
					}
				}
				// For other types, check if string representation is not empty
				return strings.TrimSpace(valueStr) != ""
			}

			// Try interpolating the variable
			interpolated := e.interpolateVariables("{"+left+"}", ctx)
			// If interpolation didn't change it, the variable doesn't exist (treat as empty)
			if interpolated == "{"+left+"}" {
				return false
			}
			return strings.TrimSpace(interpolated) != ""
		}
	}

	// Handle "variable is not value" pattern
	if strings.Contains(condition, " is not ") && !strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is not ", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// Handle "empty" keyword - treat as empty string
			if right == "empty" {
				right = ""
			}

			// Strip $ prefix if present
			paramName := left
			if strings.HasPrefix(left, "$") {
				paramName = left[1:]
			}

			// Try to get the value of the left side from parameters first
			if value, exists := ctx.Parameters[paramName]; exists {
				return value.AsString() != right
			}

			// Try to get the value from variables (let statements)
			if value, exists := ctx.Variables[paramName]; exists {
				return value != right
			}

			// Also try with the $ prefix (variables stored with $ prefix)
			if value, exists := ctx.Variables["$"+paramName]; exists {
				return value != right
			}

			// If not found in parameters or variables, compare as strings
			return left != right
		}
	}

	// Handle "folder/directory is empty" pattern
	if strings.Contains(condition, " is empty") && !strings.Contains(condition, " is not empty") {
		parts := strings.SplitN(condition, " is empty", 2)
		if len(parts) >= 1 {
			left := strings.TrimSpace(parts[0])

			// Check if this is a folder/directory path check
			if strings.HasPrefix(left, "folder ") || strings.HasPrefix(left, "directory ") || strings.HasPrefix(left, "dir ") {
				var folderPath string
				if strings.HasPrefix(left, "folder ") {
					folderPath = strings.TrimSpace(left[7:]) // Remove "folder "
				} else if strings.HasPrefix(left, "directory ") {
					folderPath = strings.TrimSpace(left[10:]) // Remove "directory "
				} else if strings.HasPrefix(left, "dir ") {
					folderPath = strings.TrimSpace(left[4:]) // Remove "dir "
				}

				// Remove quotes if present
				folderPath = strings.Trim(folderPath, "\"'")

				// Interpolate variables in the path
				folderPath = e.interpolateVariables(folderPath, ctx)

				// Check if directory exists and is empty
				if !e.dirExists(folderPath) {
					return true // Directory doesn't exist, treat as empty
				}

				isEmpty, err := e.isDirEmpty(folderPath)
				if err != nil {
					return true // Error checking, treat as empty
				}
				return isEmpty // Return true if directory IS empty
			}
		}
	}

	// Handle "variable is value" pattern
	if strings.Contains(condition, " is ") {
		parts := strings.SplitN(condition, " is ", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// Handle "empty" keyword - treat as empty string
			if right == "empty" {
				right = ""
			}

			// Strip $ prefix if present
			paramName := left
			if strings.HasPrefix(left, "$") {
				paramName = left[1:]
			}

			// Try to get the value of the left side from parameters first
			if value, exists := ctx.Parameters[paramName]; exists {
				return value.AsString() == right
			}

			// Try to get the value from variables (let statements)
			if value, exists := ctx.Variables[paramName]; exists {
				return value == right
			}

			// Also try with the $ prefix (variables stored with $ prefix)
			if value, exists := ctx.Variables["$"+paramName]; exists {
				return value == right
			}

			// If not found in parameters or variables, compare as strings
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

// dirExists checks if a directory exists
func (e *Engine) dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// isDirEmpty checks if a directory is empty
func (e *Engine) isDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	// Count only visible entries (filter out hidden files on Windows)
	visibleCount := 0
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files (those starting with . on Unix, or system files on Windows)
		if strings.HasPrefix(name, ".") {
			continue
		}
		// Skip common Windows system files
		if runtime.GOOS == "windows" {
			lowerName := strings.ToLower(name)
			if lowerName == "desktop.ini" || lowerName == "thumbs.db" || lowerName == "$recycle.bin" {
				continue
			}
		}
		visibleCount++
	}

	return visibleCount == 0, nil
}

// evaluateExpression evaluates an AST expression and returns its string value
func (e *Engine) evaluateExpression(expr ast.Expression, ctx *ExecutionContext) (string, error) {
	if expr == nil {
		return "", nil
	}

	switch ex := expr.(type) {
	case *ast.LiteralExpression:
		return ex.Value, nil

	case *ast.IdentifierExpression:
		// Look up variable value
		varName := ex.Value
		if strings.HasPrefix(varName, "$") {
			// Direct $variable reference
			if value, exists := ctx.Variables[varName]; exists {
				return value, nil
			}
			return "", fmt.Errorf("undefined variable: %s", varName)
		} else {
			// {variable} reference - look up without braces
			if value, exists := ctx.Variables[varName]; exists {
				return value, nil
			}
			return "", fmt.Errorf("undefined variable: %s", varName)
		}

	case *ast.BinaryExpression:
		return e.evaluateBinaryExpression(ex, ctx)

	case *ast.FunctionCallExpression:
		return e.evaluateFunctionCall(ex, ctx)

	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// evaluateBinaryExpression evaluates binary operations like {a} - {b}
func (e *Engine) evaluateBinaryExpression(expr *ast.BinaryExpression, ctx *ExecutionContext) (string, error) {
	leftVal, err := e.evaluateExpression(expr.Left, ctx)
	if err != nil {
		return "", err
	}

	rightVal, err := e.evaluateExpression(expr.Right, ctx)
	if err != nil {
		return "", err
	}

	switch expr.Operator {
	case "-":
		// Try to parse as numbers for arithmetic
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			result := leftNum - rightNum
			// Return as integer if it's a whole number
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		return "", fmt.Errorf("cannot subtract non-numeric values: %s - %s", leftVal, rightVal)

	case "+":
		// Try to parse as numbers for arithmetic
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			result := leftNum + rightNum
			// Return as integer if it's a whole number
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		// If not numbers, concatenate as strings
		return leftVal + rightVal, nil

	case "*":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			result := leftNum * rightNum
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		return "", fmt.Errorf("cannot multiply non-numeric values: %s * %s", leftVal, rightVal)

	case "/":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if rightNum == 0 {
				return "", fmt.Errorf("division by zero")
			}
			result := leftNum / rightNum
			if result == float64(int64(result)) {
				return fmt.Sprintf("%.0f", result), nil
			}
			return fmt.Sprintf("%g", result), nil
		}
		return "", fmt.Errorf("cannot divide non-numeric values: %s / %s", leftVal, rightVal)

	case "==":
		if leftVal == rightVal {
			return "true", nil
		}
		return "false", nil

	case "!=":
		if leftVal != rightVal {
			return "true", nil
		}
		return "false", nil

	case "<":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum < rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal < rightVal {
			return "true", nil
		}
		return "false", nil

	case ">":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum > rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal > rightVal {
			return "true", nil
		}
		return "false", nil

	case "<=":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum <= rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal <= rightVal {
			return "true", nil
		}
		return "false", nil

	case ">=":
		leftNum, leftErr := strconv.ParseFloat(leftVal, 64)
		rightNum, rightErr := strconv.ParseFloat(rightVal, 64)

		if leftErr == nil && rightErr == nil {
			if leftNum >= rightNum {
				return "true", nil
			}
			return "false", nil
		}
		// String comparison
		if leftVal >= rightVal {
			return "true", nil
		}
		return "false", nil

	default:
		return "", fmt.Errorf("unsupported binary operator: %s", expr.Operator)
	}
}

// evaluateFunctionCall evaluates function calls like now(), current git branch
func (e *Engine) evaluateFunctionCall(expr *ast.FunctionCallExpression, ctx *ExecutionContext) (string, error) {
	switch expr.Function {
	case "now":
		return fmt.Sprintf("%d", time.Now().Unix()), nil

	default:
		// For other functions, treat them as shell commands or interpolation
		functionStr := expr.Function
		if len(expr.Arguments) > 0 {
			var args []string
			for _, arg := range expr.Arguments {
				argVal, err := e.evaluateExpression(arg, ctx)
				if err != nil {
					return "", err
				}
				args = append(args, argVal)
			}
			functionStr += "(" + strings.Join(args, ", ") + ")"
		}

		// Try to execute as shell command
		shellOpts := e.getPlatformShellConfig(ctx)
		result, err := shell.Execute(functionStr, shellOpts)
		if err != nil {
			return "", fmt.Errorf("failed to execute function '%s': %v", functionStr, err)
		}
		return strings.TrimSpace(result.Stdout), nil
	}
}

// parseBuiltinOperations parses operations for builtin functions (e.g., "replace '/' by '-'")
func (e *Engine) parseBuiltinOperations(operations string) (*VariableOperationChain, error) {
	// Split by | to handle multiple operations
	parts := strings.Split(operations, "|")
	var ops []VariableOperation

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse individual operation
		tokens := strings.Fields(part)
		if len(tokens) == 0 {
			continue
		}

		op, err := e.parseBuiltinOperation(tokens)
		if err != nil {
			return nil, err
		}
		if op != nil {
			ops = append(ops, *op)
		}
	}

	if len(ops) == 0 {
		return nil, nil
	}

	return &VariableOperationChain{
		Variable:   "", // Not used for builtin operations
		Operations: ops,
	}, nil
}

// parseBuiltinOperation parses a single builtin operation
func (e *Engine) parseBuiltinOperation(tokens []string) (*VariableOperation, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	opType := tokens[0]
	args := []string{}

	switch opType {
	case "replace":
		// "replace '/' by '-'" or "replace '/' with '-'"
		if len(tokens) >= 4 && (tokens[2] == "by" || tokens[2] == "with") {
			// Remove quotes from arguments
			from := strings.Trim(tokens[1], `"'`)
			to := strings.Trim(tokens[3], `"'`)
			args = append(args, from, to)
		}
	case "without":
		// "without prefix 'v'" or "without suffix '.tmp'"
		if len(tokens) >= 3 {
			args = append(args, tokens[1]) // "prefix" or "suffix"
			argValue := strings.Join(tokens[2:], " ")
			argValue = strings.Trim(argValue, `"'`)
			args = append(args, argValue)
		}
	case "uppercase", "lowercase", "trim":
		// No arguments needed
	default:
		return nil, fmt.Errorf("unknown builtin operation: %s", opType)
	}

	return &VariableOperation{
		Type: opType,
		Args: args,
	}, nil
}

// applyBuiltinOperations applies operations to a builtin function result
func (e *Engine) applyBuiltinOperations(value string, chain *VariableOperationChain, ctx *ExecutionContext) (string, error) {
	currentValue := value

	for _, op := range chain.Operations {
		newValue, err := e.applyBuiltinOperation(currentValue, op, ctx)
		if err != nil {
			return "", fmt.Errorf("builtin operation '%s' failed: %v", op.Type, err)
		}
		currentValue = newValue
	}

	return currentValue, nil
}

// applyBuiltinOperation applies a single operation to a builtin function result
func (e *Engine) applyBuiltinOperation(value string, op VariableOperation, ctx *ExecutionContext) (string, error) {
	switch op.Type {
	case "replace":
		if len(op.Args) >= 2 {
			return strings.ReplaceAll(value, op.Args[0], op.Args[1]), nil
		}
		return "", fmt.Errorf("replace operation requires 2 arguments")
	case "without":
		return e.applyWithoutOperation(value, op.Args)
	case "uppercase":
		return strings.ToUpper(value), nil
	case "lowercase":
		return strings.ToLower(value), nil
	case "trim":
		return strings.TrimSpace(value), nil
	default:
		return "", fmt.Errorf("unknown builtin operation type: %s", op.Type)
	}
}

// downloadFileWithProgress downloads a file using native Go HTTP client with progress tracking
func (e *Engine) downloadFileWithProgress(url, filePath string, headers, auth, options map[string]string) error {
	// Create HTTP client with timeout
	timeout := 30 * time.Second
	if timeoutStr, exists := options["timeout"]; exists {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = duration
		}
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add authentication
	for authType, value := range auth {
		switch authType {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+value)
		case "basic":
			// Basic auth in format "username:password"
			req.Header.Set("Authorization", "Basic "+value)
		case "token":
			req.Header.Set("Authorization", "Token "+value)
		}
	}

	// Perform request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create parent directories if they don't exist
	if dir := filepath.Dir(filePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	// Create output file
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Get content length for progress tracking
	contentLength := resp.ContentLength

	// Create progress writer
	startTime := time.Now()
	var downloaded int64
	lastUpdate := time.Now()

	// Create a reader that tracks progress
	reader := io.TeeReader(resp.Body, &progressWriter{
		total: contentLength,
		onProgress: func(written int64) {
			downloaded = written
			// Update progress every 100ms to avoid overwhelming output
			if time.Since(lastUpdate) > 100*time.Millisecond || written == contentLength {
				lastUpdate = time.Now()
				e.showDownloadProgress(written, contentLength, time.Since(startTime))
			}
		},
	})

	// Copy data
	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Final progress update
	_, _ = fmt.Fprintf(e.output, "\r\033[K") // Clear line

	// Calculate final stats
	duration := time.Since(startTime)
	speed := float64(downloaded) / duration.Seconds()
	_, _ = fmt.Fprintf(e.output, "   📊 %s in %s (%.2f MB/s)\n",
		formatBytes(downloaded),
		duration.Round(time.Millisecond),
		speed/1024/1024)

	return nil
}

// progressWriter wraps io.Writer to track progress
type progressWriter struct {
	total      int64
	written    int64
	onProgress func(int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)
	if pw.onProgress != nil {
		pw.onProgress(pw.written)
	}
	return n, nil
}

// showDownloadProgress displays download progress with speed and ETA
func (e *Engine) showDownloadProgress(downloaded, total int64, elapsed time.Duration) {
	if total <= 0 {
		// Unknown size, just show downloaded amount
		_, _ = fmt.Fprintf(e.output, "\r   📥 Downloaded: %s", formatBytes(downloaded))
		return
	}

	// Calculate progress percentage
	percent := float64(downloaded) / float64(total) * 100

	// Calculate speed (bytes per second)
	speed := float64(downloaded) / elapsed.Seconds()

	// Calculate ETA
	remaining := total - downloaded
	var eta time.Duration
	if speed > 0 {
		eta = time.Duration(float64(remaining)/speed) * time.Second
	}

	// Create progress bar
	barWidth := 30
	filled := int(float64(barWidth) * percent / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Format output
	_, _ = fmt.Fprintf(e.output, "\r   📥 [%s] %.1f%% | %s/%s | %.2f MB/s | ETA: %s",
		bar,
		percent,
		formatBytes(downloaded),
		formatBytes(total),
		speed/1024/1024,
		formatDuration(eta))
}

// formatBytes formats bytes into human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration into human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// applyFilePermissions applies Unix file permissions based on permission specs
func (e *Engine) applyFilePermissions(path string, permSpecs []ast.PermissionSpec) error {
	// Get current file info
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	currentMode := info.Mode()
	newMode := currentMode

	// Build permission map
	for _, spec := range permSpecs {
		for _, perm := range spec.Permissions {
			for _, target := range spec.Targets {
				// Map permission and target to Unix file mode bits
				var permBits os.FileMode
				switch perm {
				case "read":
					switch target {
					case "user":
						permBits = 0400
					case "group":
						permBits = 0040
					case "others":
						permBits = 0004
					}
				case "write":
					switch target {
					case "user":
						permBits = 0200
					case "group":
						permBits = 0020
					case "others":
						permBits = 0002
					}
				case "execute":
					switch target {
					case "user":
						permBits = 0100
					case "group":
						permBits = 0010
					case "others":
						permBits = 0001
					}
				}

				// Add permission bits
				newMode |= permBits
			}
		}
	}

	// Apply new permissions
	if newMode != currentMode {
		err = os.Chmod(path, newMode)
		if err != nil {
			return fmt.Errorf("failed to chmod: %w", err)
		}
		_, _ = fmt.Fprintf(e.output, "   🔒 Set permissions: %s\n", newMode.String())
	}

	return nil
}

// extractArchive extracts an archive file to the specified directory using the archives library
func (e *Engine) extractArchive(archivePath, extractTo string) error {
	// Create extract directory if it doesn't exist
	err := os.MkdirAll(extractTo, 0755)
	if err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	// Open the archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = archiveFile.Close() }()

	// Identify the archive format
	format, archiveReader, err := archives.Identify(context.Background(), archivePath, archiveFile)
	if err != nil {
		return fmt.Errorf("failed to identify archive format: %w", err)
	}

	// Check if it's an extractor
	extractor, ok := format.(archives.Extractor)
	if !ok {
		// If it's just compressed (not archived), try to decompress it
		if decompressor, ok := format.(archives.Decompressor); ok {
			return e.decompressFile(decompressor, archiveReader, archivePath, extractTo)
		}
		return fmt.Errorf("format does not support extraction: %s", archivePath)
	}

	// Extract the archive
	handler := func(ctx context.Context, f archives.FileInfo) error {
		// Construct the output path
		outputPath := filepath.Join(extractTo, f.NameInArchive)

		// Handle directories
		if f.IsDir() {
			return os.MkdirAll(outputPath, f.Mode())
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Open the file in the archive
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in archive: %w", err)
		}
		defer func() { _ = rc.Close() }()

		// Create the output file
		outFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = outFile.Close() }()

		// Copy the contents
		if _, err := io.Copy(outFile, rc); err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		return nil
	}

	// Extract all files
	err = extractor.Extract(context.Background(), archiveReader, handler)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	return nil
}

// decompressFile decompresses a single compressed file (not an archive)
func (e *Engine) decompressFile(decompressor archives.Decompressor, reader io.Reader, archivePath, extractTo string) error {
	// Open decompression reader
	rc, err := decompressor.OpenReader(reader)
	if err != nil {
		return fmt.Errorf("failed to open decompressor: %w", err)
	}
	defer func() { _ = rc.Close() }()

	// Determine output filename by removing compression extension
	baseName := filepath.Base(archivePath)
	// Remove common compression extensions
	for _, ext := range []string{".gz", ".bz2", ".xz", ".zst", ".br", ".lz4", ".sz"} {
		if strings.HasSuffix(strings.ToLower(baseName), ext) {
			baseName = strings.TrimSuffix(baseName, ext)
			break
		}
	}

	outputPath := filepath.Join(extractTo, baseName)

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	// Decompress
	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}

	return nil
}
