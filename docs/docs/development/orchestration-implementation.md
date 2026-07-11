# Microservices Orchestration Implementation

## Overview

This document summarizes the implementation of the microservices orchestration system for Drun, as specified in `spec/microservices-orchestration.md`.

## Implemented Components

### 1. AST Nodes (`internal/ast/ast_orchestration.go`)

- **ServiceStatement**: Represents a microservice definition with all configuration options
- **OrchestrateStatement**: Represents an orchestration group
- **Supporting Types**:
  - RepositoryConfig: Git repository configuration
  - HealthCheckConfig: Health check settings (HTTP, TCP, Docker, DNS, Custom)
  - BuildConfig: Build configuration with Makefile support
  - ComposeConfig: Docker Compose configuration
  - ComposeOptions: Detailed Docker Compose options
  - EnvFileConfig: Environment file management

### 2. Lexer Tokens (`internal/lexer/token.go`)

Added 70+ new tokens for orchestration features:

- `ORCHESTRATE`, `SERVICES`, `STRATEGY`, `SEQUENTIAL`, `CIRCUIT`, `BREAKER`
- `HEALTH`, `DNS`, `TCP`, `DOMAIN`, `INTERVAL`, `RETRIES`
- `MAKEFILE`, `TARGET`, `ARGS`, `REPOSITORY`
- `ENV_FILE`, `REQUIRED`, `MISSING`, etc.

### 3. Parser (`internal/parser/parser_orchestration.go`)

Comprehensive parser implementation for:

- Service declarations with all options
- Orchestration group declarations
- Repository configuration
- Health check configuration (all types)
- Build configuration with Makefile support
- Docker Compose configuration
- Environment file configuration
- Helper functions for parsing arrays, maps, and nested structures

### 4. Domain Types (`internal/domain/orchestration/`)

#### `service.go`

- Service domain model with runtime state
- ServiceStatus enum (unknown, starting, running, healthy, unhealthy, stopping, stopped, failed)
- Health check methods (MarkHealthy, MarkUnhealthy, etc.)

#### `orchestration.go`

- Orchestration domain model with strategies
- OrchestrationStrategy enum (sequential, parallel, dependency-based)
- UpdateStrategy enum (rolling, recreate, blue-green)
- Circuit breaker state management
- Recovery configuration

#### `registry.go`

- ServiceRegistry: Thread-safe service registration and lookup
- OrchestrationRegistry: Thread-safe orchestration registration and lookup

### 5. Health Check System (`internal/healthcheck/healthcheck.go`)

Implemented all health check types:

- **HTTP**: Status code validation, custom headers
- **TCP**: Port connectivity checks
- **Docker**: Container health status inspection
- **DNS**: Domain resolution with record type validation (A, AAAA, CNAME, MX)
- **Custom**: Shell command execution

Features:

- Configurable timeouts and intervals
- Retry logic with configurable attempts
- Start period support
- Continuous monitoring with callbacks

### 6. Repository Management (`internal/repository/repository.go`)

Git repository operations:

- Clone with branch/tag selection
- SSH key support
- Update/pull operations
- Status checking
- Branch and tag detection
- Repository validation

### 7. Makefile Integration (`internal/makeexec/makeexec.go`)

Makefile execution features:

- Target execution with arguments
- Parallel job support (-j flag)
- Pre/post command execution
- Timeout handling
- Retry with exponential backoff
- Fallback command support
- Dry run capability
- Target listing and validation

### 8. Environment File Management (`internal/envfile/envfile.go`)

Environment file operations:

- Automatic .env file creation from templates
- Key-value parsing and writing
- Variable replacement
- Validation of required variables
- Backup and restore
- File merging
- Permission management (secure mode 600)
- Variable interpolation

### 9. Service Lifecycle Executor (`internal/engine/executor_orchestration.go`)

Service lifecycle management:

- Service registration from AST
- Start/stop operations
- Pre-task and post-task execution
- Repository cloning
- Environment file setup
- Build execution (Makefile or command)
- Docker Compose integration
- Health check monitoring

### 10. Orchestration Coordinator (`internal/engine/orchestration_coordinator.go`)

Advanced orchestration features:

- **Sequential Strategy**: Start services one after another
- **Parallel Strategy**: Start all services simultaneously
- **Dependency-Based Strategy**: Topological sort with dependency resolution

Key features:

- Dependency graph building
- Cycle detection
- Circuit breaker implementation
- Health monitoring with automatic recovery
- Graceful shutdown with reverse order
- Global pre/post task execution
- Failure threshold tracking
- Service status reporting

### 11. Built-in Functions (`internal/builtins/builtins_orchestration.go`)

20+ new built-in functions:

- `service_status`, `service_health`, `service_healthy`, `service_running`
- `orchestrate_status`, `orchestrate_health_status`, `orchestrate_healthy`
- `dns_resolve`, `dns_check`, `dns_validate`
- `git_status`, `git_branch`, `git_tag`
- `docker_ps`, `docker_logs`, `docker_stats`
- `compose_config`
- `make_list`, `make_dry_run`

### 12. Example Drunfiles

