# Internal Packages Guide

This directory contains the internal implementation of the drun execution engine. The codebase is organized into focused, maintainable packages.

---

## Package Overview

### 📊 Package Statistics

| Package | Files | Lines | Purpose |
|---------|-------|-------|---------|
| `ast/` | 15 | ~1,500 | Abstract Syntax Tree definitions |
| `parser/` | 26 | ~5,000 | Syntax parsing |
| `domain/` | 7 | ~840 | **Domain layer (business logic)** ✨ |
| `engine/` | 35 | ~6,500 | Execution engine |
| `lexer/` | 6 | ~800 | Tokenization |
| `builtins/` | 2 | ~200 | Built-in functions |
| `shell/` | 3 | ~300 | Shell execution |
| `detection/` | 2 | ~200 | Tool detection |
| `remote/` | 1 | ~150 | Remote file fetching |
| `cache/` | 1 | ~100 | Caching system |
| `errors/` | 1 | ~50 | Error types |
| `types/` | 3 | ~150 | Type definitions |

**Total:** ~102 files, ~15,800 lines (avg. 155 lines/file)

---

## Package Details

### 🌳 ast/ - Abstract Syntax Tree

**Purpose:** Defines all AST node types representing drun language constructs.

**Key Files:**
- `ast.go` - Core types: `Program`, `Node`, `Statement`, `Expression`
- `ast_project.go` - Project declarations
- `ast_task.go` - Task definitions
- `ast_parameter.go` - Parameter types
- `ast_control.go` - Control flow: `if`, `for`, `try/catch`
- `ast_action.go` - Base action statements
- `ast_expressions.go` - Expressions and operators

**Domain-Specific:**
- `ast_shell.go` - Shell commands
- `ast_file.go` - File operations
- `ast_docker.go` - Docker actions
- `ast_git.go` - Git actions
- `ast_http.go` - HTTP requests
- `ast_network.go` - Network operations
- `ast_variable.go` - Variable operations
- `ast_detection.go` - Tool detection

**Usage:**
```go
import "github.com/phillarmonic/drun/internal/ast"

stmt := &ast.ShellStatement{
    Command: "echo hello",
}
```

---

### 📝 parser/ - Syntax Parser

**Purpose:** Parses drun source code into AST.

**Key Files:**
- `parser.go` - Core parser with entry point
- `parser_project.go` - Parses project declarations
- `parser_task.go` - Parses task definitions
- `parser_parameter.go` - Parses parameters
- `parser_action.go` - Dispatches to action parsers
- `parser_control.go` - Parses control flow
- `parser_error.go` - Parses error handling
- `parser_helper.go` - Helper methods

**Domain-Specific Parsers:**
- `parser_shell.go` - Shell command parsing
- `parser_file.go` - File operation parsing
- `parser_docker.go` - Docker action parsing
- `parser_git.go` - Git action parsing
- `parser_http.go` - HTTP request parsing
- `parser_network.go` - Network operation parsing
- `parser_variable.go` - Variable operation parsing
- `parser_detection.go` - Detection statement parsing

**Usage:**
```go
import "github.com/phillarmonic/drun/internal/parser"

p := parser.New(lexer)
program, err := p.ParseProgram()
```

**Architecture:**
- Each parser file handles one domain
- All parsers share the core `Parser` struct
- Parsers build AST nodes from tokens

---


### 🎯 domain/ - Domain Layer

**Purpose:** Business logic layer separating domain concepts from execution concerns.

**Status:** ✅ Fully integrated with engine

**Key Files:**
- `task/task.go` - Task entity with validation
- `task/registry.go` - Task management and lookup (thread-safe)
- `task/dependencies.go` - Dependency resolution with circular detection
- `parameter/parameter.go` - Parameter entity
- `parameter/validation.go` - Parameter validation rules
- `project/project.go` - Project configuration

**Architecture:**
```
CLI Layer → Engine Layer → Domain Layer
                 ↓              ↓
           Orchestration   Business Logic
```

**Key Services:**

1. **Task Registry** (`task/registry.go`)
   - Registers and manages tasks
   - Preserves insertion order
   - Thread-safe operations
   - Namespace support

2. **Dependency Resolver** (`task/dependencies.go`)
   - Resolves task execution order
   - Detects circular dependencies
   - Topological sorting
   - Parallel/sequential grouping

3. **Parameter Validator** (`parameter/validation.go`)
   - Validates data types (string, number, boolean, list)
   - Checks constraints (from list)
   - Validates ranges (min/max)
   - Pattern matching (regex, email, semver, etc.)

**Usage in Engine:**
```go
// Engine struct holds domain services
type Engine struct {
    taskRegistry   *task.Registry
    paramValidator *parameter.Validator
    depResolver    *task.DependencyResolver
    // ...
}

// Register tasks from AST
func (e *Engine) registerTasks(tasks []*ast.TaskStatement, file string) error {
    for _, astTask := range tasks {
        domainTask := task.NewTask(astTask, "", file)
        if err := e.taskRegistry.Register(domainTask); err != nil {
            return err
        }
    }
    return nil
}

// Resolve dependencies
domainTasks, err := e.depResolver.Resolve(taskName)

// Validate parameters
domainParam := &parameter.Parameter{
    Name:        param.Name,
    DataType:    param.DataType,
    Constraints: param.Constraints,
    // ...
}
err := e.paramValidator.Validate(domainParam, typedValue)
```

