package task

import (
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/statement"
)

// Task represents a domain task entity
type Task struct {
	Name         string
	Description  string
	Parameters   []Parameter
	Dependencies []Dependency
	Body         []statement.Statement
	Namespace    string
	Source       string // File where task is defined
}

// NewTask creates a new task from AST
func NewTask(stmt *ast.TaskStatement, namespace, source string) (*Task, error) {
	// Convert body statements from AST to domain
	body, err := statement.FromASTList(stmt.Body)
	if err != nil {
		return nil, fmt.Errorf("converting task body: %w", err)
	}

	task := &Task{
		Name:        stmt.Name,
		Description: stmt.Description,
		Namespace:   namespace,
		Source:      source,
		Body:        body,
	}

	// Convert parameters
	for _, param := range stmt.Parameters {
		task.Parameters = append(task.Parameters, NewParameter(&param))
	}

	// Convert dependencies
	for _, depGroup := range stmt.Dependencies {
		for _, depItem := range depGroup.Dependencies {
			task.Dependencies = append(task.Dependencies, Dependency{
				Name:       depItem.Name,
				Parallel:   depItem.Parallel,
				Sequential: depGroup.Sequential,
			})
		}
	}

	return task, nil
}

// FullName returns the fully qualified task name (with namespace)
func (t *Task) FullName() string {
	if t.Namespace == "" {
		return t.Name
	}
	return t.Namespace + "." + t.Name
}

// HasParameter checks if task has a parameter
func (t *Task) HasParameter(name string) bool {
	for _, param := range t.Parameters {
		if param.Name == name {
			return true
		}
	}
	return false
}

// GetParameter gets a parameter by name
func (t *Task) GetParameter(name string) (*Parameter, bool) {
	for i := range t.Parameters {
		if t.Parameters[i].Name == name {
			return &t.Parameters[i], true
		}
	}
	return nil, false
}

// HasDependencies checks if task has dependencies
func (t *Task) HasDependencies() bool {
	return len(t.Dependencies) > 0
}

// Validate validates the task
func (t *Task) Validate() error {
	if t.Name == "" {
		return &TaskError{
			Task:    t.Name,
			Message: "task name cannot be empty",
		}
	}

	// Validate parameters
	for _, param := range t.Parameters {
		if err := param.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Parameter represents a task parameter
type Parameter struct {
	Name         string
	Type         string // "requires", "given", "accepts"
	DefaultValue string
	HasDefault   bool
	Required     bool
	DataType     string
	Constraints  []string
	MinValue     *float64
	MaxValue     *float64
	Pattern      string
	PatternMacro string
	EmailFormat  bool
	Variadic     bool
}

// NewParameter creates a parameter from AST
func NewParameter(stmt *ast.ParameterStatement) Parameter {
	return Parameter{
		Name:         stmt.Name,
		Type:         stmt.Type,
		DefaultValue: stmt.DefaultValue,
		HasDefault:   stmt.HasDefault,
		Required:     stmt.Required,
		DataType:     stmt.DataType,
		Constraints:  stmt.Constraints,
		MinValue:     stmt.MinValue,
		MaxValue:     stmt.MaxValue,
		Pattern:      stmt.Pattern,
		PatternMacro: stmt.PatternMacro,
		EmailFormat:  stmt.EmailFormat,
		Variadic:     stmt.Variadic,
	}
}

// Validate validates the parameter
func (p *Parameter) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("parameter name cannot be empty")
	}
	return nil
}

// Dependency represents a task dependency
type Dependency struct {
	Name       string
	Parallel   bool
	Sequential bool
}

// TaskError represents a task-related error
type TaskError struct {
	Task    string
	Message string
	Cause   error
}

func (e *TaskError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("task '%s': %s: %v", e.Task, e.Message, e.Cause)
	}
	return fmt.Sprintf("task '%s': %s", e.Task, e.Message)
}

func (e *TaskError) Unwrap() error {
	return e.Cause
}
