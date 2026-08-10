package parser

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func TestParser_ChangelogPromotion(t *testing.T) {
	tests := []struct {
		input       string
		wantPath    string
		wantVersion string
		wantDate    string
	}{
		{`promote changelog "CHANGELOG.md" to version "1.5.0"`, "CHANGELOG.md", "1.5.0", ""},
		{`promote changelog "CHANGELOG.md" to version "1.5.0" on "2026-09-01"`, "CHANGELOG.md", "1.5.0", "2026-09-01"},
		{`promote changelog "docs/CHANGELOG.md" to version "{$release_version}"`, "docs/CHANGELOG.md", "{$release_version}", ""},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n  " + tt.input + "\n"

		l := lexer.NewLexer(input)
		p := NewParser(l)
		program := p.ParseProgram()

		checkParserErrors(t, p)

		if len(program.Tasks) != 1 {
			t.Fatalf("%q: program should have 1 task. got=%d", tt.input, len(program.Tasks))
		}

		task := program.Tasks[0]
		if len(task.Body) != 1 {
			t.Fatalf("%q: task should have 1 statement. got=%d", tt.input, len(task.Body))
		}

		changelogStmt, ok := task.Body[0].(*ast.ChangelogStatement)
		if !ok {
			t.Fatalf("%q: statement should be ChangelogStatement. got=%T", tt.input, task.Body[0])
		}

		if changelogStmt.Path != tt.wantPath {
			t.Errorf("%q: path not %q. got=%q", tt.input, tt.wantPath, changelogStmt.Path)
		}
		if changelogStmt.Version != tt.wantVersion {
			t.Errorf("%q: version not %q. got=%q", tt.input, tt.wantVersion, changelogStmt.Version)
		}
		if changelogStmt.Date != tt.wantDate {
			t.Errorf("%q: date not %q. got=%q", tt.input, tt.wantDate, changelogStmt.Date)
		}
	}
}

func TestParser_ChangelogPromotionErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing path", `promote changelog to version "1.5.0"`},
		{"missing to version", `promote changelog "CHANGELOG.md"`},
		{"missing version string", `promote changelog "CHANGELOG.md" to version`},
		{"missing on date", `promote changelog "CHANGELOG.md" to version "1.5.0" on`},
	}

	for _, tt := range tests {
		input := "version: 2.0\n\ntask \"demo\":\n  " + tt.input + "\n"

		l := lexer.NewLexer(input)
		p := NewParser(l)
		p.ParseProgram()

		if len(p.Errors()) == 0 {
			t.Errorf("%s (%q): expected parser errors, got none", tt.name, tt.input)
		}
	}
}
