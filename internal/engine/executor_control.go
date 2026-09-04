package engine

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/parallel"
	"github.com/phillarmonic/drun/v2/internal/types"
)

// Domain: Control Flow Execution
// This file contains executors for:
// - Break/Continue control flow
// - Conditional statements (when/otherwise)
// - Loop statements (for each, range, line, match, parallel)
// - Loop filtering and context management

// BreakError represents a break statement execution
type BreakError struct {
	Condition string
}

// ContinueError represents a continue statement execution
type ContinueError struct {
	Condition string
}

// executeBreak executes break statements
func (e *Engine) executeBreak(breakStmt *statement.Break, ctx *ExecutionContext) error {
	condition := e.interpolateVariables(breakStmt.Condition, ctx)

	if e.dryRun {
		if condition != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would break when: %s\n", condition)
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would break\n")
		}
		return BreakError{Condition: condition}
	}

	if condition != "" {
		// Evaluate the condition
		if e.evaluateSimpleCondition(condition, ctx) {
			_, _ = fmt.Fprintf(e.output, "🔄  Breaking loop (condition: %s)\n", condition)
			return BreakError{Condition: condition}
		}
		// Condition not met, don't break
		return nil
	} else {
		_, _ = fmt.Fprintf(e.output, "🔄  Breaking loop\n")
		return BreakError{Condition: condition}
	}
}

// executeContinue executes continue statements
func (e *Engine) executeContinue(continueStmt *statement.Continue, ctx *ExecutionContext) error {
	condition := e.interpolateVariables(continueStmt.Condition, ctx)

	if e.dryRun {
		if condition != "" {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would continue if: %s\n", condition)
		} else {
			_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would continue\n")
		}
		return ContinueError{Condition: condition}
	}

	if condition != "" {
		// Evaluate the condition
		if e.evaluateSimpleCondition(condition, ctx) {
			_, _ = fmt.Fprintf(e.output, "🔄  Continuing loop (condition: %s)\n", condition)
			return ContinueError{Condition: condition}
		}
		// Condition not met, don't continue
		return nil
	} else {
		_, _ = fmt.Fprintf(e.output, "🔄  Continuing loop\n")
		return ContinueError{Condition: condition}
	}
}

// evaluateSimpleCondition evaluates simple conditions like "item == 'test'"
func (e *Engine) evaluateSimpleCondition(condition string, ctx *ExecutionContext) bool {
	// This is a simplified implementation
	// In a real implementation, you would parse and evaluate the condition properly
	// For now, we'll just return true to demonstrate the flow
	return true
}

