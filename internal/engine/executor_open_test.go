package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEngine_OpenDryRun(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "demo":
  open url "https://example.com"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)
	engine.SetDryRun(true)

	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "[DRY RUN] Would open") {
		t.Errorf("expected dry-run output, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "https://example.com") {
		t.Errorf("expected URL in output, got: %q", out.String())
	}
}

func TestEngine_OpenHeadlessCI(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")

	program, err := ParseString(`version: 2.0

task "demo":
  open url "https://example.com/docs"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "no desktop environment detected") {
		t.Errorf("expected headless warning, got: %q", output)
	}
	if !strings.Contains(output, "https://example.com/docs") {
		t.Errorf("expected URL in warning, got: %q", output)
	}
}

func TestEngine_OpenHeadlessSSH(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv("CONTINUOUS_INTEGRATION", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITLAB_CI", "")
	t.Setenv("JENKINS_URL", "")
	t.Setenv("TRAVIS", "")
	t.Setenv("CIRCLECI", "")
	t.Setenv("BUILDKITE", "")
	t.Setenv("TEAMCITY_VERSION", "")
	t.Setenv("SSH_CONNECTION", "10.0.0.1 56789 10.0.0.2 22")
	t.Setenv("SSH_TTY", "")

	program, err := ParseString(`version: 2.0

task "demo":
  open url "https://example.com"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "no desktop environment detected") {
		t.Errorf("expected headless warning, got: %q", out.String())
	}
}

func TestEngine_OpenInterpolation(t *testing.T) {
	t.Setenv("CI", "true")

	program, err := ParseString(`version: 2.0

task "demo":
  let $base = "https://example.com"
  open url "{$base}/docs"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "https://example.com/docs") {
		t.Errorf("expected interpolated URL in output, got: %q", out.String())
	}
}

func TestEngine_OpenEmptyTarget(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "demo":
  open url ""
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	err = engine.Execute(program, "demo")
	if err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
	if !strings.Contains(err.Error(), "target is empty") {
		t.Errorf("expected empty-target error, got: %v", err)
	}
}

func TestEngine_OpenRelativePathResolution(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "demo":
  open url "./report.html"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)
	engine.SetDryRun(true)

	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	expected, _ := filepath.Abs("./report.html")
	if !strings.Contains(out.String(), expected) {
		t.Errorf("expected absolute path %q in output, got: %q", expected, out.String())
	}
}

func TestHasDesktopSession_CITakesPrecedence(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	if runtime.GOOS == "linux" {
		t.Setenv("DISPLAY", ":0")
	}

	if hasDesktopSession() {
		t.Error("expected hasDesktopSession() = false when CI is set")
	}
}

func TestHasDesktopSession_SSHTakesPrecedence(t *testing.T) {
	// Clear all CI vars
	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("SSH_CONNECTION", "10.0.0.1 56789 10.0.0.2 22")
	if runtime.GOOS == "linux" {
		t.Setenv("DISPLAY", ":0")
	}

	if hasDesktopSession() {
		t.Error("expected hasDesktopSession() = false when SSH_CONNECTION is set")
	}
}

func TestHasDesktopSession_LinuxNeedsDisplay(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	for _, v := range ciEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_TTY", "")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")

	if hasDesktopSession() {
		t.Error("expected hasDesktopSession() = false on Linux without DISPLAY or WAYLAND_DISPLAY")
	}

	t.Setenv("DISPLAY", ":0")
	if !hasDesktopSession() {
		t.Error("expected hasDesktopSession() = true on Linux with DISPLAY set")
	}
}

func TestOpenerCommand_ReturnsCommand(t *testing.T) {
	cmd, err := openerCommand("https://example.com")

	switch runtime.GOOS {
	case "darwin":
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd.Path == "" {
			t.Error("expected non-empty command path")
		}
	case "linux":
		// xdg-open may or may not be available
		if err != nil {
			if !strings.Contains(err.Error(), "xdg-open") {
				t.Errorf("expected xdg-open error, got: %v", err)
			}
		}
	case "windows":
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestEngine_OpenManualTestGate(t *testing.T) {
	if os.Getenv("DRUN_TEST_OPEN") != "1" {
		t.Skip("set DRUN_TEST_OPEN=1 to run interactive open tests")
	}

	program, err := ParseString(`version: 2.0

task "demo":
  open url "https://example.com"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var out bytes.Buffer
	engine := NewEngine(&out)

	if err := engine.Execute(program, "demo"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	t.Logf("output: %s", out.String())
}
