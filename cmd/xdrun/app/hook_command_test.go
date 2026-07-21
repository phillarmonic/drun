package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSupportedHooksIncludesPreCommit(t *testing.T) {
	want := []string{"pre-commit", "commit-msg", "pre-push"}
	if len(supportedHooks) != len(want) {
		t.Fatalf("supportedHooks = %#v, want %#v", supportedHooks, want)
	}
	for i := range want {
		if supportedHooks[i] != want[i] {
			t.Fatalf("supportedHooks = %#v, want %#v", supportedHooks, want)
		}
	}
}

func TestInstallHookWritesManagedPreCommitScript(t *testing.T) {
	gitDir := filepath.Join(t.TempDir(), "hooks")
	if err := os.MkdirAll(gitDir, 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := installHook(gitDir, "pre-commit"); err != nil {
		t.Fatalf("installHook() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(gitDir, "pre-commit"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	script := string(content)
	if !strings.Contains(script, "# managed by drun") {
		t.Fatalf("pre-commit hook missing managed marker:\n%s", script)
	}
	if !strings.Contains(script, `xdrun cmd:hook run pre-commit "$@"`) {
		t.Fatalf("pre-commit hook does not run drun pre-commit validation:\n%s", script)
	}
}

func TestHookCommandInstallListUninstallPreCommitLifecycle(t *testing.T) {
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")

	withWorkingDir(t, repoDir, func() {
		runHookCommand(t, "install", "pre-commit")

		hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-commit")
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if !strings.Contains(string(content), `xdrun cmd:hook run pre-commit "$@"`) {
			t.Fatalf("pre-commit hook script did not delegate to drun:\n%s", string(content))
		}

		listOutput := captureStdout(t, func() {
			runHookCommand(t, "list")
		})
		if !strings.Contains(listOutput, "pre-commit") || !strings.Contains(listOutput, "Installed (managed by drun)") {
			t.Fatalf("hook list output did not show managed pre-commit hook:\n%s", listOutput)
		}

		runHookCommand(t, "uninstall", "pre-commit")
		if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
			t.Fatalf("pre-commit hook still exists after uninstall, stat error = %v", err)
		}
	})
}

func TestHookRunPreCommitBlocksProtectedBranch(t *testing.T) {
	repoDir := t.TempDir()
	initGitRepoWithCommit(t, repoDir, "main")
	spec := `version: 2.0

project "guarded":
  git policy:
    branch:
      protected branches: "main"

task "noop":
  info "noop"
`
	writeSpec(t, repoDir, spec)

	withWorkingDir(t, repoDir, func() {
		app := NewApp("test", "test", "test")
		app.rootCmd.SetArgs([]string{"cmd:hook", "run", "pre-commit"})
		err := app.rootCmd.Execute()
		if err == nil {
			t.Fatal("pre-commit on protected branch succeeded, want error")
		}
		if !strings.Contains(err.Error(), "branch 'main' is protected") {
			t.Fatalf("pre-commit error = %v, want protected branch message", err)
		}
	})
}

func TestHookRunPreCommitPassesWithoutProtectedBranches(t *testing.T) {
	repoDir := t.TempDir()
	initGitRepoWithCommit(t, repoDir, "main")
	spec := `version: 2.0

project "unguarded":
  git policy:
    branch:
      default branches: "main"

task "noop":
  info "noop"
`
	writeSpec(t, repoDir, spec)

	withWorkingDir(t, repoDir, func() {
		output := captureStdout(t, func() {
			runHookCommand(t, "run", "pre-commit")
		})
		if !strings.Contains(output, "Protected branch check passed") {
			t.Fatalf("pre-commit output did not show pass message:\n%s", output)
		}
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func runHookCommand(t *testing.T, args ...string) {
	t.Helper()

	app := NewApp("test", "test", "test")
	app.rootCmd.SetArgs(append([]string{"cmd:hook"}, args...))
	if err := app.rootCmd.Execute(); err != nil {
		t.Fatalf("xdrun cmd:hook %s failed: %v", strings.Join(args, " "), err)
	}
}

func initGitRepoWithCommit(t *testing.T, repoDir, branch string) {
	t.Helper()

	runGit(t, repoDir, "init")
	runGit(t, repoDir, "checkout", "-b", branch)
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "initial commit")
}

func writeSpec(t *testing.T, repoDir, spec string) {
	t.Helper()

	specDir := filepath.Join(repoDir, ".drun")
	if err := os.MkdirAll(specDir, 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "spec.drun"), []byte(spec), 0640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
