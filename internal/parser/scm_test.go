package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
)

func parseSCMTestProgram(t *testing.T, input string) *ast.Program {
	t.Helper()
	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()
	if errors := p.Errors(); len(errors) != 0 {
		t.Fatalf("parser errors: %v", errors)
	}
	return program
}

func parseSCMTestErrors(input string) []string {
	l := lexer.NewLexer(input)
	p := NewParser(l)
	p.ParseProgram()
	return p.Errors()
}

func TestParseSCMRegistryAndGitQuery(t *testing.T) {
	program := parseSCMTestProgram(t, `version: 2.0
project "php-release":
  scm:
    git:
      generic:
        php:
          https: "https://github.com/php/php-src.git"
          version tags: "php-{version}"

task "latest":
  git get latest version from php in series "8.4" as $php_version
`)
	registry, ok := program.Project.Settings[0].(*ast.SCMRegistryStatement)
	if !ok {
		t.Fatalf("setting = %T", program.Project.Settings[0])
	}
	source := registry.Technologies["git"].Providers["generic"].Sources["php"]
	if source.Default != "https" || source.VersionTags.Formats[0] != "php-{version}" {
		t.Fatalf("source = %#v", source)
	}
	query, ok := program.Tasks[0].Body[0].(*ast.GitQueryStatement)
	if !ok {
		t.Fatalf("query = %T", program.Tasks[0].Body[0])
	}
	if query.Result != "version" || query.Source != "php" || query.Series != "8.4" || query.CaptureVar != "php_version" {
		t.Fatalf("query = %#v", query)
	}
}

func TestParseExpandedSCMSource(t *testing.T) {
	program := parseSCMTestProgram(t, `version: 2.0
project "app":
  scm:
    git:
      github:
        app:
          default: ssh
          metadata: fetch
          https:
            url: "https://github.com/example/app.git"
            authentication: ambient
          ssh:
            url: "git@github.com:example/app.git"
            key: "~/.ssh/id_ed25519"
          cli:
            repository: "example/app"
            host: "github.com"
task "latest":
  git get latest tag from app using cli ordered by date as $tag
`)
	source := program.Project.Settings[0].(*ast.SCMRegistryStatement).Technologies["git"].Providers["github"].Sources["app"]
	if source.Default != "ssh" || source.Access["cli"].Host != "github.com" || source.Access["https"].Authentication != "ambient" {
		t.Fatalf("source = %#v", source)
	}
}

func TestParseExpandedVersionTagFormats(t *testing.T) {
	program := parseSCMTestProgram(t, `version: 2.0
project "app":
  scm:
    git:
      generic:
        app:
          filesystem: "."
          version tags:
            formats:
              "release-{version}"
              "legacy-{version}"
task "latest":
  git get latest version from app matching tags "runtime-{version}" matching version ">=8.4.0 <8.5.0" as $version
`)
	source := program.Project.Settings[0].(*ast.SCMRegistryStatement).Technologies["git"].Providers["generic"].Sources["app"]
	if len(source.VersionTags.Formats) != 2 || source.VersionTags.Formats[1] != "legacy-{version}" {
		t.Fatalf("formats = %#v", source.VersionTags.Formats)
	}
	query := program.Tasks[0].Body[0].(*ast.GitQueryStatement)
	if query.TagFormat != "runtime-{version}" || query.VersionMatcher != ">=8.4.0 <8.5.0" {
		t.Fatalf("query = %#v", query)
	}
}

func TestSCMValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{
			name: "multiple methods need default",
			body: `https: "https://example.test/app.git"
          ssh: "git@example.test:app.git"`,
			message: "requires default",
		},
		{
			name: "credentials cannot be configured",
			body: `https:
            url: "https://example.test/app.git"
            authentication: password`,
			message: "authentication must be ambient",
		},
		{
			name: "format requires version",
			body: `filesystem: "."
          version tags: "php-release"`,
			message: "exactly one {version}",
		},
		{
			name: "raw pattern requires named capture",
			body: `filesystem: "."
          version tags:
            pattern: "^v[0-9]+$"`,
			message: "named capture called version",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := "version: 2.0\nproject \"app\":\n  scm:\n    git:\n      generic:\n        app:\n          " + test.body + "\n"
			errors := strings.Join(parseSCMTestErrors(input), "\n")
			if !strings.Contains(errors, test.message) {
				t.Fatalf("errors = %q, want %q", errors, test.message)
			}
		})
	}
}

