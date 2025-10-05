// Package ast defines the Abstract Syntax Tree nodes for the drun v2 language.
package ast

import (
	"strings"
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

// Expression represents any expression node
type Expression interface {
	Node
	expressionNode()
}

// Program represents the root of the AST
type Program struct {
	Version   *VersionStatement
	Project   *ProjectStatement
	Tasks     []*TaskStatement
	Templates []*TaskTemplateStatement
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
	for _, template := range p.Templates {
		out.WriteString(template.String())
		out.WriteString("\n")
	}
	for _, task := range p.Tasks {
		out.WriteString(task.String())
		out.WriteString("\n")
	}
	return out.String()
}

// ProjectSetting represents a project-level setting
type ProjectSetting interface {
	Node
	projectSettingNode()
}
