package engine

import (
	"runtime"
	"testing"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
)

func TestMemoryMonitorBasic(t *testing.T) {
	program := &ast.Program{
		Version: &ast.VersionStatement{Value: "2.0"},
		Tasks:   []*ast.TaskStatement{},
	}

	monitor := NewMemoryMonitor(program)
	monitor.Start()

	// Let it run for a bit
	time.Sleep(200 * time.Millisecond)

	monitor.Stop()

	// Should not crash or panic
	t.Log("Memory monitor started and stopped successfully")
}

func TestMemoryMonitorStats(t *testing.T) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	allocMB := mem.Alloc / 1024 / 1024
	t.Logf("Current memory allocation: %d MB", allocMB)

	if allocMB > CriticalThresholdMB {
		t.Errorf("Test environment already exceeds critical threshold: %d MB > %d MB", allocMB, CriticalThresholdMB)
	}
}

func TestMemoryStatsCreation(t *testing.T) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	stats := MemoryStats{
		AllocMB:      mem.Alloc / 1024 / 1024,
		TotalAllocMB: mem.TotalAlloc / 1024 / 1024,
		SysMB:        mem.Sys / 1024 / 1024,
		NumGC:        mem.NumGC,
		Timestamp:    time.Now(),
	}

	// Check that we can read memory stats - verify raw values are readable
	if mem.Sys == 0 {
		t.Error("Unable to read system memory stats")
	}

	// Verify timestamp is set
	if stats.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	// Note: AllocMB and TotalAllocMB can be 0 if allocation is < 1MB, which is normal for small programs
	// The important thing is that we can read the stats without panicking
	t.Logf("Memory stats: Alloc=%dMB, TotalAlloc=%dMB, Sys=%dMB, NumGC=%d",
		stats.AllocMB, stats.TotalAllocMB, stats.SysMB, stats.NumGC)
	t.Logf("Raw memory: Alloc=%d bytes, TotalAlloc=%d bytes, Sys=%d bytes",
		mem.Alloc, mem.TotalAlloc, mem.Sys)
}
