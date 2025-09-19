# drun v2 Implementation Roadmap

**Version**: 2.0.0-draft  
**Last Updated**: 2025-09-19  
**Status**: In Development  

This roadmap tracks the implementation progress of features documented in the [DRUN_V2_SPECIFICATION.md](DRUN_V2_SPECIFICATION.md).

## Legend

- ✅ **Completed** - Feature is fully implemented and tested
- 🚧 **In Progress** - Feature is currently being worked on
- 📋 **Planned** - Feature is planned for implementation
- 🔍 **Research** - Feature needs research/design work
- ❌ **Blocked** - Feature is blocked by dependencies
- 🎯 **Priority** - High priority feature

---

## Core Language Features

### ✅ Lexer & Parser Foundation
- ✅ Tokenization system with semantic tokens
- ✅ AST (Abstract Syntax Tree) generation
- ✅ Error handling and reporting
- ✅ EBNF grammar definition

### ✅ Basic Execution Engine
- ✅ Direct AST execution (no compilation)
- ✅ Statement execution pipeline
- ✅ Context management and scoping
- ✅ Dry run mode support
- ✅ Verbose mode support

### ✅ Project System
- ✅ Project declarations (`project "name" version "1.0"`)
- ✅ Global project settings
- ✅ Task definitions with descriptions
- ✅ Cross-platform shell configuration
- ✅ Shell config for darwin/linux/windows

---

## Variable System

### ✅ Basic Variables
- ✅ Variable declarations (`let $var = "value"`)
- ✅ Variable assignments (`set $var to "value"`)
- ✅ Variable interpolation (`{$variable}`)
- ✅ Variable scoping (global, task, block)

### ✅ Advanced Variable Operations
- ✅ Array operations (`{$files} filtered by extension ".js"`, `{$files} sorted by name`, `{$files} first`)
- ✅ String operations (`{$version} without prefix "v"`, `{$image} split by ':'`)
- ✅ Path operations (`{$path} basename`, `{$path} dirname`, `{$path} extension`)
- ✅ Operation chaining (`{$files} filtered by extension ".js" | sorted by name`)
- ✅ For each loop integration (`for each item in $variable`)

---

## Parameter System

### ✅ Basic Parameters
- ✅ Required parameters (`requires $env`)
- ✅ Optional parameters (`given $tag defaults to "latest"`)
- ✅ CLI parameter passing (`drun task param=value`)

### ✅ Advanced Parameter Validation
- ✅ Type constraints (`as number between 1000 and 9999`)
- ✅ Pattern matching (`matching pattern "v\d+\.\d+\.\d+"`)
- ✅ Pattern macros (`matching semver`, `matching uuid`, `matching url`)
- ✅ Email format validation (`matching email format`)
- ✅ List constraints (`from ["dev", "staging", "production"]`)
- ✅ Variadic parameters (`accepts $flags as list`)

---

## Type System

### 📋 Primitive Types
- 📋 String type with validation
- 📋 Number type with range constraints
- 📋 Boolean type
- 📋 Duration type (`"5m"`, `"2h"`, `"30s"`)

### 📋 Collection Types
- 📋 Array type (`[1, 2, 3]`)
- 📋 Object type (`{name: "value", count: 42}`)

### 📋 Special Types
- 📋 Command type (executable shell commands)
- 📋 Path type with filesystem validation
- 📋 URL type with protocol validation
- ✅ Regex type for pattern matching (via pattern macros and raw patterns)
- 📋 Secret type (secure values, not logged)

### 📋 Type Inference & Validation
- 📋 Automatic type inference
- 📋 Runtime type checking
- 📋 Type constraint validation

---

## Control Flow

### ✅ Conditional Statements
- ✅ Basic if/else statements
- ✅ Conditional expressions
- ✅ Nested conditionals

### ✅ Loop Statements
- ✅ For each loops (`for each item in items`)
- ✅ Range loops (`for i in range 1 to 10`)
- ✅ Parallel execution (`in parallel`)
- ✅ Loop control (`break`, `continue`)

### ✅ Error Handling
- ✅ Try/catch/finally blocks
- ✅ Custom error types
- ✅ Error propagation
- ✅ Throw statements

---

## Shell Integration

### ✅ Shell Commands
- ✅ Single-line commands (`run "echo hello"`)
- ✅ Multiline command blocks (`run:` with indentation)
- ✅ Output capture (`capture "command" as $var`)
- ✅ Variable interpolation in commands
- ✅ Cross-platform shell configuration

