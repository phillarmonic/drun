package gitpolicy

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var conventionalCommitHeaderPattern = regexp.MustCompile(`^[a-z]+(?:-[a-z]+)*(?:\([^\r\n()]+\))?!?: [^\s\r\n].*$`)

// Policy represents the project's git conventions.
type Policy struct {
	DefaultBranches      []string
	ProtectedBranches    []string
	BranchPattern        string
	BranchTypes          []string
	CommitPattern        string
	ExtractIdentifier    bool
	CommitMinLength      int
	CommitBans           []string
	EnforceSignedCommits bool
}

// ValidationResult indicates whether validation passed, and if not, the reason.
type ValidationResult struct {
	Valid   bool
	Message string
}

// IsDefaultBranch returns true if the branch is in the default branches list.
func (p *Policy) IsDefaultBranch(branchName string) bool {
	for _, b := range p.DefaultBranches {
		if b == branchName {
			return true
		}
	}
	return false
}

// IsProtectedBranch returns true if the branch is explicitly protected.
func (p *Policy) IsProtectedBranch(branchName string) bool {
	for _, b := range p.ProtectedBranches {
		if b == branchName {
			return true
		}
	}
	return false
}

// ValidateProtectedBranchCommit rejects local commits on explicitly protected branches.
func (p *Policy) ValidateProtectedBranchCommit(branchName string) error {
	if branchName == "" {
		return errors.New("current branch is unknown")
	}

	if p.IsProtectedBranch(branchName) {
		return fmt.Errorf("branch '%s' is protected; commit on a feature branch and merge through your normal review flow", branchName)
	}

	return nil
}

// ValidateBranchName checks if a branch name conforms to the policy.
func (p *Policy) ValidateBranchName(branchName string) error {
	if p.IsDefaultBranch(branchName) {
		return nil // Default branches are exempt from naming rules
	}

	if p.BranchPattern == "" {
		return nil // No pattern enforced
	}

	// We support {type}, {identifier}, {description}
	// Let's build a regex from the pattern
	regexStr := "^" + regexp.QuoteMeta(p.BranchPattern) + "$"

	// Replace {type}
	if strings.Contains(p.BranchPattern, "{type}") {
		if len(p.BranchTypes) > 0 {
			typesRegex := "(" + strings.Join(p.BranchTypes, "|") + ")"
			regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{type}"), typesRegex, 1)
		} else {
			regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{type}"), `([a-zA-Z0-9_]+)`, 1)
		}
	}

	// Replace {identifier}
	if strings.Contains(p.BranchPattern, "{identifier}") {
		regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{identifier}"), `([a-zA-Z0-9]+-[0-9]+|[a-zA-Z0-9]+)`, 1)
	}

	// Replace {description}
	if strings.Contains(p.BranchPattern, "{description}") {
		regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{description}"), `(.+)`, 1)
	}

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return fmt.Errorf("invalid branch pattern regex: %w", err)
	}

	if !re.MatchString(branchName) {
		return fmt.Errorf("branch '%s' does not match required pattern: %s", branchName, p.BranchPattern)
	}

	return nil
}

// ExtractIdentifierFromBranch attempts to pull out {identifier} from the branch name based on the pattern.
func (p *Policy) ExtractIdentifierFromBranch(branchName string) (string, error) {
	if p.BranchPattern == "" || !strings.Contains(p.BranchPattern, "{identifier}") {
		return "", errors.New("no {identifier} in branch pattern")
	}

	if p.IsDefaultBranch(branchName) {
		return "", nil // Default branches don't have identifiers
	}

	// Build regex to capture the identifier group
	regexStr := "^" + regexp.QuoteMeta(p.BranchPattern) + "$"

	// Track which group index is the identifier
	groupIndex := -1
	currentIdx := 1

	if strings.Contains(p.BranchPattern, "{type}") {
		if len(p.BranchTypes) > 0 {
			regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{type}"), "(?:{})", 1) // don't capture or capture non-destructively, let's just make it a non-capturing group
			regexStr = strings.Replace(regexStr, "(?:{})", "(?:"+strings.Join(p.BranchTypes, "|")+")", 1)
		} else {
			regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{type}"), `(?:[a-zA-Z0-9_]+)`, 1)
		}
	}

	// Replace {identifier} with a capturing group
	idx := strings.Index(regexStr, regexp.QuoteMeta("{identifier}"))
	if idx >= 0 {
		groupIndex = currentIdx
		regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{identifier}"), `([a-zA-Z0-9]+-[0-9]+|[a-zA-Z0-9]+)`, 1)
	}

	if strings.Contains(p.BranchPattern, "{description}") {
		regexStr = strings.Replace(regexStr, regexp.QuoteMeta("{description}"), `(?:.+)`, 1)
	}

	re, err := regexp.Compile(regexStr)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(branchName)
	if len(matches) > groupIndex {
		return matches[groupIndex], nil
	}

	return "", fmt.Errorf("could not extract identifier from branch '%s'", branchName)
}

