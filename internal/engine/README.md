# Engine Package Architecture

## Overview

The `engine` package is the execution engine for drun. It has been refactored from a monolithic 5,182-line file into a modular, domain-driven architecture.

## Refactoring Results

- **Original:** `engine.go` - 5,182 lines (monolithic)
- **Current:** `engine.go` - 918 lines (orchestration core)
- **Total Reduction:** 82.3% reduction in main file size
- **Files Created:** 35 domain-specific files
- **Total Package Lines:** ~9,200 lines (well-organized)

## Package Structure

### Core Files

#### `engine.go` (918 lines)
- **Purpose:** Main orchestration and public API
- **Contents:**
  - Engine struct and constructor
  - Public methods (Execute, SetDryRun, SetVerbose, etc.)
  - Task execution orchestration
  - Statement routing
  - Project context creation

#### `context.go` (121 lines)
- **Purpose:** Execution context management
- **Contents:**
  - `ExecutionContext` - Runtime parameter and variable storage
  - `ProjectContext` - Project-level configuration and settings
  - Interface implementations for interpolation and includes packages

### Sub-Packages

#### `interpolation/` (670 lines across 4 files)
- **Purpose:** Variable and expression interpolation
- **Files:**
  - `interpolator.go` - Main interpolation engine
  - `resolvers.go` - Variable resolution logic
  - `conditional.go` - Ternary and if-then-else expressions
  - `utilities.go` - Helper functions

#### `hooks/` (91 lines)
- **Purpose:** Lifecycle hook management
- **Files:**
  - `manager.go` - Hook registration and execution

#### `includes/` (315 lines)
- **Purpose:** Remote file inclusion and caching
- **Files:**
  - `resolver.go` - Include resolution, fetching, and merging

### Executor Files (1,703 lines across 10 files)

Domain-specific statement execution:

1. **`executor_control.go` (551 lines)**
   - Conditional statements (when/otherwise)
   - Loop statements (for each, range, line, match)
   - Parallel and sequential execution
   - Break/continue control flow

2. **`executor_variables.go` (227 lines)**
   - Variable declarations (let, set)
   - Transformations (uppercase, lowercase, trim, concat, split, etc.)
   - Capture operations

3. **`executor_error.go` (166 lines)**
   - Try/catch/finally blocks
   - Throw/rethrow/ignore statements
   - Error handling and matching

4. **`executor_network.go` (150 lines)**
   - Network connectivity checks
   - Health checks and port testing
   - File downloads (HTTP/HTTPS)

5. **`executor_shell.go` (147 lines)**
   - Shell command execution
   - Multi-line shell scripts
   - Platform-specific shell configuration

6. **`executor_file.go` (125 lines)**
   - File operations (create, delete, copy, move)
   - File permission management

7. **`executor_git.go` (116 lines)**
   - Git operations (clone, commit, push, pull)
   - Branch management

8. **`executor_docker.go` (115 lines)**
   - Docker operations (build, run, push, pull)
   - Docker Compose management

9. **`executor_http.go` (72 lines)**
   - HTTP requests (GET, POST, PUT, DELETE, PATCH)
   - API interactions

10. **`executor_detection.go` (34 lines)**
    - Tool and command detection
    - Environment detection

### Helper Files (1,885 lines across 7 files)

Supporting functionality organized by domain:

1. **`helpers_builders.go` (416 lines)**
   - Command builders for Docker, Git, HTTP, Network operations
   - Shell command construction

2. **`helpers_expressions.go` (354 lines)**
   - Expression evaluation (binary, function calls)
   - Builtin operations parsing and application
   - Variable operation chains

3. **`helpers_conditions.go` (332 lines)**
   - Condition evaluation logic
   - Environment variable condition checking
   - Strict variable checking

4. **`helpers_download.go` (331 lines)**
   - Download progress tracking
   - Archive extraction
   - File permission application

5. **`helpers_detection.go` (271 lines)**
   - Detection operation execution
   - Tool availability checks
   - Version and environment detection

6. **`helpers_utilities.go` (128 lines)**
   - Array literal parsing
   - Single-line shell execution
   - Miscellaneous utilities

7. **`helpers_filesystem.go` (53 lines)**
   - File and directory existence checks
   - File size and directory empty checks

