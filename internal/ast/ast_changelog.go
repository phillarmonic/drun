package ast

import (
	"fmt"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// ChangelogStatement represents a Keep a Changelog promotion:
//
//	promote changelog "CHANGELOG.md" to version "1.5.0"
//	promote changelog "CHANGELOG.md" to version "1.5.0" on "2026-09-01"
type ChangelogStatement struct {
	Token   lexer.Token
	Path    string // Raw changelog file path (interpolated at execution)
	Version string // Raw release version (interpolated at execution)
	Date    string // Optional release date override (YYYY-MM-DD), empty means today
}

func (cs *ChangelogStatement) statementNode() {}

func (cs *ChangelogStatement) String() string {
	result := fmt.Sprintf("promote changelog %q to version %q", cs.Path, cs.Version)
	if cs.Date != "" {
		result += fmt.Sprintf(" on %q", cs.Date)
	}
	return result
}
