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
	"github.com/phillarmonic/drun/internal/cache"
	"github.com/phillarmonic/drun/internal/detection"
	"github.com/phillarmonic/drun/internal/engine/hooks"
	"github.com/phillarmonic/drun/internal/engine/includes"
	"github.com/phillarmonic/drun/internal/engine/interpolation"
	"github.com/phillarmonic/drun/internal/errors"
	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
	"github.com/phillarmonic/drun/internal/remote"
	"github.com/phillarmonic/drun/internal/shell"
	"github.com/phillarmonic/drun/internal/types"
)

// Engine executes drun v2 programs directly
type Engine struct {
	output       io.Writer
	dryRun       bool
	verbose      bool
	interpolator *interpolation.Interpolator

	// Remote includes support
	cacheManager     *cache.Manager
	githubFetcher    *remote.GitHubFetcher
	httpsFetcher     *remote.HTTPSFetcher
	drunhubFetcher   *remote.DrunhubFetcher
	includesResolver *includes.Resolver

	// Legacy regex patterns (still used by variable operations)
	quotedArgRegex *regexp.Regexp
	paramArgRegex  *regexp.Regexp
}

// ExecutionContext and ProjectContext moved to context.go

// NewEngine creates a new v2 execution engine
func NewEngine(output io.Writer) *Engine {
	if output == nil {
		output = os.Stdout
	}
	githubFetcher := remote.NewGitHubFetcher()
	interp := interpolation.NewInterpolator()

	httpsFetcher := remote.NewHTTPSFetcher()
	drunhubFetcher := remote.NewDrunhubFetcher(githubFetcher)

	e := &Engine{
		output:         output,
		dryRun:         false,
		interpolator:   interp,
		githubFetcher:  githubFetcher,
		httpsFetcher:   httpsFetcher,
		drunhubFetcher: drunhubFetcher,

		// Pre-compile regex patterns for performance (still used by variable operations)
		quotedArgRegex: regexp.MustCompile(`^([^(]+)\((.+)\)$`),
		paramArgRegex:  regexp.MustCompile(`^([^(]+)\(([^)]+)\)$`),
	}

	// Initialize includes resolver
	e.includesResolver = includes.NewResolver(
		nil, // cacheManager set later
		githubFetcher,
		httpsFetcher,
		drunhubFetcher,
		false, // verbose set later
		output,
		ParseStringWithFilename,
	)

	// Set up interpolator callbacks for variable and builtin operations
	interp.SetResolveVariableOpsCallback(func(expr string, ctx interface{}) string {
		if execCtx, ok := ctx.(*ExecutionContext); ok {
			if chain, err := e.parseVariableOperations(expr); err == nil && chain != nil {
				baseValue := interp.Interpolate(chain.Variable, execCtx)
				if result, err := e.applyVariableOperations(baseValue, chain, execCtx); err == nil {
					return result
				}
			}
		}
		return ""
	})

	interp.SetResolveBuiltinOpsCallback(func(funcName string, operations string, ctx interface{}) (string, error) {
		if execCtx, ok := ctx.(*ExecutionContext); ok {
			if result, err := builtins.CallBuiltin(funcName); err == nil {
				if chain, err := e.parseBuiltinOperations(operations); err == nil && chain != nil {
					return e.applyBuiltinOperations(result, chain, execCtx)
				}
			}
		}
		return "", fmt.Errorf("failed to resolve builtin operations")
	})

	return e
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
	e.interpolator.SetAllowUndefined(allow)
}

// SetCacheEnabled enables or disables remote include caching
func (e *Engine) SetCacheEnabled(enabled bool) error {
	var err error
	if enabled {
		e.cacheManager, err = cache.NewManager(1*time.Minute, false)
	} else {
		e.cacheManager, err = cache.NewManager(0, true) // disabled
	}
	return err
}

