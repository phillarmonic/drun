package engine

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/builtins"
	"github.com/phillarmonic/drun/internal/cache"
	"github.com/phillarmonic/drun/internal/domain/parameter"
	"github.com/phillarmonic/drun/internal/domain/statement"
	"github.com/phillarmonic/drun/internal/domain/task"
	"github.com/phillarmonic/drun/internal/engine/hooks"
	"github.com/phillarmonic/drun/internal/engine/includes"
	"github.com/phillarmonic/drun/internal/engine/interpolation"
	"github.com/phillarmonic/drun/internal/errors"
	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
	"github.com/phillarmonic/drun/internal/remote"
	"github.com/phillarmonic/drun/internal/types"
)

// Engine executes drun v2 programs directly
type Engine struct {
	output       io.Writer
	dryRun       bool
	verbose      bool
	interpolator *interpolation.Interpolator

	// Domain layer services
	taskRegistry   *task.Registry
	paramValidator *parameter.Validator
	depResolver    *task.DependencyResolver

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

	// Initialize domain services
	taskReg := task.NewRegistry()

	e := &Engine{
		output:         output,
		dryRun:         false,
		interpolator:   interp,
		githubFetcher:  githubFetcher,
		httpsFetcher:   httpsFetcher,
		drunhubFetcher: drunhubFetcher,

		// Domain services
		taskRegistry:   taskReg,
		paramValidator: parameter.NewValidator(),
		depResolver:    task.NewDependencyResolver(taskReg),

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

	// Register all tasks with domain registry
	e.taskRegistry.Clear() // Clear registry for fresh execution
	if err := e.registerTasks(program.Tasks, currentFile); err != nil {
		return fmt.Errorf("task registration failed: %v", err)
	}

	// Use domain dependency resolver
	domainTasks, err := e.depResolver.Resolve(taskName)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %v", err)
	}

	// Extract task names for execution order
	executionOrder := make([]string, len(domainTasks))
	for i, t := range domainTasks {
		executionOrder[i] = t.Name
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Execution order: %v\n", executionOrder)
	}

	// Create project context
	projectCtx, err := e.createProjectContext(program.Project, currentFile)
	if err != nil {
		return fmt.Errorf("creating project context: %w", err)
	}

	// Create execution context with parameters
	ctx := &ExecutionContext{
		Parameters:  make(map[string]*types.Value, 8), // Pre-allocate for typical parameter count
		Variables:   make(map[string]string, 16),      // Pre-allocate for typical variable count
		Project:     projectCtx,
		CurrentFile: currentFile,
		Program:     program,
	}

	// Execute drun setup hooks
	if ctx.Project != nil && ctx.Project.HookManager != nil {
		for _, hook := range ctx.Project.HookManager.GetSetupHooks() {
			// Convert domain statement to AST for execution (temporary bridge)
			astHook, err := statement.ToAST(hook)
			if err != nil {
				return fmt.Errorf("converting setup hook: %w", err)
			}
			if err := e.executeStatement(astHook, ctx); err != nil {
				return fmt.Errorf("drun setup hook failed: %v", err)
			}
		}
	}

	// Execute all tasks in dependency order
	for _, currentTaskName := range executionOrder {
		// Get the domain task from the registry
		domainTask, err := e.taskRegistry.Get(currentTaskName)
		if err != nil {
			return fmt.Errorf("task '%s' not found in registry: %w", currentTaskName, err)
		}

		// Find the AST task for parameter setup (still needed temporarily)
		var astTask *ast.TaskStatement
		for _, task := range program.Tasks {
			if task.Name == currentTaskName {
				astTask = task
				break
			}
		}
		if astTask == nil {
			return fmt.Errorf("task '%s' not found in AST", currentTaskName)
		}

		// Set up parameters for this specific task
		if err := e.setupTaskParameters(astTask, params, ctx); err != nil {
			return err
		}

		// Set current task name for globals access
		ctx.CurrentTask = currentTaskName

		// Execute before hooks only for the target task
		if currentTaskName == taskName && ctx.Project != nil && ctx.Project.HookManager != nil {
			for _, hook := range ctx.Project.HookManager.GetBeforeHooks() {
				// Convert domain statement to AST for execution (temporary bridge)
				astHook, err := statement.ToAST(hook)
				if err != nil {
					return fmt.Errorf("converting before hook: %w", err)
				}
				if err := e.executeStatement(astHook, ctx); err != nil {
					return fmt.Errorf("before hook failed: %v", err)
				}
			}
		}

		// Convert domain task body to AST for execution (temporary bridge)
		astBody, err := statement.ToASTList(domainTask.Body)
		if err != nil {
			return fmt.Errorf("converting task body for '%s': %w", currentTaskName, err)
		}

		// Create a temporary AST task for execution
		tempASTTask := &ast.TaskStatement{
			Name:        domainTask.Name,
			Description: domainTask.Description,
			Body:        astBody,
		}

		// Execute the task
		if err := e.executeTask(tempASTTask, ctx); err != nil {
			return fmt.Errorf("task '%s' failed: %v", currentTaskName, err)
		}

		// Execute after hooks only for the target task
		if currentTaskName == taskName && ctx.Project != nil && ctx.Project.HookManager != nil {
			for _, hook := range ctx.Project.HookManager.GetAfterHooks() {
				// Convert domain statement to AST for execution (temporary bridge)
				astHook, hookErr := statement.ToAST(hook)
				if hookErr != nil {
					_, _ = fmt.Fprintf(e.output, "⚠️  After hook conversion failed: %v\n", hookErr)
					continue
				}
				if hookErr := e.executeStatement(astHook, ctx); hookErr != nil {
					_, _ = fmt.Fprintf(e.output, "⚠️  After hook failed: %v\n", hookErr)
				}
			}
		}
	}

	// Execute drun teardown hooks
	if ctx.Project != nil && ctx.Project.HookManager != nil {
		for _, hook := range ctx.Project.HookManager.GetTeardownHooks() {
			// Convert domain statement to AST for execution (temporary bridge)
			astHook, hookErr := statement.ToAST(hook)
			if hookErr != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Drun teardown hook conversion failed: %v\n", hookErr)
				continue
			}
			if hookErr := e.executeStatement(astHook, ctx); hookErr != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Drun teardown hook failed: %v\n", hookErr)
			}
		}
	}

	return nil
}

