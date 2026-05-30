package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
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

	var stdoutPipe, stderrPipe io.ReadCloser
	if opts.CaptureOutput {
		var err error
		stdoutPipe, err = cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		stderrPipe, err = cmd.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
		}
	} else if opts.StreamOutput && opts.Output != nil {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	stopForward := forwardSignals(cmd)
	defer stopForward()

	if opts.CaptureOutput {
		var stdoutBuf, stderrBuf bytes.Buffer

		stdoutDone := make(chan error, 1)
		stderrDone := make(chan error, 1)

		var stdoutWriter io.Writer = &stdoutBuf
		var stderrWriter io.Writer = &stderrBuf

		if opts.StreamOutput && opts.Output != nil {
			stdoutWriter = io.MultiWriter(opts.Output, &stdoutBuf)
			stderrWriter = io.MultiWriter(opts.Output, &stderrBuf)
		}

		go func() {
			_, err := io.Copy(stdoutWriter, stdoutPipe)
			stdoutDone <- err
		}()

		go func() {
			_, err := io.Copy(stderrWriter, stderrPipe)
			stderrDone <- err
		}()

		// Finish reading pipes before Wait(). Per os/exec.Cmd.StdoutPipe, Wait
		// closes the pipe after the command exits; calling Wait before io.Copy
		// completes races and can yield "read |0: file already closed".
		stdoutErr := <-stdoutDone
		stderrErr := <-stderrDone
		err := cmd.Wait()

		if stdoutErr != nil && stdoutErr != io.EOF {
			return nil, fmt.Errorf("failed reading stdout: %w", stdoutErr)
		}
		if stderrErr != nil && stderrErr != io.EOF {
			return nil, fmt.Errorf("failed reading stderr: %w", stderrErr)
		}

		result.Stdout = strings.TrimRight(stdoutBuf.String(), "\r\n")
		result.Stderr = strings.TrimRight(stderrBuf.String(), "\r\n")

		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitError.ExitCode()
			} else {
				return nil, fmt.Errorf("command execution failed: %w", err)
			}
		}
	} else {
		if err := cmd.Wait(); err != nil {
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

func forwardSignals(cmd *exec.Cmd) func() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case sig, ok := <-signalCh:
				if !ok {
					return
				}
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
		signal.Stop(signalCh)
		close(signalCh)
	}
}
