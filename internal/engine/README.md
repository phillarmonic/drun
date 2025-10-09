# Engine Package Architecture

## Overview

The `engine` package is the execution engine for drun. It uses a **modular, domain-driven architecture** with clear separation of concerns through specialized components: Planner, Executor, and supporting subsystems.

## Current Architecture

- **Core Orchestration:** `engine.go` - Main engine coordinating execution flow
- **Execution Planning:** `planner/` - Dependency resolution and execution plan generation
- **Task Execution:** `executor/` - Task and hook execution with domain statements
- **Configuration:** Options-based dependency injection for testability
- **Subsystems:** Interpolation, hooks, includes as focused packages

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

### Component Packages

#### `planner/` (Execution Planning)
- **Purpose:** Dependency resolution and execution plan generation
- **Files:**
  - `planner.go` - Main planner with Plan() method
  - `planner_test.go` - Planner unit tests
- **Key Types:**
  - `ExecutionPlan` - Comprehensive execution plan with all metadata
  - `TaskPlan` - Individual task with domain statements and parameters
  - `HookPlan` - Lifecycle hooks (setup, before, after, teardown)
  - `ProjectContext` - Project-level information for planning

**Benefits:**
- Single upfront dependency resolution
- Deterministic execution order
- No redundant AST scans
- Rich debugging information

#### `executor/` (Task Execution)
- **Purpose:** Execute tasks and lifecycle hooks using domain statements
- **Files:**
  - `executor.go` - Main executor
  - `executor_test.go` - Executor unit tests
- **Key Types:**
  - `Executor` - Handles task and hook execution
  - `DomainStatementExecutor` - Interface for statement execution
- **Features:**
  - Direct domain statement execution
  - Lifecycle hook management
  - Dry-run support
  - Error handling

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

## Dependency Injection & Configuration

### Options-Based Constructor

The engine supports pluggable infrastructure through `NewEngineWithOptions`:

```go
// Example: Custom configuration
engine := NewEngineWithOptions(
    WithOutput(customWriter),
    WithTaskRegistry(customRegistry),
    WithParamValidator(customValidator),
    WithDepResolver(customResolver),
    WithCacheManager(customCache),
    WithVerbose(true),
    WithDryRun(false),
)
```

### Available Options (`options.go`)

- `WithOutput(io.Writer)` - Custom output writer
- `WithTaskRegistry(*task.Registry)` - Custom task registry
- `WithParamValidator(*parameter.Validator)` - Custom parameter validator
- `WithDepResolver(*task.DependencyResolver)` - Custom dependency resolver
- `WithCacheManager(*cache.Manager)` - Custom cache manager
- `WithVerbose(bool)` - Enable verbose output
- `WithDryRun(bool)` - Enable dry-run mode
- `WithAllowUndefinedVars(bool)` - Allow undefined variables

### Default Configuration

When options are omitted, sensible defaults are applied via `applyDefaults()`:
- Standard output writer
- New task registry
- Default validators and resolvers
- GitHub, HTTPS, and Drunhub fetchers
- Standard interpolator

## Architecture Benefits

### Code Quality
✅ **Modular Design** - Clear component boundaries (Planner, Executor, Engine)  
✅ **Domain-Driven** - Business logic separated from AST  
✅ **Explicit Planning** - Upfront execution plan eliminates waste  
✅ **Dependency Injection** - Pluggable infrastructure for testing  

### Development Experience
✅ **Easier Navigation** - Logical package organization  
✅ **Better Testability** - Components tested in isolation  
✅ **Clear Extension Points** - Add features without breaking changes  
✅ **Rich Debugging** - Plan visualization and diagnostics  

### Performance & Reliability
✅ **Optimized Execution** - Single AST scan, no redundant work  
✅ **All Examples Passing** - 60/60 examples verified  
✅ **All Unit Tests Passing** - Comprehensive test coverage  
✅ **Zero Regressions** - Backward compatible  

## Debug & Diagnostics

### Execution Plan Visualization

The engine supports comprehensive debugging through execution plan exports:

**Available Formats:**
- **Graphviz DOT** - For rendering with `dot` command
- **Mermaid** - For markdown diagrams
- **JSON** - For programmatic analysis

**CLI Usage:**
```bash
# View plan in terminal
xdrun --debug --debug-domain --debug-plan -f myfile.drun

# Export all formats
xdrun --debug --debug-domain \
  --debug-export-graph plan \
  --debug-export-mermaid plan \
  --debug-export-json plan \
  -f myfile.drun
```

**Plan Information:**
- Complete execution order
- Task dependencies
- Parameter metadata
- Hook integration points
- Project and namespace info

## Future Enhancements

Potential areas for further improvement:

1. **Plan Caching** - Cache execution plans for warm-start performance
2. **Interactive Debugger** - Step-through execution with breakpoints
3. **Plan Diff Tool** - Compare execution plans across changes
4. **Web UI** - Interactive plan visualization dashboard
5. **Performance Profiling** - Built-in performance metrics

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

**Last Updated:** October 9, 2025  
**Status:** ✅ Production - Modular architecture with debug diagnostics  
**Test Coverage:** 60/60 examples passing, all unit tests passing
