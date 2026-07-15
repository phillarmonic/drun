package ast

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// SCMRegistryStatement is the project-level SCM → technology → provider registry.
type SCMRegistryStatement struct {
	Token        lexer.Token
	Technologies map[string]*SCMTechnology
}

type SCMTechnology struct {
	Name      string
	Providers map[string]*SCMProvider
}

type SCMProvider struct {
	Name    string
	Sources map[string]*SCMSource
}

type SCMSource struct {
	Alias       string
	Provider    string
	Default     string
	Metadata    string
	Access      map[string]*SCMAccessProfile
	VersionTags *VersionTagContract
}

type SCMAccessProfile struct {
	Method         string
	URL            string
	Repository     string
	Host           string
	Authentication string
	Key            string
	Path           string
}

type VersionTagContract struct {
	Preset  string
	Formats []string
	Pattern string
}

func (s *SCMRegistryStatement) statementNode()      {}
func (s *SCMRegistryStatement) projectSettingNode() {}
func (s *SCMRegistryStatement) String() string {
	var out strings.Builder
	out.WriteString("scm:")
	techNames := sortedKeys(s.Technologies)
	for _, techName := range techNames {
		tech := s.Technologies[techName]
		fmt.Fprintf(&out, "\n  %s:", techName)
		for _, providerName := range sortedKeys(tech.Providers) {
			provider := tech.Providers[providerName]
			fmt.Fprintf(&out, "\n    %s:", providerName)
			for _, alias := range sortedKeys(provider.Sources) {
				fmt.Fprintf(&out, "\n      %s:", alias)
			}
		}
	}
	return out.String()
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// GitQueryStatement captures a value derived from a registered Git source.
type GitQueryStatement struct {
	Token          lexer.Token
	Result         string // tag or version
	Source         string
	AccessMethod   string
	TagPreset      string
	TagFormat      string
	TagPattern     string
	Series         string
	VersionMatcher string
	OrderBy        string // version or date
	AllowFetch     bool
	CaptureVar     string
}

func (s *GitQueryStatement) statementNode() {}
func (s *GitQueryStatement) String() string {
	out := fmt.Sprintf("git get latest %s from %s", s.Result, s.Source)
	if s.AccessMethod != "" {
		out += " using " + s.AccessMethod
	}
	if s.TagPattern != "" {
		out += fmt.Sprintf(" matching tags pattern %q", s.TagPattern)
	} else if s.TagFormat != "" {
		out += fmt.Sprintf(" matching tags %q", s.TagFormat)
	} else if s.TagPreset != "" {
		out += " matching tags " + s.TagPreset
	}
	if s.Series != "" {
		out += fmt.Sprintf(" in series %q", s.Series)
	} else if s.VersionMatcher != "" {
		out += fmt.Sprintf(" matching version %q", s.VersionMatcher)
	}
	if s.OrderBy != "" && s.OrderBy != "version" {
		out += " ordered by " + s.OrderBy
	}
	if s.AllowFetch {
		out += " allow fetch"
	}
	return out + " as $" + s.CaptureVar
}
