package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

// executeTry executes try/catch/finally blocks
func (e *Engine) executeTry(tryStmt *statement.Try, ctx *ExecutionContext) error {
	var tryError error
	var finallyError error

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute try block\n")

		// Execute try body in dry run (domain statements)
		for _, stmt := range tryStmt.TryBody {
			if err := e.executeStatement(stmt, ctx); err != nil {
				_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would catch error: %v\n", err)
				break
			}
		}

		if len(tryStmt.CatchClauses) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute catch blocks if needed\n")
		}

		if len(tryStmt.FinallyBody) > 0 {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute finally block\n")
		}

		return nil
	}

	// Execute try block (domain statements)
	_, _ = fmt.Fprintf(e.output, "🔄  Executing try block\n")
	for _, stmt := range tryStmt.TryBody {
		if err := e.executeStatement(stmt, ctx); err != nil {
			tryError = err
			_, _ = fmt.Fprintf(e.output, "⚠️  Error in try block: %v\n", err)
			break
		}
	}

	// Execute catch blocks if there was an error
	handled := false
	if tryError != nil {
		for _, catchClause := range tryStmt.CatchClauses {
			if !e.shouldHandleError(tryError, catchClause) {
				continue
			}

			_, _ = fmt.Fprintf(e.output, "🔧 Handling error with catch block\n")

			// Set error variable if specified
			if catchClause.ErrorVar != "" {
				ctx.Variables[catchClause.ErrorVar] = tryError.Error()
				_, _ = fmt.Fprintf(e.output, "📦  Captured error in variable '%s'\n", catchClause.ErrorVar)
			}

			handled = true

			// Track the error being handled so a `rethrow` inside the catch
			// body can re-raise it with its message intact.
			savedCaughtError := ctx.CurrentCaughtError
			ctx.CurrentCaughtError = tryError

			// Execute the catch body. An error raised inside it is NOT
			// swallowed: it replaces the original error and propagates out of
			// the try statement.
			var catchBodyErr error
			for _, stmt := range catchClause.Body {
				if err := e.executeStatement(stmt, ctx); err != nil {
					catchBodyErr = err
					_, _ = fmt.Fprintf(e.output, "⚠️  Error in catch block: %v\n", err)
					break
				}
			}
			ctx.CurrentCaughtError = savedCaughtError

			if catchBodyErr != nil {
				tryError = catchBodyErr
			} else {
				tryError = nil // Error was handled successfully
			}
			break // only the first matching clause runs
		}
	}

	// Report how the error (if any) resolved.
	switch {
	case handled && tryError == nil:
		_, _ = fmt.Fprintf(e.output, "✅  Error handled successfully\n")
	case !handled && tryError != nil:
		_, _ = fmt.Fprintf(e.output, "❌  Unhandled error: %v\n", tryError)
	case tryError == nil:
		_, _ = fmt.Fprintf(e.output, "✅  Try block completed successfully\n")
	}
	// handled && tryError != nil means the matching catch body raised its own
	// error; the "Error in catch block" message above is the only report.

	// Always execute finally block (domain statements)
	if len(tryStmt.FinallyBody) > 0 {
		_, _ = fmt.Fprintf(e.output, "🔄  Executing finally block\n")
		for _, stmt := range tryStmt.FinallyBody {
			if err := e.executeStatement(stmt, ctx); err != nil {
				finallyError = err
				_, _ = fmt.Fprintf(e.output, "⚠️  Error in finally block: %v\n", err)
				break
			}
		}

		if finallyError == nil {
			_, _ = fmt.Fprintf(e.output, "✅  Finally block completed successfully\n")
		}
	}

	// Return the most relevant error
	if finallyError != nil {
		return finallyError // Finally errors take precedence
	}
	return tryError // Original error (if not handled)
}

// executeThrow executes throw, rethrow, and ignore statements
func (e *Engine) executeThrow(throwStmt *statement.Throw, ctx *ExecutionContext) error {
	if e.dryRun {
		switch throwStmt.Action {
		case "throw":
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would throw error: %s\n", throwStmt.Message)
		case "rethrow":
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would rethrow current error\n")
		case "ignore":
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would ignore current error\n")
		}
		return nil
	}

	switch throwStmt.Action {
	case "throw":
		message := e.interpolateVariables(throwStmt.Message, ctx)
		_, _ = fmt.Fprintf(e.output, "💥  Throwing error: %s\n", message)
		return fmt.Errorf("thrown error: %s", message)
	case "rethrow":
		if ctx == nil || ctx.CurrentCaughtError == nil {
			return errors.New("rethrow: no active caught error (rethrow only works inside a catch block)")
		}
		_, _ = fmt.Fprintf(e.output, "🔄  Rethrowing current error: %v\n", ctx.CurrentCaughtError)
		return ctx.CurrentCaughtError
	case "ignore":
		if ctx == nil || ctx.CurrentCaughtError == nil {
			return errors.New("ignore: bare 'ignore' only makes sense inside a catch block (it marks the caught error as handled); it cannot suppress a failing statement, because a command that fails already aborts the task")
		}
		_, _ = fmt.Fprintf(e.output, "🤐 Ignoring current error: %v\n", ctx.CurrentCaughtError)
		return nil // Explicit documentation-only no-op: the caught error is handled
	default:
		return fmt.Errorf("unknown throw action: %s", throwStmt.Action)
	}
}

// shouldHandleError checks if a catch clause should handle the given error
func (e *Engine) shouldHandleError(err error, catchClause statement.CatchClause) bool {
	// If no specific error type is specified, catch all errors
	if catchClause.ErrorType == "" {
		return true
	}

	// Simple error type matching based on error message content
	// In a more sophisticated implementation, we'd have typed errors
	errorMsg := strings.ToLower(err.Error())
	errorType := strings.ToLower(catchClause.ErrorType)

	switch errorType {
	case "filenotfounderror", "filenotfound":
		return strings.Contains(errorMsg, "no such file") ||
			strings.Contains(errorMsg, "not found") ||
			strings.Contains(errorMsg, "does not exist")
	case "shellerror", "commanderror":
		return strings.Contains(errorMsg, "command") ||
			strings.Contains(errorMsg, "shell") ||
			strings.Contains(errorMsg, "exit")
	case "permissionerror", "permission":
		return strings.Contains(errorMsg, "permission") ||
			strings.Contains(errorMsg, "access denied")
	default:
		// For custom error types, do a simple string match
		return strings.Contains(errorMsg, errorType)
	}
}
