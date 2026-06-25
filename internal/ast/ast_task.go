package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// TaskStatement represents a task definition
type TaskStatement struct {
	Token        lexer.Token
	Name         string
	Mode         string
	Description  string
	Annotations  []Annotation
	Parameters   []ParameterStatement
	Dependencies []DependencyGroup
	Body         []Statement
}

func (ts *TaskStatement) statementNode() {}
func (ts *TaskStatement) String() string {
	var out strings.Builder
	for _, annotation := range ts.Annotations {
		out.WriteString(annotation.String())
		out.WriteString("\n")
	}
	fmt.Fprintf(&out, "task \"%s\"", ts.Name)
	if ts.Mode != "" {
		fmt.Fprintf(&out, " mode \"%s\"", ts.Mode)
	}
	if ts.Description != "" {
		fmt.Fprintf(&out, " means \"%s\"", ts.Description)
	}
	out.WriteString(":\n")

	for _, dep := range ts.Dependencies {
		fmt.Fprintf(&out, "  %s\n", dep.String())
	}

	for _, param := range ts.Parameters {
		fmt.Fprintf(&out, "  %s\n", param.String())
	}

	for _, stmt := range ts.Body {
		fmt.Fprintf(&out, "  %s\n", stmt.String())
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
	fmt.Fprintf(&out, "call task \"%s\"", tcs.TaskName)
	if len(tcs.Parameters) > 0 {
		out.WriteString(" with")
		for key, value := range tcs.Parameters {
			fmt.Fprintf(&out, " %s=\"%s\"", key, value)
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
	Annotations []Annotation
	Parameters  []ParameterStatement
	Body        []Statement
}

func (tts *TaskTemplateStatement) statementNode() {}
func (tts *TaskTemplateStatement) String() string {
	var out strings.Builder
	for _, annotation := range tts.Annotations {
		out.WriteString(annotation.String())
		out.WriteString("\n")
	}
	fmt.Fprintf(&out, "template task \"%s\"", tts.Name)
	if tts.Description != "" {
		fmt.Fprintf(&out, " means \"%s\"", tts.Description)
	}
	out.WriteString(":\n")

	for _, param := range tts.Parameters {
		fmt.Fprintf(&out, "  %s\n", param.String())
	}

	for _, stmt := range tts.Body {
		fmt.Fprintf(&out, "  %s\n", stmt.String())
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
	fmt.Fprintf(&out, "task \"%s\" from template \"%s\"", tfts.Name, tfts.TemplateName)

	if len(tfts.Overrides) > 0 {
		out.WriteString(":\n  with")
		for key, value := range tfts.Overrides {
			fmt.Fprintf(&out, " %s=\"%s\"", key, value)
		}
	}

	return out.String()
}
