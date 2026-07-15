package engine

import (
	"context"
	"fmt"
	"os"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/scm"
)

func (e *Engine) executeGitQuery(query *statement.GitQuery, ctx *ExecutionContext) error {
	if ctx.Project == nil || ctx.Project.SCMRegistry == nil {
		return fmt.Errorf("git query source %q cannot be resolved: project does not declare scm → git", query.Source)
	}
	if e.dryRun {
		value := fmt.Sprintf("[DRY RUN] latest %s from %s", query.Result, query.Source)
		ctx.Variables[query.CaptureVar] = value
		method := query.AccessMethod
		if method == "" {
			method = "source default"
		}
		order := query.OrderBy
		if order == "" {
			order = "version"
		}
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would get latest %s from Git source %s using %s, ordered by %s, as $%s\n", query.Result, query.Source, method, order, query.CaptureVar)
		return nil
	}
	registry, err := scm.RegistryFromAST(ctx.Project.SCMRegistry)
	if err != nil {
		return err
	}
	baseDir := ctx.OriginalWorkingDir
	if baseDir == "" {
		baseDir, _ = os.Getwd()
	}
	resolver := scm.GitSourceResolver{
		Registry: registry.Git(), Adapters: scm.DefaultGitAdapters(), BaseDir: baseDir,
		Expand: func(value string) (string, error) { return e.interpolateVariablesWithError(value, ctx) },
	}
	source, err := resolver.Registry.Source(query.Source)
	if err != nil {
		return err
	}
	needsMetadata := query.OrderBy == "date"
	session, err := resolver.Open(context.Background(), query.Source, query.AccessMethod, needsMetadata && query.AllowFetch)
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()
	tagQuery := scm.GitTagQuery{
		Result: query.Result, TagPreset: query.TagPreset, TagFormat: query.TagFormat,
		TagPattern: query.TagPattern, Series: query.Series, VersionMatcher: query.VersionMatcher,
		OrderBy: query.OrderBy,
	}
	result, err := tagQuery.Execute(context.Background(), session, source)
	if err != nil {
		return fmt.Errorf("querying Git source %q: %w", query.Source, err)
	}
	value := result.Value(query.Result)
	ctx.Variables[query.CaptureVar] = value
	_, _ = fmt.Fprintf(e.output, "📦  Captured latest Git %s %q from %s as $%s\n", query.Result, value, query.Source, query.CaptureVar)
	return nil
}
