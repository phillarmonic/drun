package statements

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
)

// TaskStatement represents a task definition
type TaskStatement struct {
	Token        lexer.Token
	Name         string
	Description  string
	Parameters   []*ParameterStatement
	Dependencies []*DependencyGroup
	Body         []ast.Statement
}

func (ts *TaskStatement) statementNode() {}
func (ts *TaskStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("task \"%s\"", ts.Name))
	if ts.Description != "" {
		out.WriteString(fmt.Sprintf(" means \"%s\"", ts.Description))
	}
	out.WriteString(":\n")

	for _, dep := range ts.Dependencies {
		out.WriteString(fmt.Sprintf("  %s\n", dep.String()))
	}

	for _, param := range ts.Parameters {
		out.WriteString(fmt.Sprintf("  %s\n", param.String()))
	}

	for _, stmt := range ts.Body {
		out.WriteString(fmt.Sprintf("  %s\n", stmt.String()))
	}
	return out.String()
}

// TaskCallStatement represents calling another task
type TaskCallStatement struct {
	Token      lexer.Token
	TaskName   string
	Parameters map[string]string
}

func (tcs *TaskCallStatement) statementNode() {}
func (tcs *TaskCallStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("call task \"%s\"", tcs.TaskName))
	if len(tcs.Parameters) > 0 {
		out.WriteString(" with")
		for key, value := range tcs.Parameters {
			out.WriteString(fmt.Sprintf(" %s=\"%s\"", key, value))
		}
	}
	return out.String()
}

// DependencyGroup represents a group of dependencies with execution semantics
type DependencyGroup struct {
	Token        lexer.Token
	Dependencies []DependencyItem
	Sequential   bool
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
	Name     string
	Parallel bool
}

func (di *DependencyItem) String() string {
	if di.Parallel {
		return di.Name + " in parallel"
	}
	return di.Name
}

// TaskTemplateStatement represents a task template definition
type TaskTemplateStatement struct {
	Token       lexer.Token
	Name        string
	Description string
	Parameters  []*ParameterStatement
	Body        []ast.Statement
}

func (tts *TaskTemplateStatement) statementNode() {}
func (tts *TaskTemplateStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("template task \"%s\"", tts.Name))
	if tts.Description != "" {
		out.WriteString(fmt.Sprintf(" means \"%s\"", tts.Description))
	}
	out.WriteString(":\n")

	for _, param := range tts.Parameters {
		out.WriteString(fmt.Sprintf("  %s\n", param.String()))
	}

	for _, stmt := range tts.Body {
		out.WriteString(fmt.Sprintf("  %s\n", stmt.String()))
	}
	return out.String()
}

// TaskFromTemplateStatement represents a task instantiated from a template
type TaskFromTemplateStatement struct {
	Token        lexer.Token
	Name         string
	TemplateName string
	Overrides    map[string]string
}

func (tfts *TaskFromTemplateStatement) statementNode() {}
func (tfts *TaskFromTemplateStatement) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("task \"%s\" from template \"%s\"", tfts.Name, tfts.TemplateName))

	if len(tfts.Overrides) > 0 {
		out.WriteString(":\n  with")
		for key, value := range tfts.Overrides {
			out.WriteString(fmt.Sprintf(" %s=\"%s\"", key, value))
		}
	}

	return out.String()
}
