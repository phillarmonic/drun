package shell

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func usesPowerShell(shellPath string) bool {
	shellPath = strings.ToLower(filepath.Base(shellPath))
	return shellPath == "powershell.exe" || shellPath == "pwsh.exe"
}

func TestExecuteSimple(t *testing.T) {
	// Test simple command execution
	output, err := ExecuteSimple("echo 'Hello, World!'")
	if err != nil {
		t.Fatalf("ExecuteSimple failed: %v", err)
	}

	expected := "Hello, World!"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestExecute_Success(t *testing.T) {
	opts := DefaultOptions()
	opts.CaptureOutput = true

	result, err := Execute("echo 'test output'", opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected command to succeed")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "test output" {
		t.Errorf("Expected 'test output', got %q", result.Stdout)
	}

	if result.Duration <= 0 {
		t.Errorf("Expected positive duration, got %v", result.Duration)
	}
}

func TestExecute_Failure(t *testing.T) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.IgnoreErrors = true // Don't return error for non-zero exit

	result, err := Execute("exit 1", opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Errorf("Expected command to fail")
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}
}

func TestExecute_WithEnvironment(t *testing.T) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.Environment = map[string]string{
		"TEST_VAR": "test_value_123",
	}

	var cmd string
	if usesPowerShell(opts.Shell) {
		cmd = "echo $env:TEST_VAR"
	} else {
		cmd = "echo $TEST_VAR"
	}

	result, err := Execute(cmd, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Stdout != "test_value_123" {
		t.Errorf("Expected 'test_value_123', got %q", result.Stdout)
	}
}

func TestExecute_WithWorkingDir(t *testing.T) {
	opts := DefaultOptions()
	opts.CaptureOutput = true

	var expectedDir, cmd string
	if runtime.GOOS == "windows" {
		// Use a directory that exists on Windows regardless of the selected shell.
		expectedDir = os.Getenv("TEMP")
		if expectedDir == "" {
			expectedDir = "C:\\Windows\\Temp"
		}
		opts.WorkingDir = expectedDir
	} else {
		expectedDir = "/tmp"
		opts.WorkingDir = expectedDir
	}

	if usesPowerShell(opts.Shell) {
		cmd = "Get-Location | Select-Object -ExpandProperty Path"
	} else {
		cmd = "pwd"
	}

	result, err := Execute(cmd, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if runtime.GOOS == "windows" {
		// On Windows, check if we're in a temp directory (path normalization can vary)
		actualPath := strings.TrimSpace(result.Stdout)
		lowerPath := strings.ToLower(actualPath)
		if usesPowerShell(opts.Shell) {
			if !strings.Contains(lowerPath, "temp") {
				t.Errorf("Expected output to contain 'temp' directory, got %q", actualPath)
			}
		} else if lowerPath != "/tmp" && !strings.Contains(lowerPath, "temp") {
			t.Errorf("Expected output to contain 'temp' directory, got %q", actualPath)
		}
	} else {
		if result.Stdout != expectedDir {
			t.Errorf("Expected %q, got %q", expectedDir, result.Stdout)
		}
	}
}

func TestExecute_WithTimeout(t *testing.T) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.Timeout = 100 * time.Millisecond
	opts.IgnoreErrors = true

	start := time.Now()
	result, err := Execute("sleep 1", opts)
	duration := time.Since(start)

	// Command should be killed by timeout
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Errorf("Expected command to fail due to timeout")
	}

	// Should complete within reasonable time (allowing some buffer for CI environments)
	if duration > 2*time.Second {
		t.Errorf("Command took too long, timeout may not be working: %v", duration)
	}
}

func TestExecuteWithOutput(t *testing.T) {
	var output bytes.Buffer

	result, err := ExecuteWithOutput("echo 'streaming test'", &output)
	if err != nil {
		t.Fatalf("ExecuteWithOutput failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected command to succeed")
	}

	// Check that output was streamed
	outputStr := output.String()
	if !strings.Contains(outputStr, "streaming test") {
		t.Errorf("Expected output to contain 'streaming test', got %q", outputStr)
	}

	// Check that result also captured the output
	if result.Stdout != "streaming test" {
		t.Errorf("Expected captured output 'streaming test', got %q", result.Stdout)
	}
}

