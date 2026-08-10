package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phillarmonic/drun/v2/internal/changelog"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

func (e *Engine) executeChangelog(stmt *statement.Changelog, ctx *ExecutionContext) error {
	path := e.interpolateVariables(stmt.Path, ctx)
	version := e.interpolateVariables(stmt.Version, ctx)

	date := time.Now()
	if stmt.Date != "" {
		parsed, err := changelog.ParseDate(e.interpolateVariables(stmt.Date, ctx))
		if err != nil {
			return fmt.Errorf("promote changelog %q: %w", path, err)
		}
		date = parsed
	}

	// #nosec G304 -- the Drun program explicitly supplies the path.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("promote changelog %q: %w", path, err)
	}

	updated, err := changelog.Promote(string(data), version, date)
	if err != nil {
		return fmt.Errorf("promote changelog %q: %w", path, err)
	}
	normalized, _ := changelog.NormalizeVersion(version)

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would promote changelog %s unreleased entries to version %s (%s)\n", path, normalized, date.Format("2006-01-02"))
		return nil
	}

	if string(data) == updated {
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅  Changelog %s is already up to date\n", path)
		}
		return nil
	}

	if err := writeFileAtomic(path, []byte(updated)); err != nil {
		return fmt.Errorf("promote changelog %q: %w", path, err)
	}
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "✅  Promoted changelog %s unreleased entries to version %s (%s)\n", path, normalized, date.Format("2006-01-02"))
	}
	return nil
}

// writeFileAtomic replaces path with data using a same-directory temporary
// file, preserving the original file permissions.
func writeFileAtomic(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".drun-changelog-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
