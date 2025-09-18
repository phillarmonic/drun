package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/v2/lexer"
)

// Node represents any node in the AST
type Node interface {
	String() string
}

// Statement represents any statement node
type Statement interface {
	Node
	statementNode()
}

// Program represents the root of the AST
type Program struct {
	Version *VersionStatement
	Project *ProjectStatement
	Tasks   []*TaskStatement
}

func (p *Program) String() string {
	var out strings.Builder
	if p.Version != nil {
		out.WriteString(p.Version.String())
		out.WriteString("\n")
	}
	if p.Project != nil {
		out.WriteString(p.Project.String())
		out.WriteString("\n")
	}
	for _, task := range p.Tasks {
		out.WriteString(task.String())
		out.WriteString("\n")
	}
	return out.String()
}

// VersionStatement represents a version declaration
type VersionStatement struct {
	Token lexer.Token // the VERSION token
	Value string      // the version number (e.g., "2.0")
}

func (vs *VersionStatement) statementNode() {}
func (vs *VersionStatement) String() string {
	return fmt.Sprintf("version: %s", vs.Value)
}

// ProjectStatement represents a project declaration
type ProjectStatement struct {
	Token    lexer.Token      // the PROJECT token
	Name     string           // project name
	Version  string           // optional project version
	Settings []ProjectSetting // project settings (set, include, hooks)
}

func (ps *ProjectStatement) statementNode() {}
func (ps *ProjectStatement) String() string {
	var out strings.Builder
	out.WriteString("project ")
	out.WriteString(ps.Name)
	if ps.Version != "" {
		out.WriteString(" version ")
		out.WriteString(ps.Version)
	}
	out.WriteString(":")
	for _, setting := range ps.Settings {
		out.WriteString("\n  ")
		out.WriteString(setting.String())
	}
	return out.String()
}

// ProjectSetting represents a project-level setting
type ProjectSetting interface {
	Node
	projectSettingNode()
}

// SetStatement represents a project setting (set key to value)
type SetStatement struct {
	Token lexer.Token // the SET token
	Key   string      // setting key
	Value string      // setting value
}

func (ss *SetStatement) statementNode()      {}
func (ss *SetStatement) projectSettingNode() {}
func (ss *SetStatement) String() string {
	return fmt.Sprintf("set %s to %s", ss.Key, ss.Value)
}

// IncludeStatement represents an include directive
type IncludeStatement struct {
	Token lexer.Token // the INCLUDE token
	Path  string      // path to include
}

func (is *IncludeStatement) statementNode()      {}
func (is *IncludeStatement) projectSettingNode() {}
func (is *IncludeStatement) String() string {
	return fmt.Sprintf("include %s", is.Path)
}

// LifecycleHook represents before/after hooks
type LifecycleHook struct {
	Token lexer.Token // the BEFORE or AFTER token
	Type  string      // "before" or "after"
	Scope string      // "any" for global hooks
	Body  []Statement // hook body statements
}

func (lh *LifecycleHook) statementNode()      {}
func (lh *LifecycleHook) projectSettingNode() {}
func (lh *LifecycleHook) String() string {
	var out strings.Builder
	out.WriteString(lh.Type)
	out.WriteString(" ")
	out.WriteString(lh.Scope)
	out.WriteString(" task:")
	for _, stmt := range lh.Body {
		out.WriteString("\n    ")
		out.WriteString(stmt.String())
	}
	return out.String()
}

// TaskStatement represents a task definition
type TaskStatement struct {
	Token        lexer.Token          // the TASK token
	Name         string               // task name
	Description  string               // optional description after "means"
	Parameters   []ParameterStatement // parameter declarations
	Dependencies []DependencyGroup    // dependency declarations
	Body         []Statement          // statements in the task body (actions, conditionals, loops)
}

func (ts *TaskStatement) statementNode() {}
func (ts *TaskStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("task \"%s\"", ts.Name))
	if ts.Description != "" {
		out.WriteString(fmt.Sprintf(" means \"%s\"", ts.Description))
	}
	out.WriteString(":\n")

	// Add dependencies
	for _, dep := range ts.Dependencies {
		out.WriteString(fmt.Sprintf("  %s\n", dep.String()))
	}

	// Add parameters
	for _, param := range ts.Parameters {
		out.WriteString(fmt.Sprintf("  %s\n", param.String()))
	}

	// Add body statements
	for _, stmt := range ts.Body {
		out.WriteString(fmt.Sprintf("  %s\n", stmt.String()))
	}
	return out.String()
}

