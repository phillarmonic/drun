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
	Tasks   []*TaskStatement
}

func (p *Program) String() string {
	var out strings.Builder
	if p.Version != nil {
		out.WriteString(p.Version.String())
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

// TaskStatement represents a task definition
type TaskStatement struct {
	Token       lexer.Token          // the TASK token
	Name        string               // task name
	Description string               // optional description after "means"
	Parameters  []ParameterStatement // parameter declarations
	Body        []Statement          // statements in the task body (actions, conditionals, loops)
}

func (ts *TaskStatement) statementNode() {}
func (ts *TaskStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("task \"%s\"", ts.Name))
	if ts.Description != "" {
		out.WriteString(fmt.Sprintf(" means \"%s\"", ts.Description))
	}
	out.WriteString(":\n")

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

// ParameterStatement represents parameter declarations (requires, given, accepts)
type ParameterStatement struct {
	Token        lexer.Token // the parameter token (REQUIRES, GIVEN, ACCEPTS)
	Type         string      // "requires", "given", "accepts"
	Name         string      // parameter name
	DefaultValue string      // default value (for "given")
	Constraints  []string    // constraints like ["dev", "staging", "production"]
	DataType     string      // "string", "list", "number", etc.
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
	Token    lexer.Token // the FOR token
	Variable string      // loop variable name
	Iterable string      // what to iterate over
	Parallel bool        // whether to run in parallel
	Body     []Statement // statements in the loop body
}

func (ls *LoopStatement) statementNode() {}
func (ls *LoopStatement) String() string {
	var out strings.Builder
	out.WriteString("for each ")
	out.WriteString(ls.Variable)
	out.WriteString(" in ")
	out.WriteString(ls.Iterable)

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
