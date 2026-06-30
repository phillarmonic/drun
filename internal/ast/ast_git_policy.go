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

	if gp.BranchPattern != "" || len(gp.DefaultBranches) > 0 || len(gp.BranchTypes) > 0 {
		out.WriteString("\n  branch:")
		if len(gp.DefaultBranches) > 0 {
			out.WriteString("\n    default branches: ")
			for i, b := range gp.DefaultBranches {
				if i > 0 {
					out.WriteString(", ")
				}
				fmt.Fprintf(&out, "\"%s\"", b)
			}
		}
		if gp.BranchPattern != "" {
			fmt.Fprintf(&out, "\n    naming: \"%s\"", gp.BranchPattern)
		}
		if len(gp.BranchTypes) > 0 {
			out.WriteString("\n    types: ")
			for i, t := range gp.BranchTypes {
				if i > 0 {
					out.WriteString(", ")
				}
				fmt.Fprintf(&out, "\"%s\"", t)
			}
		}
	}

	if gp.CommitPattern != "" || gp.ExtractIdentifier || gp.CommitMinLength > 0 || len(gp.CommitBans) > 0 || gp.EnforceSignedCommits {
		out.WriteString("\n  commit:")
		if gp.CommitPattern != "" {
			fmt.Fprintf(&out, "\n    messages: \"%s\"", gp.CommitPattern)
		}
		if len(gp.CommitBans) > 0 {
			out.WriteString("\n    ban: ")
			for i, ban := range gp.CommitBans {
				if i > 0 {
					out.WriteString(", ")
				}
				fmt.Fprintf(&out, "\"%s\"", ban)
			}
		}
		if gp.CommitMinLength > 0 {
			fmt.Fprintf(&out, "\n    min length: %d", gp.CommitMinLength)
		}
		if gp.ExtractIdentifier {
			out.WriteString("\n    extract identifier from branch")
		}
		if gp.EnforceSignedCommits {
			out.WriteString("\n    enforce signed commits")
		}
	}

	return out.String()
}
