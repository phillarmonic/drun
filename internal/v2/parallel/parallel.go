package parallel

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/phillarmonic/drun/internal/v2/ast"
)

// ExecutionResult represents the result of a parallel execution
type ExecutionResult struct {
	Index    int           // index of the item in the original list
	Item     string        // the item being processed
	Error    error         // error if execution failed
	Duration time.Duration // how long the execution took
	Output   string        // captured output from the execution
}

// ParallelExecutor manages parallel execution of loop bodies
type ParallelExecutor struct {
	maxWorkers int
	failFast   bool
	output     io.Writer
	dryRun     bool
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(maxWorkers int, failFast bool, output io.Writer, dryRun bool) *ParallelExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 10 // default reasonable limit
	}
	return &ParallelExecutor{
		maxWorkers: maxWorkers,
		failFast:   failFast,
		output:     output,
		dryRun:     dryRun,
	}
}

// ExecuteLoop executes a loop body in parallel for each item
func (pe *ParallelExecutor) ExecuteLoop(
	items []string,
	variable string,
	body []ast.Statement,
	executor func([]ast.Statement, map[string]string) error,
) ([]ExecutionResult, error) {
	if pe.dryRun {
		return pe.executeDryRun(items, variable, body)
	}

	return pe.executeParallel(items, variable, body, executor)
}

// executeDryRun simulates parallel execution in dry run mode
func (pe *ParallelExecutor) executeDryRun(items []string, variable string, body []ast.Statement) ([]ExecutionResult, error) {
	_, _ = fmt.Fprintf(pe.output, "[DRY RUN] Would execute %d items in parallel (max workers: %d)\n",
		len(items), pe.maxWorkers)

	results := make([]ExecutionResult, len(items))
	for i, item := range items {
		_, _ = fmt.Fprintf(pe.output, "[DRY RUN] Worker %d: %s = %s\n", i+1, variable, item)
		results[i] = ExecutionResult{
			Index:    i,
			Item:     item,
			Duration: time.Millisecond * 100, // simulate execution time
		}
	}

	_, _ = fmt.Fprintf(pe.output, "[DRY RUN] All %d parallel executions would complete\n", len(items))
	return results, nil
}

// executeParallel executes the loop body in parallel for each item
func (pe *ParallelExecutor) executeParallel(
	items []string,
	variable string,
	body []ast.Statement,
	executor func([]ast.Statement, map[string]string) error,
) ([]ExecutionResult, error) {
	numItems := len(items)
	if numItems == 0 {
		return []ExecutionResult{}, nil
	}

	// Determine actual number of workers
	workers := pe.maxWorkers
	if workers > numItems {
		workers = numItems
	}

	_, _ = fmt.Fprintf(pe.output, "🔄 Starting parallel execution: %d items, %d workers\n", numItems, workers)

	// Create channels for work distribution and result collection
	workChan := make(chan workItem, numItems)
	resultChan := make(chan ExecutionResult, numItems)

	// Context for cancellation (fail-fast)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go pe.worker(ctx, i+1, workChan, resultChan, variable, body, executor, &wg)
	}

	// Send work items
	go func() {
		defer close(workChan)
		for i, item := range items {
			select {
			case workChan <- workItem{index: i, item: item}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	results := make([]ExecutionResult, numItems)
	var firstError error
	completedCount := 0

	for i := 0; i < numItems; i++ {
		select {
		case result := <-resultChan:
			results[result.Index] = result
			completedCount++

			if result.Error != nil {
				_, _ = fmt.Fprintf(pe.output, "❌ Worker failed on item %d (%s): %v\n",
					result.Index+1, result.Item, result.Error)

				if pe.failFast && firstError == nil {
					firstError = result.Error
					cancel() // stop all workers
				}
			} else {
				_, _ = fmt.Fprintf(pe.output, "✅ Worker completed item %d (%s) in %v\n",
					result.Index+1, result.Item, result.Duration)
			}

		case <-ctx.Done():
			// Context cancelled due to fail-fast
			goto collectRemaining
		}
	}

collectRemaining:
	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Drain any remaining results
	for result := range resultChan {
		if result.Index < len(results) {
			results[result.Index] = result
			completedCount++
		}
	}

	_, _ = fmt.Fprintf(pe.output, "🏁 Parallel execution completed: %d/%d items processed\n",
		completedCount, numItems)

	// Count errors
	errorCount := 0
	for _, result := range results {
		if result.Error != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		_, _ = fmt.Fprintf(pe.output, "⚠️  %d items failed during parallel execution\n", errorCount)

		if pe.failFast && firstError != nil {
			return results, fmt.Errorf("parallel execution failed (fail-fast): %v", firstError)
		}

		return results, fmt.Errorf("parallel execution completed with %d errors", errorCount)
	}

	return results, nil
}

// workItem represents a single item of work
type workItem struct {
	index int
	item  string
}

// worker processes work items from the work channel
func (pe *ParallelExecutor) worker(
	ctx context.Context,
	workerID int,
	workChan <-chan workItem,
	resultChan chan<- ExecutionResult,
	variable string,
	body []ast.Statement,
	executor func([]ast.Statement, map[string]string) error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case work, ok := <-workChan:
			if !ok {
				return // channel closed
			}

			// Execute the work item
			result := pe.executeWorkItem(workerID, work, variable, body, executor)

			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// executeWorkItem executes the loop body for a single item
func (pe *ParallelExecutor) executeWorkItem(
	workerID int,
	work workItem,
	variable string,
	body []ast.Statement,
	executor func([]ast.Statement, map[string]string) error,
) ExecutionResult {
	start := time.Now()

	// Create context with the loop variable
	context := map[string]string{
		variable: work.item,
	}

	// Execute the loop body
	err := executor(body, context)
	duration := time.Since(start)

	return ExecutionResult{
		Index:    work.index,
		Item:     work.item,
		Error:    err,
		Duration: duration,
	}
}

// ProgressTracker tracks progress of parallel execution
type ProgressTracker struct {
	total     int
	completed int
	failed    int
	mu        sync.Mutex
	output    io.Writer
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int, output io.Writer) *ProgressTracker {
	return &ProgressTracker{
		total:  total,
		output: output,
	}
}

// Update updates the progress tracker
func (pt *ProgressTracker) Update(success bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.completed++
	if !success {
		pt.failed++
	}

	// Print progress every 10% or on completion
	percentage := (pt.completed * 100) / pt.total
	if pt.completed == pt.total || pt.completed%max(1, pt.total/10) == 0 {
		_, _ = fmt.Fprintf(pt.output, "📊 Progress: %d/%d (%d%%) - %d failed\n",
			pt.completed, pt.total, percentage, pt.failed)
	}
}

// IsComplete returns whether all items have been processed
func (pt *ProgressTracker) IsComplete() bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.completed >= pt.total
}

// GetStats returns current statistics
func (pt *ProgressTracker) GetStats() (completed, failed, total int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.completed, pt.failed, pt.total
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
