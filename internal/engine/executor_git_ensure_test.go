package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

func TestExecuteGitEnsureVersionSucceedsAndCapturesLatest(t *testing.T) {
	repository := gitEnsureTestRepository(t, []string{"runtime-1.9.0", "runtime-1.10.0", "runtime-2.0.0-rc1"})
	source := gitEnsureTestSource(repository, `git ensure $release_version is newer than latest version from runtime
    using filesystem
    matching tags "runtime-{version}"
    as $latest_version
  info "Latest was {$latest_version}"`)
	program := parseGitEnsureTestProgram(t, source)
	output := &bytes.Buffer{}
	if err := NewEngine(output).ExecuteWithParams(program, "release", map[string]string{"release_version": "1.11.0"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Latest was 1.10.0") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestExecuteGitEnsureVersionFailureModes(t *testing.T) {
	repository := gitEnsureTestRepository(t, []string{"v1.2.3", "v1.10.0"})
	tests := []struct {
		name      string
		candidate string
		want      string
	}{
		{name: "equal", candidate: "1.10.0", want: "already tagged"},
		{name: "older", candidate: "1.9.9", want: `older than latest version "1.10.0"`},
		{name: "invalid", candidate: "v1.11.0", want: "not a stable MAJOR.MINOR.PATCH version"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			program := parseGitEnsureTestProgram(t, gitEnsureTestSource(repository, `git ensure $release_version is newer than latest version from runtime using filesystem`))
			err := NewEngine(&bytes.Buffer{}).ExecuteWithParams(program, "release", map[string]string{"release_version": test.candidate})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestExecuteGitEnsureVersionNoStableTagsIsResolutionError(t *testing.T) {
	repository := gitEnsureTestRepository(t, []string{"latest", "v2.0.0-rc1"})
	program := parseGitEnsureTestProgram(t, gitEnsureTestSource(repository, `git ensure "2.0.0" is newer than latest version from runtime using filesystem`))
	err := NewEngine(&bytes.Buffer{}).Execute(program, "release")
	if err == nil || !strings.Contains(err.Error(), "cannot resolve latest stable version") || !strings.Contains(err.Error(), "no stable version tags") {
		t.Fatalf("error = %v", err)
	}
}

func TestExecuteGitEnsureVersionDryRunDoesNotOpenOrCapture(t *testing.T) {
	program := parseGitEnsureTestProgram(t, gitEnsureTestSource("/definitely/missing", `git ensure "2.0.0" is newer than latest version from runtime as $latest_version`))
	output := &bytes.Buffer{}
	engine := NewEngine(output)
	engine.SetDryRun(true)
	if err := engine.Execute(program, "release"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "left unchanged") {
		t.Fatalf("output = %q", output.String())
	}
}

func parseGitEnsureTestProgram(t *testing.T, source string) *ast.Program {
	t.Helper()
	p := parser.NewParser(lexer.NewLexer(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v\n%s", p.Errors(), source)
	}
	return program
}

func gitEnsureTestSource(repository, statement string) string {
	return fmt.Sprintf(`version: 2.0
project "release":
  scm:
    git:
      generic:
        runtime:
          default: https
          https: "https://example.test/runtime.git"
          filesystem: %q
task "release":
  given $release_version defaults to "2.0.0"
  %s
`, repository, statement)
}

func gitEnsureTestRepository(t *testing.T, tags []string) string {
	t.Helper()
	repository := t.TempDir()
	gitQueryTestCommand(t, repository, "init")
	gitQueryTestCommand(t, repository, "config", "user.name", "Drun Test")
	gitQueryTestCommand(t, repository, "config", "user.email", "drun@example.test")
	if err := os.WriteFile(filepath.Join(repository, "README"), []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitQueryTestCommand(t, repository, "add", "README")
	gitQueryTestCommand(t, repository, "commit", "-m", "initial")
	for _, tag := range tags {
		gitQueryTestCommand(t, repository, "tag", tag)
	}
	return repository
}
