# Memory Monitor

The drun engine includes an automatic memory monitoring system to detect and handle runaway execution scenarios, such as infinite loops or uncontrolled recursion.

## Overview

The memory monitor runs in the background during task execution and tracks memory usage. If memory consumption exceeds safe thresholds, it automatically dumps diagnostic information and terminates execution to prevent system crashes.

## Thresholds

### Warning Threshold: 100 MB
- When memory usage exceeds 100 MB, a warning is logged to stderr
- Execution continues normally
- Warning is logged only once per execution

### Critical Threshold: 500 MB
- When memory usage exceeds 500 MB, execution is immediately terminated
- Diagnostic information is dumped to files
- Process exits with code 1

### Check Interval
- Memory is checked every 100 milliseconds during execution
- Minimal performance impact on normal execution

## Diagnostic Output

When the critical threshold is exceeded, the monitor creates two files:

### 1. JSON Dump File
**Filename:** `drun-crash-dump-YYYYMMDD-HHMMSS.json`

Contains:

- Detailed memory statistics (allocation, total allocated, system memory, GC count)
- Complete AST program structure
- Runtime information (Go version, goroutines, CPU, OS/arch)
- Timestamp of the crash

### 2. Text Summary File
**Filename:** `drun-crash-summary-YYYYMMDD-HHMMSS.txt`

Human-readable summary containing:

- Memory usage statistics
- Runtime information
- Program metadata (version, task count, project name)
- The runtime's goroutineleak profile, which lists every goroutine that is
  blocked forever together with the drun labels of the worker that owns it
  (see [Goroutine labels](#goroutine-labels))
- Reference to the full JSON dump file

The same profile is also written next to the dump as
`drun-goroutineleak-YYYYMMDD-HHMMSS.pprof`, which `go tool pprof` reads.

## Goroutine labels

Every goroutine drun spawns carries pprof labels, so a panic or `SIGQUIT`
traceback names the worker instead of an anonymous goroutine:

| Label | Meaning |
| --- | --- |
| `drun.component` | The subsystem, for example `parallel-worker`, `parallel-feeder`, `parallel-collector`, `memory-monitor`, `orchestration-health`, `orchestration-recovery`, `orchestration-service-start` |
| `drun.worker` | The 1-based parallel worker index |
| `drun.variable` | The loop variable a parallel worker is running |
| `drun.orchestration` | The orchestration an engine goroutine belongs to |
| `drun.service` | The service an orchestration start goroutine is starting |

Labels are attached once per goroutine, never once per work item, and the
goroutineleak profile is only captured on the crash path, so nothing is written
while debug capture is off.

## Usage

The memory monitor is automatically enabled for all task executions. No configuration is required.

## Example Scenarios

### Infinite Loop Detection
```drun
# This would trigger the monitor if it causes memory growth
task "bad-task":
  call task hello-world  # Without quotes, creates infinite parsing loop
```

When detected:

```text
  WARNING: Memory usage is high (100 MB)

 CRITICAL: Memory usage exceeded 500 MB (current: 523 MB)
Diagnostic information dumped to drun-crash-dump-20250930-143022.json
This likely indicates an infinite loop or runaway recursion.
```

### Normal Execution
Normal task execution, even with intensive operations, should stay well below the thresholds:

```drun
task "intensive":
  for each $i in ["1", "2", "3", "4", "5", "6", "7", "8", "9", "10"]:
    for each $j in ["1", "2", "3", "4", "5", "6", "7", "8", "9", "10"]:
      info "Processing {$i}-{$j}"
```

This completes normally without triggering any warnings.

## Implementation Details

- **Goroutine-based**: Runs in a separate goroutine with minimal overhead
- **Context-aware**: Properly cleaned up when execution completes
- **Thread-safe**: Uses Go's runtime.ReadMemStats for accurate measurements
- **Zero configuration**: Automatically enabled, no setup required

## Troubleshooting

If you encounter a memory dump:

1. **Check the summary file** for quick diagnosis
2. **Review the JSON dump** for detailed AST and execution state
3. **Look for**:
   - Infinite loops (tasks calling themselves)
   - Unbounded recursion
   - Parser bugs (e.g., unquoted kebab-case task names)
   - Large data structure creation in loops

## Testing

The memory monitor includes unit tests:

```bash
go test ./internal/engine -run TestMemoryMonitor -v
```

## Performance Impact

- **CPU overhead**: ~0.1% (one check every 100ms)
- **Memory overhead**: Negligible (<1 MB)
- **Latency impact**: None (runs in background goroutine)
