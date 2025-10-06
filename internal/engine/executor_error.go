package engine

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
)

// executeTry executes try/catch/finally blocks
func (e *Engine) executeTry(tryStmt *ast.TryStatement, ctx *ExecutionContext) error {
	var tryError error
	var finallyError error

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute try block\n")

		// Execute try body in dry run
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

	// Execute try block
	_, _ = fmt.Fprintf(e.output, "🔄 Executing try block\n")
	for _, stmt := range tryStmt.TryBody {
		if err := e.executeStatement(stmt, ctx); err != nil {
			tryError = err
			_, _ = fmt.Fprintf(e.output, "⚠️  Error in try block: %v\n", err)
			break
		}
	}

	// Execute catch blocks if there was an error
	if tryError != nil {
		handled := false
		for _, catchClause := range tryStmt.CatchClauses {
			if e.shouldHandleError(tryError, catchClause) {
				_, _ = fmt.Fprintf(e.output, "🔧 Handling error with catch block\n")

				// Set error variable if specified
				if catchClause.ErrorVar != "" {
					ctx.Variables[catchClause.ErrorVar] = tryError.Error()
					_, _ = fmt.Fprintf(e.output, "📦 Captured error in variable '%s'\n", catchClause.ErrorVar)
				}

				// Execute catch body
				for _, stmt := range catchClause.Body {
					if err := e.executeStatement(stmt, ctx); err != nil {
						// Error in catch block - this becomes the new error
						tryError = err
						break
					}
				}

				handled = true
				break
			}
		}

		if !handled {
			_, _ = fmt.Fprintf(e.output, "❌ Unhandled error: %v\n", tryError)
		} else {
			_, _ = fmt.Fprintf(e.output, "✅ Error handled successfully\n")
			tryError = nil // Error was handled
		}
	} else {
		_, _ = fmt.Fprintf(e.output, "✅ Try block completed successfully\n")
	}

	// Always execute finally block
	if len(tryStmt.FinallyBody) > 0 {
		_, _ = fmt.Fprintf(e.output, "🔄 Executing finally block\n")
		for _, stmt := range tryStmt.FinallyBody {
			if err := e.executeStatement(stmt, ctx); err != nil {
				finallyError = err
				_, _ = fmt.Fprintf(e.output, "⚠️  Error in finally block: %v\n", err)
				break
			}
		}

		if finallyError == nil {
			_, _ = fmt.Fprintf(e.output, "✅ Finally block completed successfully\n")
		}
	}

	// Return the most relevant error
	if finallyError != nil {
		return finallyError // Finally errors take precedence
	}
	return tryError // Original error (if not handled)
}

// executeThrow executes throw, rethrow, and ignore statements
func (e *Engine) executeThrow(throwStmt *ast.ThrowStatement, ctx *ExecutionContext) error {
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
		_, _ = fmt.Fprintf(e.output, "💥 Throwing error: %s\n", message)
		return fmt.Errorf("thrown error: %s", message)
	case "rethrow":
		_, _ = fmt.Fprintf(e.output, "🔄 Rethrowing current error\n")
		// In a real implementation, we'd need to track the current error context
		return fmt.Errorf("rethrown error")
	case "ignore":
		_, _ = fmt.Fprintf(e.output, "🤐 Ignoring current error\n")
		return nil // Ignore effectively suppresses the error
	default:
		return fmt.Errorf("unknown throw action: %s", throwStmt.Action)
	}
}

// shouldHandleError checks if a catch clause should handle the given error
func (e *Engine) shouldHandleError(err error, catchClause ast.CatchClause) bool {
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