### ✅ Shell Actions
- ✅ `run` - Execute and stream output
- ✅ `exec` - Execute command
- ✅ `shell` - Shell command execution
- ✅ `capture` - Capture command output

---

## Built-in Actions

### ✅ Status & Logging Actions
- ✅ `step` - Process step indicator
- ✅ `info` - Informational messages
- ✅ `warn` - Warning messages
- ✅ `error` - Error messages (non-fatal)
- ✅ `success` - Success messages
- ✅ `fail` - Failure messages (fatal)

### ✅ Docker Actions (High Priority)
- ✅ `build docker image "name:tag"` - Build container images
- ✅ `push image "name" to "registry"` - Push to registries
- ✅ `pull image "name"` - Pull images
- ✅ `run container "image" on port 8080` - Run containers
- ✅ `stop container "name"` - Stop containers
- ✅ `remove container "name"` - Remove containers
- ✅ `start docker compose services` - Compose operations
- ✅ `scale docker compose service "name" to 3` - Service scaling

### ✅ Git Actions (High Priority)
- ✅ `commit changes with message "text"` - Commit operations
- ✅ `create branch "name"` - Branch management
- ✅ `checkout branch "name"` - Branch switching
- ✅ `merge branch "name"` - Branch merging
- ✅ `push to branch "name"` - Push operations
- ✅ `create tag "v1.0.0"` - Tag management
- ✅ `push tag "name"` - Tag pushing

### 📋 Kubernetes Actions
- 📋 `deploy "image" to kubernetes` - Deploy applications
- 📋 `scale deployment "name" to 5 replicas` - Scaling
- 📋 `rollback deployment "name"` - Rollback operations
- 📋 `wait for rollout of deployment "name"` - Status waiting
- 📋 `expose deployment "name" on port 8080` - Service exposure
- 📋 `apply kubernetes manifests from "path"` - Manifest application
- 📋 `get pods in namespace "name"` - Resource inspection

### ✅ File System Actions
- ✅ `copy "src" to "dest"` - File copying
- ✅ `move "old" to "new"` - File moving
- ✅ `remove "file"` - File deletion
- ✅ `backup "file" as "backup-{now.date}"` - File backup
- ✅ `create directory "path"` - Directory creation
- ✅ `check if file "path" exists` - File existence
- ✅ `get size of file "path"` - File information

### 📋 Network Actions
- 📋 `send GET request to "url"` - HTTP GET
- 📋 `send POST request to "url" with data {...}` - HTTP POST
- 📋 `download "url" to "path"` - File downloads
- 📋 `check health of service at "url"` - Health checks
- 📋 `wait for service at "url" to be ready` - Service waiting
- 📋 `check if port 8080 is open on "host"` - Port checking
- 📋 `test connection to "host" on port 5432` - Connectivity testing

### 📋 Progress & Timing Actions
- 📋 `start progress "message"` - Progress indicators
- 📋 `update progress to 50% with message "text"` - Progress updates
- 📋 `finish progress with "message"` - Progress completion
- 📋 `start timer "name"` - Timing operations
- 📋 `stop timer "name"` - Timer stopping
- 📋 `show elapsed time for "name"` - Time display

---

## Smart Detection System

### ✅ Tool Detection
- ✅ Basic tool availability (`if docker is available`)
- ✅ Tool version checking (`if node version >= "16"`)
- ✅ Environment detection (`when in ci environment`)
- ✅ Quoted tool names (`if "docker compose" is available`)

### ✅ DRY Tool Detection
- ✅ Tool variant detection (`detect available "docker compose" or "docker-compose" as $cmd`)
- ✅ Variable capture for consistent usage
- ✅ Multiple alternatives support (`tool1 or tool2 or tool3`)
- ✅ Cross-platform compatibility

### 📋 Enhanced Detection
- 📋 Project type detection (automatic)
- 📋 Framework detection (symfony, laravel, rails)
- 📋 Build tool detection (webpack, vite)
- 📋 Package manager detection (npm, yarn, pnpm)

---

## HTTP Integration

### ✅ Basic HTTP Actions
- ✅ HTTP requests with different methods
- ✅ Request headers and authentication
- ✅ JSON request/response handling
- ✅ Response status checking

### ✅ Advanced HTTP Features
- ✅ File uploads and downloads
- ✅ Response parsing and extraction
- ✅ Retry logic and error handling
- 📋 Webhook integration

---

## Security & Secrets

