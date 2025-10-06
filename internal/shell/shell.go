package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Result represents the result of a shell command execution
type Result struct {
	Command  string        // The command that was executed
	ExitCode int           // Exit code of the command
	Stdout   string        // Standard output
	Stderr   string        // Standard error
	Duration time.Duration // How long the command took
	Success  bool          // Whether the command succeeded (exit code 0)
}

// Options configures shell command execution
type Options struct {
	WorkingDir    string            // Working directory for the command
	Environment   map[string]string // Additional environment variables
	Timeout       time.Duration     // Command timeout (0 = no timeout)
	CaptureOutput bool              // Whether to capture stdout/stderr
	StreamOutput  bool              // Whether to stream output in real-time
	Output        io.Writer         // Where to stream output (if StreamOutput is true)
	Shell         string            // Shell to use (default: /bin/sh)
	IgnoreErrors  bool              // Whether to ignore non-zero exit codes
}

// DefaultOptions returns sensible default options
func DefaultOptions() *Options {
	// Use platform-appropriate shell defaults
	defaultShell := "/bin/sh"
	switch runtime.GOOS {
	case "darwin":
		defaultShell = "/bin/zsh"
	case "linux":
		defaultShell = "/bin/bash"
	case "windows":
		defaultShell = "powershell.exe"
	}

	return &Options{
		WorkingDir:    "",
		Environment:   make(map[string]string, 8), // Pre-allocate for typical env var count
		Timeout:       0,                          // No timeout - allow tasks to run as long as necessary
		CaptureOutput: true,
		StreamOutput:  false,
		Output:        os.Stdout,
		Shell:         defaultShell,
		IgnoreErrors:  false,
	}
}

// Execute runs a shell command with the given options
func Execute(command string, opts *Options) (*Result, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	start := time.Now()

	// Create context with timeout if specified
	var ctx context.Context
	var cancel context.CancelFunc

	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	// Create the command
	cmd := exec.CommandContext(ctx, opts.Shell, "-c", command)

	// Explicitly set stdin to nil to prevent commands from hanging waiting for input
	// This is important for non-interactive command execution
	cmd.Stdin = nil

	// Set working directory
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	// Set environment variables
	if len(opts.Environment) > 0 {
		env := os.Environ()
		for key, value := range opts.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	result := &Result{
		Command: command,
	}

	// Handle output capture and streaming
	if opts.CaptureOutput {
		var stdoutBuf, stderrBuf strings.Builder
		// Pre-allocate buffers with reasonable capacity to reduce allocations
		stdoutBuf.Grow(1024)
		stderrBuf.Grow(512)

		if opts.StreamOutput && opts.Output != nil {
			// Stream and capture simultaneously
			stdoutPipe, err := cmd.StdoutPipe()
			if err != nil {
				return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
			}

			stderrPipe, err := cmd.StderrPipe()
			if err != nil {
				return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
			}

			// Start the command
			if err := cmd.Start(); err != nil {
				return nil, fmt.Errorf("failed to start command: %w", err)
			}

			// Use channels to synchronize goroutines
			stdoutDone := make(chan bool)
			stderrDone := make(chan bool)

			// Stream stdout
			go func() {
				defer close(stdoutDone)
				scanner := bufio.NewScanner(stdoutPipe)
				for scanner.Scan() {
					line := scanner.Text()
					// More efficient string building
					stdoutBuf.WriteString(line)
					stdoutBuf.WriteByte('\n')
					_, _ = fmt.Fprintln(opts.Output, line)
				}
			}()

			// Stream stderr
			go func() {
				defer close(stderrDone)
				scanner := bufio.NewScanner(stderrPipe)
				for scanner.Scan() {
					line := scanner.Text()
					// More efficient string building
					stderrBuf.WriteString(line)
					stderrBuf.WriteByte('\n')
					_, _ = fmt.Fprintln(opts.Output, line)
				}
			}()

			// Wait for command completion
			err = cmd.Wait()

			// Wait for both goroutines to complete
			<-stdoutDone
			<-stderrDone

			result.Stdout = strings.TrimSuffix(stdoutBuf.String(), "\n")
			result.Stderr = strings.TrimSuffix(stderrBuf.String(), "\n")

			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					result.ExitCode = exitError.ExitCode()
				} else {
					return nil, fmt.Errorf("command execution failed: %w", err)
				}
			}
		} else {
			// Just capture without streaming
			stdout, err := cmd.Output()
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					result.ExitCode = exitError.ExitCode()
					result.Stderr = string(exitError.Stderr)
				} else {
					return nil, fmt.Errorf("command execution failed: %w", err)
				}
			}
			result.Stdout = strings.TrimSpace(string(stdout))
		}
	} else {
		// No capture, just run
		if opts.StreamOutput && opts.Output != nil {
			cmd.Stdout = opts.Output
			cmd.Stderr = opts.Output
		}

		err := cmd.Run()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitError.ExitCode()
			} else {
				return nil, fmt.Errorf("command execution failed: %w", err)
			}
		}
	}

	result.Duration = time.Since(start)
	result.Success = result.ExitCode == 0

	// Check if we should treat this as an error
	if !result.Success && !opts.IgnoreErrors {
		return result, fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	return result, nil
}

// ExecuteSimple runs a command with default options and returns just the output
func ExecuteSimple(command string) (string, error) {
	result, err := Execute(command, DefaultOptions())
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// ExecuteWithOutput runs a command and streams output to the given writer
func ExecuteWithOutput(command string, output io.Writer) (*Result, error) {
	opts := DefaultOptions()
	opts.StreamOutput = true
	opts.Output = output
	return Execute(command, opts)
}
