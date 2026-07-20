package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// VersionConstraint represents a single version constraint (e.g., >= "2.27")
type VersionConstraint struct {
	Operator string // ">=", ">", "<=", "<"
	Version  string // "2.27", "3.0", etc.
}

// ToolRequirement represents a tool requirement with optional version constraints
type ToolRequirement struct {
	Name          string              // tool name (e.g., "gosec", "golangci-lint")
	Constraints   []VersionConstraint // zero or more version constraints
	AutoProvision bool                // whether drun may provision the tool automatically
}

func (tr *ToolRequirement) String() string {
	var out strings.Builder
	out.WriteString(tr.Name)
	for _, c := range tr.Constraints {
		fmt.Fprintf(&out, " %s \"%s\"", c.Operator, c.Version)
	}
	if tr.AutoProvision {
		out.WriteString(" provision")
	}
	return out.String()
}

// TaskToolSources represents a "from tasks:" source clause inside a
// "requires tools:" block.
type TaskToolSources struct {
	Token lexer.Token
	Tasks []string
}

func (tts *TaskToolSources) String() string {
	var out strings.Builder
	out.WriteString("from tasks:")
	for _, task := range tts.Tasks {
		fmt.Fprintf(&out, "\n  %s", task)
	}
	return out.String()
}

// RequiresToolsStatement represents a "requires tools:" block
// This can appear in both project settings and task bodies.
type RequiresToolsStatement struct {
	Token       lexer.Token
	Tools       []ToolRequirement
	TaskSources []TaskToolSources
}

func (rts *RequiresToolsStatement) statementNode()      {}
func (rts *RequiresToolsStatement) projectSettingNode() {}
func (rts *RequiresToolsStatement) String() string {
	var out strings.Builder
	out.WriteString("requires tools:")
	for _, tool := range rts.Tools {
		out.WriteString("\n  ")
		out.WriteString(tool.String())
	}
	for _, source := range rts.TaskSources {
		sourceString := strings.ReplaceAll(source.String(), "\n", "\n  ")
		out.WriteString("\n  ")
		out.WriteString(sourceString)
	}
	return out.String()
}

// ProvisioningSourcesStatement represents a project-level "provisioning sources:" block.
type ProvisioningSourcesStatement struct {
	Token   lexer.Token
	Sources []string
}

func (pss *ProvisioningSourcesStatement) statementNode()      {}
func (pss *ProvisioningSourcesStatement) projectSettingNode() {}
func (pss *ProvisioningSourcesStatement) String() string {
	var out strings.Builder
	out.WriteString("provisioning sources:")
	for _, source := range pss.Sources {
		fmt.Fprintf(&out, "\n  %q", source)
	}
	return out.String()
}
