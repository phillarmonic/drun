package parallel

import (
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

// TestWorkersCarryTracebackLabels parks every worker inside the loop body and
// then reads the all-goroutines stack dump. Go 1.27 prints pprof labels next to
// the goroutine state, which is what makes a crash dump name the worker.
func TestWorkersCarryTracebackLabels(t *testing.T) {
	executor := NewParallelExecutor(2, false, io.Discard, false, false)

	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	runBody := func([]statement.Statement, map[string]string) error {
		entered <- struct{}{}
		<-release
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := executor.ExecuteLoop([]string{"alpha", "beta"}, "item", nil, runBody); err != nil {
			t.Errorf("ExecuteLoop error = %v", err)
		}
	}()

	for range 2 {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			close(release)
			t.Fatal("workers did not reach the loop body")
		}
	}

	dump := make([]byte, 1<<20)
	// pprof quotes label values that are not bare identifiers (a hyphen is
	// enough), so drop the quotes before looking for them.
	stack := strings.ReplaceAll(string(dump[:runtime.Stack(dump, true)]), `"`, "")
	close(release)
	<-done

	for _, want := range []string{
		"drun.component: parallel-worker",
		"drun.variable: item",
		"drun.worker: 1",
	} {
		if !strings.Contains(stack, want) {
			t.Errorf("stack dump is missing %q", want)
		}
	}
}