// Cleanup removes temporary files created during execution
func (e *Engine) Cleanup() {
	if e.includesResolver != nil {
		e.includesResolver.Cleanup()
	}
	if e.cacheManager != nil {
		_ = e.cacheManager.Close()
	}
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
		Project:     e.createProjectContext(program.Project, currentFile),
		CurrentFile: currentFile,
		Program:     program,
	}

	// Execute drun setup hooks
	if ctx.Project != nil && ctx.Project.HookManager != nil {
		for _, hook := range ctx.Project.HookManager.GetSetupHooks() {
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
		if currentTaskName == taskName && ctx.Project != nil && ctx.Project.HookManager != nil {
			for _, hook := range ctx.Project.HookManager.GetBeforeHooks() {
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
		if currentTaskName == taskName && ctx.Project != nil && ctx.Project.HookManager != nil {
			for _, hook := range ctx.Project.HookManager.GetAfterHooks() {
				if hookErr := e.executeStatement(hook, ctx); hookErr != nil {
					_, _ = fmt.Fprintf(e.output, "⚠️  After hook failed: %v\n", hookErr)
				}
			}
		}
	}

	// Execute drun teardown hooks
	if ctx.Project != nil && ctx.Project.HookManager != nil {
		for _, hook := range ctx.Project.HookManager.GetTeardownHooks() {
			if hookErr := e.executeStatement(hook, ctx); hookErr != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Drun teardown hook failed: %v\n", hookErr)
			}
		}
	}

	return nil
}

// setupTaskParameters sets up parameters for a specific task
func (e *Engine) setupTaskParameters(task *ast.TaskStatement, params map[string]string, ctx *ExecutionContext) error {
	// First, add project-level parameters if they exist
	if ctx.Project != nil && ctx.Project.Parameters != nil {
		for paramName, projectParam := range ctx.Project.Parameters {
			// Check if user provided a value via CLI
			var rawValue string
			var hasValue bool

			if providedValue, exists := params[paramName]; exists {
				rawValue = providedValue
				hasValue = true
			} else if projectParam.HasDefault {
				rawValue = e.interpolateVariables(projectParam.DefaultValue, ctx)
				hasValue = true
			}

			if hasValue {
				// Determine parameter type
				paramType, err := types.ParseParameterType(projectParam.DataType)
				if err != nil {
					paramType = types.InferType(rawValue)
				}

				// Create typed value
				typedValue, err := types.NewValue(paramType, rawValue)
				if err != nil {
					return errors.NewParameterValidationError(fmt.Sprintf("project parameter '%s': invalid %s value '%s': %v",
						paramName, paramType, rawValue, err))
				}

				// Validate constraints
				if err := typedValue.ValidateConstraints(projectParam.Constraints); err != nil {
					return errors.NewParameterValidationError(fmt.Sprintf("project parameter '%s': %v", paramName, err))
				}

				if err := typedValue.ValidateAdvancedConstraints(projectParam.MinValue, projectParam.MaxValue, projectParam.Pattern, projectParam.PatternMacro, projectParam.EmailFormat); err != nil {
					return errors.NewParameterValidationError(fmt.Sprintf("project parameter '%s': %v", paramName, err))
				}

				ctx.Parameters[paramName] = typedValue
			}
		}
	}

	// Set up task-specific parameters with defaults and validation
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
func (e *Engine) createProjectContext(project *ast.ProjectStatement, currentFile string) *ProjectContext {
	if project == nil {
		return nil
	}

	ctx := &ProjectContext{
		Name:              project.Name,
		Version:           project.Version,
		Settings:          make(map[string]string, 8),                         // Pre-allocate for typical settings count
		Parameters:        make(map[string]*ast.ProjectParameterStatement, 8), // Pre-allocate for project parameters
		Snippets:          make(map[string]*ast.SnippetStatement, 8),          // Pre-allocate for snippets
		HookManager:       hooks.NewManager(),                                 // Initialize hooks manager
		ShellConfigs:      make(map[string]*ast.PlatformShellConfig, 4),       // Pre-allocate for typical platform count
		IncludedSnippets:  make(map[string]*ast.SnippetStatement, 16),         // Pre-allocate for included snippets
		IncludedTemplates: make(map[string]*ast.TaskTemplateStatement, 16),    // Pre-allocate for included templates
		IncludedTasks:     make(map[string]*ast.TaskStatement, 16),            // Pre-allocate for included tasks
		IncludedFiles:     make(map[string]bool, 4),                           // Pre-allocate for included files
	}

	// Process project settings
	for _, setting := range project.Settings {
		switch s := setting.(type) {
		case *ast.SetStatement:
			// Convert expression to string representation
			if s.Value != nil {
				ctx.Settings[s.Key] = s.Value.String()
			}
		case *ast.ProjectParameterStatement:
			// Store project-level parameter
			ctx.Parameters[s.Name] = s
		case *ast.SnippetStatement:
			// Store snippet for later use
			ctx.Snippets[s.Name] = s
		case *ast.LifecycleHook:
			switch s.Type {
			case "before":
				ctx.HookManager.RegisterBeforeHooks(s.Body)
			case "after":
				ctx.HookManager.RegisterAfterHooks(s.Body)
			case "setup":
				ctx.HookManager.RegisterSetupHooks(s.Body)
			case "teardown":
				ctx.HookManager.RegisterTeardownHooks(s.Body)
			}
		case *ast.ShellConfigStatement:
			// Store shell configurations for each platform
			for platform, config := range s.Platforms {
				ctx.ShellConfigs[platform] = config
			}
		case *ast.IncludeStatement:
			// Process include statement
			e.includesResolver.ProcessInclude(ctx, s, currentFile)
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
	case *ast.UseSnippetStatement:
		return e.executeUseSnippet(s, ctx)
	case *ast.TaskFromTemplateStatement:
		return e.executeTaskFromTemplate(s, ctx)
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
		// Optional line breaks - only add if explicitly requested
		if action.LineBreakBefore {
			_, _ = fmt.Fprintln(e.output)
		}

		// Print the box
		boxWidth := len(interpolatedMessage) + 4
		topLine := "┌" + strings.Repeat("─", boxWidth-2) + "┐"
		middleLine := "│ " + interpolatedMessage + " │"
		bottomLine := "└" + strings.Repeat("─", boxWidth-2) + "┘"
		_, _ = fmt.Fprintf(e.output, "%s\n%s\n%s\n", topLine, middleLine, bottomLine)

		// Optional line break after
		if action.LineBreakAfter {
			_, _ = fmt.Fprintln(e.output)
		}
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

	// Find the task or template to call
	var targetTask *ast.TaskStatement

	var taskNamespace string // Track the namespace for transitive resolution

	// Check if it's a namespaced reference (contains dot)
	if strings.Contains(callStmt.TaskName, ".") && ctx.Project != nil {
		// Extract namespace from the task name (e.g., "docker.build" -> "docker")
		parts := strings.SplitN(callStmt.TaskName, ".", 2)
		taskNamespace = parts[0]

		// First check included templates
		if template, exists := ctx.Project.IncludedTemplates[callStmt.TaskName]; exists {
			// Convert template to task for execution
			targetTask = &ast.TaskStatement{
				Token:       template.Token,
				Name:        template.Name,
				Description: template.Description,
				Parameters:  template.Parameters,
				Body:        template.Body,
			}
		} else if task, exists := ctx.Project.IncludedTasks[callStmt.TaskName]; exists {
			// Check included tasks
			targetTask = task
		}
	} else {
		// Local task/template - check local templates first
		for _, template := range ctx.Program.Templates {
			if template.Name == callStmt.TaskName {
				// Convert template to task for execution
				targetTask = &ast.TaskStatement{
					Token:       template.Token,
					Name:        template.Name,
					Description: template.Description,
					Parameters:  template.Parameters,
					Body:        template.Body,
				}
				break
			}
		}

		// If not a template, check regular tasks
		if targetTask == nil {
			for _, task := range ctx.Program.Tasks {
				if task.Name == callStmt.TaskName {
					targetTask = task
					break
				}
			}
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task '%s' not found", callStmt.TaskName)
	}

	// Create a new execution context for the called task
	callCtx := &ExecutionContext{
		Parameters:       make(map[string]*types.Value, 8),
		Variables:        make(map[string]string, 16),
		Project:          ctx.Project,
		CurrentFile:      ctx.CurrentFile,
		CurrentTask:      callStmt.TaskName,
		CurrentNamespace: taskNamespace, // Set namespace for transitive resolution
		Program:          ctx.Program,
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

// executeUseSnippet executes a snippet by running its body statements
func (e *Engine) executeUseSnippet(useStmt *ast.UseSnippetStatement, ctx *ExecutionContext) error {
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute snippet: %s\n", useStmt.SnippetName)
		return nil
	}

	// Find the snippet in the project context
	if ctx.Project == nil {
		return fmt.Errorf("snippet '%s' not found: no project context", useStmt.SnippetName)
	}

	var snippet *ast.SnippetStatement
	var exists bool

	// Check if it's a namespaced reference (contains dot)
	if strings.Contains(useStmt.SnippetName, ".") {
		// Look in included snippets
		snippet, exists = ctx.Project.IncludedSnippets[useStmt.SnippetName]
	} else {
		// Try current namespace first (for transitive resolution)
		if ctx.CurrentNamespace != "" {
			namespacedName := ctx.CurrentNamespace + "." + useStmt.SnippetName
			snippet, exists = ctx.Project.IncludedSnippets[namespacedName]
		}

		// If not found in namespace, look in local snippets
		if !exists {
			snippet, exists = ctx.Project.Snippets[useStmt.SnippetName]
		}
	}

	if !exists {
		return fmt.Errorf("snippet '%s' not found", useStmt.SnippetName)
	}

	// Execute all statements in the snippet body
	for _, stmt := range snippet.Body {
		if err := e.executeStatement(stmt, ctx); err != nil {
			return fmt.Errorf("error executing snippet '%s': %w", useStmt.SnippetName, err)
		}
	}

	return nil
}

// executeTaskFromTemplate instantiates and executes a task from a template
func (e *Engine) executeTaskFromTemplate(tfts *ast.TaskFromTemplateStatement, ctx *ExecutionContext) error {
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would instantiate task '%s' from template '%s'\n", tfts.Name, tfts.TemplateName)
		return nil
	}

	// Find the template in the program
	if ctx.Program == nil || ctx.Program.Templates == nil {
		return fmt.Errorf("template '%s' not found: no templates defined", tfts.TemplateName)
	}

	var template *ast.TaskTemplateStatement
	for _, tmpl := range ctx.Program.Templates {
		if tmpl.Name == tfts.TemplateName {
			template = tmpl
			break
		}
	}

	if template == nil {
		return fmt.Errorf("template '%s' not found", tfts.TemplateName)
	}

	// Create a new execution context for the instantiated task
	taskCtx := &ExecutionContext{
		Parameters:  make(map[string]*types.Value, 8),
		Variables:   make(map[string]string, 16),
		Project:     ctx.Project,
		CurrentFile: ctx.CurrentFile,
		CurrentTask: tfts.Name,
		Program:     ctx.Program,
	}

	// Copy current variables to the new context
	for k, v := range ctx.Variables {
		taskCtx.Variables[k] = v
	}

	// Set up parameters from the template with overrides from the instantiation
	for _, param := range template.Parameters {
		var rawValue string
		var hasValue bool

		// Check if there's an override value
		if overrideValue, exists := tfts.Overrides[param.Name]; exists {
			rawValue = overrideValue
			hasValue = true
		} else if param.HasDefault {
			// Use the template's default value
			rawValue = e.interpolateVariables(param.DefaultValue, taskCtx)
			hasValue = true
		} else if param.Required {
			return fmt.Errorf("required parameter '%s' not provided for template '%s'", param.Name, tfts.TemplateName)
		}

		// Create typed value if we have a value
		if hasValue {
			paramType, err := types.ParseParameterType(param.DataType)
			if err != nil {
				paramType = types.InferType(rawValue)
			}

			typedValue, err := types.NewValue(paramType, rawValue)
			if err != nil {
				return fmt.Errorf("parameter '%s': invalid %s value '%s': %v",
					param.Name, paramType, rawValue, err)
			}

			if err := typedValue.ValidateConstraints(param.Constraints); err != nil {
				return fmt.Errorf("parameter '%s': %v", param.Name, err)
			}

			if err := typedValue.ValidateAdvancedConstraints(param.MinValue, param.MaxValue, param.Pattern, param.PatternMacro, param.EmailFormat); err != nil {
				return fmt.Errorf("parameter '%s': %v", param.Name, err)
			}

			taskCtx.Parameters[param.Name] = typedValue
		}
	}

	// Execute the template body
	for _, stmt := range template.Body {
		if err := e.executeStatement(stmt, taskCtx); err != nil {
			return fmt.Errorf("error executing template task '%s': %w", tfts.Name, err)
		}
	}

	// Copy back any new variables that might have been set
	for k, v := range taskCtx.Variables {
		ctx.Variables[k] = v
	}

	return nil
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

// executeTry executes a try/catch/finally statement
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
	// Build list of all tools to check (primary + alternatives)
	toolsToCheck := []string{stmt.Target}
	toolsToCheck = append(toolsToCheck, stmt.Alternatives...)

	// Check availability for all tools
	var conditionMet bool
	var conditionText string

	if stmt.Condition == "not_available" {
		// For "not available": condition is true if ANY tool is not available (OR logic)
		conditionMet = false
		for _, tool := range toolsToCheck {
			if !detector.IsToolAvailable(tool) {
				conditionMet = true
				break
			}
		}

		if len(toolsToCheck) == 1 {
			conditionText = fmt.Sprintf("%s is not available", stmt.Target)
		} else {
			toolNames := strings.Join(toolsToCheck, ", ")
			conditionText = fmt.Sprintf("any of [%s] is not available", toolNames)
		}
	} else {
		// For "is available": condition is true if ALL tools are available (AND logic)
		conditionMet = true
		for _, tool := range toolsToCheck {
			if !detector.IsToolAvailable(tool) {
				conditionMet = false
				break
			}
		}

		if len(toolsToCheck) == 1 {
			conditionText = fmt.Sprintf("%s is available", stmt.Target)
		} else {
			toolNames := strings.Join(toolsToCheck, ", ")
			conditionText = fmt.Sprintf("all of [%s] are available", toolNames)
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would check if %s: %t\n", conditionText, conditionMet)
		if conditionMet {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute if body\n")
			for _, bodyStmt := range stmt.Body {
				if err := e.executeStatement(bodyStmt, ctx); err != nil {
					return err
				}
			}
		} else if len(stmt.ElseBody) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute else body\n")
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

func (e BreakError) Error() string {
	if e.Condition != "" {
		return "break when " + e.Condition
	}
	return "break"
}

func (e ContinueError) Error() string {
	if e.Condition != "" {
		return "continue if " + e.Condition
	}
	return "continue"
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
	return e.interpolator.Interpolate(message, ctx)
}

// interpolateVariablesWithError replaces {variable} placeholders with actual values and returns any undefined variable errors
func (e *Engine) interpolateVariablesWithError(message string, ctx *ExecutionContext) (string, error) {
	return e.interpolator.InterpolateWithError(message, ctx)
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

// evaluateEnvCondition evaluates environment variable conditionals
// Handles: "VARNAME exists", "VARNAME is "value"", "VARNAME is not empty",
// "VARNAME exists and is not empty", "VARNAME exists and is "value""
func (e *Engine) evaluateEnvCondition(condition string, ctx *ExecutionContext) bool {
	condition = strings.TrimSpace(condition)

	// Extract variable name first
	var varName string
	var rest string

	// Find the first space to separate var name from the condition
	spaceIdx := strings.Index(condition, " ")
	if spaceIdx == -1 {
		// No space, just the variable name (shouldn't happen in valid syntax)
		varName = condition
		rest = ""
	} else {
		varName = condition[:spaceIdx]
		rest = strings.TrimSpace(condition[spaceIdx+1:])
	}

	// Handle compound conditions with "and" - must come after we have the varName
	if strings.Contains(rest, " and ") {
		parts := strings.SplitN(rest, " and ", 2)
		if len(parts) == 2 {
			// Evaluate first condition with varName
			left := e.evaluateEnvConditionWithVar(varName, strings.TrimSpace(parts[0]), ctx)
			if !left {
				return false // Short-circuit if first condition fails
			}
			// Evaluate second condition with same varName
			right := e.evaluateEnvConditionWithVar(varName, strings.TrimSpace(parts[1]), ctx)
			return right
		}
	}

	// Single condition evaluation
	return e.evaluateEnvConditionWithVar(varName, rest, ctx)
}

// evaluateEnvConditionWithVar evaluates a single env condition for a given variable
func (e *Engine) evaluateEnvConditionWithVar(varName string, condition string, ctx *ExecutionContext) bool {
	condition = strings.TrimSpace(condition)

	// Get the environment variable value
	envValue, envExists := os.LookupEnv(varName)

	// Handle "exists" check
	if condition == "exists" || condition == "" {
		return envExists
	}

	// Handle "is not empty" check
	if condition == "is not empty" {
		return envExists && strings.TrimSpace(envValue) != ""
	}

	// Handle "is empty" check
	if condition == "is empty" {
		return !envExists || strings.TrimSpace(envValue) == ""
	}

	// Handle "is "value"" check
	if strings.HasPrefix(condition, "is ") && !strings.HasPrefix(condition, "is not ") {
		expectedValue := strings.TrimSpace(condition[3:])
		// Remove quotes if present
		expectedValue = strings.Trim(expectedValue, "\"'")
		return envExists && envValue == expectedValue
	}

	// Handle "is not "value"" check
	if strings.HasPrefix(condition, "is not ") && !strings.HasPrefix(condition, "is not empty") {
		expectedValue := strings.TrimSpace(condition[7:])
		// Remove quotes if present
		expectedValue = strings.Trim(expectedValue, "\"'")
		return !envExists || envValue != expectedValue
	}

	// Default: check if environment variable exists
	return envExists
}

// evaluateCondition evaluates condition expressions
func (e *Engine) evaluateCondition(condition string, ctx *ExecutionContext) bool {
	// Simple condition evaluation
	// Handle various patterns like "variable is value", "variable is not empty", etc.

	// Handle environment variable conditionals
	if strings.HasPrefix(condition, "env ") {
		return e.evaluateEnvCondition(strings.TrimPrefix(condition, "env "), ctx)
	}

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

	case *ast.ArrayLiteral:
		// Convert array literal to bracket-enclosed comma-separated string
		// This preserves the array format so loops can properly split it
		var elements []string
		for _, elem := range ex.Elements {
			val, err := e.evaluateExpression(elem, ctx)
			if err != nil {
				return "", err
			}
			elements = append(elements, val)
		}
		return "[" + strings.Join(elements, ",") + "]", nil

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
