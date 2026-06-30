package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteTaskNamesReturnsBuiltinCommandsWithoutConfig(t *testing.T) {
	tempRoot := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	app := NewApp("test", "test", "test")

	completions, directive := CompleteTaskNames(app.rootCmd, nil, "")
	expectedDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	if directive != expectedDirective {
		t.Fatalf("CompleteTaskNames() directive = %v, want %v", directive, expectedDirective)
	}
	if len(completions) == 0 {
		t.Fatalf("CompleteTaskNames() returned no completions")
	}
	if !containsCompletion(completions, "cmd:skill") {
		t.Fatalf("CompleteTaskNames() missing cmd:skill in %#v", completions)
	}
	if containsCompletion(completions, "cmd:completion") {
		t.Fatalf("CompleteTaskNames() should hide cmd:completion from default suggestions: %#v", completions)
	}
}

func TestCompleteTaskNamesPrefersCmdNamespaceForCmdPrefix(t *testing.T) {
	tempRoot := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := os.MkdirAll(".drun", 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(".drun", "spec.drun"), []byte(`
version: 2.0

project "completion-test" version "1.0":
task "ci" means "Run CI":
	info "ci"
`), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	app := NewApp("test", "test", "test")

	completions, _ := CompleteTaskNames(app.rootCmd, nil, "c")
	if len(completions) == 0 {
		t.Fatalf("CompleteTaskNames() returned no completions")
	}
	if !containsCompletion(completions, "cmd:completion") {
		t.Fatalf("CompleteTaskNames() missing cmd:completion in %#v", completions)
	}
	if !containsCompletion(completions, "cmd:lsp") {
		t.Fatalf("CompleteTaskNames() missing cmd:lsp in %#v", completions)
	}
	if containsCompletion(completions, "ci") {
		t.Fatalf("CompleteTaskNames() included ci task for cmd namespace prefix: %#v", completions)
	}
}

func TestCompleteTaskNamesKeepsTasksBeforeBuiltins(t *testing.T) {
	tempRoot := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := os.MkdirAll(".drun", 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(".drun", "spec.drun"), []byte(generateStarterConfig(true)), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	app := NewApp("test", "test", "test")

	completions, directive := CompleteTaskNames(app.rootCmd, nil, "")
	expectedDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	if directive != expectedDirective {
		t.Fatalf("CompleteTaskNames() directive = %v, want %v", directive, expectedDirective)
	}
	if len(completions) < 2 {
		t.Fatalf("CompleteTaskNames() returned too few completions: %#v", completions)
	}
	if !strings.HasPrefix(completions[0], "default\t") {
		t.Fatalf("CompleteTaskNames() first completion = %q, want task before builtins", completions[0])
	}
	if !containsCompletion(completions, "cmd:skill") {
		t.Fatalf("CompleteTaskNames() missing cmd:skill in %#v", completions)
	}
	if containsCompletion(completions, "cmd:completion") {
		t.Fatalf("CompleteTaskNames() should hide cmd:completion from default suggestions: %#v", completions)
	}
	if containsCompletion(completions, "cmd:dump-env") {
		t.Fatalf("CompleteTaskNames() should hide cmd:dump-env from default suggestions: %#v", completions)
	}
	if containsCompletion(completions, "cmd:lsp") {
		t.Fatalf("CompleteTaskNames() should hide cmd:lsp from default suggestions: %#v", completions)
	}
}

func containsCompletion(completions []string, prefix string) bool {
	for _, completion := range completions {
		if strings.HasPrefix(completion, prefix+"\t") || completion == prefix {
			return true
		}
	}
	return false
}