// registerTasks registers all tasks from the program into the domain registry
func (e *Engine) registerTasks(tasks []*ast.TaskStatement, currentFile string) error {
	for _, astTask := range tasks {
		domainTask, err := task.NewTask(astTask, "", currentFile)
		if err != nil {
			return fmt.Errorf("converting task %s: %w", astTask.Name, err)
		}
		if err := e.taskRegistry.Register(domainTask); err != nil {
			return err
		}
	}
	return nil
}

// setupTaskParameters sets up parameters for a specific task
func (e *Engine) setupTaskParameters(task *ast.TaskStatement, params map[string]string, ctx *ExecutionContext) error {
	// First, add included/namespaced parameters from includes (e.g., docker.registry)
	if ctx.Project != nil && ctx.Project.IncludedParams != nil {
		for namespacedName, projectParam := range ctx.Project.IncludedParams {
			// Use the namespaced name as the key (e.g., "docker.registry")
			// Check if user provided a value via CLI (they won't, but check anyway)
			var rawValue string
			var hasValue bool

			// Included params won't be provided via CLI, so just use defaults
			if projectParam.HasDefault {
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
					return errors.NewParameterValidationError(fmt.Sprintf("included parameter '%s': invalid %s value '%s': %v",
						namespacedName, paramType, rawValue, err))
				}

				// Store with namespaced key so it can be accessed as $params.docker.registry
				ctx.Parameters[namespacedName] = typedValue
			}
		}
	}

	// Then, add project-level parameters if they exist
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

				// Convert AST project parameter to domain parameter and validate using domain validator
				domainParam := &parameter.Parameter{
					Name:         paramName,
					Type:         "given", // project parameters are "given" type
					DefaultValue: projectParam.DefaultValue,
					HasDefault:   projectParam.HasDefault,
					Required:     false, // project parameters always have defaults
					DataType:     projectParam.DataType,
					Constraints:  projectParam.Constraints,
					MinValue:     projectParam.MinValue,
					MaxValue:     projectParam.MaxValue,
					Pattern:      projectParam.Pattern,
					PatternMacro: projectParam.PatternMacro,
					EmailFormat:  projectParam.EmailFormat,
				}

				// Use domain validator
				if err := e.paramValidator.Validate(domainParam, typedValue); err != nil {
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

			// Convert AST parameter to domain parameter and validate using domain validator
			domainParam := &parameter.Parameter{
				Name:         param.Name,
				Type:         param.Type,
				DefaultValue: param.DefaultValue,
				HasDefault:   param.HasDefault,
				Required:     param.Required,
				DataType:     param.DataType,
				Constraints:  param.Constraints,
				MinValue:     param.MinValue,
				MaxValue:     param.MaxValue,
				Pattern:      param.Pattern,
				PatternMacro: param.PatternMacro,
				EmailFormat:  param.EmailFormat,
				Variadic:     param.Variadic,
			}

			// Use domain validator
			if err := e.paramValidator.Validate(domainParam, typedValue); err != nil {
				return errors.NewParameterValidationError(fmt.Sprintf("parameter '%s': %v", param.Name, err))
			}

			ctx.Parameters[param.Name] = typedValue
		}
	}

	return nil
}

