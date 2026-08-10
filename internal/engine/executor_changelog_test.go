package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

const changelogFixture = `# Changelog

## [Unreleased]

### Added

- New thing.

[Unreleased]: https://github.com/acme/widget/compare/v1.4.0...HEAD
`

func TestExecuteChangelogPromoteAndDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(changelogFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &ExecutionContext{Variables: map[string]string{"release_version": "1.5.0"}}
	out := &bytes.Buffer{}
	e := NewEngine(out)

	e.SetDryRun(true)
	stmt := &statement.Changelog{Path: path, Version: "{$release_version}", Date: "2026-08-10"}
	if err := e.executeChangelog(stmt, ctx); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[DRY RUN]") {
		t.Fatalf("dry run output = %q", out.String())
	}
	data, _ := os.ReadFile(path)
	if string(data) != changelogFixture {
		t.Fatal("dry run mutated file")
	}

	e.SetDryRun(false)
	if err := e.executeChangelog(stmt, ctx); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	got := string(data)
	for _, want := range []string{
		"## [Unreleased]\n\n### Added\n\n## [1.5.0] - 2026-08-10",
		"[Unreleased]: https://github.com/acme/widget/compare/v1.5.0...HEAD",
		"[1.5.0]: https://github.com/acme/widget/compare/v1.4.0...v1.5.0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("promoted changelog missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestExecuteChangelogDefaultsToToday(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(changelogFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEngine(&bytes.Buffer{})
	ctx := &ExecutionContext{Variables: map[string]string{}}
	if err := e.executeChangelog(&statement.Changelog{Path: path, Version: "1.5.0"}, ctx); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := "## [1.5.0] - " + time.Now().Format("2006-01-02")
	if !strings.Contains(string(data), want) {
		t.Fatalf("promoted changelog missing %q\ngot:\n%s", want, data)
	}
}

func TestExecuteChangelogErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(changelogFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEngine(&bytes.Buffer{})
	ctx := &ExecutionContext{Variables: map[string]string{}}

	if err := e.executeChangelog(&statement.Changelog{Path: path, Version: "1.5.0", Date: "2026-02-30"}, ctx); err == nil ||
		!strings.Contains(err.Error(), "not a valid calendar date") {
		t.Fatalf("invalid date error = %v", err)
	}
	if err := e.executeChangelog(&statement.Changelog{Path: path, Version: "banana"}, ctx); err == nil ||
		!strings.Contains(err.Error(), "not a semantic version") {
		t.Fatalf("invalid version error = %v", err)
	}
	if err := e.executeChangelog(&statement.Changelog{Path: filepath.Join(dir, "MISSING.md"), Version: "1.5.0"}, ctx); err == nil {
		t.Fatal("missing file: expected error, got none")
	}
}

func TestExecuteChangelogRerunIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(changelogFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	e := NewEngine(&bytes.Buffer{})
	ctx := &ExecutionContext{Variables: map[string]string{}}
	stmt := &statement.Changelog{Path: path, Version: "1.5.0", Date: "2026-08-10"}

	if err := e.executeChangelog(stmt, ctx); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)

	// Re-running the same release preparation with an emptied Unreleased
	// section must not fail or change the file.
	if err := e.executeChangelog(stmt, ctx); err != nil {
		t.Fatalf("re-run with empty unreleased section: %v", err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("re-run mutated the changelog")
	}

	// Entries added after the first run merge into the existing section.
	updated := strings.Replace(string(second), "## [Unreleased]\n\n### Added",
		"## [Unreleased]\n\n### Added\n\n- Late addition.", 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := e.executeChangelog(stmt, ctx); err != nil {
		t.Fatalf("re-run with new entries: %v", err)
	}
	merged, _ := os.ReadFile(path)
	text := string(merged)
	if !strings.Contains(text, "- New thing.\n- Late addition.") {
		t.Fatalf("late entry was not merged into the existing section\ngot:\n%s", text)
	}
	if strings.Count(text, "## [1.5.0]") != 1 {
		t.Fatalf("expected exactly one release section\ngot:\n%s", text)
	}
	if strings.Contains(text, "## [1.5.0] - 2026-08-10") == false {
		t.Fatalf("existing release date must be preserved\ngot:\n%s", text)
	}
}

func TestChangelogPromotionEndToEnd(t *testing.T) {
	dir := t.TempDir()
	changelogPath := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(changelogPath, []byte(changelogFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	source := `version: 2.0

task "release":
  requires $next
  promote changelog "` + changelogPath + `" to version "{$next}" on "2026-08-10"
`
	specPath := filepath.Join(dir, "release.drun")
	if err := os.WriteFile(specPath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	p := parser.NewParser(lexer.NewLexer(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	engine := NewEngine(&bytes.Buffer{})
	if err := engine.ExecuteWithParamsAndFile(program, "release", map[string]string{"next": "1.5.0"}, specPath); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, _ := os.ReadFile(changelogPath)
	if !strings.Contains(string(data), "## [1.5.0] - 2026-08-10") {
		t.Fatalf("end-to-end promotion failed\ngot:\n%s", data)
	}
}
