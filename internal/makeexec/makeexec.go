package makeexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/domain/orchestration"
)

// Executor executes Makefile targets
type Executor struct {
	workDir string
}

// NewExecutor creates a new Makefile executor
func NewExecutor(workDir string) *Executor {
	return &Executor{
		workDir: workDir,
	}
}

// Execute executes a Makefile target
func (e *Executor) Execute(ctx context.Context, config *orchestration.BuildConfig, servicePath string) error {
	// Determine working directory
	workDir := servicePath
	if config.WorkingDirectory != "" {
		workDir = config.WorkingDirectory
	}

	fullPath := filepath.Join(e.workDir, workDir)

	// Check if Makefile exists
	makefilePath := filepath.Join(fullPath, config.Makefile)
	if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
		return fmt.Errorf("makefile not found at %s", makefilePath)
	}

	// Execute pre-make commands if any
	if len(config.PreMakeCommands) > 0 {
		for _, cmdStr := range config.PreMakeCommands {
			if err := e.executeCommand(ctx, cmdStr, fullPath); err != nil {
				return fmt.Errorf("pre-make command failed: %w", err)
			}
		}
	}

	// Build make command
	args := []string{"-f", config.Makefile}

	// Add parallel jobs flag if specified
	if config.ParallelJobs > 0 {
		args = append(args, fmt.Sprintf("-j%d", config.ParallelJobs))
	}

	// Add make target
	if config.MakeTarget != "" {
		args = append(args, config.MakeTarget)
	}

	// Add make arguments
	args = append(args, config.MakeArgs...)

	// Create command
	// #nosec G204 -- build execution intentionally invokes make with user-selected targets and flags.
	cmd := exec.CommandContext(ctx, "make", args...)
	cmd.Dir = fullPath
	cmd.Env = os.Environ()

	// Set timeout if specified
	if config.MakefileTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.MakefileTimeout)
		defer cancel()
		// #nosec G204 -- build execution intentionally invokes make with user-selected targets and flags.
		cmd = exec.CommandContext(ctx, "make", args...)
		cmd.Dir = fullPath
		cmd.Env = os.Environ()
	}

	// Set verbose output
	if config.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Execute make command
	output, err := cmd.CombinedOutput()
	if err != nil {
		if config.RetryOnFailure && config.MaxRetries > 0 {
			// Retry with exponential backoff
			return e.executeWithRetries(ctx, config, servicePath)
		}

		if config.FallbackCommand != "" {
			// Execute fallback command
			return e.executeCommand(ctx, config.FallbackCommand, fullPath)
		}

		return fmt.Errorf("make command failed: %w\nOutput: %s", err, string(output))
	}

	// Execute post-make commands if any
	if len(config.PostMakeCommands) > 0 {
		for _, cmdStr := range config.PostMakeCommands {
			if err := e.executeCommand(ctx, cmdStr, fullPath); err != nil {
				return fmt.Errorf("post-make command failed: %w", err)
			}
		}
	}

	return nil
}

// executeWithRetries executes a make command with retries
func (e *Executor) executeWithRetries(ctx context.Context, config *orchestration.BuildConfig, servicePath string) error {
	var lastErr error

	for i := 0; i < config.MaxRetries; i++ {
		// Wait before retry
		if i > 0 && config.RetryDelay > 0 {
			select {
			case <-time.After(config.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Try executing
		err := e.Execute(ctx, config, servicePath)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	// All retries failed
	if config.FallbackCommand != "" {
		// Execute fallback command
		workDir := servicePath
		if config.WorkingDirectory != "" {
			workDir = config.WorkingDirectory
		}
		fullPath := filepath.Join(e.workDir, workDir)

		if err := e.executeCommand(ctx, config.FallbackCommand, fullPath); err != nil {
			return fmt.Errorf("make command failed after %d retries and fallback failed: %w", config.MaxRetries, err)
		}
		return nil
	}

	return fmt.Errorf("make command failed after %d retries: %w", config.MaxRetries, lastErr)
}

// executeCommand executes a shell command
func (e *Executor) executeCommand(ctx context.Context, cmdStr, workDir string) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// #nosec G204 -- pre/post/fallback commands are explicitly configured build commands.
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ListTargets lists available Makefile targets
func (e *Executor) ListTargets(ctx context.Context, makefilePath string) ([]string, error) {
	fullPath := filepath.Join(e.workDir, makefilePath)

	// Check if Makefile exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("makefile not found at %s", fullPath)
	}

	// Use make -qp to list targets
	// #nosec G204 -- make target discovery intentionally inspects the selected Makefile path.
	cmd := exec.CommandContext(ctx, "make", "-qp", "-f", fullPath)
	cmd.Dir = filepath.Dir(fullPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// make -qp returns non-zero exit code, but still outputs target list
		if len(output) == 0 {
			return nil, fmt.Errorf("failed to list targets: %w", err)
		}
	}

	// Parse output to extract targets
	targets := []string{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Look for target definitions (lines with colons)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, ".") && !strings.Contains(line, "=") {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				target := strings.TrimSpace(parts[0])
				if target != "" && !strings.Contains(target, " ") {
					targets = append(targets, target)
				}
			}
		}
	}

	return targets, nil
}

// DryRun performs a dry run of a Makefile target
func (e *Executor) DryRun(ctx context.Context, config *orchestration.BuildConfig, servicePath string) (string, error) {
	// Determine working directory
	workDir := servicePath
	if config.WorkingDirectory != "" {
		workDir = config.WorkingDirectory
	}

	fullPath := filepath.Join(e.workDir, workDir)

	// Check if Makefile exists
	makefilePath := filepath.Join(fullPath, config.Makefile)
	if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("makefile not found at %s", makefilePath)
	}

	// Build make command with --dry-run flag
	args := []string{"-f", config.Makefile, "--dry-run"}

	// Add make target
	if config.MakeTarget != "" {
		args = append(args, config.MakeTarget)
	}

	// Add make arguments
	args = append(args, config.MakeArgs...)

	// Create command
	// #nosec G204 -- dry-run intentionally invokes make with user-selected targets and flags.
	cmd := exec.CommandContext(ctx, "make", args...)
	cmd.Dir = fullPath
	cmd.Env = os.Environ()

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("dry run failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// Validate validates that a Makefile exists and is valid
func (e *Executor) Validate(ctx context.Context, makefilePath string) error {
	fullPath := filepath.Join(e.workDir, makefilePath)

	// Check if Makefile exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("makefile not found at %s", fullPath)
	}

	// Try to parse the Makefile using make --dry-run
	// #nosec G204 -- validation intentionally invokes make against the selected Makefile path.
	cmd := exec.CommandContext(ctx, "make", "-f", fullPath, "--dry-run")
	cmd.Dir = filepath.Dir(fullPath)

	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("makefile validation failed: %w", err)
	}

	return nil
}