// ActionStatement represents an action call (info, step, success, etc.)
type ActionStatement struct {
	Token   lexer.Token // the action token (INFO, STEP, SUCCESS, etc.)
	Action  string      // action name (info, step, success, etc.)
	Message string      // the message string
}

func (as *ActionStatement) statementNode() {}
func (as *ActionStatement) String() string {
	return fmt.Sprintf("%s \"%s\"", as.Action, as.Message)
}

// ShellStatement represents shell command execution
type ShellStatement struct {
	Token        lexer.Token // the shell token (RUN, EXEC, SHELL, CAPTURE)
	Action       string      // "run", "exec", "shell", "capture"
	Command      string      // the shell command to execute
	CaptureVar   string      // variable name to capture output (for capture action)
	StreamOutput bool        // whether to stream output in real-time
}

func (ss *ShellStatement) statementNode() {}
func (ss *ShellStatement) String() string {
	if ss.CaptureVar != "" {
		return fmt.Sprintf("%s \"%s\" as %s", ss.Action, ss.Command, ss.CaptureVar)
	}
	return fmt.Sprintf("%s \"%s\"", ss.Action, ss.Command)
}

// FileStatement represents file system operations
type FileStatement struct {
	Token      lexer.Token // the file operation token (CREATE, COPY, MOVE, DELETE, READ, WRITE, APPEND)
	Action     string      // "create", "copy", "move", "delete", "read", "write", "append"
	Target     string      // target file/directory path
	Source     string      // source path (for copy/move operations)
	Content    string      // content (for write/append operations)
	IsDir      bool        // whether the operation is on a directory
	CaptureVar string      // variable name to capture content (for read operations)
}

func (fs *FileStatement) statementNode() {}
func (fs *FileStatement) String() string {
	switch fs.Action {
	case "create":
		if fs.IsDir {
			return fmt.Sprintf("create dir \"%s\"", fs.Target)
		}
		return fmt.Sprintf("create file \"%s\"", fs.Target)
	case "copy":
		return fmt.Sprintf("copy \"%s\" to \"%s\"", fs.Source, fs.Target)
	case "move":
		return fmt.Sprintf("move \"%s\" to \"%s\"", fs.Source, fs.Target)
	case "delete":
		if fs.IsDir {
			return fmt.Sprintf("delete dir \"%s\"", fs.Target)
		}
		return fmt.Sprintf("delete file \"%s\"", fs.Target)
	case "read":
		if fs.CaptureVar != "" {
			return fmt.Sprintf("read file \"%s\" as %s", fs.Target, fs.CaptureVar)
		}
		return fmt.Sprintf("read file \"%s\"", fs.Target)
	case "write":
		return fmt.Sprintf("write \"%s\" to file \"%s\"", fs.Content, fs.Target)
	case "append":
		return fmt.Sprintf("append \"%s\" to file \"%s\"", fs.Content, fs.Target)
	default:
		return fmt.Sprintf("%s \"%s\"", fs.Action, fs.Target)
	}
}

// TryStatement represents try/catch/finally error handling blocks
type TryStatement struct {
	Token        lexer.Token   // the TRY token
	TryBody      []Statement   // statements in the try block
	CatchClauses []CatchClause // catch clauses
	FinallyBody  []Statement   // statements in the finally block (optional)
}

func (ts *TryStatement) statementNode() {}
func (ts *TryStatement) String() string {
	var out strings.Builder
	out.WriteString("try:")

	for _, stmt := range ts.TryBody {
		out.WriteString("\n  ")
		out.WriteString(stmt.String())
	}

	for _, catch := range ts.CatchClauses {
		out.WriteString("\n")
		out.WriteString(catch.String())
	}

	if len(ts.FinallyBody) > 0 {
		out.WriteString("\nfinally:")
		for _, stmt := range ts.FinallyBody {
			out.WriteString("\n  ")
			out.WriteString(stmt.String())
		}
	}

	return out.String()
}

// CatchClause represents a catch clause within a try statement
type CatchClause struct {
	Token     lexer.Token // the CATCH token
	ErrorType string      // specific error type to catch (optional)
	ErrorVar  string      // variable to capture error (optional)
	Body      []Statement // statements in the catch block
}

