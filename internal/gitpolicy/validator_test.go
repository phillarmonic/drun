package gitpolicy

import (
	"testing"
)

func TestPolicy_ValidateBranchName(t *testing.T) {
	policy := &Policy{
		DefaultBranches: []string{"master", "develop"},
		BranchPattern:   "{type}/{identifier}-{description}",
		BranchTypes:     []string{"feat", "fix", "hotfix", "chore"},
	}

	tests := []struct {
		name       string
		branchName string
		wantErr    bool
	}{
		{"default branch master", "master", false},
		{"default branch develop", "develop", false},
		{"valid feat branch", "feat/PHIL-01-added-hello", false},
		{"valid fix branch with long id", "fix/CORE-1234-bug-fix", false},
		{"valid chore without dash id", "chore/hello-world", false}, // identifier becomes "hello"
		{"invalid type", "test/PHIL-01-foo", true},
		{"missing type", "PHIL-01-foo", true},
		{"missing description", "feat/PHIL", true}, // requires description after dash
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := policy.ValidateBranchName(tt.branchName); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBranchName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPolicy_ExtractIdentifier(t *testing.T) {
	policy := &Policy{
		DefaultBranches: []string{"master", "develop"},
		BranchPattern:   "{type}/{identifier}-{description}",
		BranchTypes:     []string{"feat", "fix", "hotfix", "chore"},
	}

	tests := []struct {
		name       string
		branchName string
		want       string
		wantErr    bool
	}{
		{"feat branch", "feat/PHIL-01-hello", "PHIL-01", false},
		{"fix branch", "fix/CORE-123-hello", "CORE-123", false},
		{"default branch", "master", "", false},
		{"invalid branch", "feat/hello", "", true}, // "hello" is captured as id, but there is no description dash
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := policy.ExtractIdentifierFromBranch(tt.branchName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractIdentifierFromBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractIdentifierFromBranch() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolicy_ValidateCommitMessage(t *testing.T) {
	policy := &Policy{
		CommitMinLength:   10,
		CommitBans:        []string{"WIP", "wip", "fixup"},
		CommitPattern:     "{identifier}: {message}",
		ExtractIdentifier: true,
	}

	tests := []struct {
		name       string
		msg        string
		branchName string
		wantErr    bool
	}{
		{"valid with identifier", "PHIL-01: added new feature", "feat/PHIL-01-feature", false},
		{"too short", "PHIL-01:", "feat/PHIL-01-feature", true},
		{"banned word", "WIP", "feat/PHIL-01-feature", true},
		{"missing identifier from branch", "added feature", "feat/PHIL-01-feature", true},
		{"valid without branch extraction (enforces some id format)", "CORE-12: fixed bug", "", false},
		{"invalid format without branch extraction", "fixed bug", "", true},
		{"default branch skips extraction but requires id pattern", "CORE-99: hello", "master", false},
		{"default branch invalid pattern", "hello world", "master", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := policy.ValidateCommitMessage(tt.msg, tt.branchName); (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommitMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPolicy_ValidateCommitMessage_ConventionalCommits(t *testing.T) {
	policy := &Policy{
		BranchPattern:     "{type}/{identifier}-{description}",
		BranchTypes:       []string{"feat", "fix", "chore"},
		CommitMinLength:   10,
		CommitBans:        []string{"WIP", "wip", "fixup"},
		CommitPattern:     "conventional commits",
		ExtractIdentifier: true,
	}

	tests := []struct {
		name       string
		msg        string
		branchName string
		wantErr    bool
	}{
		{"valid conventional header", "feat: add parser support", "", false},
		{"valid conventional with scope and breaking change", "feat(parser)!: PHIL-01 support conventional commits", "feat/PHIL-01-parser", false},
		{"valid conventional with body", "fix(engine): PHIL-01 reject invalid headers\n\nAdds strict header validation.", "fix/PHIL-01-engine", false},
		{"missing type separator", "feat add parser support", "", true},
		{"missing description", "feat:", "", true},
		{"uppercase type rejected", "Feat: add parser support", "", true},
		{"missing branch identifier", "feat(parser): add parser support", "feat/PHIL-01-parser", true},
		{"banned still rejected", "WIP", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := policy.ValidateCommitMessage(tt.msg, tt.branchName); (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommitMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