func (e *Engine) executeConditional(stmt *statement.Conditional, ctx *ExecutionContext) error {
	// In strict mode, check for undefined variables in the condition
	// This checks both bare $var references and {var} interpolations
	if e.interpolator.IsStrictMode() {
		// Check for undefined variables in {var} interpolations
		if _, err := e.interpolateVariablesWithError(stmt.Condition, ctx); err != nil {
			return fmt.Errorf("in %s condition: %w", stmt.ConditionType, err)
		}
		// Check for undefined bare $var references (e.g., "when $var is value")
		if err := e.checkConditionForUndefinedVars(stmt.Condition, ctx); err != nil {
			return fmt.Errorf("in %s condition: %w", stmt.ConditionType, err)
		}
	}

	// File comparisons are exact byte comparisons and may return actionable I/O
	// errors. Version-aware conditions use numeric MAJOR.MINOR.PATCH ordering.
	// Port conditions run a native TCP probe. Docker conditions query the
	// daemon for resource existence. Other condition families continue
	// through the general evaluator.
	conditionResult, handled, err := e.evaluateFileComparisonCondition(stmt.Condition, ctx)
	if err != nil {
		return fmt.Errorf("in %s condition: %w", stmt.ConditionType, err)
	}
	if !handled {
		conditionResult, handled, err = e.evaluateSemanticVersionCondition(stmt.Condition, ctx)
		if err != nil {
			return fmt.Errorf("in %s condition: %w", stmt.ConditionType, err)
		}
	}
	if !handled {
		conditionResult, handled, err = e.evaluatePortCondition(stmt.Condition, ctx)
		if err != nil {
			return fmt.Errorf("in %s condition: %w", stmt.ConditionType, err)
		}
	}
	if !handled {
		conditionResult, handled, err = e.evaluateDockerCondition(stmt.Condition, ctx)
		if err != nil {
			return fmt.Errorf("in %s condition: %w", stmt.ConditionType, err)
		}
	}
	if !handled {
		conditionResult = e.evaluateCondition(stmt.Condition, ctx)
	}

	if conditionResult {
		// Execute the main body (domain statements)
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, ctx); err != nil {
				return err
			}
		}
	} else if len(stmt.ElseBody) > 0 {
		// Execute the else body if condition is false (domain statements)
		for _, elseStmt := range stmt.ElseBody {
			if err := e.executeStatement(elseStmt, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeLoop executes loop statements (for each)
func (e *Engine) executeLoop(stmt *statement.Loop, ctx *ExecutionContext) error {
	// If LoopType is not set, default to "each"
	loopType := stmt.LoopType
	if loopType == "" {
		loopType = "each"
	}

	switch loopType {
	case "range":
		return e.executeRangeLoop(stmt, ctx)
	case "line":
		return e.executeLineLoop(stmt, ctx)
	case "match":
		return e.executeMatchLoop(stmt, ctx)
	default: // "each"
		return e.executeEachLoop(stmt, ctx)
	}
}

// executeSequentialLoop executes loop items sequentially
func (e *Engine) executeSequentialLoop(stmt *statement.Loop, items []string, ctx *ExecutionContext) error {
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "🔄  Executing %d items sequentially\n", len(items))
	}

	for i, item := range items {
		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "📋 Processing item %d/%d: %s\n", i+1, len(items), item)
		}

		// Create a new context with the loop variable
		loopCtx := e.createLoopContext(ctx, stmt.Variable, item)

		// Execute the loop body (domain statements)
		for _, bodyStmt := range stmt.Body {
			if err := e.executeStatement(bodyStmt, loopCtx); err != nil {
				// Check for break/continue control flow
				if breakErr, ok := err.(BreakError); ok {
					if e.verbose {
						_, _ = fmt.Fprintf(e.output, "🔄  Breaking loop: %s\n", breakErr.Error())
					}
					return nil // Break out of the entire loop
				}
				if continueErr, ok := err.(ContinueError); ok {
					if e.verbose {
						_, _ = fmt.Fprintf(e.output, "🔄  Continuing loop: %s\n", continueErr.Error())
					}
					break // Break out of the body execution, continue to next item
				}
				return fmt.Errorf("error processing item '%s': %v", item, err)
			}
		}
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "✅  Sequential loop completed: %d items processed\n", len(items))
	}
	return nil
}

// executeParallelLoop executes loop items in parallel
func (e *Engine) executeParallelLoop(stmt *statement.Loop, items []string, ctx *ExecutionContext) error {
	// Determine parallel execution settings
	maxWorkers := stmt.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 5 // reasonable default
	}

	failFast := stmt.FailFast

	// Create parallel executor
	executor := parallel.NewParallelExecutor(maxWorkers, failFast, e.output, e.dryRun, e.verbose)

	// Define the execution function for each item (domain statements)
	executeItem := func(body []statement.Statement, variables map[string]string) error {
		// Create a new context for this parallel execution
		loopCtx := &ExecutionContext{
			Parameters: make(map[string]*types.Value, len(ctx.Parameters)+len(variables)), // Pre-allocate for parent + new variables
			Variables:  make(map[string]string, len(ctx.Variables)+len(variables)),        // Pre-allocate for parent + new variables
			Project:    ctx.Project,                                                       // inherit project context
		}

		// Copy existing parameters and variables
		for k, v := range ctx.Parameters {
			loopCtx.Parameters[k] = v
		}
		for k, v := range ctx.Variables {
			loopCtx.Variables[k] = v
		}

		// Add the variables from the parallel executor
		for k, v := range variables {
			loopCtx.Variables[k] = v
			// Also add as a typed parameter for compatibility
			if itemValue, err := types.NewValue(types.StringType, v); err == nil {
				loopCtx.Parameters[k] = itemValue
			}
		}

		// Execute the loop body (domain statements)
		for _, bodyStmt := range body {
			if err := e.executeStatement(bodyStmt, loopCtx); err != nil {
				return err
			}
		}

		return nil
	}

	// Execute in parallel
	results, err := executor.ExecuteLoop(items, stmt.Variable, stmt.Body, executeItem)
	// Report results
	if err != nil {
		// Count successful executions
		successCount := 0
		for _, result := range results {
			if result.Error == nil {
				successCount++
			}
		}

		if e.verbose {
			_, _ = fmt.Fprintf(e.output, "⚠️  Parallel loop completed with errors: %d/%d successful\n",
				successCount, len(items))
		}
		return err
	}

	return nil
}

