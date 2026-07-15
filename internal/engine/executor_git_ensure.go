package engine

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/scm"
)

func (e *Engine) executeGitEnsureVersion(guard *statement.GitEnsureVersion, ctx *ExecutionContext) error {
	candidateValue, err := e.gitEnsureCandidateValue(guard, ctx)
	if err != nil {
		return err
	}
	candidate, err := scm.ParseVersion(candidateValue)
	if err != nil {
		return fmt.Errorf("git ensure candidate: %w", err)
	}
	if ctx.Project == nil || ctx.Project.SCMRegistry == nil {
		return fmt.Errorf("git ensure source %q cannot be resolved: project does not declare scm → git", guard.Source)
	}
	registry, err := scm.RegistryFromAST(ctx.Project.SCMRegistry)
	if err != nil {
		return fmt.Errorf("git ensure source %q cannot be resolved: %w", guard.Source, err)
	}
	source, err := registry.Git().Source(guard.Source)
	if err != nil {
		return fmt.Errorf("git ensure source %q cannot be resolved: %w", guard.Source, err)
	}
	method := guard.AccessMethod
	if method == "" {
		method = source.Default
	}
	if _, err := source.AccessProfile(method); err != nil {
		return fmt.Errorf("git ensure source %q cannot be resolved: %w", guard.Source, err)
	}
	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would ensure version %s is newer than the latest stable version from Git source %s using %s%s%s\n",
			candidate.Raw, guard.Source, method, gitEnsureContractSummary(guard), gitEnsureCaptureSummary(guard))
		return nil
	}

	baseDir := ctx.OriginalWorkingDir
	if baseDir == "" {
		baseDir, _ = os.Getwd()
	}
	resolver := scm.GitSourceResolver{
		Registry: registry.Git(), Adapters: scm.DefaultGitAdapters(), BaseDir: baseDir,
		Expand: func(value string) (string, error) { return e.interpolateVariablesWithError(value, ctx) },
	}
	session, err := resolver.Open(context.Background(), guard.Source, guard.AccessMethod, false)
	if err != nil {
		return fmt.Errorf("git ensure source %q cannot be resolved: %w", guard.Source, err)
	}
	defer func() { _ = session.Close() }()
	query := scm.GitTagQuery{
		Result: "version", TagPreset: guard.TagPreset, TagFormat: guard.TagFormat,
		TagPattern: guard.TagPattern, OrderBy: "version",
	}
	latestResult, err := query.Execute(context.Background(), session, source)
	if err != nil {
		return fmt.Errorf("git ensure source %q cannot resolve latest stable version: %w", guard.Source, err)
	}
	latest := latestResult.Version
	switch candidate.Compare(latest) {
	case 0:
		return fmt.Errorf("release version %q is already tagged on Git source %q", candidate.Raw, guard.Source)
	case -1:
		return fmt.Errorf("release version %q is older than latest version %q from Git source %q", candidate.Raw, latest.Raw, guard.Source)
	}
	if guard.CaptureVar != "" {
		ctx.Variables[guard.CaptureVar] = latest.Raw
		_, _ = fmt.Fprintf(e.output, "✅  Version %s is newer than latest version %s from %s; captured latest as $%s\n", candidate.Raw, latest.Raw, guard.Source, guard.CaptureVar)
	} else {
		_, _ = fmt.Fprintf(e.output, "✅  Version %s is newer than latest version %s from %s\n", candidate.Raw, latest.Raw, guard.Source)
	}
	return nil
}

func (e *Engine) gitEnsureCandidateValue(guard *statement.GitEnsureVersion, ctx *ExecutionContext) (string, error) {
	value := guard.Candidate
	if guard.CandidateIsVariable {
		value = "{" + value + "}"
	}
	resolved, err := e.interpolateVariablesWithError(value, ctx)
	if err != nil {
		return "", fmt.Errorf("resolving git ensure candidate: %w", err)
	}
	return resolved, nil
}

func gitEnsureContractSummary(guard *statement.GitEnsureVersion) string {
	switch {
	case guard.TagPattern != "":
		return ", matching the inline tag pattern"
	case guard.TagFormat != "":
		return fmt.Sprintf(", matching tags %q", guard.TagFormat)
	case guard.TagPreset != "":
		return ", matching tags " + guard.TagPreset
	default:
		return ", using the source/default version-tag contract"
	}
}

func gitEnsureCaptureSummary(guard *statement.GitEnsureVersion) string {
	if strings.TrimSpace(guard.CaptureVar) == "" {
		return ""
	}
	return ", with latest-version capture $" + guard.CaptureVar + " left unchanged"
}