Created 6 comprehensive examples:

1. **64-microservices-basic.drun**: Basic microservices setup
2. **65-microservices-repository-cloning.drun**: Repository cloning and management
3. **66-microservices-env-file-management.drun**: Environment file handling
4. **67-microservices-global-tasks.drun**: Global pre/post tasks
5. **68-microservices-dns-health-check.drun**: DNS health checks and domain validation
6. **69-microservices-complete.drun**: Complete e-commerce platform example

## Features Implemented

### Core Features

-  Service definitions with dependencies
-  Orchestration groups with multiple strategies
-  Health monitoring (HTTP, TCP, Docker, DNS, Custom)
-  Circuit breaker pattern
-  Dependency resolution with topological sort
-  Repository cloning and management
-  Makefile integration
-  Environment file management
-  Global pre/post tasks
-  Service-specific pre/post tasks

### Docker Compose Integration

-  Custom compose file support
-  Project name configuration
-  Force recreate option
-  Build before start
-  Pull policy (always, missing, never)
-  Wait for services
-  Scaling configuration
-  Timeout settings

### Health Checks

-  HTTP with status code validation
-  TCP port connectivity
-  Docker container health
-  DNS resolution with record types
-  Custom command execution
-  Configurable retries and intervals
-  Start period support

### Repository Management

-  Clone if missing
-  Update on start
-  Branch selection
-  Tag selection
-  SSH key support
-  Status checking

### Build System

-  Makefile execution
-  Make targets and arguments
-  Shell command execution (supports multiline strings)
-  Line continuation with backslash
-  Variable interpolation in build commands
-  Parallel jobs
-  Pre/post commands
-  Retry on failure
-  Fallback commands
-  Dry run support

### Environment Files

-  Required validation
-  Template copying
-  Variable replacement
-  Secure permissions
-  Task-based setup
-  Validation

### Orchestration Strategies

-  Sequential: Services start one after another
-  Parallel: All services start simultaneously
-  Dependency-based: Topological sort based on dependencies

### Circuit Breaker

-  Failure threshold tracking
-  Automatic service stopping on failure
-  Recovery timeout
-  Circuit state management
-  Health monitoring

## Usage Example

```drun
version: 2.0

# Define a service
service "api" in "./services/api" means "REST API":
    depends on ["database", "redis"]
    repository:
        url "https://github.com/company/api.git"
        branch "main"
        clone true  # default, can be omitted
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
        timeout "10s"
        interval "15s"
        retries 3
        condition "200"
    build:
        required true
        command "npm install && npm test && npm run build"
        # Alternatively, use Makefile:
        # makefile "Makefile"
        # make_target "build"
    compose:
        file "docker-compose.yml"
    environment:
        NODE_ENV "production"

# Define orchestration
orchestrate "production" means "Production services":
    services ["database", "redis", "api"]
    strategy "dependency-based"
    circuit_breaker true
    stop_on_failure true
    health_check_interval "30s"

# Start services
task "start" means "Start production":
    orchestrate "production" start
    success "Services started!"
```

## Architecture

The orchestration system follows a layered architecture:

1. **AST Layer**: Parses drunfiles into structured AST nodes
2. **Domain Layer**: Business logic and state management
3. **Infrastructure Layer**: Health checks, repository management, makefile execution
4. **Engine Layer**: Orchestration coordination and service lifecycle
5. **Built-in Functions**: User-facing API for status and operations

## Key Design Decisions

1. **Thread-Safe Registries**: All service and orchestration registries use sync.RWMutex for concurrent access
2. **Context-Based Execution**: All operations support context cancellation for proper cleanup
3. **Modular Health Checks**: Each health check type is independently implemented
4. **Topological Sort**: Dependency resolution uses Kahn's algorithm for cycle detection
5. **Circuit Breaker**: Implements the circuit breaker pattern with configurable thresholds
6. **Graceful Degradation**: Services can continue running even if health checks fail (configurable)

## Future Enhancements

While the core specification is implemented, the following enhancements could be added:

1. **Testing**: Comprehensive unit and integration tests
2. **Kubernetes Integration**: Native k8s support beyond Docker Compose
3. **Service Mesh**: Istio/Linkerd integration
4. **Advanced Monitoring**: Prometheus/Grafana integration
5. **Blue-Green Deployments**: Advanced deployment strategies
6. **Canary Releases**: Gradual rollout support
7. **Auto-scaling**: Dynamic scaling based on metrics

## Integration with Existing Drun

The orchestration system is designed to integrate seamlessly with existing Drun features:

- Task system for pre/post operations
- Variable interpolation for dynamic configuration
- Error handling with try/catch
- Control flow (if/when/for)
- Secret management for credentials
- File operations for setup tasks

## Status

 **Implementation Complete**

All core features from the specification have been implemented:

- AST nodes and parser
- Domain models
- Health check system
- Repository management
- Makefile integration
- Environment file management
- Service lifecycle executor
- Orchestration coordinator
- Circuit breaker
- Built-in functions
- Comprehensive examples

The system is ready for integration testing and real-world usage.