### 📋 Secrets Management
- 📋 Secret definitions with sources
- 📋 Environment variable secrets (`env://VAR_NAME`)
- 📋 File-based secrets (`file://path/to/secret`)
- 📋 Secure secret usage (not logged)
- 📋 Required vs optional secrets
- 📋 HashiCorp Vault integration (`vault://path`)

### 📋 Security Features
- 📋 Secure variable interpolation
- 📋 Secret masking in logs
- 📋 Audit trail for secret access

---

## Developer Experience

### ✅ CLI Features
- ✅ Task listing and discovery
- ✅ Help system and descriptions
- ✅ Parameter validation and help
- ✅ Dry run mode
- ✅ Verbose output mode

### 📋 Advanced CLI Features
- 📋 Shell completion (bash, zsh, fish, PowerShell)
- 📋 Interactive parameter prompting
- 📋 Task dependency visualization
- 📋 Performance profiling and metrics

### 📋 IDE Integration
- 📋 Language Server Protocol (LSP)
- 📋 Syntax highlighting
- 📋 IntelliSense and completion
- 📋 Error diagnostics
- 📋 Refactoring support

---

## Testing & Quality

### ✅ Core Testing
- ✅ Unit tests for lexer/parser
- ✅ Integration tests for engine
- ✅ Example file validation
- ✅ Regression testing

### 📋 Advanced Testing
- 📋 End-to-end testing framework
- 📋 Performance benchmarks
- 📋 Cross-platform testing
- 📋 Memory leak detection

---

## Documentation

### ✅ Core Documentation
- ✅ Language specification (DRUN_V2_SPECIFICATION.md)
- ✅ README with examples
- ✅ Grammar documentation (EBNF)
- ✅ Feature examples

### 📋 Enhanced Documentation
- 📋 Tutorial series
- 📋 Best practices guide
- 📋 Migration guide from v1
- 📋 API reference
- 📋 Troubleshooting guide

---

## Performance & Optimization

### 📋 Performance Features
- 📋 Parallel task execution
- 📋 Caching system for includes
- 📋 Lazy evaluation
- 📋 Memory optimization
- 📋 Startup time optimization

### 📋 Monitoring & Observability
- 📋 Execution metrics
- 📋 Performance profiling
- 📋 Resource usage tracking
- 📋 Execution tracing

---

## Release & Distribution

### ✅ Build System
- ✅ Cross-platform builds
- ✅ Static binary generation
- ✅ Release automation

### 📋 Distribution
- 📋 Package manager integration (brew, apt, chocolatey)
- 📋 Container images
- 📋 Auto-update mechanism
- 📋 Version management

---

## Migration & Compatibility

### 📋 v1 Compatibility
- 📋 v1 YAML format support
- 📋 Migration tooling
- 📋 Hybrid v1/v2 projects
- 📋 Deprecation warnings

### 📋 Ecosystem Integration
- 📋 CI/CD platform integration
- 📋 Container orchestration
- 📋 Cloud platform support
- 📋 Third-party tool integration

---

## Implementation Phases

### 🎯 Phase 1: Core Semantic Actions (Current Focus)
**Priority**: High  
**Timeline**: Q4 2025  

- ✅ Docker semantic actions
- ✅ Git semantic actions  
- ✅ File system operations
- ✅ Enhanced HTTP actions

### 📋 Phase 2: Advanced Language Features
**Priority**: Medium  
**Timeline**: Q1 2026  

- Type system implementation
- Advanced variable operations
- Parameter validation system
- Progress tracking system

### 📋 Phase 3: Developer Experience
**Priority**: Medium  
**Timeline**: Q2 2026  

- IDE integration (LSP)
- Shell completion
- Enhanced CLI features
- Testing framework

### 📋 Phase 4: Enterprise Features
**Priority**: Lower  
**Timeline**: Q3 2026  

- Secrets management
- Security features
- Performance optimization
- Monitoring & observability

---

## Contributing

To contribute to this roadmap:

1. **Pick a feature** from the 📋 Planned items
2. **Update status** to 🚧 In Progress
3. **Create implementation** following the specification
4. **Add tests** and documentation
5. **Update status** to ✅ Completed

### Current Priorities

The highest impact features to implement next:

1. **🎯 Docker Actions** - Most commonly used in automation
2. **🎯 Git Actions** - Essential for CI/CD workflows  
3. **File System Actions** - Basic operations needed by many tasks
4. **Enhanced HTTP Actions** - API integration is crucial

---

**Last Updated**: 2025-09-19  
**Next Review**: Weekly during active development