func (cc *CatchClause) String() string {
	var out strings.Builder
	out.WriteString("catch")

	if cc.ErrorType != "" {
		out.WriteString(" ")
		out.WriteString(cc.ErrorType)
	}

	if cc.ErrorVar != "" {
		out.WriteString(" as ")
		out.WriteString(cc.ErrorVar)
	}

	out.WriteString(":")

	for _, stmt := range cc.Body {
		out.WriteString("\n  ")
		out.WriteString(stmt.String())
	}

	return out.String()
}

// ThrowStatement represents throw and rethrow statements
type ThrowStatement struct {
	Token   lexer.Token // the THROW or RETHROW token
	Action  string      // "throw", "rethrow", or "ignore"
	Message string      // error message (for throw)
}

func (ts *ThrowStatement) statementNode() {}
func (ts *ThrowStatement) String() string {
	switch ts.Action {
	case "throw":
		return fmt.Sprintf("throw \"%s\"", ts.Message)
	case "rethrow":
		return "rethrow"
	case "ignore":
		return "ignore"
	default:
		return ts.Action
	}
}

// DependencyGroup represents a group of dependencies with execution semantics
type DependencyGroup struct {
	Token        lexer.Token      // the DEPENDS token
	Dependencies []DependencyItem // list of dependencies in this group
	Sequential   bool             // true for "and" dependencies, false for "," dependencies
}

func (dg *DependencyGroup) statementNode() {}
func (dg *DependencyGroup) String() string {
	var out strings.Builder
	out.WriteString("depends on ")

	for i, dep := range dg.Dependencies {
		if i > 0 {
			if dg.Sequential {
				out.WriteString(" and ")
			} else {
				out.WriteString(", ")
			}
		}
		out.WriteString(dep.String())
	}

	return out.String()
}

// DependencyItem represents a single dependency
type DependencyItem struct {
	Name     string // task name
	Parallel bool   // whether this dependency can run in parallel
}

func (di *DependencyItem) String() string {
	if di.Parallel {
		return di.Name + " in parallel"
	}
	return di.Name
}

// DockerStatement represents Docker operations
type DockerStatement struct {
	Token     lexer.Token       // the DOCKER token
	Operation string            // "build", "push", "pull", "run", "stop", etc.
	Resource  string            // "image", "container", "compose"
	Name      string            // image/container name
	Options   map[string]string // additional options (from, to, as, etc.)
}

func (ds *DockerStatement) statementNode() {}
func (ds *DockerStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("docker %s %s", ds.Operation, ds.Resource))
	if ds.Name != "" {
		out.WriteString(fmt.Sprintf(" \"%s\"", ds.Name))
	}

	// Add options
	for key, value := range ds.Options {
		out.WriteString(fmt.Sprintf(" %s \"%s\"", key, value))
	}

	return out.String()
}

// GitStatement represents Git operations
type GitStatement struct {
	Token     lexer.Token       // the GIT token
	Operation string            // "clone", "init", "add", "commit", "push", "pull", etc.
	Resource  string            // "repository", "branch", "files", "changes", "tag", etc.
	Name      string            // repository URL, branch name, file pattern, etc.
	Options   map[string]string // additional options (to, from, with, into, etc.)
}

func (gs *GitStatement) statementNode() {}
func (gs *GitStatement) String() string {
	var out strings.Builder
	out.WriteString("git " + gs.Operation)

	if gs.Resource != "" {
		out.WriteString(" " + gs.Resource)
	}

	if gs.Name != "" {
		out.WriteString(fmt.Sprintf(" \"%s\"", gs.Name))
	}

	// Add options
	for key, value := range gs.Options {
		out.WriteString(fmt.Sprintf(" %s \"%s\"", key, value))
	}

	return out.String()
}

// HTTPStatement represents HTTP operations
type HTTPStatement struct {
	Token   lexer.Token       // the HTTP token or method token
	Method  string            // "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"
	URL     string            // the target URL
	Body    string            // request body (for POST, PUT, PATCH)
	Headers map[string]string // HTTP headers
	Auth    map[string]string // authentication options
	Options map[string]string // additional options (timeout, retry, etc.)
}

func (hs *HTTPStatement) statementNode() {}
func (hs *HTTPStatement) String() string {
	var out strings.Builder
	out.WriteString(strings.ToLower(hs.Method) + " request")

	if hs.URL != "" {
		out.WriteString(fmt.Sprintf(" to \"%s\"", hs.URL))
	}

	// Add headers
	for key, value := range hs.Headers {
		out.WriteString(fmt.Sprintf(" with header \"%s: %s\"", key, value))
	}

	// Add body
	if hs.Body != "" {
		out.WriteString(fmt.Sprintf(" with body \"%s\"", hs.Body))
	}

	// Add auth
	for key, value := range hs.Auth {
		out.WriteString(fmt.Sprintf(" with %s \"%s\"", key, value))
	}

	// Add options
	for key, value := range hs.Options {
		out.WriteString(fmt.Sprintf(" %s \"%s\"", key, value))
	}

	return out.String()
}