**Design Principles:**
- Domain entities are independent of AST
- Business rules stay in domain layer
- Engine orchestrates, domain validates
- Easily testable in isolation

**Test Coverage:**
- `task/task_test.go` - 32 tests
- `task/registry_test.go` - 16 tests  
- `task/dependencies_test.go` - 21 tests
- `parameter/validation_test.go` - 17 tests

**When to Use Domain Layer:**
- ✅ Adding new validation rules
- ✅ Extending task/parameter properties
- ✅ Adding business logic operations
- ❌ AST changes (use `ast/` instead)
- ❌ Execution logic (use `engine/` instead)

---

### ⚙️ engine/ - Execution Engine

**Purpose:** Executes AST by orchestrating executors and subsystems.

#### Core Files

- **`engine.go`** (911 lines) - Main engine, orchestration
- **`context.go`** - Execution context and project context

#### Subsystems

**`interpolation/`** - Variable interpolation
- `interpolator.go` - Main interpolation logic
- `resolvers.go` - Variable resolution
- `conditional.go` - Conditional interpolation
- `utilities.go` - Helper functions

**`hooks/`** - Lifecycle hooks
- `manager.go` - Hook registration and execution

**`includes/`** - Include resolution
- `resolver.go` - Remote/local include handling

#### Executors

Each executor handles one type of statement:

- `executor_error.go` - `try/catch/finally`, `throw`
- `executor_control.go` - `if`, `for`, `break`, `continue`
- `executor_variables.go` - `let`, `set`, `capture`
- `executor_shell.go` - Shell command execution
- `executor_file.go` - File operations
- `executor_network.go` - Network operations, health checks
- `executor_docker.go` - Docker actions
- `executor_git.go` - Git actions
- `executor_http.go` - HTTP requests
- `executor_detection.go` - Tool detection

#### Helpers

Domain-specific helper functions:

- `helpers_builders.go` - Command builders
- `helpers_conditions.go` - Condition evaluation
- `helpers_detection.go` - Detection helpers
- `helpers_download.go` - File download with progress
- `helpers_expressions.go` - Expression evaluation
- `helpers_filesystem.go` - Filesystem utilities
- `helpers_utilities.go` - General utilities

**Usage:**
```go
import "github.com/phillarmonic/drun/internal/engine"

eng := engine.NewEngine(os.Stdout)
eng.LoadProject(program)
err := eng.RunTask("build", params)
```

**Architecture:**
```
Engine (orchestrator)
├── Interpolation System (variable resolution)
├── Hooks System (lifecycle hooks)
├── Includes System (file inclusion)
└── Executors (statement execution)
    ├── Error Executor
    ├── Control Executor
    ├── Variable Executor
    ├── Shell Executor
    ├── File Executor
    ├── Network Executor
    ├── Docker Executor
    ├── Git Executor
    ├── HTTP Executor
    └── Detection Executor
```

---

### 🔤 lexer/ - Lexical Analysis

**Purpose:** Tokenizes drun source code.

**Key Files:**
- `lexer.go` - Main lexer implementation
- `tokens.go` - Token type definitions
- `keywords.go` - Keyword mappings
- `semantic_tokens.go` - Semantic token support
- `position.go` - Position tracking
- `errors.go` - Lexer error types

**Usage:**
```go
import "github.com/phillarmonic/drun/internal/lexer"

l := lexer.New(source)
token := l.NextToken()
```

---

### 🛠️ Support Packages

#### builtins/ - Built-in Functions

Functions like `now()`, `env()`, string operations.

```go
import "github.com/phillarmonic/drun/internal/builtins"

result, err := builtins.Apply("uppercase", "hello")
```

#### shell/ - Shell Execution

Cross-platform shell command execution.

```go
import "github.com/phillarmonic/drun/internal/shell"

result, err := shell.Execute("echo hello", opts)
```

#### detection/ - Tool Detection

Detects installed tools and versions.

```go
import "github.com/phillarmonic/drun/internal/detection"

available := detection.IsToolAvailable("docker")
version, _ := detection.GetToolVersion("node")
```

#### remote/ - Remote Fetching

Fetches files from remote sources (HTTP, GitHub).

```go
import "github.com/phillarmonic/drun/internal/remote"

content, err := remote.FetchContent("https://...")
```

#### cache/ - Caching System

Caches downloaded files and parsed content.

```go
import "github.com/phillarmonic/drun/internal/cache"

manager := cache.NewManager()
manager.Set("key", data)
```

#### errors/ - Error Types

Defines custom error types with context.

```go
import "github.com/phillarmonic/drun/internal/errors"

err := errors.NewExecutionError("failed", ctx)
```

#### types/ - Type Definitions

Common type definitions and utilities.

```go
import "github.com/phillarmonic/drun/internal/types"

value := types.StringValue("hello")
```

