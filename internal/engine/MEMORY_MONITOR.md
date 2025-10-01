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
- Reference to the full JSON dump file

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
```
⚠️  WARNING: Memory usage is high (100 MB)

❌ CRITICAL: Memory usage exceeded 500 MB (current: 523 MB)
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
