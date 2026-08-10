package engine

import (
	"fmt"
	"strconv"
	"time"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

// Domain: Wait Execution
// This file contains the executor for fixed-duration waits (wait 5 seconds)

// executeWait executes a fixed-duration wait (wait 5 seconds, wait {retries} minutes)
func (e *Engine) executeWait(waitStmt *statement.Wait, ctx *ExecutionContext) error {
	// Interpolate variables in the duration value (e.g. wait {retries} seconds)
	value, err := e.interpolateVariablesWithError(waitStmt.Value, ctx)
	if err != nil {
		return err
	}

	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid wait duration %q: expected a number", value)
	}
	if amount < 0 {
		return fmt.Errorf("invalid wait duration %q: must not be negative", value)
	}

	var unit time.Duration
	switch waitStmt.Unit {
	case "second":
		unit = time.Second
	case "minute":
		unit = time.Minute
	case "hour":
		unit = time.Hour
	default:
		return fmt.Errorf("invalid wait unit %q: expected second(s), minute(s), or hour(s)", waitStmt.Unit)
	}

	duration := time.Duration(amount * float64(unit))

	// formatDuration floors to whole seconds; show milliseconds for sub-second waits
	display := formatDuration(duration)
	if duration < time.Second {
		display = fmt.Sprintf("%dms", duration.Milliseconds())
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would wait %s\n", display)
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "⏳  Waiting %s...\n", display)
	time.Sleep(duration)
	return nil
}
