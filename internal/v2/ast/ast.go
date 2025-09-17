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
	Token       lexer.Token       // the TASK token
	Name        string            // task name
	Description string            // optional description after "means"
	Body        []ActionStatement // statements in the task body
}

func (ts *TaskStatement) statementNode() {}
func (ts *TaskStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("task \"%s\"", ts.Name))
	if ts.Description != "" {
		out.WriteString(fmt.Sprintf(" means \"%s\"", ts.Description))
	}
	out.WriteString(":\n")
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
