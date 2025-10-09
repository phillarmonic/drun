package planner

import (
	"encoding/json"
	"fmt"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/statement"
	"github.com/phillarmonic/drun/internal/domain/task"
)

// HookPlan represents hooks to execute at various lifecycle points
type HookPlan struct {
	SetupHooks    []statement.Statement
	TeardownHooks []statement.Statement
	BeforeHooks   []statement.Statement
	AfterHooks    []statement.Statement
}

// TaskPlan represents a single task in the execution plan
type TaskPlan struct {
	Name        string
	Description string
	Namespace   string
	Source      string
	Parameters  []task.Parameter
	Body        []statement.Statement
}

// ExecutionPlan represents a complete, deterministic execution plan
// All dependencies, hooks, and includes are resolved upfront
type ExecutionPlan struct {
	// Target task being executed
	TargetTask string

	// Ordered list of tasks to execute (includes dependencies)
	ExecutionOrder []string

	// Fully resolved task plans (domain-level, no AST)
	Tasks map[string]*TaskPlan

	// Lifecycle hooks resolved from project
	Hooks *HookPlan

	// Project metadata
	ProjectName    string
	ProjectVersion string

	// Namespace tracking for includes
	Namespaces map[string]bool // Set of namespaces used
}

// Planner orchestrates dependency resolution and produces execution plans
type Planner struct {
	taskRegistry *task.Registry
	depResolver  *task.DependencyResolver
}

// NewPlanner creates a new execution planner
func NewPlanner(taskRegistry *task.Registry, depResolver *task.DependencyResolver) *Planner {
	return &Planner{
		taskRegistry: taskRegistry,
		depResolver:  depResolver,
	}
}

// ProjectContext provides project-level information for planning
type ProjectContext struct {
	Name          string
	Version       string
	SetupHooks    []statement.Statement
	TeardownHooks []statement.Statement
	BeforeHooks   []statement.Statement
	AfterHooks    []statement.Statement
}

// Plan creates a comprehensive execution plan for the given task
func (p *Planner) Plan(taskName string, program *ast.Program, projectCtx *ProjectContext) (*ExecutionPlan, error) {
	// Resolve dependencies using domain resolver
	domainTasks, err := p.depResolver.Resolve(taskName)
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Build execution order
	executionOrder := make([]string, len(domainTasks))
	taskPlans := make(map[string]*TaskPlan)
	namespaces := make(map[string]bool)

	for i, domainTask := range domainTasks {
		executionOrder[i] = domainTask.Name

		// Create TaskPlan from domain task
		taskPlans[domainTask.Name] = &TaskPlan{
			Name:        domainTask.Name,
			Description: domainTask.Description,
			Namespace:   domainTask.Namespace,
			Source:      domainTask.Source,
			Parameters:  domainTask.Parameters,
			Body:        domainTask.Body,
		}

		// Track namespaces
		if domainTask.Namespace != "" {
			namespaces[domainTask.Namespace] = true
		}
	}

	// Build hook plan from project context
	var hookPlan *HookPlan
	if projectCtx != nil {
		hookPlan = &HookPlan{
			SetupHooks:    projectCtx.SetupHooks,
			TeardownHooks: projectCtx.TeardownHooks,
			BeforeHooks:   projectCtx.BeforeHooks,
			AfterHooks:    projectCtx.AfterHooks,
		}
	}

	plan := &ExecutionPlan{
		TargetTask:     taskName,
		ExecutionOrder: executionOrder,
		Tasks:          taskPlans,
		Hooks:          hookPlan,
		Namespaces:     namespaces,
	}

	// Set project metadata if available
	if projectCtx != nil {
		plan.ProjectName = projectCtx.Name
		plan.ProjectVersion = projectCtx.Version
	}

	return plan, nil
}

// GetTask retrieves a task plan from the execution plan
func (ep *ExecutionPlan) GetTask(name string) (*TaskPlan, error) {
	t, ok := ep.Tasks[name]
	if !ok {
		return nil, fmt.Errorf("task '%s' not found in execution plan", name)
	}
	return t, nil
}

// ToJSON serializes the execution plan to JSON (for debugging/visualization)
func (ep *ExecutionPlan) ToJSON() (string, error) {
	// Create a serializable version (excluding AST tasks)
	type SerializablePlan struct {
		TargetTask     string
		ExecutionOrder []string
		ProjectName    string
		ProjectVersion string
		Namespaces     []string
		TaskCount      int
		HasHooks       bool
	}

	namespaceList := make([]string, 0, len(ep.Namespaces))
	for ns := range ep.Namespaces {
		namespaceList = append(namespaceList, ns)
	}

	plan := SerializablePlan{
		TargetTask:     ep.TargetTask,
		ExecutionOrder: ep.ExecutionOrder,
		ProjectName:    ep.ProjectName,
		ProjectVersion: ep.ProjectVersion,
		Namespaces:     namespaceList,
		TaskCount:      len(ep.Tasks),
		HasHooks:       ep.Hooks != nil,
	}

	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize plan: %w", err)
	}

	return string(data), nil
}

// Helper methods for debug utilities

// GetTargetTask returns the target task name
func (ep *ExecutionPlan) GetTargetTask() string {
	return ep.TargetTask
}

// GetExecutionOrder returns the execution order
func (ep *ExecutionPlan) GetExecutionOrder() []string {
	return ep.ExecutionOrder
}

// GetTaskCount returns the number of tasks in the plan
func (ep *ExecutionPlan) GetTaskCount() int {
	return len(ep.Tasks)
}

// GetProjectName returns the project name
func (ep *ExecutionPlan) GetProjectName() string {
	return ep.ProjectName
}

// GetProjectVersion returns the project version
func (ep *ExecutionPlan) GetProjectVersion() string {
	return ep.ProjectVersion
}

// GetNamespaces returns a list of namespaces used in the plan
func (ep *ExecutionPlan) GetNamespaces() []string {
	namespaces := make([]string, 0, len(ep.Namespaces))
	for ns := range ep.Namespaces {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}
