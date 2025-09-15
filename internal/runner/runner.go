package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/phillarmonic/drun/internal/model"
	"github.com/phillarmonic/drun/internal/shell"
	"github.com/phillarmonic/drun/internal/tmpl"
)

// Runner executes plans with logging and timeout support
type Runner struct {
	shellSelector  *shell.Selector
	templateEngine *tmpl.Engine
	output         io.Writer
	dryRun         bool
	explain        bool
}

// NewRunner creates a new runner
func NewRunner(shellSelector *shell.Selector, templateEngine *tmpl.Engine, output io.Writer) *Runner {
	return &Runner{
		shellSelector:  shellSelector,
		templateEngine: templateEngine,
		output:         output,
	}
}

// SetDryRun enables or disables dry-run mode
func (r *Runner) SetDryRun(dryRun bool) {
	r.dryRun = dryRun
}

// SetExplain enables or disables explain mode
func (r *Runner) SetExplain(explain bool) {
	r.explain = explain
}

// Execute executes an execution plan
func (r *Runner) Execute(plan *model.ExecutionPlan, jobs int) error {
	if len(plan.Nodes) == 0 {
		return nil
	}

	if jobs <= 1 {
		// Sequential execution
		return r.executeSequential(plan)
	}

	// Parallel execution
	return r.executeParallel(plan, jobs)
}

// executeSequential executes nodes sequentially
func (r *Runner) executeSequential(plan *model.ExecutionPlan) error {
	for i, node := range plan.Nodes {
		if err := r.executeNode(&node, i+1, len(plan.Nodes)); err != nil {
			return err
		}
	}
	return nil
}

// executeParallel executes nodes in parallel where possible
func (r *Runner) executeParallel(plan *model.ExecutionPlan, jobs int) error {
	// For now, implement a simple parallel execution
	// In a full implementation, this would use the DAG to determine which nodes can run in parallel

	semaphore := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstError error

	for i, node := range plan.Nodes {
		wg.Add(1)
		go func(n model.PlanNode, idx int) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			if err := r.executeNode(&n, idx+1, len(plan.Nodes)); err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = err
				}
				mu.Unlock()
			}
		}(node, i)
	}

	wg.Wait()
	return firstError
}

// executeNode executes a single node
func (r *Runner) executeNode(node *model.PlanNode, current, total int) error {
	r.logf("[%d/%d] %s", current, total, node.ID)

	// Render recipe-specific environment variables
	if err := r.renderRecipeEnvironment(node.Context); err != nil {
		return fmt.Errorf("failed to render recipe environment for '%s': %w", node.ID, err)
	}

	// Render the step with the template engine
	renderedStep, err := r.templateEngine.RenderStep(node.Step, node.Context)
	if err != nil {
		return fmt.Errorf("failed to render step for recipe '%s': %w", node.ID, err)
	}

	// Select appropriate shell
	sh, err := r.shellSelector.Select(node.Recipe.Shell, node.Context.OS)
	if err != nil {
		return fmt.Errorf("failed to select shell for recipe '%s': %w", node.ID, err)
	}

	// Build the script
	script := renderedStep.String()

	if r.explain || r.dryRun {
		r.logf("Recipe: %s", node.ID)
		r.logf("Working Directory: %s", node.Recipe.WorkingDir)
		r.logf("Shell: %s %v", sh.Cmd, sh.Args)
		r.logf("Script:")
		for i, line := range strings.Split(script, "\n") {
			r.logf("  %d: %s", i+1, line)
		}
		r.logf("Environment:")
		for k, v := range node.Context.Env {
			if r.isSecret(k) {
				r.logf("  %s=***", k)
			} else {
				r.logf("  %s=%s", k, v)
			}
		}
		r.logf("")

		if r.dryRun {
			return nil // Don't actually execute in dry-run mode
		}
	}

	// Execute the script
	return r.executeScript(sh, script, node.Recipe.WorkingDir, node.Context.Env, node.Recipe.Timeout, node.Recipe.IgnoreError)
}

// executeScript executes a script with the given shell
func (r *Runner) executeScript(sh *shell.Shell, script, workingDir string, env map[string]string, timeout time.Duration, ignoreError bool) error {
	// Build command
	args := sh.BuildCommand(script)
	cmd := exec.Command(sh.Cmd, args...)

	// Set working directory
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set environment
	cmd.Env = os.Environ() // Start with current environment
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up output redirection
	cmd.Stdout = r.output
	cmd.Stderr = r.output

	// Set up timeout context
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && !ignoreError {
			return fmt.Errorf("command failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		// Timeout occurred, kill the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Errorf("command timed out after %v", timeout)
	}
}

// logf logs a formatted message
func (r *Runner) logf(format string, args ...interface{}) {
	fmt.Fprintf(r.output, format+"\n", args...)
}

// renderRecipeEnvironment renders recipe-specific environment variables
func (r *Runner) renderRecipeEnvironment(ctx *model.ExecutionContext) error {
	// Render environment variables that contain templates
	for k, v := range ctx.Env {
		if strings.Contains(v, "{{") {
			rendered, err := r.templateEngine.Render(v, ctx)
			if err != nil {
				return fmt.Errorf("failed to render environment variable %s: %w", k, err)
			}
			ctx.Env[k] = rendered
		}
	}
	return nil
}

// isSecret checks if an environment variable name indicates it contains a secret
func (r *Runner) isSecret(name string) bool {
	name = strings.ToUpper(name)
	secretKeywords := []string{"TOKEN", "SECRET", "PASSWORD", "KEY", "PASS"}

	for _, keyword := range secretKeywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}

	return false
}