// DetectionStatement represents smart detection operations
type DetectionStatement struct {
	Token     lexer.Token // the DETECT token or condition token
	Type      string      // "detect", "if_available", "when_environment", etc.
	Target    string      // what to detect: "docker", "node", "ci", etc.
	Condition string      // condition type: "available", "version", ">=", etc.
	Value     string      // expected value for comparisons
	Body      []Statement // statements to execute if condition is true
	ElseBody  []Statement // statements to execute if condition is false (optional)
}

func (ds *DetectionStatement) statementNode() {}
func (ds *DetectionStatement) String() string {
	var out strings.Builder

	switch ds.Type {
	case "detect":
		out.WriteString("detect " + ds.Target)
		if ds.Condition != "" {
			out.WriteString(" " + ds.Condition)
		}
		if ds.Value != "" {
			out.WriteString(" " + ds.Value)
		}
	case "if_available":
		out.WriteString("if " + ds.Target + " is available")
	case "when_environment":
		out.WriteString("when in " + ds.Target + " environment")
	case "if_version":
		out.WriteString("if " + ds.Target + " version " + ds.Condition + " " + ds.Value)
	default:
		out.WriteString(ds.Type + " " + ds.Target)
	}

	if len(ds.Body) > 0 {
		out.WriteString(":")
		for _, stmt := range ds.Body {
			out.WriteString("\n  " + stmt.String())
		}
	}

	if len(ds.ElseBody) > 0 {
		out.WriteString("\nelse:")
		for _, stmt := range ds.ElseBody {
			out.WriteString("\n  " + stmt.String())
		}
	}

	return out.String()
}

// BreakStatement represents break statements in loops
type BreakStatement struct {
	Token     lexer.Token // the BREAK token
	Condition string      // optional condition (for "break when condition")
}

func (bs *BreakStatement) statementNode() {}
func (bs *BreakStatement) String() string {
	if bs.Condition != "" {
		return "break when " + bs.Condition
	}
	return "break"
}

// ContinueStatement represents continue statements in loops
type ContinueStatement struct {
	Token     lexer.Token // the CONTINUE token
	Condition string      // optional condition (for "continue if condition")
}

func (cs *ContinueStatement) statementNode() {}
func (cs *ContinueStatement) String() string {
	if cs.Condition != "" {
		return "continue if " + cs.Condition
	}
	return "continue"
}

// VariableStatement represents variable operations (let, set, transform)
type VariableStatement struct {
	Token     lexer.Token // the LET, SET, or TRANSFORM token
	Operation string      // "let", "set", "transform"
	Variable  string      // variable name
	Value     string      // value or expression
	Function  string      // function name for operations (concat, split, etc.)
	Arguments []string    // function arguments
}

func (vs *VariableStatement) statementNode() {}
func (vs *VariableStatement) String() string {
	var out strings.Builder

	switch vs.Operation {
	case "let":
		out.WriteString("let ")
		out.WriteString(vs.Variable)
		out.WriteString(" = ")
		out.WriteString(vs.Value)
	case "set":
		out.WriteString("set ")
		out.WriteString(vs.Variable)
		out.WriteString(" to ")
		out.WriteString(vs.Value)
	case "transform":
		out.WriteString("transform ")
		out.WriteString(vs.Variable)
		out.WriteString(" with ")
		out.WriteString(vs.Function)
		if len(vs.Arguments) > 0 {
			out.WriteString(" ")
			out.WriteString(strings.Join(vs.Arguments, " "))
		}
	default:
		out.WriteString(vs.Operation)
		out.WriteString(" ")
		out.WriteString(vs.Variable)
		if vs.Value != "" {
			out.WriteString(" ")
			out.WriteString(vs.Value)
		}
	}

	return out.String()
}