// executeRangeLoop executes range loops
func (e *Engine) executeRangeLoop(stmt *statement.Loop, ctx *ExecutionContext) error {
	start := e.interpolateVariables(stmt.RangeStart, ctx)
	end := e.interpolateVariables(stmt.RangeEnd, ctx)
	step := "1"
	if stmt.RangeStep != "" {
		step = e.interpolateVariables(stmt.RangeStep, ctx)
	}

	// Parse the interpolated bounds into integers so they actually drive
	// iteration. Non-numeric bounds are a programming error, not something to
	// paper over with a default range.
	startInt, err := strconv.Atoi(strings.TrimSpace(start))
	if err != nil {
		return fmt.Errorf("range loop: start bound %q is not a valid integer", start)
	}
	endInt, err := strconv.Atoi(strings.TrimSpace(end))
	if err != nil {
		return fmt.Errorf("range loop: end bound %q is not a valid integer", end)
	}
	stepInt, err := strconv.Atoi(strings.TrimSpace(step))
	if err != nil {
		return fmt.Errorf("range loop: step %q is not a valid integer", step)
	}
	if stepInt == 0 {
		return fmt.Errorf("range loop: step cannot be zero (would loop forever)")
	}

	// Build the real range. Positive steps count up to (and including) end;
	// negative steps count down to (and including) end. A range whose step
	// points away from end simply yields no items.
	var items []string
	if stepInt > 0 {
		for i := startInt; i <= endInt; i += stepInt {
			items = append(items, fmt.Sprintf("%d", i))
		}
	} else {
		for i := startInt; i >= endInt; i += stepInt {
			items = append(items, fmt.Sprintf("%d", i))
		}
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute range loop from %s to %s step %s (%d items)\n", start, end, step, len(items))
		return nil
	}

	_, _ = fmt.Fprintf(e.output, "🔄  Executing range loop from %s to %s step %s (%d items)\n", start, end, step, len(items))

	// Apply filter if present
	if stmt.Filter != nil {
		items = e.applyFilter(items, stmt.Filter, ctx)
	}

	// Execute loop
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, items, ctx)
	}
	return e.executeSequentialLoop(stmt, items, ctx)
}

// executeLineLoop executes line-by-line file processing loops
func (e *Engine) executeLineLoop(stmt *statement.Loop, ctx *ExecutionContext) error {
	filename, err := e.interpolateVariablesWithError(stmt.Iterable, ctx)
	if err != nil {
		return fmt.Errorf("line loop: file path: %w", err)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would read lines from file: %s\n", filename)
		return nil
	}

	// Resolve relative paths against the current working directory
	// (use-workdir aware), then read the real file line by line.
	path := e.resolveFilesystemPath(filename, ctx)

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("line loop: cannot read file %q: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	var lines []string
	scanner := bufio.NewScanner(file)
	// Allow lines longer than the 64 KiB scanner default.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("line loop: reading file %q: %w", filename, err)
	}

	_, _ = fmt.Fprintf(e.output, "📄 Reading lines from file: %s (%d lines)\n", filename, len(lines))

	// Apply filter if present
	if stmt.Filter != nil {
		lines = e.applyFilter(lines, stmt.Filter, ctx)
	}

	// Execute loop
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, lines, ctx)
	}
	return e.executeSequentialLoop(stmt, lines, ctx)
}

