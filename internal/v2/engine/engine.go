package engine

import (
	"fmt"
	"io"
	"os"
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

// Execute runs a v2 program
func (e *Engine) Execute(program *ast.Program, taskName string) error {
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

	return e.executeTask(targetTask)
}

// executeTask executes a single task
func (e *Engine) executeTask(task *ast.TaskStatement) error {
	if e.dryRun {
		fmt.Fprintf(e.output, "[DRY RUN] Would execute task: %s\n", task.Name)
		if task.Description != "" {
			fmt.Fprintf(e.output, "[DRY RUN] Description: %s\n", task.Description)
		}
		for _, action := range task.Body {
			fmt.Fprintf(e.output, "[DRY RUN] %s: %s\n", action.Action, action.Message)
		}
		return nil
	}

	// Execute each action in the task body
	for _, action := range task.Body {
		if err := e.executeAction(&action); err != nil {
			return fmt.Errorf("failed to execute action '%s': %v", action.Action, err)
		}
	}

	return nil
}

// executeAction executes a single action statement
func (e *Engine) executeAction(action *ast.ActionStatement) error {
	// Map actions to output with appropriate formatting and emojis
	switch action.Action {
	case "info":
		fmt.Fprintf(e.output, "ℹ️  %s\n", action.Message)
	case "step":
		fmt.Fprintf(e.output, "🚀 %s\n", action.Message)
	case "warn":
		fmt.Fprintf(e.output, "⚠️  %s\n", action.Message)
	case "error":
		fmt.Fprintf(e.output, "❌ %s\n", action.Message)
	case "success":
		fmt.Fprintf(e.output, "✅ %s\n", action.Message)
	case "fail":
		fmt.Fprintf(e.output, "💥 %s\n", action.Message)
		return fmt.Errorf("task failed: %s", action.Message)
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
