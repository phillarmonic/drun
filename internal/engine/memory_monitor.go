package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/phillarmonic/drun/internal/ast"
)

const (
	// Memory thresholds
	WarningThresholdMB  = 100 // 100 MB - log warning
	CriticalThresholdMB = 500 // 500 MB - dump and exit
	CheckIntervalMS     = 100 // Check every 100ms
)

// MemoryMonitor monitors memory usage and dumps diagnostics if threshold exceeded
type MemoryMonitor struct {
	program       *ast.Program
	ctx           context.Context
	cancel        context.CancelFunc
	warningLogged bool
}

// MemoryStats holds memory usage information
type MemoryStats struct {
	AllocMB      uint64    `json:"alloc_mb"`
	TotalAllocMB uint64    `json:"total_alloc_mb"`
	SysMB        uint64    `json:"sys_mb"`
	NumGC        uint32    `json:"num_gc"`
	Timestamp    time.Time `json:"timestamp"`
}

// DiagnosticDump contains all diagnostic information
type DiagnosticDump struct {
	MemoryStats MemoryStats            `json:"memory_stats"`
	Program     *ast.Program           `json:"program"`
	RuntimeInfo map[string]interface{} `json:"runtime_info"`
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(program *ast.Program) *MemoryMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &MemoryMonitor{
		program: program,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins monitoring memory usage
func (m *MemoryMonitor) Start() {
	go m.monitorLoop()
}

// Stop stops the memory monitor
func (m *MemoryMonitor) Stop() {
	m.cancel()
}

// monitorLoop runs the monitoring loop
func (m *MemoryMonitor) monitorLoop() {
	ticker := time.NewTicker(CheckIntervalMS * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkMemory()
		}
	}
}

// checkMemory checks current memory usage and takes action if needed
func (m *MemoryMonitor) checkMemory() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	allocMB := mem.Alloc / 1024 / 1024

	// Critical threshold - dump and exit
	if allocMB > CriticalThresholdMB {
		m.dumpDiagnostics(mem)
		fmt.Fprintf(os.Stderr, "\n❌ CRITICAL: Memory usage exceeded %d MB (current: %d MB)\n", CriticalThresholdMB, allocMB)
		fmt.Fprintf(os.Stderr, "Diagnostic information dumped to drun-crash-dump.json\n")
		fmt.Fprintf(os.Stderr, "This likely indicates an infinite loop or runaway recursion.\n")
		os.Exit(1)
	}

	// Warning threshold - log once
	if allocMB > WarningThresholdMB && !m.warningLogged {
		fmt.Fprintf(os.Stderr, "\n⚠️  WARNING: Memory usage is high (%d MB)\n", allocMB)
		m.warningLogged = true
	}
}

// dumpDiagnostics dumps diagnostic information to a file
func (m *MemoryMonitor) dumpDiagnostics(mem runtime.MemStats) {
	stats := MemoryStats{
		AllocMB:      mem.Alloc / 1024 / 1024,
		TotalAllocMB: mem.TotalAlloc / 1024 / 1024,
		SysMB:        mem.Sys / 1024 / 1024,
		NumGC:        mem.NumGC,
		Timestamp:    time.Now(),
	}

	dump := DiagnosticDump{
		MemoryStats: stats,
		Program:     m.program,
		RuntimeInfo: map[string]interface{}{
			"go_version":    runtime.Version(),
			"num_goroutine": runtime.NumGoroutine(),
			"num_cpu":       runtime.NumCPU(),
			"os":            runtime.GOOS,
			"arch":          runtime.GOARCH,
		},
	}

	// Create dump file
	filename := fmt.Sprintf("drun-crash-dump-%s.json", time.Now().Format("20060102-150405"))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create dump file: %v\n", err)
		return
	}
	defer func() { _ = file.Close() }()

	// Write JSON dump
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dump); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write dump: %v\n", err)
		return
	}

	// Also create a simple text summary
	summaryFile := fmt.Sprintf("drun-crash-summary-%s.txt", time.Now().Format("20060102-150405"))
	f, err := os.Create(summaryFile)
	if err == nil {
		defer func() { _ = f.Close() }()
		_, _ = fmt.Fprintf(f, "DRUN CRASH DUMP SUMMARY\n")
		_, _ = fmt.Fprintf(f, "======================\n\n")
		_, _ = fmt.Fprintf(f, "Timestamp: %s\n", stats.Timestamp.Format(time.RFC3339))
		_, _ = fmt.Fprintf(f, "Memory Allocated: %d MB\n", stats.AllocMB)
		_, _ = fmt.Fprintf(f, "Total Allocated: %d MB\n", stats.TotalAllocMB)
		_, _ = fmt.Fprintf(f, "System Memory: %d MB\n", stats.SysMB)
		_, _ = fmt.Fprintf(f, "Garbage Collections: %d\n", stats.NumGC)
		_, _ = fmt.Fprintf(f, "\nRuntime Info:\n")
		_, _ = fmt.Fprintf(f, "  Go Version: %s\n", runtime.Version())
		_, _ = fmt.Fprintf(f, "  Goroutines: %d\n", runtime.NumGoroutine())
		_, _ = fmt.Fprintf(f, "  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		_, _ = fmt.Fprintf(f, "\nProgram Info:\n")
		if m.program.Version != nil {
			_, _ = fmt.Fprintf(f, "  Version: %s\n", m.program.Version.Value)
		}
		_, _ = fmt.Fprintf(f, "  Tasks: %d\n", len(m.program.Tasks))
		if m.program.Project != nil {
			_, _ = fmt.Fprintf(f, "  Project: %s\n", m.program.Project.Name)
		}
		_, _ = fmt.Fprintf(f, "\nFull details in: %s\n", filename)
	}

	// Get absolute path for user-friendly message
	absPath, _ := filepath.Abs(filename)
	fmt.Fprintf(os.Stderr, "Diagnostic dump written to: %s\n", absPath)
}