func TestGitQueryValidationErrors(t *testing.T) {
	base := `version: 2.0
project "app":
  scm:
    git:
      generic:
        app:
          filesystem: "."
task "latest":
  %s
`
	tests := []struct {
		statement string
		message   string
	}{
		{`git get latest version from app in series "8.4" matching version ">=8.0.0" as $version`, "mutually exclusive"},
		{`git get latest version from app in series "8.4.2" as $version`, "major or major.minor"},
		{`git get latest version from app matching version "~8.4" as $version`, "invalid version constraint"},
		{`git get latest version from app ordered by alphabetically as $version`, "version or date"},
		{`git get latest version from app matching tags unknown_preset as $version`, "unknown version tag preset"},
		{`git get latest version from app matching tags pattern "^v[0-9]+$" as $version`, "named capture called version"},
	}
	for _, test := range tests {
		errors := strings.Join(parseSCMTestErrors(fmt.Sprintf(base, test.statement)), "\n")
		if !strings.Contains(errors, test.message) {
			t.Errorf("%s: errors = %q, want %q", test.statement, errors, test.message)
		}
	}
}

func TestParseMultilineGitQuery(t *testing.T) {
	program := parseSCMTestProgram(t, `version: 2.0
project "app":
  scm:
    git:
      generic:
        app:
          filesystem: "."
task "latest":
  git get latest version from app
    matching tags "runtime-{version}"
    in series "8.4"
    ordered by date
    allow fetch
    as $version
`)
	query := program.Tasks[0].Body[0].(*ast.GitQueryStatement)
	if query.TagFormat != "runtime-{version}" || query.Series != "8.4" || query.OrderBy != "date" || !query.AllowFetch || query.CaptureVar != "version" {
		t.Fatalf("query = %#v", query)
	}
}

func TestParseGitEnsureVersion(t *testing.T) {
	program := parseSCMTestProgram(t, `version: 2.0
project "app":
  scm:
    git:
      generic:
        runtime:
          default: https
          https: "https://example.test/runtime.git"
          ssh: "git@example.test:runtime.git"
task "release":
  git ensure $release_version is newer than latest version from runtime
    using ssh
    matching tags "runtime-{version}"
    as $latest_version
`)
	guard, ok := program.Tasks[0].Body[0].(*ast.GitEnsureVersionStatement)
	if !ok {
		t.Fatalf("statement = %T", program.Tasks[0].Body[0])
	}
	if guard.Candidate != "$release_version" || !guard.CandidateIsVariable || guard.Source != "runtime" || guard.AccessMethod != "ssh" || guard.TagFormat != "runtime-{version}" || guard.CaptureVar != "latest_version" {
		t.Fatalf("guard = %#v", guard)
	}
	if got := guard.String(); got != `git ensure $release_version is newer than latest version from runtime using ssh matching tags "runtime-{version}" as $latest_version` {
		t.Fatalf("String() = %q", got)
	}
}

func TestParseMinimalGitEnsureVersionWithoutCapture(t *testing.T) {
	program := parseSCMTestProgram(t, `version: 2.0
project "app":
  scm:
    git:
      generic:
        app:
          filesystem: "."
task "release":
  git ensure "2.0.0" is newer than latest version from app
  info "continues"
`)
	if len(program.Tasks[0].Body) != 2 {
		t.Fatalf("body length = %d, want 2", len(program.Tasks[0].Body))
	}
	guard := program.Tasks[0].Body[0].(*ast.GitEnsureVersionStatement)
	if guard.Candidate != "2.0.0" || guard.CandidateIsVariable || guard.CaptureVar != "" {
		t.Fatalf("guard = %#v", guard)
	}
}

func TestGitEnsureVersionValidationErrors(t *testing.T) {
	base := `version: 2.0
project "app":
  scm:
    git:
      generic:
        app:
          filesystem: "."
task "release":
  %s
`
	tests := []struct {
		statement string
		message   string
	}{
		{`git ensure 2.0 is newer than latest version from app`, "variable or string"},
		{`git ensure "2.0.0" is newer than latest tag from app`, `expected "version"`},
		{`git ensure "2.0.0" is newer than latest version from app in series "2"`, "unexpected git ensure modifier"},
		{`git ensure "2.0.0" is newer than latest version from app matching version ">1.0.0"`, `expected "tags"`},
		{`git ensure "2.0.0" is newer than latest version from app matching tags unknown`, "unknown version tag preset"},
		{`git ensure "2.0.0" is newer than latest version from app matching tags pattern "^v[0-9]+$"`, "named capture called version"},
		{`git ensure "2.0.0" is newer than latest version from app matching tags semver using filesystem`, "must appear once before"},
	}
	for _, test := range tests {
		errors := strings.Join(parseSCMTestErrors(fmt.Sprintf(base, test.statement)), "\n")
		if !strings.Contains(errors, test.message) {
			t.Errorf("%s: errors = %q, want %q", test.statement, errors, test.message)
		}
	}
}

func TestSCMAliasesAreUniqueWithinTechnology(t *testing.T) {
	errors := strings.Join(parseSCMTestErrors(`version: 2.0
project "app":
  scm:
    git:
      github:
        shared:
          https: "https://github.com/example/shared.git"
      generic:
        shared:
          filesystem: "."
`), "\n")
	if !strings.Contains(errors, `duplicate git SCM alias "shared"`) {
		t.Fatalf("errors = %q", errors)
	}
}
