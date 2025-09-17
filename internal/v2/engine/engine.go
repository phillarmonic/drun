package engine

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/internal/v2/ast"
	"github.com/phillarmonic/drun/internal/v2/builtins"
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
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute task: %s\n", task.Name)
		if task.Description != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Description: %s\n", task.Description)
		}
		for _, stmt := range task.Body {
			if action, ok := stmt.(*ast.ActionStatement); ok {
				interpolatedMessage := e.interpolateVariables(action.Message, ctx)
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] %s: %s\n", action.Action, interpolatedMessage)
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
					if result, err := builtins.CallBuiltin(funcName, paramValue); err == nil {
						return result
					}
				}
			}
		}
	}

	// 4. Check for simple parameter lookup
	if ctx != nil {
		if value, exists := ctx.Parameters[expr]; exists {
			return value
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
	// For now, we'll implement a simple mock loop
	// In a real implementation, we'd need to:
	// 1. Resolve the iterable (could be a parameter, file list, etc.)
	// 2. Iterate over each item
	// 3. Set the loop variable in a new context
	// 4. Execute the body for each iteration

	// Mock implementation: assume iterable is a parameter containing comma-separated values
	iterableValue, exists := ctx.Parameters[stmt.Iterable]
	if !exists {
		return fmt.Errorf("iterable '%s' not found in parameters", stmt.Iterable)
	}

	// Split by comma to get items (simple implementation)
	items := strings.Split(iterableValue, ",")

	for _, item := range items {
		// Create a new context with the loop variable
		loopCtx := &ExecutionContext{
			Parameters: make(map[string]string),
		}

		// Copy existing parameters
		for k, v := range ctx.Parameters {
			loopCtx.Parameters[k] = v
		}

		// Set the loop variable
		loopCtx.Parameters[stmt.Variable] = strings.TrimSpace(item)

		// Execute the loop body
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, loopCtx); err != nil {
				return err
			}
		}
	}

	return nil
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
				return value == right
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
