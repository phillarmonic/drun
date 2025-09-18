package parallel

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
)

func TestParallelExecutor_DryRun(t *testing.T) {
	var output bytes.Buffer
	executor := NewParallelExecutor(3, false, &output, true)

	items := []string{"item1", "item2", "item3", "item4"}
	variable := "item"
	body := []ast.Statement{} // empty body for test

	executeFunc := func(body []ast.Statement, variables map[string]string) error {
		return nil // no-op for dry run test
	}

	results, err := executor.ExecuteLoop(items, variable, body, executeFunc)
	if err != nil {
		t.Fatalf("ExecuteLoop failed: %v", err)
	}

	if len(results) != len(items) {
		t.Errorf("Expected %d results, got %d", len(items), len(results))
	}

	outputStr := output.String()
	expectedParts := []string{
		"[DRY RUN] Would execute 4 items in parallel",
		"max workers: 3",
		"Worker 1: item = item1",
		"Worker 2: item = item2",
		"Worker 3: item = item3",
		"Worker 4: item = item4",
		"All 4 parallel executions would complete",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestParallelExecutor_SuccessfulExecution(t *testing.T) {
	var output bytes.Buffer
	executor := NewParallelExecutor(2, false, &output, false)

	items := []string{"task1", "task2", "task3"}
	variable := "task"
	body := []ast.Statement{} // empty body for test

	executedItems := make(map[string]bool)
	executeFunc := func(body []ast.Statement, variables map[string]string) error {
		item := variables[variable]
		executedItems[item] = true
		time.Sleep(10 * time.Millisecond) // simulate work
		return nil
	}

	results, err := executor.ExecuteLoop(items, variable, body, executeFunc)
	if err != nil {
		t.Fatalf("ExecuteLoop failed: %v", err)
	}

	if len(results) != len(items) {
		t.Errorf("Expected %d results, got %d", len(items), len(results))
	}

	// Check that all items were executed
	for _, item := range items {
		if !executedItems[item] {
			t.Errorf("Item %s was not executed", item)
		}
	}

	// Check that all results are successful
	for i, result := range results {
		if result.Error != nil {
			t.Errorf("Result %d has error: %v", i, result.Error)
		}
		if result.Duration <= 0 {
			t.Errorf("Result %d has invalid duration: %v", i, result.Duration)
		}
	}

	outputStr := output.String()
	expectedParts := []string{
		"Starting parallel execution: 3 items, 2 workers",
		"Worker completed item",
		"Parallel execution completed: 3/3 items processed",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}
}

func TestParallelExecutor_WithErrors(t *testing.T) {
	var output bytes.Buffer
	executor := NewParallelExecutor(2, false, &output, false)

	items := []string{"success1", "error1", "success2", "error2"}
	variable := "item"
	body := []ast.Statement{} // empty body for test

	executeFunc := func(body []ast.Statement, variables map[string]string) error {
		item := variables[variable]
		if strings.Contains(item, "error") {
			return fmt.Errorf("simulated error for %s", item)
		}
		time.Sleep(5 * time.Millisecond) // simulate work
		return nil
	}

	results, err := executor.ExecuteLoop(items, variable, body, executeFunc)

	// Should return error due to failures
	if err == nil {
		t.Fatalf("Expected error due to failures, got nil")
	}

	if len(results) != len(items) {
		t.Errorf("Expected %d results, got %d", len(items), len(results))
	}

	// Count successful and failed results
	successCount := 0
	errorCount := 0
	for _, result := range results {
		if result.Error == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	if successCount != 2 {
		t.Errorf("Expected 2 successful results, got %d", successCount)
	}

	if errorCount != 2 {
		t.Errorf("Expected 2 error results, got %d", errorCount)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Worker failed on item") {
		t.Errorf("Expected output to contain failure messages")
	}
}

func TestParallelExecutor_FailFast(t *testing.T) {
	var output bytes.Buffer
	executor := NewParallelExecutor(2, true, &output, false) // fail-fast enabled

	items := []string{"success1", "error1", "success2", "success3"}
	variable := "item"
	body := []ast.Statement{} // empty body for test

	executeFunc := func(body []ast.Statement, variables map[string]string) error {
		item := variables[variable]
		if strings.Contains(item, "error") {
			return fmt.Errorf("simulated error for %s", item)
		}
		time.Sleep(20 * time.Millisecond) // simulate work
		return nil
	}

	_, err := executor.ExecuteLoop(items, variable, body, executeFunc)

	// Should return error due to fail-fast
	if err == nil {
		t.Fatalf("Expected error due to fail-fast, got nil")
	}

	if !strings.Contains(err.Error(), "fail-fast") {
		t.Errorf("Expected fail-fast error, got: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Worker failed on item") {
		t.Errorf("Expected output to contain failure messages")
	}
}

func TestParallelExecutor_EmptyItems(t *testing.T) {
	var output bytes.Buffer
	executor := NewParallelExecutor(2, false, &output, false)

	items := []string{}
	variable := "item"
	body := []ast.Statement{} // empty body for test

	executeFunc := func(body []ast.Statement, variables map[string]string) error {
		return nil
	}

	results, err := executor.ExecuteLoop(items, variable, body, executeFunc)
	if err != nil {
		t.Fatalf("ExecuteLoop failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty items, got %d", len(results))
	}
}

func TestParallelExecutor_WorkerLimiting(t *testing.T) {
	var output bytes.Buffer
	executor := NewParallelExecutor(2, false, &output, false)

	// More items than workers
	items := []string{"item1", "item2", "item3", "item4", "item5"}
	variable := "item"
	body := []ast.Statement{} // empty body for test

	concurrentCount := 0
	maxConcurrent := 0
	var mu sync.Mutex

	executeFunc := func(body []ast.Statement, variables map[string]string) error {
		mu.Lock()
		concurrentCount++
		if concurrentCount > maxConcurrent {
			maxConcurrent = concurrentCount
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond) // simulate work

		mu.Lock()
		concurrentCount--
		mu.Unlock()

		return nil
	}

	results, err := executor.ExecuteLoop(items, variable, body, executeFunc)
	if err != nil {
		t.Fatalf("ExecuteLoop failed: %v", err)
	}

	if len(results) != len(items) {
		t.Errorf("Expected %d results, got %d", len(items), len(results))
	}

	// Should not exceed the worker limit
	if maxConcurrent > 2 {
		t.Errorf("Expected max concurrent workers to be 2, got %d", maxConcurrent)
	}
}

func TestProgressTracker(t *testing.T) {
	var output bytes.Buffer
	tracker := NewProgressTracker(10, &output)

	// Test initial state
	completed, failed, total := tracker.GetStats()
	if completed != 0 || failed != 0 || total != 10 {
		t.Errorf("Expected (0, 0, 10), got (%d, %d, %d)", completed, failed, total)
	}

	if tracker.IsComplete() {
		t.Errorf("Expected tracker to not be complete initially")
	}

	// Update with successes and failures
	for i := 0; i < 7; i++ {
		tracker.Update(true) // success
	}
	for i := 0; i < 3; i++ {
		tracker.Update(false) // failure
	}

	// Check final state
	completed, failed, total = tracker.GetStats()
	if completed != 10 || failed != 3 || total != 10 {
		t.Errorf("Expected (10, 3, 10), got (%d, %d, %d)", completed, failed, total)
	}

	if !tracker.IsComplete() {
		t.Errorf("Expected tracker to be complete")
	}

	// Check output contains progress messages
	outputStr := output.String()
	if !strings.Contains(outputStr, "Progress:") {
		t.Errorf("Expected output to contain progress messages")
	}
}