// ParameterStatement represents parameter declarations (requires, given, accepts)
type ParameterStatement struct {
	Token        lexer.Token // the parameter token (REQUIRES, GIVEN, ACCEPTS)
	Type         string      // "requires", "given", "accepts"
	Name         string      // parameter name
	DefaultValue string      // default value (for "given")
	Constraints  []string    // constraints like ["dev", "staging", "production"]
	DataType     string      // "string", "number", "boolean", "list", etc.
	Required     bool        // true for "requires", false for "given"/"accepts"
}

func (ps *ParameterStatement) statementNode() {}
func (ps *ParameterStatement) String() string {
	var out strings.Builder
	out.WriteString(ps.Type)
	out.WriteString(" ")
	out.WriteString(ps.Name)

	if ps.DefaultValue != "" {
		out.WriteString(" defaults to ")
		out.WriteString(ps.DefaultValue)
	}

	if len(ps.Constraints) > 0 {
		out.WriteString(" from [")
		out.WriteString(strings.Join(ps.Constraints, ", "))
		out.WriteString("]")
	}

	if ps.DataType != "" && ps.DataType != "string" {
		out.WriteString(" as ")
		out.WriteString(ps.DataType)
	}

	return out.String()
}

// ConditionalStatement represents when/if statements
type ConditionalStatement struct {
	Token     lexer.Token // the conditional token (WHEN, IF)
	Type      string      // "when", "if"
	Condition string      // the condition expression
	Body      []Statement // statements in the conditional body
	ElseBody  []Statement // statements in the else body (for if statements)
}

func (cs *ConditionalStatement) statementNode() {}
func (cs *ConditionalStatement) String() string {
	var out strings.Builder
	out.WriteString(cs.Type)
	out.WriteString(" ")
	out.WriteString(cs.Condition)
	out.WriteString(":\n")

	for _, stmt := range cs.Body {
		out.WriteString("  ")
		out.WriteString(stmt.String())
		out.WriteString("\n")
	}

	if len(cs.ElseBody) > 0 {
		out.WriteString("else:\n")
		for _, stmt := range cs.ElseBody {
			out.WriteString("  ")
			out.WriteString(stmt.String())
			out.WriteString("\n")
		}
	}

	return out.String()
}

// LoopStatement represents for each loops
type LoopStatement struct {
	Token      lexer.Token       // the FOR token
	Type       string            // "each", "range", "line", "match"
	Variable   string            // loop variable name
	Iterable   string            // what to iterate over
	RangeStart string            // start value for range loops
	RangeEnd   string            // end value for range loops
	RangeStep  string            // step value for range loops (optional)
	Filter     *FilterExpression // filter condition (optional)
	Parallel   bool              // whether to run in parallel
	MaxWorkers int               // maximum number of parallel workers (0 = unlimited)
	FailFast   bool              // whether to stop on first error
	Body       []Statement       // statements in the loop body
}

// FilterExpression represents filter conditions in loops
type FilterExpression struct {
	Variable string // variable being filtered
	Operator string // "contains", "starts", "ends", "matches", "==", "!=", etc.
	Value    string // value to compare against
}

func (ls *LoopStatement) statementNode() {}
func (ls *LoopStatement) String() string {
	var out strings.Builder

	switch ls.Type {
	case "range":
		out.WriteString("for ")
		out.WriteString(ls.Variable)
		out.WriteString(" in range ")
		out.WriteString(ls.RangeStart)
		out.WriteString(" to ")
		out.WriteString(ls.RangeEnd)
		if ls.RangeStep != "" {
			out.WriteString(" step ")
			out.WriteString(ls.RangeStep)
		}
	case "line":
		out.WriteString("for each line ")
		out.WriteString(ls.Variable)
		out.WriteString(" in file ")
		out.WriteString(ls.Iterable)
	case "match":
		out.WriteString("for each match ")
		out.WriteString(ls.Variable)
		out.WriteString(" in pattern ")
		out.WriteString(ls.Iterable)
	default: // "each"
		out.WriteString("for each ")
		out.WriteString(ls.Variable)
		out.WriteString(" in ")
		out.WriteString(ls.Iterable)
	}

	if ls.Filter != nil {
		out.WriteString(" where ")
		out.WriteString(ls.Filter.Variable)
		out.WriteString(" ")
		out.WriteString(ls.Filter.Operator)
		out.WriteString(" ")
		out.WriteString(ls.Filter.Value)
	}

	if ls.Parallel {
		out.WriteString(" in parallel")
	}

	out.WriteString(":\n")

	for _, stmt := range ls.Body {
		out.WriteString("  ")
		out.WriteString(stmt.String())
		out.WriteString("\n")
	}

	return out.String()
}
