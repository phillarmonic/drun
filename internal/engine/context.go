package engine

import (
	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/engine/interpolation"
	"github.com/phillarmonic/drun/internal/types"
)

// ExecutionContext holds parameter values and other runtime context
type ExecutionContext struct {
	Parameters       map[string]*types.Value // parameter name -> typed value
	Variables        map[string]string       // captured variables from shell commands
	Project          *ProjectContext         // project-level settings and hooks
	CurrentFile      string                  // path to the current drun file being executed
	CurrentTask      string                  // name of the currently executing task
	CurrentNamespace string                  // namespace of currently executing task/template (for transitive resolution)
	Program          *ast.Program            // the AST program being executed
}

// Implement interpolation.Context interface
func (ctx *ExecutionContext) GetParameters() map[string]*types.Value {
	if ctx == nil {
		return nil
	}
	return ctx.Parameters
}

func (ctx *ExecutionContext) GetVariables() map[string]string {
	if ctx == nil {
		return nil
	}
	return ctx.Variables
}

func (ctx *ExecutionContext) GetProject() interpolation.ProjectContext {
	if ctx == nil || ctx.Project == nil {
		return nil
	}
	return ctx.Project
}

func (ctx *ExecutionContext) GetCurrentFile() string {
	if ctx == nil {
		return ""
	}
	return ctx.CurrentFile
}

func (ctx *ExecutionContext) GetCurrentTask() string {
	if ctx == nil {
		return ""
	}
	return ctx.CurrentTask
}

// ProjectContext holds project-level configuration
type ProjectContext struct {
	Name              string                                    // project name
	Version           string                                    // project version
	Settings          map[string]string                         // project settings (set key to value)
	Parameters        map[string]*ast.ProjectParameterStatement // project-level shared parameters
	Snippets          map[string]*ast.SnippetStatement          // reusable code snippets
	BeforeHooks       []ast.Statement                           // before any task hooks
	AfterHooks        []ast.Statement                           // after any task hooks
	SetupHooks        []ast.Statement                           // on drun setup hooks
	TeardownHooks     []ast.Statement                           // on drun teardown hooks
	ShellConfigs      map[string]*ast.PlatformShellConfig       // platform-specific shell configurations
	IncludedSnippets  map[string]*ast.SnippetStatement          // namespaced snippets: "docker.login-check"
	IncludedTemplates map[string]*ast.TaskTemplateStatement     // namespaced templates: "docker.build"
	IncludedTasks     map[string]*ast.TaskStatement             // namespaced tasks: "docker.deploy"
	IncludedFiles     map[string]bool                           // track included files to prevent circular includes
}

// Implement interpolation.ProjectContext interface
func (pc *ProjectContext) GetName() string {
	if pc == nil {
		return ""
	}
	return pc.Name
}

func (pc *ProjectContext) GetVersion() string {
	if pc == nil {
		return ""
	}
	return pc.Version
}

func (pc *ProjectContext) GetSettings() map[string]string {
	if pc == nil {
		return nil
	}
	return pc.Settings
}
