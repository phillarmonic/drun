# Domain Layer

The domain layer contains business logic and domain entities extracted from the execution engine. This layer provides clean abstractions for tasks, parameters, and project configuration with clear separation of concerns.

## Package Structure

```
internal/domain/
├── task/
│   ├── task.go           # Task entity with operations
│   ├── registry.go       # Task registration and lookup
│   └── dependencies.go   # Dependency resolution logic
├── parameter/
│   ├── parameter.go      # Parameter entity
│   └── validation.go     # Parameter validation & constraints
└── project/
    ├── project.go        # Project entity
    └── settings.go       # Settings management
```

## Domain Entities

### Task Domain

The task domain manages task entities, their registration, and dependency resolution:

- **Task**: Represents a drun task with metadata, parameters, dependencies, and body
- **Registry**: Thread-safe task registration and lookup with namespace support
- **DependencyResolver**: Handles circular dependency detection and topological sorting

### Parameter Domain  

The parameter domain provides parameter validation with advanced constraints:

- **Parameter**: Parameter entity with type information and constraints
- **Validator**: Validates parameter values against data types, patterns, and ranges

### Project Domain

The project domain manages project-level configuration:

- **Project**: Project entity with settings, shell configs, and lifecycle hooks
- **SettingsManager**: Manages project settings with defaults

## Design Principles

1. **Domain-Driven Design**: Business logic separated from infrastructure
2. **Single Responsibility**: Each package has one clear purpose  
3. **Testability**: Domain logic can be tested in isolation
4. **Thread Safety**: Registry provides concurrent access support
5. **Type Safety**: Strong typing with validation

## Usage

The domain layer is ready for integration with the engine. Currently, it exists as standalone packages that can be used independently or integrated into the execution flow.

### Example: Task Registry

```go
import "github.com/phillarmonic/drun/internal/domain/task"

// Create registry
registry := task.NewRegistry()

// Register task
domainTask := task.NewTask(astTask, "namespace", "source.drun")
err := registry.Register(domainTask)

// Lookup task
foundTask, err := registry.Get("taskName")

// List all tasks
allTasks := registry.List()
```

### Example: Parameter Validation

```go
import "github.com/phillarmonic/drun/internal/domain/parameter"

// Create validator
validator := parameter.NewValidator()

// Define parameter with constraints
param := &parameter.Parameter{
    Name:       "port",
    DataType:   "number",
    MinValue:   &minVal,  // 1
    MaxValue:   &maxVal,  // 65535
}

// Validate value
value := types.NewValue(types.NumberType, "8080")
err := validator.Validate(param, value)
```

## Future Integration

The domain layer is designed to be integrated into the engine for:

- Cleaner business logic separation
- Better testability of validation rules
- More maintainable dependency resolution
- Explicit domain boundaries

Integration can be done gradually without breaking existing functionality.

## Statistics

- **Total Lines**: 839
- **Files**: 7  
- **Packages**: 3
- **Average File Size**: 120 lines

---

**Created**: Phase 5 Refactoring - October 2025  
**Status**: ✅ Complete - Ready for use

