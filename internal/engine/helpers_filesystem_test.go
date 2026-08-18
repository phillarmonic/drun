package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFilesystemPathExpandsHomeAndEnvironmentVariables(t *testing.T) {
	home := t.TempDir()
	workingDir := t.TempDir()
	environmentRoot := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DRUN_TEST_PATH_ROOT", environmentRoot)

	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{OriginalWorkingDir: workingDir}
	tests := map[string]struct {
		path string
		want string
	}{
		"home directory": {
			path: "~",
			want: home,
		},
		"home-relative path": {
			path: "~/Library/Audio/Plug-Ins/VST3",
			want: filepath.Join(home, "Library", "Audio", "Plug-Ins", "VST3"),
		},
		"unbraced environment variable": {
			path: "$DRUN_TEST_PATH_ROOT/plugins",
			want: filepath.Join(environmentRoot, "plugins"),
		},
		"braced environment variable": {
			path: "${DRUN_TEST_PATH_ROOT}/plugins",
			want: filepath.Join(environmentRoot, "plugins"),
		},
		"undefined environment variable stays literal": {
			path: "$DRUN_UNDEFINED_PATH/plugins",
			want: filepath.Join(workingDir, "$DRUN_UNDEFINED_PATH", "plugins"),
		},
		"named user home stays literal": {
			path: "~generic/plugins",
			want: filepath.Join(workingDir, "~generic", "plugins"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := engine.resolveFilesystemPath(test.path, ctx); got != test.want {
				t.Fatalf("resolveFilesystemPath(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}

func TestFilesystemConditionsExpandHomeAndEnvironmentVariables(t *testing.T) {
	home := t.TempDir()
	pluginDir := filepath.Join(home, "Library", "Audio", "Plug-Ins", "VST3")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{OriginalWorkingDir: t.TempDir()}
	for _, condition := range []string{
		`folder "~/Library/Audio/Plug-Ins/VST3" exists`,
		`folder "$HOME/Library/Audio/Plug-Ins/VST3" exists`,
		`folder "${HOME}/Library/Audio/Plug-Ins/VST3" exists`,
	} {
		t.Run(condition, func(t *testing.T) {
			if !engine.evaluateCondition(condition, ctx) {
				t.Fatalf("condition %q should be true", condition)
			}
		})
	}
}

func TestCreateFileAndDirectoryExpandPathsRecursively(t *testing.T) {
	home := t.TempDir()
	environmentRoot := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DRUN_CREATE_ROOT", environmentRoot)

	program, err := ParseString(`version: 2.0

task "create paths":
  create directory "~/nested/from-home"
  create file "$DRUN_CREATE_ROOT/nested/unbraced/file.txt"
  create directory "${DRUN_CREATE_ROOT}/nested/braced/directory"`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if err := NewEngine(io.Discard).Execute(program, "create paths"); err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	for _, path := range []string{
		filepath.Join(home, "nested", "from-home"),
		filepath.Join(environmentRoot, "nested", "braced", "directory"),
	} {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Fatalf("expected recursively created directory %q, stat error: %v", path, err)
		}
	}
	file := filepath.Join(environmentRoot, "nested", "unbraced", "file.txt")
	info, err := os.Stat(file)
	if err != nil || info.IsDir() {
		t.Fatalf("expected recursively created file %q, stat error: %v", file, err)
	}
}

func TestFilesystemNounsAndMissingPredicatesAreInterchangeable(t *testing.T) {
	root := t.TempDir()
	existingDirectory := filepath.Join(root, "existing-directory")
	if err := os.Mkdir(existingDirectory, 0o755); err != nil {
		t.Fatalf("create existing directory: %v", err)
	}
	existingFile := filepath.Join(root, "existing-file")
	if err := os.WriteFile(existingFile, []byte("present"), 0o600); err != nil {
		t.Fatalf("create existing file: %v", err)
	}
	missing := filepath.Join(root, "missing")
	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{}

	for noun, existing := range map[string]string{
		"file":      existingFile,
		"folder":    existingDirectory,
		"directory": existingDirectory,
		"dir":       existingDirectory,
	} {
		for _, predicate := range []string{"not exists", "does not exist"} {
			condition := fmt.Sprintf(`%s %q %s`, noun, missing, predicate)
			if !engine.evaluateCondition(condition, ctx) {
				t.Errorf("condition %q should be true", condition)
			}
			condition = fmt.Sprintf(`%s %q %s`, noun, existing, predicate)
			if engine.evaluateCondition(condition, ctx) {
				t.Errorf("condition %q should be false", condition)
			}
		}
	}
}