// executeMatchLoop executes pattern matching loops.
//
// Syntax: for each match <var> in pattern "<regex>" of $subject
// The regex is compiled and every non-overlapping match found in the
// subject variable's value is bound to <var> for one iteration.
func (e *Engine) executeMatchLoop(stmt *statement.Loop, ctx *ExecutionContext) error {
	pattern := e.interpolateVariables(stmt.Iterable, ctx)

	if stmt.Subject == "" {
		return fmt.Errorf("match loop: no subject; use: for each match <var> in pattern %q of $subject", pattern)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would find matches for pattern: %s in %s\n", pattern, stmt.Subject)
		return nil
	}

	// Resolve the subject variable ($name, name, or a typed parameter).
	subjectRef := stmt.Subject
	var subject string
	if value, exists := ctx.Variables[subjectRef]; exists {
		subject = value
	} else if value, exists := ctx.Variables[strings.TrimPrefix(subjectRef, "$")]; exists {
		subject = value
	} else if param, exists := ctx.Parameters[strings.TrimPrefix(subjectRef, "$")]; exists {
		subject = param.AsString()
	} else {
		return fmt.Errorf("match loop: subject variable %q not found", subjectRef)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("match loop: invalid pattern %q: %w", pattern, err)
	}

	matches := re.FindAllString(subject, -1)

	_, _ = fmt.Fprintf(e.output, "🔍  Finding matches for pattern: %s in %s (%d matches)\n", pattern, subjectRef, len(matches))

	// Apply filter if present
	if stmt.Filter != nil {
		matches = e.applyFilter(matches, stmt.Filter, ctx)
	}

	// Execute loop
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, matches, ctx)
	}
	return e.executeSequentialLoop(stmt, matches, ctx)
}

// splitIterableString splits a raw string iterable into items. Values that
// contain a comma are treated as comma-separated lists (each item trimmed);
// otherwise whitespace separates items. Array-literal strings ([...]) are
// parsed by the caller before this helper is reached.
func splitIterableString(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		items := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				items = append(items, trimmed)
			}
		}
		return items
	}

	return strings.Fields(s)
}

// executeEachLoop executes traditional each loops
func (e *Engine) executeEachLoop(stmt *statement.Loop, ctx *ExecutionContext) error {
	// Resolve the iterable (could be a parameter, variable, array literal, etc.)
	var items []string

	// Check if it's an array literal (starts with '[')
	if strings.HasPrefix(stmt.Iterable, "[") && strings.HasSuffix(stmt.Iterable, "]") {
		// Parse array literal
		items = e.parseArrayLiteralString(stmt.Iterable)
	} else if strings.HasPrefix(stmt.Iterable, "$globals.") {
		// Handle $globals.key syntax for project settings (check this before general $ variables)
		if ctx.Project != nil && ctx.Project.Settings != nil {
			key := stmt.Iterable[9:] // Remove "$globals." prefix
			if projectValue, exists := ctx.Project.Settings[key]; exists {
				// Handle project setting (could be array or string)
				if strings.HasPrefix(projectValue, "[") && strings.HasSuffix(projectValue, "]") {
					// It's an array literal stored as a string
					items = e.parseArrayLiteralString(projectValue)
				} else {
					// It's a regular string, split by whitespace
					iterableStr := strings.TrimSpace(projectValue)
					if iterableStr == "" {
						_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
						return nil
					}
					items = splitIterableString(iterableStr)
				}
			} else {
				return fmt.Errorf("project setting '%s' not found", key)
			}
		} else {
			return fmt.Errorf("no project defined for $globals access")
		}
	} else if strings.HasPrefix(stmt.Iterable, "$") {
		// Variable reference
		var iterableStr string
		// Try both with and without $ prefix to handle different storage methods
		if value, exists := ctx.Variables[stmt.Iterable]; exists {
			iterableStr = value
		} else if value, exists := ctx.Variables[stmt.Iterable[1:]]; exists {
			iterableStr = value
		} else if param, exists := ctx.Parameters[stmt.Iterable[1:]]; exists {
			// Also check parameters (without the $ prefix)
			iterableStr = param.AsString()
		} else {
			return fmt.Errorf("variable '%s' not found", stmt.Iterable)
		}

		// Check if it's an array literal or a space-separated list
		iterableStr = strings.TrimSpace(iterableStr)
		if iterableStr == "" {
			_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
			return nil
		}

		// Check if it's an array literal (starts with '[' and ends with ']')
		if strings.HasPrefix(iterableStr, "[") && strings.HasSuffix(iterableStr, "]") {
			items = e.parseArrayLiteralString(iterableStr)
		} else {
			items = splitIterableString(iterableStr) // Split iterable string into items
		}
	} else {
		// Check if it's a legacy direct project setting access (for backward compatibility)
		if ctx.Project != nil && ctx.Project.Settings != nil {
			if projectValue, exists := ctx.Project.Settings[stmt.Iterable]; exists {
				// Handle project setting (could be array or string) - but warn about deprecated usage
				_, _ = fmt.Fprintf(e.output, "⚠️  Warning: Direct project setting access '%s' is deprecated. Use '$globals.%s' instead.\n", stmt.Iterable, stmt.Iterable)
				if strings.HasPrefix(projectValue, "[") && strings.HasSuffix(projectValue, "]") {
					// It's an array literal stored as a string
					items = e.parseArrayLiteralString(projectValue)
				} else {
					// It's a regular string, split by whitespace
					iterableStr := strings.TrimSpace(projectValue)
					if iterableStr == "" {
						_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
						return nil
					}
					items = splitIterableString(iterableStr)
				}
			} else {
				// Parameter reference
				iterableValue, exists := ctx.Parameters[stmt.Iterable]
				if !exists {
					return fmt.Errorf("iterable '%s' not found in parameters or project settings", stmt.Iterable)
				}
				iterableStr := iterableValue.AsString()

				// Split by space to get items (for our variable operations system)
				iterableStr = strings.TrimSpace(iterableStr)
				if iterableStr == "" {
					_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
					return nil
				}

				items = splitIterableString(iterableStr) // Split iterable string into items
			}
		} else {
			// Parameter reference (no project)
			iterableValue, exists := ctx.Parameters[stmt.Iterable]
			if !exists {
				return fmt.Errorf("iterable '%s' not found in parameters", stmt.Iterable)
			}
			iterableStr := iterableValue.AsString()

			// Split by space to get items (for our variable operations system)
			iterableStr = strings.TrimSpace(iterableStr)
			if iterableStr == "" {
				_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
				return nil
			}

			items = splitIterableString(iterableStr) // Split iterable string into items
		}
	}

	if len(items) == 0 {
		_, _ = fmt.Fprintf(e.output, "ℹ️  No items to process in loop\n")
		return nil
	}

	// Apply filter if present
	if stmt.Filter != nil {
		items = e.applyFilter(items, stmt.Filter, ctx)
	}

	// Check if this should run in parallel
	if stmt.Parallel {
		return e.executeParallelLoop(stmt, items, ctx)
	}

	// Sequential execution
	return e.executeSequentialLoop(stmt, items, ctx)
}

