package engine

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/domain/task"
	"github.com/phillarmonic/drun/v2/internal/provisioning"
	"github.com/phillarmonic/drun/v2/internal/shell"
)

type toolDetector interface {
	IsToolAvailable(tool string) bool
	GetToolVersion(tool string) string
	CompareVersion(version1, operator, version2 string) bool
}

type provisioningResolver interface {
	ResolveRequirement(ctx context.Context, req statement.ToolRequirement, sources provisioning.SourceSet) (*provisioning.Resolution, error)
}

type versionMismatch struct {
	currentVersion string
	constraint     statement.VersionConstraint
}

// Domain: Tool Requirements Execution
// This file contains the executor for "requires tools:" blocks.
// Tool requirements are checked eagerly — if a tool is missing or doesn't
// meet the version constraints, execution fails immediately.

// executeRequiresTools checks that all required tools are available and meet version constraints.
func (e *Engine) executeRequiresTools(stmt *statement.RequiresTools, ctx *ExecutionContext) error {
	tools := append([]statement.ToolRequirement(nil), stmt.Tools...)
	if len(stmt.TaskRefs) > 0 {
		inherited, err := task.ResolveInheritedProjectToolRequirements(e.taskRegistry, stmt.TaskRefs)
		if err != nil {
			return err
		}
		tools = append(tools, inherited...)
	}
	return e.checkToolRequirements(e.newToolDetector(), tools, ctx.Project, ctx)
}

// checkToolRequirements validates a list of tool requirements against the system.
// This is shared between task-level execution and project-level startup checks.
func (e *Engine) checkToolRequirements(detector toolDetector, tools []statement.ToolRequirement, projectCtx *ProjectContext, execCtx *ExecutionContext) error {
	for _, tool := range tools {
		if err := e.checkSingleToolRequirement(detector, tool, projectCtx, execCtx); err != nil {
			return err
		}
	}

	return nil
}

// checkProjectToolRequirements checks project-level tool requirements at startup.
func (e *Engine) checkProjectToolRequirements(projectCtx *ProjectContext) error {
	if projectCtx == nil || (len(projectCtx.RequiredTools) == 0 && len(projectCtx.RequiredToolTaskRefs) == 0) {
		return nil
	}
	if len(projectCtx.RequiredToolTaskRefs) > 0 {
		inherited, err := task.ResolveInheritedProjectToolRequirements(e.taskRegistry, projectCtx.RequiredToolTaskRefs)
		if err != nil {
			return err
		}
		projectCtx.RequiredTools = append(projectCtx.RequiredTools, inherited...)
		projectCtx.RequiredToolTaskRefs = nil
	}

	return e.checkToolRequirements(e.newToolDetector(), projectCtx.RequiredTools, projectCtx, nil)
}

func (e *Engine) checkSingleToolRequirement(detector toolDetector, tool statement.ToolRequirement, projectCtx *ProjectContext, execCtx *ExecutionContext) error {
	if !detector.IsToolAvailable(tool.Name) {
		if tool.AutoProvision {
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] 🔧 Would provision missing tool '%s'\n", tool.Name)
				return nil
			}
			return e.provisionAndRecheck(tool, projectCtx, execCtx, "required tool is not installed")
		}
		if e.dryRun {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] ❌ Required tool '%s' is not installed\n", tool.Name)
			return nil
		}
		return fmt.Errorf("required tool '%s' is not installed", tool.Name)
	}

	currentVersion, mismatch, err := evaluateToolVersion(detector, tool)
	if err != nil {
		if e.dryRun {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] ⚠️  Could not determine version for '%s'\n", tool.Name)
			return nil
		}
		return err
	}

	if mismatch != nil {
		if tool.AutoProvision {
			if e.dryRun {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] 🔧 Would provision '%s' to satisfy %s %s\n",
					tool.Name, mismatch.constraint.Operator, mismatch.constraint.Version)
				return nil
			}
			if !e.allowToolVersionChanges {
				_, _ = fmt.Fprintf(e.output, "⚠️  Tool '%s' version %s does not satisfy %s %s; refusing to change the installed version without --allow-tool-version-changes\n",
					tool.Name, mismatch.currentVersion, mismatch.constraint.Operator, mismatch.constraint.Version)
				return fmt.Errorf("required tool '%s' version %s does not satisfy constraint %s %s; rerun with --allow-tool-version-changes to allow provisioning to change installed versions",
					tool.Name, mismatch.currentVersion, mismatch.constraint.Operator, mismatch.constraint.Version)
			}
			return e.provisionAndRecheck(tool, projectCtx, execCtx,
				fmt.Sprintf("tool version %s does not satisfy %s %s", mismatch.currentVersion, mismatch.constraint.Operator, mismatch.constraint.Version))
		}
		if e.dryRun {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] ❌ Tool '%s' version %s does not satisfy %s %s\n",
				tool.Name, mismatch.currentVersion, mismatch.constraint.Operator, mismatch.constraint.Version)
			return nil
		}
		return fmt.Errorf("required tool '%s' version %s does not satisfy constraint %s %s",
			tool.Name, mismatch.currentVersion, mismatch.constraint.Operator, mismatch.constraint.Version)
	}

	if len(tool.Constraints) > 0 {
		if e.verbose || e.dryRun {
			_, _ = fmt.Fprintf(e.output, "✅  %s %s (%s)\n",
				tool.Name, currentVersion, formatConstraints(tool.Constraints))
		}
		return nil
	}

	if e.verbose || e.dryRun {
		_, _ = fmt.Fprintf(e.output, "✅  %s is available\n", tool.Name)
	}
	return nil
}

