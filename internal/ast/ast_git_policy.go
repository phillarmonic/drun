package ast

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/lexer"
)

// GitPolicyStatement is a project-level setting for git conventions.
// It defines branch naming rules, commit message patterns, banned messages,
// and signing requirements.
type GitPolicyStatement struct {
	Token                lexer.Token
	DefaultBranches      []string
	BranchPattern        string   // e.g. "{type}/{identifier}-{description}"
	BranchTypes          []string // e.g. ["feat", "fix", "hotfix", "chore"]
	CommitPattern        string   // e.g. "{identifier}: {message}"
	ExtractIdentifier    bool     // extract identifier from branch name
	CommitMinLength      int      // minimum commit message length (0 = no limit)
	CommitBans           []string // banned commit message patterns (exact match)
	EnforceSignedCommits bool
}

func (gp *GitPolicyStatement) statementNode()      {}
func (gp *GitPolicyStatement) projectSettingNode() {}
func (gp *GitPolicyStatement) String() string {
	var out strings.Builder
	out.WriteString("git policy:")

	if len(gp.DefaultBranches) > 0 {
		out.WriteString("\n  default branches:")
		for _, b := range gp.DefaultBranches {
			fmt.Fprintf(&out, "\n    %s", b)
		}
	}

	if gp.BranchPattern != "" {
		out.WriteString("\n  branch naming:")
		fmt.Fprintf(&out, "\n    pattern \"%s\"", gp.BranchPattern)
		if len(gp.BranchTypes) > 0 {
			fmt.Fprintf(&out, "\n    types: %s", strings.Join(gp.BranchTypes, ", "))
		}
	}

	if gp.CommitPattern != "" || gp.ExtractIdentifier || gp.CommitMinLength > 0 || len(gp.CommitBans) > 0 {
		out.WriteString("\n  commit messages:")
		if gp.CommitPattern != "" {
			fmt.Fprintf(&out, "\n    pattern \"%s\"", gp.CommitPattern)
		}
		if gp.ExtractIdentifier {
			out.WriteString("\n    extract identifier from branch")
		}
		if gp.CommitMinLength > 0 {
			fmt.Fprintf(&out, "\n    min length %d", gp.CommitMinLength)
		}
		for _, ban := range gp.CommitBans {
			fmt.Fprintf(&out, "\n    ban \"%s\"", ban)
		}
	}

	if gp.EnforceSignedCommits {
		out.WriteString("\n  enforce signed commits")
	}

	return out.String()
}