// createProjectContext creates a project context from the project statement
func (e *Engine) createProjectContext(project *ast.ProjectStatement, currentFile string) (*ProjectContext, error) {
	if project == nil {
		return nil, nil
	}

	ctx := &ProjectContext{
		Name:              project.Name,
		Version:           project.Version,
		Settings:          make(map[string]string, 8),                          // Pre-allocate for typical settings count
		Parameters:        make(map[string]*ast.ProjectParameterStatement, 8),  // Pre-allocate for project parameters
		Snippets:          make(map[string]*ast.SnippetStatement, 8),           // Pre-allocate for snippets
		HookManager:       hooks.NewManager(),                                  // Initialize hooks manager
		ShellConfigs:      make(map[string]*ast.PlatformShellConfig, 4),        // Pre-allocate for typical platform count
		IncludedSnippets:  make(map[string]*ast.SnippetStatement, 16),          // Pre-allocate for included snippets
		IncludedTemplates: make(map[string]*ast.TaskTemplateStatement, 16),     // Pre-allocate for included templates
		IncludedTasks:     make(map[string]*ast.TaskStatement, 16),             // Pre-allocate for included tasks
		IncludedSettings:  make(map[string]string, 16),                         // Pre-allocate for included settings
		IncludedParams:    make(map[string]*ast.ProjectParameterStatement, 16), // Pre-allocate for included parameters
		IncludedFiles:     make(map[string]bool, 4),                            // Pre-allocate for included files
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
			// Convert AST statements to domain statements
			domainBody, err := statement.FromASTList(s.Body)
			if err != nil {
				return nil, fmt.Errorf("converting %s hook body: %w", s.Type, err)
			}

			switch s.Type {
			case "before":
				ctx.HookManager.RegisterBeforeHooks(domainBody)
			case "after":
				ctx.HookManager.RegisterAfterHooks(domainBody)
			case "setup":
				ctx.HookManager.RegisterSetupHooks(domainBody)
			case "teardown":
				ctx.HookManager.RegisterTeardownHooks(domainBody)
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

	return ctx, nil
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

	var snippetNamespace string // Track if snippet is from an included project

	// Check if it's a namespaced reference (contains dot)
	if strings.Contains(useStmt.SnippetName, ".") {
		// Look in included snippets
		snippet, exists = ctx.Project.IncludedSnippets[useStmt.SnippetName]
		// Extract namespace (everything before the last dot)
		if exists {
			parts := strings.Split(useStmt.SnippetName, ".")
			if len(parts) > 1 {
				snippetNamespace = parts[0]
			}
		}
	} else {
		// Try current namespace first (for transitive resolution)
		if ctx.CurrentNamespace != "" {
			namespacedName := ctx.CurrentNamespace + "." + useStmt.SnippetName
			snippet, exists = ctx.Project.IncludedSnippets[namespacedName]
			if exists {
				snippetNamespace = ctx.CurrentNamespace
			}
		}

		// If not found in namespace, look in local snippets
		if !exists {
			snippet, exists = ctx.Project.Snippets[useStmt.SnippetName]
		}
	}

	if !exists {
		return fmt.Errorf("snippet '%s' not found", useStmt.SnippetName)
	}

	// Save the current namespace and set new one if snippet is from included project
	oldNamespace := ctx.CurrentNamespace
	if snippetNamespace != "" {
		ctx.CurrentNamespace = snippetNamespace
	}
	defer func() {
		ctx.CurrentNamespace = oldNamespace
	}()

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

			// Convert AST parameter to domain parameter and validate using domain validator
			domainParam := &parameter.Parameter{
				Name:         param.Name,
				Type:         param.Type,
				DefaultValue: param.DefaultValue,
				HasDefault:   param.HasDefault,
				Required:     param.Required,
				DataType:     param.DataType,
				Constraints:  param.Constraints,
				MinValue:     param.MinValue,
				MaxValue:     param.MaxValue,
				Pattern:      param.Pattern,
				PatternMacro: param.PatternMacro,
				EmailFormat:  param.EmailFormat,
				Variadic:     param.Variadic,
			}

			// Use domain validator
			if err := e.paramValidator.Validate(domainParam, typedValue); err != nil {
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
	// Register tasks with domain registry for listing
	e.taskRegistry.Clear()
	_ = e.registerTasks(program.Tasks, "")

	// Get tasks from domain registry
	domainTasks := e.taskRegistry.List()

	tasks := make([]TaskInfo, 0, len(domainTasks))
	for _, domainTask := range domainTasks {
		info := TaskInfo{
			Name:        domainTask.Name,
			Description: domainTask.Description,
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
