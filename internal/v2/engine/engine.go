package engine

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/lexer"
	"github.com/phillarmonic/drun/internal/v2/parser"
)

// Engine executes drun v2 programs directly
type Engine struct {
	output io.Writer
	dryRun bool
}

// ExecutionContext holds parameter values and other runtime context
type ExecutionContext struct {
	Parameters map[string]string // parameter name -> value
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

	// Find the requested task
	var targetTask *ast.TaskStatement
	for _, task := range program.Tasks {
		if task.Name == taskName {
			targetTask = task
			break
		}
	}

	if targetTask == nil {
		return fmt.Errorf("task '%s' not found", taskName)
	}

	// Create execution context with parameters
	ctx := &ExecutionContext{
		Parameters: make(map[string]string),
	}

	// Set up parameters with defaults and validation
	for _, param := range targetTask.Parameters {
		var value string
		var hasValue bool

		if providedValue, exists := params[param.Name]; exists {
			value = providedValue
			hasValue = true
		} else if param.DefaultValue != "" {
			value = param.DefaultValue
			hasValue = true
		} else if param.Required {
			return fmt.Errorf("required parameter '%s' not provided", param.Name)
		}

		// Validate constraints if value is provided
		if hasValue {
			if err := e.validateParameterConstraints(param, value); err != nil {
				return err
			}
			ctx.Parameters[param.Name] = value
		}
	}

	return e.executeTask(targetTask, ctx)
}

// executeTask executes a single task with the given context
func (e *Engine) executeTask(task *ast.TaskStatement, ctx *ExecutionContext) error {
	if e.dryRun {
		fmt.Fprintf(e.output, "[DRY RUN] Would execute task: %s\n", task.Name)
		if task.Description != "" {
			fmt.Fprintf(e.output, "[DRY RUN] Description: %s\n", task.Description)
		}
		for _, stmt := range task.Body {
			if action, ok := stmt.(*ast.ActionStatement); ok {
				interpolatedMessage := e.interpolateVariables(action.Message, ctx)
				fmt.Fprintf(e.output, "[DRY RUN] %s: %s\n", action.Action, interpolatedMessage)
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
	case *ast.ParameterStatement:
		// Parameters are handled during task setup, not execution
		return nil
	case *ast.ConditionalStatement:
		// TODO: Implement conditional execution
		return fmt.Errorf("conditional statements not yet implemented")
	case *ast.LoopStatement:
		// TODO: Implement loop execution
		return fmt.Errorf("loop statements not yet implemented")
	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// executeAction executes a single action statement
func (e *Engine) executeAction(action *ast.ActionStatement, ctx *ExecutionContext) error {
	// Interpolate variables in the message
	interpolatedMessage := e.interpolateVariables(action.Message, ctx)

	// Map actions to output with appropriate formatting and emojis
	switch action.Action {
	case "info":
		fmt.Fprintf(e.output, "ℹ️  %s\n", interpolatedMessage)
	case "step":
		fmt.Fprintf(e.output, "🚀 %s\n", interpolatedMessage)
	case "warn":
		fmt.Fprintf(e.output, "⚠️  %s\n", interpolatedMessage)
	case "error":
		fmt.Fprintf(e.output, "❌ %s\n", interpolatedMessage)
	case "success":
		fmt.Fprintf(e.output, "✅ %s\n", interpolatedMessage)
	case "fail":
		fmt.Fprintf(e.output, "💥 %s\n", interpolatedMessage)
		return fmt.Errorf("task failed: %s", interpolatedMessage)
	default:
		return fmt.Errorf("unknown action: %s", action.Action)
	}

	return nil
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
	if ctx == nil || len(ctx.Parameters) == 0 {
		return message
	}

	// Use regex to find {variable} patterns
	re := regexp.MustCompile(`\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(message, func(match string) string {
		// Extract variable name (remove { and })
		varName := match[1 : len(match)-1]

		// Look up the variable value
		if value, exists := ctx.Parameters[varName]; exists {
			return value
		}

		// If variable not found, return the original placeholder
		return match
	})
}

// validateParameterConstraints validates parameter values against their constraints
func (e *Engine) validateParameterConstraints(param ast.ParameterStatement, value string) error {
	// Check constraints (e.g., from ["dev", "staging", "production"])
	if len(param.Constraints) > 0 {
		for _, constraint := range param.Constraints {
			if value == constraint {
				return nil // Value is valid
			}
		}
		return fmt.Errorf("parameter '%s' value '%s' is not valid. Must be one of: %v",
			param.Name, value, param.Constraints)
	}

	// TODO: Add more validation types (data type validation, regex patterns, etc.)

	return nil
}