func evaluateToolVersion(detector toolDetector, tool statement.ToolRequirement) (string, *versionMismatch, error) {
	if len(tool.Constraints) == 0 {
		return "", nil, nil
	}

	currentVersion := detector.GetToolVersion(tool.Name)
	if currentVersion == "" {
		return "", nil, fmt.Errorf("required tool '%s' is installed but version could not be determined (needed: %s)",
			tool.Name, formatConstraints(tool.Constraints))
	}

	for _, constraint := range tool.Constraints {
		if !detector.CompareVersion(currentVersion, constraint.Operator, constraint.Version) {
			return currentVersion, &versionMismatch{
				currentVersion: currentVersion,
				constraint:     constraint,
			}, nil
		}
	}

	return currentVersion, nil, nil
}

func (e *Engine) provisionAndRecheck(tool statement.ToolRequirement, projectCtx *ProjectContext, execCtx *ExecutionContext, reason string) error {
	resolver := e.newProvisioningResolver(e.provisioningWorkingDir(execCtx))
	resolution, err := resolver.ResolveRequirement(context.Background(), tool, provisioning.SourceSet{
		Project: provisioningSourcesFromProject(projectCtx),
		User:    append([]string(nil), e.userProvisioningSources...),
	})
	if err != nil {
		return fmt.Errorf("resolve provisioning for tool '%s': %w", tool.Name, err)
	}

	command := resolution.InstallCommand()
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔧 Provisioning '%s' because %s\n", tool.Name, reason)
		_, _ = fmt.Fprintf(e.output, "   source: %s\n", resolution.Source)
		_, _ = fmt.Fprintf(e.output, "   command: %s\n", command)
	}

	if err := e.provisionCommandRunner(command, execCtx); err != nil {
		return fmt.Errorf("provision tool '%s': %w", tool.Name, err)
	}

	refreshedDetector := e.newToolDetector()
	if err := e.checkSingleToolRequirement(refreshedDetector, withoutAutoProvision(tool), projectCtx, execCtx); err != nil {
		return fmt.Errorf("post-provision check for tool '%s' failed: %w", tool.Name, err)
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "✅  Provisioned '%s' successfully\n", tool.Name)
	}
	return nil
}

func (e *Engine) runProvisioningCommand(command string, execCtx *ExecutionContext) error {
	opts := shell.DefaultOptions()
	if execCtx != nil {
		opts = e.getPlatformShellConfig(execCtx)
	}
	opts.CaptureOutput = true
	opts.StreamOutput = true
	opts.Output = e.output
	opts.WorkingDir = e.provisioningWorkingDir(execCtx)

	_, err := shell.Execute(command, opts)
	return err
}

func (e *Engine) provisioningWorkingDir(execCtx *ExecutionContext) string {
	if execCtx != nil {
		if execCtx.WorkingDir != "" {
			return execCtx.WorkingDir
		}
		if execCtx.OriginalWorkingDir != "" {
			return execCtx.OriginalWorkingDir
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func provisioningSourcesFromProject(projectCtx *ProjectContext) []string {
	if projectCtx == nil || len(projectCtx.ProvisioningSources) == 0 {
		return nil
	}
	return append([]string(nil), projectCtx.ProvisioningSources...)
}

func withoutAutoProvision(tool statement.ToolRequirement) statement.ToolRequirement {
	tool.AutoProvision = false
	return tool
}

// formatConstraints formats a slice of version constraints for error messages.
func formatConstraints(constraints []statement.VersionConstraint) string {
	parts := make([]string, len(constraints))
	for i, c := range constraints {
		parts[i] = c.Operator + " " + c.Version
	}
	return strings.Join(parts, ", ")
}
