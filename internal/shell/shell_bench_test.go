package shell

import (
	"io"
	"testing"
	"time"
)

// BenchmarkExecuteSimple benchmarks simple command execution
func BenchmarkExecuteSimple(b *testing.B) {
	opts := DefaultOptions()
	opts.Output = io.Discard

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Execute("echo 'test'", opts)
	}
}

// BenchmarkExecuteWithCapture benchmarks command execution with output capture
func BenchmarkExecuteWithCapture(b *testing.B) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.Output = io.Discard

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Execute("echo 'test output line'", opts)
	}
}

// BenchmarkExecuteMultilineOutput benchmarks command with multiline output
func BenchmarkExecuteMultilineOutput(b *testing.B) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.Output = io.Discard

	// Command that produces multiple lines of output
	command := "printf 'line1\\nline2\\nline3\\nline4\\nline5\\n'"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Execute(command, opts)
	}
}

// BenchmarkDefaultOptions benchmarks the creation of default options
func BenchmarkDefaultOptions(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts := DefaultOptions()
		_ = opts
	}
}

// BenchmarkExecuteWithEnvironment benchmarks command execution with environment variables
func BenchmarkExecuteWithEnvironment(b *testing.B) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.Output = io.Discard
	opts.Environment = map[string]string{
		"TEST_VAR1": "value1",
		"TEST_VAR2": "value2",
		"TEST_VAR3": "value3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Execute("echo $TEST_VAR1", opts)
	}
}

// BenchmarkExecuteWithTimeout benchmarks command execution with timeout
func BenchmarkExecuteWithTimeout(b *testing.B) {
	opts := DefaultOptions()
	opts.CaptureOutput = true
	opts.Output = io.Discard
	opts.Timeout = 5 * time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Execute("echo 'test with timeout'", opts)
	}
}
