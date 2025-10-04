package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/lexer"
)

// VersionStatement represents a version declaration
type VersionStatement struct {
	Token lexer.Token
	Value string
}

func (vs *VersionStatement) statementNode() {}
func (vs *VersionStatement) String() string {
	return fmt.Sprintf("version: %s", vs.Value)
}

// ProjectStatement represents a project declaration
type ProjectStatement struct {
	Token    lexer.Token
	Name     string
	Version  string
	Settings []ProjectSetting
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

// SetStatement represents a project setting (set key to value)
type SetStatement struct {
	Token lexer.Token
	Key   string
	Value Expression
}

func (ss *SetStatement) statementNode()      {}
func (ss *SetStatement) projectSettingNode() {}
func (ss *SetStatement) String() string {
	if ss.Value != nil {
		return fmt.Sprintf("set %s to %s", ss.Key, ss.Value.String())
	}
	return fmt.Sprintf("set %s to <nil>", ss.Key)
}

// IncludeStatement represents an include directive
type IncludeStatement struct {
	Token     lexer.Token
	Path      string
	Selectors []string
	Namespace string
}

func (is *IncludeStatement) statementNode()      {}
func (is *IncludeStatement) projectSettingNode() {}
func (is *IncludeStatement) String() string {
	var out strings.Builder
	if len(is.Selectors) > 0 {
		out.WriteString(fmt.Sprintf("include %s from %s", strings.Join(is.Selectors, ", "), is.Path))
	} else {
		out.WriteString(fmt.Sprintf("include %s", is.Path))
	}
	if is.Namespace != "" {
		out.WriteString(fmt.Sprintf(" as %s", is.Namespace))
	}
	return out.String()
}

// ShellConfigStatement represents shell configuration for different platforms
type ShellConfigStatement struct {
	Token     lexer.Token
	Platforms map[string]*PlatformShellConfig
}

func (scs *ShellConfigStatement) statementNode()      {}
func (scs *ShellConfigStatement) projectSettingNode() {}
func (scs *ShellConfigStatement) String() string {
	var out strings.Builder
	out.WriteString("shell config:")
	for platform, config := range scs.Platforms {
		out.WriteString(fmt.Sprintf("\n  %s:", platform))
		out.WriteString(fmt.Sprintf("\n    executable: \"%s\"", config.Executable))
		if len(config.Args) > 0 {
			out.WriteString("\n    args:")
			for _, arg := range config.Args {
				out.WriteString(fmt.Sprintf("\n      - \"%s\"", arg))
			}
		}
		if len(config.Environment) > 0 {
			out.WriteString("\n    environment:")
			for key, value := range config.Environment {
				out.WriteString(fmt.Sprintf("\n      %s: \"%s\"", key, value))
			}
		}
	}
	return out.String()
}

// PlatformShellConfig represents shell configuration for a specific platform
type PlatformShellConfig struct {
	Executable  string
	Args        []string
	Environment map[string]string
}

// LifecycleHook represents lifecycle hooks
type LifecycleHook struct {
	Token lexer.Token
	Type  string // "before", "after", "setup", or "teardown"
	Scope string // "any" for task hooks, "drun" for tool hooks
	Body  []Statement
}

func (lh *LifecycleHook) statementNode()      {}
func (lh *LifecycleHook) projectSettingNode() {}
func (lh *LifecycleHook) String() string {
	var out strings.Builder
	if lh.Scope == "drun" {
		out.WriteString("on ")
		out.WriteString(lh.Scope)
		out.WriteString(" ")
		out.WriteString(lh.Type)
		out.WriteString(":")
	} else {
		out.WriteString(lh.Type)
		out.WriteString(" ")
		out.WriteString(lh.Scope)
		out.WriteString(" task:")
	}
	for _, stmt := range lh.Body {
		out.WriteString("\n    ")
		out.WriteString(stmt.String())
	}
	return out.String()
}