// ValidateCommitMessage checks if a commit message conforms to the policy.
// It uses branchName if extract identifier is enabled.
func (p *Policy) ValidateCommitMessage(msg, branchName string) error {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return errors.New("commit message is empty")
	}

	if p.CommitMinLength > 0 && len(msg) < p.CommitMinLength {
		return fmt.Errorf("commit message is too short (minimum %d characters)", p.CommitMinLength)
	}

	for _, ban := range p.CommitBans {
		if msg == ban { // exact match for banned patterns like "WIP"
			return fmt.Errorf("commit message '%s' is banned", ban)
		}
	}

	if p.CommitPattern == "" {
		return nil
	}

	// Extract identifier if needed
	identifier := ""
	if p.ExtractIdentifier && branchName != "" {
		id, err := p.ExtractIdentifierFromBranch(branchName)
		if err == nil && id != "" {
			identifier = id
		}
	}

	if isConventionalCommitPattern(p.CommitPattern) {
		if err := validateConventionalCommitMessage(msg); err != nil {
			return err
		}
		if identifier != "" && !strings.Contains(msg, identifier) {
			return fmt.Errorf("commit message must include branch identifier '%s'", identifier)
		}
		return nil
	}

	// Fast path: if pattern is exactly "{identifier}: {message}"
	if p.CommitPattern == "{identifier}: {message}" {
		if identifier != "" {
			prefix := identifier + ":"
			if !strings.HasPrefix(msg, prefix) {
				return fmt.Errorf("commit message must start with '%s'", prefix)
			}
		} else if strings.Contains(p.CommitPattern, "{identifier}") {
			// They wanted an identifier but we couldn't get one (e.g. on default branch).
			// We can either strictly require it or allow any format if no identifier exists.
			// Let's enforce that it has SOME identifier if the pattern requires it.
			if !regexp.MustCompile(`^[a-zA-Z0-9]+-[0-9]+:`).MatchString(msg) && !regexp.MustCompile(`^[a-zA-Z0-9]+:`).MatchString(msg) {
				return errors.New("commit message does not match required pattern: {identifier}: {message}")
			}
		}
		return nil
	}

	// Custom pattern logic...
	// For now we'll just check if the prefix matches the identifier if present
	if identifier != "" && strings.Contains(p.CommitPattern, "{identifier}") {
		prefixParts := strings.Split(p.CommitPattern, "{identifier}")
		if len(prefixParts) > 0 && prefixParts[0] != "" {
			if !strings.HasPrefix(msg, prefixParts[0]+identifier) {
				return fmt.Errorf("commit message must follow pattern '%s'", p.CommitPattern)
			}
		} else {
			if !strings.HasPrefix(msg, identifier) {
				return fmt.Errorf("commit message must start with identifier '%s'", identifier)
			}
		}
	}

	return nil
}

func isConventionalCommitPattern(pattern string) bool {
	switch strings.ToLower(strings.TrimSpace(pattern)) {
	case "conventional", "conventional commit", "conventional commits", "conventional-commits":
		return true
	default:
		return false
	}
}

func validateConventionalCommitMessage(msg string) error {
	header := msg
	if idx := strings.IndexAny(msg, "\r\n"); idx >= 0 {
		header = msg[:idx]
	}
	header = strings.TrimSpace(header)
	if !conventionalCommitHeaderPattern.MatchString(header) {
		return errors.New("commit message must follow Conventional Commits format: type(scope): description")
	}
	return nil
}
