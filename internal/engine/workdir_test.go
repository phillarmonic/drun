package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/lexer"
	parserPkg "github.com/phillarmonic/drun/internal/parser"
)

// parseForWorkdirTest is a helper to parse a drun string in engine tests.
func parseForWorkdirTest(t *testing.T, input string) *ast.Program {
	t.Helper()
	l := lexer.NewLexer(input)
	p := parserPkg.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}
	if program == nil {
		t.Fatal("ParseProgram() returned nil")
	}
	return program
}

// TestUseWorkdirExecution verifies that `use workdir` actually changes the
// working directory used by subsequent shell commands in the same task.
func TestUseWorkdirExecution(t *testing.T) {
	// Create a temporary directory with a known file inside a subdirectory
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "mysubdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a sentinel file in subDir
	sentinelFile := filepath.Join(subDir, "sentinel.txt")
	if err := os.WriteFile(sentinelFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to tmpDir so we have a known cwd
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	input := `version: 2.0

task "check":
    use workdir "mysubdir"
    run "ls sentinel.txt"
`
	program := parseForWorkdirTest(t, input)

	var out bytes.Buffer
	eng := NewEngine(&out)
	if err := eng.Execute(program, "check"); err != nil {
		t.Fatalf("Execution failed: %v\nOutput:\n%s", err, out.String())
	}

	if !strings.Contains(out.String(), "sentinel.txt") {
		t.Errorf("Expected output to contain 'sentinel.txt', got:\n%s", out.String())
	}
}

// TestUseWorkdirDoesNotLeak verifies that workdir changes are scoped to the
// task that sets them and don't affect subsequent tasks.
func TestUseWorkdirDoesNotLeak(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	input := `version: 2.0

task "setter":
    use workdir "subdir"
    run "pwd"

task "checker":
    depends on setter
    capture from shell "pwd" as $current
    info "pwd: {$current}"
`
	program := parseForWorkdirTest(t, input)

	var out bytes.Buffer
	eng := NewEngine(&out)
	if err := eng.Execute(program, "checker"); err != nil {
		t.Fatalf("Execution failed: %v\nOutput:\n%s", err, out.String())
	}

	output := out.String()
	// The info line should print tmpDir, NOT subDir
	if strings.Contains(output, "pwd: "+subDir) {
		t.Errorf("workdir leaked into dependent task! Output:\n%s", output)
	}
	if !strings.Contains(output, tmpDir) {
		t.Errorf("Expected tmpDir %q in output, got:\n%s", tmpDir, output)
	}
}

// TestUseWorkdirAbsolutePath verifies absolute paths work directly.
func TestUseWorkdirAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "abs")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "marker"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Use the absolute path directly in the drun source
	input := `version: 2.0

task "abs-test":
    use workdir "` + subDir + `"
    run "ls marker"
`
	program := parseForWorkdirTest(t, input)

	var out bytes.Buffer
	eng := NewEngine(&out)
	if err := eng.Execute(program, "abs-test"); err != nil {
		t.Fatalf("Execution failed: %v\nOutput:\n%s", err, out.String())
	}

	if !strings.Contains(out.String(), "marker") {
		t.Errorf("Expected output to contain 'marker', got:\n%s", out.String())
	}
}

// TestUseWorkdirNonExistentFails verifies that using a nonexistent directory produces an error.
func TestUseWorkdirNonExistentFails(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	input := `version: 2.0

task "fail":
    use workdir "does-not-exist"
    run "pwd"
`
	program := parseForWorkdirTest(t, input)

	var out bytes.Buffer
	eng := NewEngine(&out)
	err := eng.Execute(program, "fail")
	if err == nil {
		t.Fatal("Expected an error for nonexistent directory, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error to mention 'does not exist', got: %v", err)
	}
}

// TestUseWorkdirRelativePathsResolveFromOriginalCwd verifies that two consecutive
// `use workdir` calls with relative paths both resolve from the original cwd
// (not chained from each other).
func TestUseWorkdirRelativePathsResolveFromOriginalCwd(t *testing.T) {
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "a")
	dirB := filepath.Join(tmpDir, "b")
	for _, d := range []string{dirA, dirB} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dirA, "in-a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "in-b"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	input := `version: 2.0

task "resolve-test":
    use workdir "a"
    run "ls in-a"
    use workdir "b"
    run "ls in-b"
`
	program := parseForWorkdirTest(t, input)

	var out bytes.Buffer
	eng := NewEngine(&out)
	if err := eng.Execute(program, "resolve-test"); err != nil {
		t.Fatalf("Execution failed: %v\nOutput:\n%s", err, out.String())
	}

	output := out.String()
	if !strings.Contains(output, "in-a") {
		t.Errorf("Expected 'in-a' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "in-b") {
		t.Errorf("Expected 'in-b' in output, got:\n%s", output)
	}
}

// TestUseWorkdirDryRun verifies that dry-run mode logs the intent without errors.
func TestUseWorkdirDryRun(t *testing.T) {
	input := `version: 2.0

task "build":
    use workdir "frontend"
    run "npm run build"
`
	program := parseForWorkdirTest(t, input)

	var out bytes.Buffer
	eng := NewEngine(&out)
	eng.SetDryRun(true)
	if err := eng.Execute(program, "build"); err != nil {
		t.Fatalf("Dry-run execution failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Would set working directory") {
		t.Errorf("Expected dry-run to log workdir intent, got:\n%s", output)
	}
	if !strings.Contains(output, "frontend") {
		t.Errorf("Expected dry-run output to mention 'frontend', got:\n%s", output)
	}
}