### Legacy Files

- `dependency.go` - Dependency resolution
- `memory_monitor.go` - Memory usage monitoring
- `variable_operations.go` - Legacy variable operations (to be refactored)

## Architecture Principles

### 1. Domain-Driven Design
Each file is organized around a specific domain or responsibility:
- Executors handle specific statement types
- Helpers provide supporting functionality
- Sub-packages encapsulate complex subsystems

### 2. Single Responsibility
- Each file has a clear, focused purpose
- Methods are grouped by their domain
- No file exceeds ~600 lines

### 3. Separation of Concerns
- Orchestration (engine.go) is separate from execution (executors)
- Supporting systems (interpolation, hooks, includes) are isolated
- Helper methods are categorized by functionality

### 4. Testability
- Smaller, focused files are easier to test
- Clear boundaries make mocking simpler
- Domain separation enables targeted testing

## Key Interfaces

### Engine Public API
```go
// Core execution
Execute(program *ast.Program, taskName string) error
ExecuteWithParams(program *ast.Program, taskName string, params map[string]string) error

// Configuration
SetDryRun(dryRun bool)
SetVerbose(verbose bool)
SetAllowUndefinedVars(allow bool)
SetCacheEnabled(enabled bool) error

// Cleanup
Cleanup()

// Task listing
ListTasks(program *ast.Program) []TaskInfo
```

### Execution Context
```go
type ExecutionContext struct {
    Parameters       map[string]*types.Value
    Variables        map[string]string
    Project          *ProjectContext
    CurrentFile      string
    CurrentTask      string
    CurrentNamespace string
    Program          *ast.Program
}
```

### Project Context
```go
type ProjectContext struct {
    Name              string
    Version           string
    Settings          map[string]string
    Parameters        map[string]*ast.ProjectParameterStatement
    Snippets          map[string]*ast.SnippetStatement
    HookManager       *hooks.Manager
    ShellConfigs      map[string]*ast.PlatformShellConfig
    IncludedSnippets  map[string]*ast.SnippetStatement
    IncludedTemplates map[string]*ast.TaskTemplateStatement
    IncludedTasks     map[string]*ast.TaskStatement
    IncludedFiles     map[string]bool
}
```

## Execution Flow

1. **Parse** → AST Program
2. **Create Context** → Project + Execution contexts
3. **Resolve Dependencies** → Task ordering
4. **Execute Hooks** → Setup hooks
5. **Execute Task** → Statement-by-statement execution
6. **Route Statements** → Appropriate executors
7. **Execute Hooks** → Teardown hooks
8. **Cleanup** → Resource cleanup

## Benefits of Refactoring

### Code Quality
✅ Reduced complexity (5,182 → 918 lines in core)  
✅ Improved readability (clear domain boundaries)  
✅ Better maintainability (focused files)  
✅ Enhanced testability (isolated domains)  

### Development Experience
✅ Easier to navigate (logical file organization)  
✅ Faster to understand (smaller, focused files)  
✅ Simpler to extend (clear extension points)  
✅ Reduced merge conflicts (distributed changes)  

### Performance
✅ No performance impact (same logic, better organized)  
✅ All 58 examples passing  
✅ All unit tests passing  
✅ Zero regressions  

## Future Enhancements

Potential areas for further improvement:

1. **Extract more sub-packages** for complex domains (e.g., detection, actions)
2. **Interface extraction** for better mocking and testing
3. **Dependency injection** for cleaner initialization
4. **Event system** for better observability
5. **Plugin architecture** for extensibility

## Maintenance Guidelines

### Adding New Executors
1. Create new `executor_<domain>.go` file
2. Add domain header comment
3. Implement executor methods
4. Update `executeStatement` router in `engine.go`
5. Add tests

### Adding New Helpers
1. Identify the appropriate `helpers_<category>.go` file
2. Add helper methods with clear documentation
3. Update imports if needed
4. Add tests

### Modifying Core Orchestration
1. Changes to `engine.go` should be minimal
2. Prefer extracting to executors or helpers
3. Maintain clear separation of concerns
4. Update documentation

---

**Last Updated:** Phase 3 Refactoring - October 2025  
**Status:** ✅ Complete - All tests passing, zero regressions