func TestExecute_MultilineOutput(t *testing.T) {
	opts := DefaultOptions()
	opts.CaptureOutput = true

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo line1; echo line2; echo line3"
	} else {
		cmd = "echo 'line1'; echo 'line2'; echo 'line3'"
	}

	result, err := Execute(cmd, opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Normalize line endings for cross-platform compatibility
	normalizedOutput := strings.ReplaceAll(result.Stdout, "\r\n", "\n")
	expected := "line1\nline2\nline3"
	if normalizedOutput != expected {
		t.Errorf("Expected %q, got %q (normalized: %q)", expected, result.Stdout, normalizedOutput)
	}
}

func TestExecute_StderrCapture(t *testing.T) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.IgnoreErrors = true

	result, err := Execute("echo 'error message' >&2; exit 1", opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Errorf("Expected command to fail")
	}

	if !strings.Contains(result.Stderr, "error message") {
		t.Errorf("Expected stderr to contain 'error message', got %q", result.Stderr)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	// Check that we get a platform-appropriate shell.
	switch runtime.GOOS {
	case "darwin":
		if opts.Shell != "/bin/zsh" {
			t.Errorf("Expected /bin/zsh on darwin, got %q", opts.Shell)
		}
	case "linux":
		if opts.Shell != "/bin/bash" {
			t.Errorf("Expected /bin/bash on linux, got %q", opts.Shell)
		}
	case "windows":
		gitBash := detectGitBash()
		if gitBash != "" {
			if opts.Shell != gitBash {
				t.Errorf("Expected Git Bash %q on windows, got %q", gitBash, opts.Shell)
			}
		} else if opts.Shell != "powershell.exe" {
			t.Errorf("Expected powershell.exe fallback on windows, got %q", opts.Shell)
		}
	default:
		if opts.Shell != "/bin/sh" {
			t.Errorf("Expected /bin/sh fallback, got %q", opts.Shell)
		}
	}

	if opts.Timeout != 0 {
		t.Errorf("Expected default timeout 0 (no timeout), got %v", opts.Timeout)
	}

	if !opts.CaptureOutput {
		t.Errorf("Expected CaptureOutput to be true by default")
	}

	if opts.StreamOutput {
		t.Errorf("Expected StreamOutput to be false by default")
	}

	if opts.Attached {
		t.Errorf("Expected Attached to be false by default")
	}
}

func TestBuildCommand_AttachedUsesTTYOnUnix(t *testing.T) {
	opts := DefaultOptions()
	opts.Attached = true

	cmd := buildCommand(context.Background(), "echo test", opts)

	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if !strings.HasSuffix(cmd.Path, "/script") && cmd.Path != "script" {
			t.Fatalf("Expected attached mode to use script, got %q", cmd.Path)
		}
		return
	}

	if filepath.Base(cmd.Path) != filepath.Base(opts.Shell) {
		t.Fatalf("Expected fallback shell %q, got %q", opts.Shell, cmd.Path)
	}
}

func TestBuildCommand_DefaultUsesShell(t *testing.T) {
	opts := DefaultOptions()

	cmd := buildCommand(context.Background(), "echo test", opts)

	if filepath.Base(cmd.Path) != filepath.Base(opts.Shell) {
		t.Fatalf("Expected shell %q, got %q", opts.Shell, cmd.Path)
	}
}

func TestExecute_ImmediateErrorWithStderr(t *testing.T) {
	// This test verifies that commands that fail immediately with stderr output
	// don't hang waiting for stdin (which was the bug causing xdrun to hang)
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.StreamOutput = true
	opts.Output = &bytes.Buffer{}
	opts.IgnoreErrors = true
	opts.Timeout = 5 * time.Second // Set a timeout to ensure we don't hang forever

	// Simulate a command that writes to stderr and exits immediately (like docker compose exec with invalid user)
	var cmd string
	if usesPowerShell(opts.Shell) {
		cmd = "Write-Error 'Error response from daemon: unable to find user'; exit 1"
	} else {
		cmd = "echo 'Error response from daemon: unable to find user' >&2; exit 1"
	}

	start := time.Now()
	result, err := Execute(cmd, opts)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should complete quickly, not hang
	if duration > 2*time.Second {
		t.Errorf("Command took too long (%v), may be hanging waiting for stdin", duration)
	}

	if result.Success {
		t.Errorf("Expected command to fail")
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "Error response from daemon") && !strings.Contains(result.Stderr, "unable to find user") {
		t.Errorf("Expected stderr to contain error message, got %q", result.Stderr)
	}
}
