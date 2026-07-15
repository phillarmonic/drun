package engine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

func TestExecuteGitQueryCapturesLatestVersion(t *testing.T) {
	repository := t.TempDir()
	gitQueryTestCommand(t, repository, "init")
	gitQueryTestCommand(t, repository, "config", "user.name", "Drun Test")
	gitQueryTestCommand(t, repository, "config", "user.email", "drun@example.test")
	if err := os.WriteFile(filepath.Join(repository, "README"), []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitQueryTestCommand(t, repository, "add", "README")
	gitQueryTestCommand(t, repository, "commit", "-m", "initial")
	for _, tag := range []string{"php-8.4.22", "php-8.4.23", "php-8.4.24RC", "php-8.5.0"} {
		gitQueryTestCommand(t, repository, "tag", tag)
	}
	source := fmt.Sprintf(`version: 2.0
project "php":
  scm:
    git:
      generic:
        php:
          filesystem: %q
          version tags: "php-{version}"
task "latest":
  git get latest version from php in series "8.4" as $php_version
  info "PHP {$php_version}"
`, repository)
	p := parser.NewParser(lexer.NewLexer(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	output := &bytes.Buffer{}
	if err := NewEngine(output).Execute(program, "latest"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("PHP 8.4.23")) {
		t.Fatalf("output = %q", output.String())
	}
}

func TestExecuteGitQueryDryRunDoesNotOpenSource(t *testing.T) {
	source := `version: 2.0
project "app":
  scm:
    git:
      generic:
        app:
          filesystem: "/definitely/missing"
task "latest":
  git get latest tag from app as $tag
`
	p := parser.NewParser(lexer.NewLexer(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	engine := NewEngine(&bytes.Buffer{})
	engine.SetDryRun(true)
	if err := engine.Execute(program, "latest"); err != nil {
		t.Fatal(err)
	}
}

func gitQueryTestCommand(t *testing.T, directory string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}