---

## Architecture Principles

### 1. Single Responsibility

Each package/file has ONE clear purpose:
- ✅ `parser_docker.go` - Only Docker parsing
- ✅ `executor_git.go` - Only Git execution
- ✅ `helpers_download.go` - Only download helpers

### 2. Dependency Direction

```
CLI → Engine → Parser → Lexer → AST
          ↓
    Support Packages
```

Higher-level packages depend on lower-level ones, never the reverse.

### 3. Domain Organization

Files grouped by domain, not by technical layer:
- All Docker-related code: `ast_docker.go`, `parser_docker.go`, `executor_docker.go`
- All Git-related code: `ast_git.go`, `parser_git.go`, `executor_git.go`

### 4. Testability

Every package can be tested independently:
```go
// Test parser without engine
parser := parser.New(lexer.New("task build"))
program, err := parser.ParseProgram()

// Test executor without full engine
ctx := &ExecutionContext{...}
err := executeDocker(stmt, ctx)
```

---

## Common Patterns

### Adding a New Action Type

1. **Define AST** in `internal/ast/ast_yourfeature.go`
2. **Add Parser** in `internal/parser/parser_yourfeature.go`
3. **Add Executor** in `internal/engine/executor_yourfeature.go`
4. **Wire it up** in `engine.go` and `parser.go`

### Accessing Variables

```go
// In executor
value, exists := ctx.GetVariable("myvar")
if !exists {
    return fmt.Errorf("variable not found")
}
```

### Interpolating Strings

```go
// In executor
interpolated, err := e.interpolator.InterpolateString(input, ctx)
if err != nil {
    return err
}
```

### Executing Shell Commands

```go
// In executor
result, err := shell.Execute(command, e.getShellOpts(ctx))
if err != nil {
    return fmt.Errorf("command failed: %w", err)
}
```

---

## File Size Guidelines

After refactoring, we maintain these size limits:

| File Type | Target Size | Max Size |
|-----------|-------------|----------|
| AST definitions | 100-200 lines | 300 lines |
| Parsers | 200-300 lines | 500 lines |
| Executors | 150-250 lines | 400 lines |
| Helpers | 100-200 lines | 300 lines |
| Core orchestration | 200-500 lines | 1000 lines |

**Current Status:** ✅ All files within guidelines

---

## Navigation Tips

### Finding Code by Feature

Looking for Docker support?
1. AST: `internal/ast/ast_docker.go`
2. Parser: `internal/parser/parser_docker.go`
3. Executor: `internal/engine/executor_docker.go`

### Understanding Execution Flow

1. Start at `cmd/drun/main.go` - entry point
2. Follow to `engine.go` - orchestration
3. Look at specific executor - action execution
4. Check helpers - supporting functions

### Debugging

1. **Lexer issues:** Check `internal/lexer/`
2. **Parsing errors:** Check `internal/parser/parser_*.go`
3. **Runtime errors:** Check `internal/engine/executor_*.go`
4. **Variable issues:** Check `internal/engine/interpolation/`

---

## Testing Strategy

### Unit Tests

Each package has its own tests:
```
internal/parser/parser_docker_test.go
internal/engine/executor_docker_test.go
internal/ast/ast_test.go
```

### Integration Tests

Engine tests in `internal/engine/*_test.go`:
- `strict_variables_test.go`
- `loop_scoping_test.go`
- `matrix_execution_test.go`
- `when_otherwise_test.go`

### Regression Tests

All 62 example files in `examples/` must pass.

---

## Performance Considerations

### Hot Paths

Most frequently executed code:
1. Variable interpolation (`interpolation/`)
2. Statement execution (`executor_*.go`)
3. Context lookups (`context.go`)

### Optimization Points

- ✅ Variable lookups cached in context
- ✅ Includes cached by resolver
- ✅ Regex patterns compiled once
- ✅ Shell opts reused across commands

---

## Refactoring History

### Before Refactoring

```
internal/
├── ast.go (1,133 lines) ❌
├── parser.go (4,874 lines) ❌
├── engine.go (5,179 lines) ❌
└── ... (support packages)
```

### After (Current)

```
internal/
├── ast/ (15 files, ~100-200 lines each) ✅
├── parser/ (26 files, ~200-300 lines each) ✅
├── engine/ (36 files, ~150-300 lines each) ✅
└── ... (support packages)
```

**Improvement:**
- 📊 10x better file sizes
- 🎯 Clear responsibilities
- 🧪 Easier to test
- 📖 Easier to understand
- 🚀 Faster to modify

---

## Contributing

When adding new code:

1. ✅ Follow existing patterns
2. ✅ Keep files under 500 lines
3. ✅ Group by domain, not layer
4. ✅ Add tests for new code
5. ✅ Update documentation

---

## Related Documentation

- **[ARCHITECTURE.md](../ARCHITECTURE.md)** - System architecture overview
- **[spec/](../spec/)** - Refactoring specifications
- **[DRUN_V2_SPECIFICATION.md](../DRUN_V2_SPECIFICATION.md)** - Language spec

---

*Last Updated: October 5, 2025*  
*Current Version*
