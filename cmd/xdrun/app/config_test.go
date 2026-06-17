package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

func TestInferProjectNameFromWorkingDirUsesFolderName(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "sample-service")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if got := inferProjectNameFromWorkingDir(); got != "sample-service" {
		t.Fatalf("inferProjectNameFromWorkingDir() = %q, want %q", got, "sample-service")
	}
}

func TestGenerateStarterConfigUsesWorkingDirectoryName(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "starter-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	config := generateStarterConfig(false)
	if !strings.Contains(config, `project "starter-app" version "1.0":`) {
		t.Fatalf("generateStarterConfig() did not embed working directory name:\n%s", config)
	}

	assertGeneratedConfigParses(t, config)
}

func TestGenerateStarterConfigMinimalContainsOnlyWelcomeTask(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "minimal-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	config := generateStarterConfig(true)
	if !strings.Contains(config, `project "minimal-app" version "1.0":`) {
		t.Fatalf("generateStarterConfig(true) did not embed working directory name:\n%s", config)
	}
	if !strings.Contains(config, `task "default" means "Welcome":`) {
		t.Fatalf("generateStarterConfig(true) did not include default welcome task:\n%s", config)
	}
	if strings.Contains(config, `task "hello" means "Say hello":`) {
		t.Fatalf("generateStarterConfig(true) should not include extra tasks:\n%s", config)
	}

	assertGeneratedConfigParses(t, config)
}

func assertGeneratedConfigParses(t *testing.T, config string) {
	t.Helper()

	l := lexer.NewLexer(config)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if program == nil {
		t.Fatal("generated config did not parse")
	}
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("generated config parse errors: %v\n%s", errs, config)
	}
}
