package engine

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/builtins"
	"github.com/phillarmonic/drun/internal/v2/fileops"
	"github.com/phillarmonic/drun/internal/v2/lexer"
	"github.com/phillarmonic/drun/internal/v2/parallel"
	"github.com/phillarmonic/drun/internal/v2/parser"
	"github.com/phillarmonic/drun/internal/v2/shell"
	"github.com/phillarmonic/drun/internal/v2/types"
)

// Engine executes drun v2 programs directly
type Engine struct {
	output io.Writer
	dryRun bool
}

// ExecutionContext holds parameter values and other runtime context
type ExecutionContext struct {
	Parameters map[string]*types.Value // parameter name -> typed value
	Variables  map[string]string       // captured variables from shell commands
	Project    *ProjectContext         // project-level settings and hooks
}

// ProjectContext holds project-level configuration
type ProjectContext struct {
	Name        string            // project name
	Version     string            // project version
	Settings    map[string]string // project settings (set key to value)
	BeforeHooks []ast.Statement   // before any task hooks
	AfterHooks  []ast.Statement   // after any task hooks
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
		} else if param.DefaultValue != "" {
			rawValue = param.DefaultValue
			hasValue = true
		} else if param.Required {
			return fmt.Errorf("required parameter '%s' not provided", param.Name)
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
				return fmt.Errorf("parameter '%s': invalid %s value '%s': %w",
					param.Name, paramType, rawValue, err)
			}

			// Validate constraints
			if err := typedValue.ValidateConstraints(param.Constraints); err != nil {
				return fmt.Errorf("parameter '%s': %w", param.Name, err)
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
		Name:        project.Name,
		Version:     project.Version,
		Settings:    make(map[string]string),
		BeforeHooks: []ast.Statement{},
		AfterHooks:  []ast.Statement{},
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
		}
	}

	return ctx
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
	// Interpolate variables in the command
	interpolatedCommand := e.interpolateVariables(shellStmt.Command, ctx)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute shell command: %s\n", interpolatedCommand)
		if shellStmt.CaptureVar != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would capture output as: %s\n", shellStmt.CaptureVar)
		}
		return nil
	}

	// Configure shell options based on the action type
	opts := shell.DefaultOptions()
	opts.CaptureOutput = true
	opts.StreamOutput = shellStmt.StreamOutput
	opts.Output = e.output

	// Show what we're about to execute
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
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute Docker command: docker %s %s", operation, resource)
		if name != "" {
			_, _ = fmt.Fprintf(e.output, " %s", name)
		}
		for key, value := range options {
			_, _ = fmt.Fprintf(e.output, " %s %s", key, value)
		}
		_, _ = fmt.Fprintf(e.output, "\n")
		return nil
	}

	// Build Docker command
	var dockerCmd []string
	dockerCmd = append(dockerCmd, "docker")

	// Handle Docker Compose separately
	if operation == "compose" {
		dockerCmd = append(dockerCmd, "compose")
		if command, exists := options["command"]; exists {
			dockerCmd = append(dockerCmd, command)
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
	}

	// Execute Docker command
	_, _ = fmt.Fprintf(e.output, "🐳 Running Docker: %s\n", strings.Join(dockerCmd, " "))

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
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute Git command: ")
		e.buildGitCommand(operation, resource, name, options, true)
		return nil
	}

	// Build and execute Git command
	_, _ = fmt.Fprintf(e.output, "🔗 Running Git: ")
	e.buildGitCommand(operation, resource, name, options, false)

	// For now, we'll simulate the command execution
	// In a real implementation, you would use exec.Command to run the git command
	// cmd := exec.Command("git", args...)
	// return cmd.Run()

	return nil
}

// buildGitCommand builds and displays the git command
func (e *Engine) buildGitCommand(operation, resource, name string, options map[string]string, dryRun bool) {
	var gitCmd []string
	gitCmd = append(gitCmd, "git")

	switch operation {
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
			if resource == "branch" {
				gitCmd = append(gitCmd, "branch", "--show-current")
			} else if resource == "commit" {
				gitCmd = append(gitCmd, "rev-parse", "HEAD")
			}
		} else {
			gitCmd = append(gitCmd, "show")
		}

	case "create":
		// git create branch "name"
		if resource == "branch" && name != "" {
			gitCmd = append(gitCmd, "checkout", "-b", name)
		}

	case "switch":
		// git switch to branch "name"
		if resource == "branch" && name != "" {
			gitCmd = append(gitCmd, "checkout", name)
		}

	case "delete":
		// git delete branch "name"
		if resource == "branch" && name != "" {
			gitCmd = append(gitCmd, "branch", "-d", name)
		}

	case "merge":
		// git merge branch "name" into "target"
		gitCmd = append(gitCmd, "merge")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}
	}

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "%s\n", strings.Join(gitCmd, " "))
	} else {
		_, _ = fmt.Fprintf(e.output, "%s\n", strings.Join(gitCmd, " "))
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
	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

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

		// Try to resolve the content
		if result := e.resolveExpression(content, ctx); result != "" {
			return result
		}

		// If nothing worked, return the original placeholder
		return match
	})
}

// resolveExpression resolves various types of expressions
func (e *Engine) resolveExpression(expr string, ctx *ExecutionContext) string {
	// 1. Check if it's a simple builtin function call (no arguments)
	if builtins.IsBuiltin(expr) {
		if result, err := builtins.CallBuiltin(expr); err == nil {
			return result
		}
	}

	// 2. Check for function calls with quoted string arguments
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

	// 3. Check for function calls with parameter arguments
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

	// 4. Check for simple parameter lookup
	if ctx != nil {
		if value, exists := ctx.Parameters[expr]; exists {
			return value.AsString()
		}
		// Also check captured variables
		if value, exists := ctx.Variables[expr]; exists {
			return value
		}
		// Check project settings
		if ctx.Project != nil {
			if value, exists := ctx.Project.Settings[expr]; exists {
				return value
			}
			// Check special project variables
			if expr == "version" && ctx.Project.Version != "" {
				return ctx.Project.Version
			}
			if expr == "project" && ctx.Project.Name != "" {
				return ctx.Project.Name
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
	// Resolve the iterable (could be a parameter, file list, etc.)
	iterableValue, exists := ctx.Parameters[stmt.Iterable]
	if !exists {
		return fmt.Errorf("iterable '%s' not found in parameters", stmt.Iterable)
	}

	// Split by comma to get items (simple implementation)
	iterableStr := strings.TrimSpace(iterableValue.AsString())
	if iterableStr == "" {
		_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
		return nil
	}

	items := strings.Split(iterableStr, ",")

	// Trim whitespace from items
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}

	if len(items) == 0 {
		_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
		return nil
	}

	// Check if this should run in parallel
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, items, ctx)
	}

	// Sequential execution
	return e.executeSequentialLoop(stmt, items, ctx)
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