// applyFilter applies filter conditions to a list of items
func (e *Engine) applyFilter(items []string, filter *statement.Filter, ctx *ExecutionContext) []string {
	var filtered []string

	filterValue := e.interpolateVariables(filter.Value, ctx)

	for _, item := range items {
		match := false

		switch filter.Operator {
		case "contains":
			match = strings.Contains(item, filterValue)
		case "starts", "starts with":
			match = strings.HasPrefix(item, filterValue)
		case "ends", "ends with":
			match = strings.HasSuffix(item, filterValue)
		case "matches":
			// In a real implementation, you would use regex
			match = strings.Contains(item, filterValue)
		case "==":
			match = item == filterValue
		case "!=":
			match = item != filterValue
		default:
			// For other operators, just include the item
			match = true
		}

		if match {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) != len(items) {
		_, _ = fmt.Fprintf(e.output, "🔍  Filter applied: %d items match condition '%s %s %s'\n",
			len(filtered), filter.Variable, filter.Operator, filterValue)
	}

	return filtered
}

// createLoopContext creates a new execution context for a loop iteration
func (e *Engine) createLoopContext(ctx *ExecutionContext, variable, value string) *ExecutionContext {
	loopCtx := &ExecutionContext{
		Parameters: make(map[string]*types.Value, len(ctx.Parameters)+1), // Pre-allocate for parent + loop variable
		Variables:  make(map[string]string, len(ctx.Variables)+1),        // Pre-allocate for parent + loop variable
		Project:    ctx.Project,                                          // inherit project context
	}

	// Copy existing parameters and variables
	for k, v := range ctx.Parameters {
		loopCtx.Parameters[k] = v
	}
	for k, v := range ctx.Variables {
		loopCtx.Variables[k] = v
	}

	// Set the loop variable as a string type
	itemValue, _ := types.NewValue(types.StringType, value)
	loopCtx.Parameters[variable] = itemValue
	loopCtx.Variables[variable] = value

	return loopCtx
}
