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

func TestCompleteTaskNamesCompletesSelectedTaskParameterWithEquals(t *testing.T) {
	withCompletionSpec(t, `
version: 2.0

task "prepare-release" means "Prepare a release":
  requires $version as string matching pattern "^[0-9]+\\.[0-9]+\\.[0-9]+$"
  given $channel from ["stable", "beta"] defaults to "stable"
  requires $port as number between 1000 and 9999
  info "ready"
`)

	app := NewApp("test", "test", "test")
	completions, directive := CompleteTaskNames(app.rootCmd, []string{"prepare-release"}, "v")

	expectedDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoSpace
	if directive != expectedDirective {
		t.Fatalf("CompleteTaskNames() directive = %v, want %v", directive, expectedDirective)
	}
	if len(completions) != 1 {
		t.Fatalf("CompleteTaskNames() completions = %#v, want one version completion", completions)
	}
	if !strings.HasPrefix(completions[0], "version=\t[parameter] required string, pattern: ") {
		t.Fatalf("CompleteTaskNames() completion = %q, want version= with required pattern metadata", completions[0])
	}
}

func TestCompleteTaskNamesCompletesUnusedParametersForPartialTaskName(t *testing.T) {
	withCompletionSpec(t, `
version: 2.0

task "prepare-release" means "Prepare a release":
  requires $version
  given $channel from ["stable", "beta"] defaults to "stable"
  requires $port as number between 1000 and 9999
  info "ready"
`)

	app := NewApp("test", "test", "test")
	completions, directive := CompleteTaskNames(app.rootCmd, []string{"prepare-r", "version=1.2.3"}, "")

	expectedDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder | cobra.ShellCompDirectiveNoSpace
	if directive != expectedDirective {
		t.Fatalf("CompleteTaskNames() directive = %v, want %v", directive, expectedDirective)
	}
	if containsCompletion(completions, "version=") {
		t.Fatalf("CompleteTaskNames() repeated an already supplied parameter: %#v", completions)
	}
	if !containsCompletion(completions, "channel=") {
		t.Fatalf("CompleteTaskNames() missing optional channel parameter: %#v", completions)
	}
	if !containsCompletion(completions, "port=") {
		t.Fatalf("CompleteTaskNames() missing required port parameter: %#v", completions)
	}
	if completion := findCompletion(completions, "channel="); !strings.Contains(completion, "optional string, default: stable, one of: stable, beta") {
		t.Fatalf("CompleteTaskNames() channel metadata = %q", completion)
	}
	if completion := findCompletion(completions, "port="); !strings.Contains(completion, "required number, range: 1000-9999") {
		t.Fatalf("CompleteTaskNames() port metadata = %q", completion)
	}
}

func TestCompleteTaskNamesDoesNotCompleteParameterValuesYet(t *testing.T) {
	withCompletionSpec(t, `
version: 2.0

task "deploy" means "Deploy":
  requires $environment from ["dev", "production"]
  info "deploying"
`)

	app := NewApp("test", "test", "test")
	completions, directive := CompleteTaskNames(app.rootCmd, []string{"deploy"}, "environment=")

	if len(completions) != 0 {
		t.Fatalf("CompleteTaskNames() completions = %#v, want no value completions", completions)
	}
	expectedDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
	if directive != expectedDirective {
		t.Fatalf("CompleteTaskNames() directive = %v, want %v", directive, expectedDirective)
	}
}

func withCompletionSpec(t *testing.T, source string) {
	t.Helper()
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
	if err := os.WriteFile(filepath.Join(".drun", "spec.drun"), []byte(source), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
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

func findCompletion(completions []string, prefix string) string {
	for _, completion := range completions {
		if strings.HasPrefix(completion, prefix+"\t") || completion == prefix {
			return completion
		}
	}
	return ""
}
