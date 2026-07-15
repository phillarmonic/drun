package engine

import (
	"fmt"
	"os"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/filevalue"
)

func (e *Engine) executeFileValue(stmt *statement.FileValue, ctx *ExecutionContext) error {
	format := stmt.Format
	selector := e.interpolateVariables(stmt.Selector, ctx)
	target := e.interpolateVariables(stmt.Target, ctx)

	switch stmt.Operation {
	case "get":
		value, err := filevalue.ReadFile(format, selector, target)
		if err != nil {
			return fmt.Errorf("get %s %q from %q: %w", format, selector, target, err)
		}
		ctx.Variables[stmt.CaptureVar] = value.Text
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "📦  Captured %s %q from %s as $%s\n", format, selector, target, stmt.CaptureVar)
		}
		return nil

	case "check":
		actual, err := filevalue.ReadFile(format, selector, target)
		if err != nil {
			return fmt.Errorf("check %s %q in %q: %w", format, selector, target, err)
		}
		expected := e.interpolateVariables(stmt.Expected, ctx)
		matched := actual.Text == expected
		if stmt.Comparison == "differs" {
			matched = !matched
		}
		if !matched {
			operator := "equal"
			if stmt.Comparison == "differs" {
				operator = "differ from"
			}
			return fmt.Errorf("file value check failed: %s %q in %q expected to %s %q, actual %q", format, selector, target, operator, expected, actual.Text)
		}
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "✅  File value check passed: %s %q in %s\n", format, selector, target)
		}
		return nil

	case "update":
		value := e.interpolateVariables(stmt.Value, ctx)
		if e.dryRun {
			// Validate the complete prospective edit while leaving the file untouched.
			// #nosec G304 -- the Drun program explicitly supplies the path.
			data, err := os.ReadFile(target)
			if err != nil {
				return err
			}
			if _, _, err := filevalue.Update(format, selector, data, value, stmt.MissingPolicy, stmt.ValueType); err != nil {
				return fmt.Errorf("update %s %q in %q: %w", format, selector, target, err)
			}
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would update %s %q in %s to %q\n", format, selector, target, value)
			return nil
		}
		changed, _, err := filevalue.UpdateFile(format, selector, target, value, stmt.MissingPolicy, stmt.ValueType)
		if err != nil {
			return fmt.Errorf("update %s %q in %q: %w", format, selector, target, err)
		}
		if e.verbose {
			if changed {
				_, _ = fmt.Fprintf(e.output, "✅  Updated %s %q in %s\n", format, selector, target)
			} else {
				_, _ = fmt.Fprintf(e.output, "✅  %s %q in %s already has the requested value\n", format, selector, target)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown file value operation %q", stmt.Operation)
	}
}
