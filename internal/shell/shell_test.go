package shell

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

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

	result, err := Execute("echo $TEST_VAR", opts)
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
	opts.WorkingDir = "/tmp"

	result, err := Execute("pwd", opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Stdout != "/tmp" {
		t.Errorf("Expected '/tmp', got %q", result.Stdout)
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

	result, err := Execute("echo 'line1'; echo 'line2'; echo 'line3'", opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expected := "line1\nline2\nline3"
	if result.Stdout != expected {
		t.Errorf("Expected %q, got %q", expected, result.Stdout)
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

	if opts.Shell != "/bin/sh" {
		t.Errorf("Expected default shell '/bin/sh', got %q", opts.Shell)
	}

	if opts.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", opts.Timeout)
	}

	if !opts.CaptureOutput {
		t.Errorf("Expected CaptureOutput to be true by default")
	}

	if opts.StreamOutput {
		t.Errorf("Expected StreamOutput to be false by default")
	}
}
