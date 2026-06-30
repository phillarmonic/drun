package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/gitpolicy"
)

// executeGitValidate executes an inline "git validate" statement.
func (e *Engine) executeGitValidate(stmt *statement.GitValidate, ctx *ExecutionContext) error {
	if ctx.Project == nil || ctx.Project.GitPolicy == nil {
		if e.dryRun {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would validate git %s, but no git policy is configured\n", stmt.Target)
			return nil
		}
		return fmt.Errorf("cannot validate git %s: no git policy configured in project settings", stmt.Target)
	}

	policy := toGitPolicyModel(ctx.Project.GitPolicy)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] git validate %s\n", stmt.Target)
		return nil
	}

	switch stmt.Target {
	case "branch_name":
		return e.validateBranchName(policy)
	case "commit_message":
		return e.validateCommitMessage(policy, stmt.Value)
	case "signed_commits":
		return e.validateSignedCommits()
	case "all":
		if err := e.validateBranchName(policy); err != nil {
			return err
		}
		// For all, we only validate the last commit message if no explicit value is given
		if err := e.validateCommitMessage(policy, stmt.Value); err != nil {
			return err
		}
		if err := e.validateSignedCommits(); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown git validate target: %s", stmt.Target)
	}
}

func toGitPolicyModel(s *statement.GitPolicy) *gitpolicy.Policy {
	return &gitpolicy.Policy{
		DefaultBranches:      s.DefaultBranches,
		BranchPattern:        s.BranchPattern,
		BranchTypes:          s.BranchTypes,
		CommitPattern:        s.CommitPattern,
		ExtractIdentifier:    s.ExtractIdentifier,
		CommitMinLength:      s.CommitMinLength,
		CommitBans:           s.CommitBans,
		EnforceSignedCommits: s.EnforceSignedCommits,
	}
}

func (e *Engine) validateBranchName(policy *gitpolicy.Policy) error {
	// Get current branch
	branchName, err := e.RunGitCommandOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	branchName = strings.TrimSpace(branchName)

	if err := policy.ValidateBranchName(branchName); err != nil {
		return err
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "✅  Branch name '%s' is valid\n", branchName)
	}
	return nil
}

func (e *Engine) validateCommitMessage(policy *gitpolicy.Policy, explicitMsg string) error {
	var msg string
	if explicitMsg != "" {
		msg = explicitMsg
	} else {
		// Get last commit message
		var err error
		msg, err = e.RunGitCommandOutput("log", "-1", "--pretty=%B")
		if err != nil {
			return fmt.Errorf("failed to get last commit message: %w", err)
		}
	}

	// Get current branch for identifier extraction
	branchName, _ := e.RunGitCommandOutput("rev-parse", "--abbrev-ref", "HEAD")
	branchName = strings.TrimSpace(branchName)

	if err := policy.ValidateCommitMessage(msg, branchName); err != nil {
		return err
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "✅  Commit message is valid\n")
	}
	return nil
}

func (e *Engine) validateSignedCommits() error {
	// Let's check the last commit for a signature
	// %G? outputs G for good signature, B for bad, U for good/untrusted, N for no signature, etc.
	out, err := e.RunGitCommandOutput("log", "-1", "--format=%G?")
	if err != nil {
		return fmt.Errorf("failed to check commit signature: %w", err)
	}

	sigStatus := strings.TrimSpace(out)
	if sigStatus != "G" && sigStatus != "U" {
		return fmt.Errorf("commit is not properly signed (signature status: '%s')", sigStatus)
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "✅  Commit is signed\n")
	}
	return nil
}

// RunGitCommandOutput runs a git command and gets its output
func (e *Engine) RunGitCommandOutput(args ...string) (string, error) {
	cmdArgs := append([]string{}, args...)
	
	// Create command
	cmd := exec.Command("git", cmdArgs...)

	// We disable prompt/tty requirements for pure output capture
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), exitErr.Stderr)
		}
		return "", err
	}

	return string(out), nil
}
