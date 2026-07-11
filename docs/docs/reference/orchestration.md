# Microservices orchestration specification

## Microservices Orchestration

drun v2 includes a comprehensive microservices orchestration system for managing multi-service architectures with Docker Compose integration, health monitoring, and visual progress feedback.

### Overview

The orchestration system allows you to:

- Define services with dependencies and health checks
- Group services into orchestration units
- Manage service lifecycles with semantic actions
- Monitor health and status in real-time
- Handle errors with circuit breaker support
- Visualize progress with BuildKit-style display

### Service Declaration

Services represent Docker Compose projects that can be orchestrated together.

#### Syntax

```drun
service "<name>" in "<path>":
    [depends on ["service1", "service2", ...]]
    [repository:
        url "<url>"
        [branch "<branch>"]
        [tag "<tag>"]
        [ssh_key "<path>"]
        [clone <boolean>]  # defaults to true, can be omitted
        [update_on_start <boolean>]]
    [build:
        required <boolean>
        [command "<command>"]
        [makefile "<path>"]
        [make_target "<target>"]
        [make_args ["arg1", "arg2", ...]]
        [makefile_timeout "<duration>"]
        [retry_on_failure <boolean>]
        [max_retries <number>]
        [retry_delay "<duration>"]
        [fallback_command "<command>"]]
    [health check:
        type "<type>"
        endpoint "<endpoint>"
        [timeout "<duration>"]
        [interval "<duration>"]
        [retries <number>]
        [condition "<condition>"]]
```

#### Example

```drun
service "api" in "./services/api":
    depends on ["database", "redis"]
    repository:
        url "https://github.com/acme/api.git"
        branch "main"
        clone true  # default, can be omitted
        update_on_start false
    build:
        required true
        command "npm install && npm run build"
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
        timeout "10s"
        interval "2s"
        retries 5
        condition "200"
```

#### Properties

- **name**: Unique identifier for the service
- **path**: Directory containing docker-compose.yml
- **depends on**: List of service dependencies (optional)
- **repository**: Git repository configuration (optional)
  - **url**: Repository URL (required if repository is specified)
  - **branch**: Branch to checkout (optional)
  - **tag**: Tag to checkout (optional, mutually exclusive with branch)
  - **ssh_key**: Path to SSH key for private repositories (optional)
  - **clone**: Auto-clone missing repositories (defaults to `true`, can be omitted)
  - **update_on_start**: Pull latest changes on start (defaults to `false`)
- **build**: Build configuration (optional)
  - **required**: Whether the build is mandatory (defaults to `false`)
  - **command**: Shell command to execute (supports multiline strings)
  - **allocate_tty**: Allocate a pseudo-TTY for the command (defaults to `false`)
  - **makefile**: Path to Makefile (alternative to command)
  - **make_target**: Makefile target to execute
  - **make_args**: Additional arguments to pass to make
  - **makefile_timeout**: Maximum time for make command execution
  - **retry_on_failure**: Retry on build failure (defaults to `false`)
  - **max_retries**: Maximum number of retry attempts
  - **retry_delay**: Delay between retries
  - **fallback_command**: Command to run if make fails
- **health check**: Health check configuration (optional)

### Build Configuration

The build configuration allows you to specify custom build commands or Makefile targets to execute before starting a service.

#### Simple Build Command

```drun
service "api" in "./services/api":
    build:
        required true
        command "npm install && npm run build"
```

#### Multiline Build Commands

**The `command` field supports multiline strings**, allowing complex multi-step build processes:

```drun
service "backend" in "./backend":
    build:
        required true
        command "echo 'Installing dependencies...'
npm install
echo 'Running tests...'
npm test
echo 'Building application...'
npm run build
echo 'Build complete!'"
```

#### Line Continuation

Use backslash (`\`) for line continuation to join lines without newlines:

```drun
service "frontend" in "./frontend":
    build:
        required true
        command "docker build \
            --tag myapp:latest \
            --build-arg ENV=production \
            --build-arg VERSION=1.0.0 \
            ."
```

#### Makefile-Based Build

```drun
service "api" in "./services/api":
    build:
        required true
        makefile "Makefile"
        make_target "build"
        make_args ["ENV=production", "VERBOSE=1"]
        makefile_timeout "10m"
        retry_on_failure true
        max_retries 2
        retry_delay "5s"
        fallback_command "docker compose build"
```

#### Build with Variable Interpolation

```drun
project "myapp" version "1.0":
    parameter $environment defaults to "development"
    parameter $version defaults to "1.0.0"

service "api" in "./api":
    build:
        required true
        command "echo 'Building for {$environment}...'
go mod download
go test ./...
go build -ldflags=\"-X main.version={$version}\" -o bin/api
echo 'Build complete for version {$version}'"
```

#### Build with TTY Allocation

For commands that require interactive terminal access (like `docker compose exec`):

```drun
service "gateway" in "./gateway":
    compose file "docker-compose.dev.yml"
    build:
        required true
        allocate_tty true
        command "make build && make init"
    health check:
        type "http"
        endpoint "http://localhost:93/"
```

**When to use `allocate_tty`:**
-  When your build uses `docker compose exec` to run commands inside containers
-  When scripts require a TTY (check for "input device is not a TTY" errors)
-  When commands need interactive terminal features
-  Not needed for regular shell commands, docker build, or make

#### Complex Real-World Example

```drun
service "web-app" in "./webapp":
    repository:
        url "git@github.com:acme/webapp.git"
        branch "main"
        clone true
    build:
        required true
        command "echo 'Installing frontend dependencies...'
cd frontend && npm ci
echo 'Building frontend assets...'
npm run build
cd ..
echo 'Installing backend dependencies...'
cd backend && go mod download
echo 'Running backend tests...'
go test ./...
echo 'Building backend binary...'
go build -o ../bin/server
cd ..
echo 'Build pipeline complete!'"
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
```

**Key Features:**
- **Multiline Support**: Write complex multi-step commands naturally
- **Line Continuation**: Use `\` to join long single commands
- **Variable Interpolation**: Use `{$var}` syntax in build commands
- **Escaped Quotes**: Use `\"` for quotes within commands
- **Make Integration**: Alternative Makefile-based builds with fallback
- **Retry Logic**: Automatic retries with configurable delays

### Health Check Types

#### HTTP Health Check

Checks an HTTP endpoint for successful response:

```drun
health check:
    type "http"
    endpoint "http://localhost:8080/health"
    timeout "10s"
    interval "2s"
    retries 5
    condition "200"  # Expected HTTP status code
```

#### TCP Health Check

Checks if a TCP port is accepting connections:

```drun
health check:
    type "tcp"
    endpoint "localhost:5432"
    timeout "5s"
    interval "1s"
    retries 10
```

#### Docker Health Check

Uses Docker's native health check status:

```drun
health check:
    type "docker"
    container "container-name"
    timeout "30s"
    interval "2s"
```

#### DNS Health Check

Checks if a hostname resolves:

```drun
health check:
    type "dns"
    endpoint "api.example.com"
    timeout "5s"
    retries 3
```

#### Custom Health Check

Runs a custom command:

```drun
health check:
    type "custom"
    command "curl -f http://localhost:8080/ready || exit 1"
    timeout "5s"
    interval "2s"
    retries 5
```

### Orchestration Groups

Orchestration groups define collections of services with shared lifecycle management.

#### Syntax

```drun
orchestrate "<name>":
    services ["service1", "service2", ...]
    [strategy "<strategy>"]
    [circuit <boolean>]
    [health_check_interval "<duration>"]
```

#### Example

```drun
orchestrate "full_stack":
    services ["database", "redis", "api", "frontend"]
    strategy "dependency-based"
    circuit_breaker true
    health_check_interval "30s"
    git_ssh_key "~/.ssh/id_rsa"
    dns_checks ["api.local", "db.local", "frontend.local"]
```

#### Properties

- **services**: List of services to orchestrate
- **strategy**: Startup strategy (sequential, parallel, dependency-based)
- **circuit_breaker**: Enable circuit breaker behaviour (stops dependent services on failure)
- **stop_on_failure**: Always stop services when any service fails
- **health_check_interval**: Background health check cadence once started
- **startup_timeout** / **shutdown_timeout**: Global orchestration-level timeouts
- **makefile_order** / **makefile_timeout**: Cross-service build sequencing and timeout
- **clone_order** / **clone_timeout**: Repository cloning sequencing and timeout
- **pre_task** / **post_task**: Task hooks executed before start / after stop
- **git_ssh_key**: Default SSH key path for all Git operations (services can override)
- **dns_checks**: Array of domains to validate DNS resolution before orchestration actions

#### Startup Strategies

**Sequential**: Start services one by one in declaration order

```drun
orchestrate "simple":
    services ["a", "b", "c"]
    strategy "sequential"
```

**Dependency-Based** (Recommended): Start based on dependency graph

```drun
orchestrate "smart":
    services ["frontend", "api", "database"]
    strategy "dependency-based"
```

**Parallel**: Start all services simultaneously

```drun
orchestrate "workers":
    services ["worker1", "worker2", "worker3"]
    strategy "parallel"
```

#### Git SSH Key Configuration

Orchestrations can specify a default SSH key for all Git repository operations. This key is used as a fallback for any service that doesn't specify its own SSH key.

```drun
orchestrate "microservices":
    services ["api", "frontend", "worker"]
    strategy "dependency-based"
    git_ssh_key "~/.ssh/id_rsa"
```

**Behavior:**
- Services without their own `ssh_key` configuration will use the orchestration's key
- Services with their own `ssh_key` configuration override the orchestration default
- Supports path expansion (`~` for home directory)
- Applied to all Git operations: clone, fetch, pull

#### DNS Resolution Checks

Orchestrations can validate that specific domains resolve before starting services. This is useful for catching missing `/etc/hosts` entries in local development environments.

```drun
orchestrate "local_stack":
    services ["database", "api", "frontend"]
    strategy "dependency-based"
    dns_checks [
        "api.local",
        "db.local",
        "frontend.local"
    ]
```

**Behavior:**
- DNS checks run before `start`, `up`, `down`, and `status` actions
- Each domain has a 500ms timeout for fast failure detection
- Failures are non-blocking warnings (orchestration continues)
- Silent when all domains resolve (only shows output on failures)
- Helpful warning message suggests adding entries to `/etc/hosts`

**Example Output (on failure):**
```text
 DNS resolution check:
    api.local - not resolvable

  DNS resolution failed for: api.local
These domains may need to be added to your /etc/hosts file
```

### Orchestration Actions

Use orchestration actions within task bodies to manage services.

#### Syntax

```drun
orchestrate "<group_name>" <action> [services ["service1", ...]]
```

#### Available Actions

- `start` - Start services (skip if running and no updates detected)
- `up` - Bring up services with fresh rebuild and repo updates on default branches
- `stop` - Stop all services in reverse order
- `restart` - Stop then start services
- `recreate` - Force a fresh deployment by running `down → build → start`
- `status` - Show status of all services
- `show endpoints` / `endpoints` - List all service endpoints (URLs from health checks)
- `health` / `health_check` - Re-evaluate service health and report failures
- `build` - Build service images
- `pull` - Pull latest images
- `down` - Stop and remove containers
- `logs` - Stream logs for the selected services (supports filters)
- `clone repositories` - Produce the repository cloning plan (dry-run execution)
- `update repositories` - Update repositories to latest version (optionally filter by branch)
- `list branches` - List all repositories with their current branch
- `list branches "branch name"` - Show all repositories checked out on the specified branch
- `switch branch to default` - Switch a specific service (or all) to its default branch
- `set all branches to default` - Set all services to their default branch

**Difference between `start` and `up`:**

| Feature | `start` | `up` |
|---------|---------|------|
| Skip if healthy |  Yes |  No |
| Check for updates |  Yes |  Yes |
| Force repo updates on main/master |  No |  Yes |
| Force rebuild |  No |  Yes |
| **Use for** | Quick restarts | Development, fresh deploys |

#### Resume from a Specific Service

You can resume an orchestration from a specific service using the `starting from` modifier. This is useful when an orchestration fails partway through and you've fixed the issue. The system will verify that all dependencies before the specified service are running and healthy before starting from that point.

**Syntax:**
```drun
orchestrate "<group_name>" <action> starting from "<service_name>"
orchestrate "<group_name>" <action> starting from {$variable}
orchestrate "<group_name>" <action> starting from $variable
```

**Example:**
```drun
task "up" means "Start the stack":
    given $service defaults to empty

    when $service is not empty:
        # Resume from a specific service if provided
        orchestrate "my_stack" up starting from {$service}
    otherwise:
        # Start the full stack
        orchestrate "my_stack" up
```

**Behavior:**
1. Verifies all dependencies before `$service` are running and healthy
2. If any dependency is not running or healthy, fails with an error
3. If all dependencies are satisfied, starts from `$service` onwards in dependency order

**Usage:**
```bash
# Start full stack
xdrun up

# Resume from 'api' service (assumes database, cache are already running)
xdrun up service=api
```

#### Examples

```drun
task "start":
    orchestrate "my_stack" start

task "up":
    orchestrate "my_stack" up

task "stop":
    orchestrate "my_stack" stop

task "restart_api":
    orchestrate "my_stack" restart services ["api"]

task "recreate_api":
    orchestrate "my_stack" recreate services ["api"] with cache "false"

task "update_repos":
    # Update all repositories
    orchestrate "my_stack" update repositories

task "update_main_branch":
    # Update only repositories on main/master branch
    orchestrate "my_stack" update repositories with branch "main"

task "status":
    orchestrate "my_stack" status

task "endpoints":
    orchestrate "my_stack" show endpoints

task "show_api_logs":
    orchestrate "my_stack" logs service "api"

task "check_branches":
    # List all repositories with their current branch
    orchestrate "my_stack" list branches

task "check_main_branch":
    # Show all repositories checked out on "main" branch
    orchestrate "my_stack" list branches "main"

task "switch_service_branch":
    # Switch a specific service to its default branch
    orchestrate "my_stack" switch branch to default service "api"

task "switch_all_branches":
    # Switch all services to their default branch
    orchestrate "my_stack" set all branches to default
```

Service filters can be supplied inline (`services ["api"]`, `service "api"`) or via CLI parameters (`xdrun logs service=api`). Filters accept literal strings, variables (`$service`), or interpolated values (`{$service}`) and are validated against the orchestration's service registry.

#### Branch Management Actions

The branch management actions help you keep repositories aligned with their default branches:

**`list branches`** - Lists all repositories with their current branch:
- Shows all repositories with their current branch name
- Lists services without repository configuration
- Provides a summary count

**`list branches "branch name"`** - Shows repositories on a specific branch:
- Filters to show only repositories checked out on the specified branch
- Useful for finding which services are on a particular branch
- Normalizes branch names (main/master are treated as equivalent)

**`switch branch to default`** - Switches repositories to their default branch:
- If a `service` filter is provided, only switches that service
- If no filter is provided, switches all services
- Skips repositories with uncommitted changes (safety feature)
- Automatically pulls latest changes after switching
- The default branch is detected from the remote repository

**`set all branches to default`** - Sets all services to their default branch:
- Same behavior as `switch branch to default` but applies to all services
- Useful for resetting all repositories to their default state

**Safety Features:**
- Repositories with uncommitted changes are skipped (you must commit or stash changes first)
- The default branch is automatically detected from `origin/HEAD` or by checking for `main`/`master` branches
- After switching, the latest changes are pulled from the default branch

**Variable support in service filters:**
```drun
task "restart_service":
    given $service defaults to empty

    when $service is not empty:
        # All three syntaxes work:
        orchestrate "stack" restart service "api"        # Literal
        orchestrate "stack" restart service $service     # Variable
        orchestrate "stack" restart service {$service}   # Interpolation
```

The `build` and `recreate` actions accept a `with cache "false"` modifier to disable Docker's build cache (`docker compose build --no-cache`) for services that need a completely fresh image.

You can retrieve an orchestration's service list in tasks via the builtin expression `{orchestrate services "stack_name"}` which yields an array literal suitable for loops:

```drun
let $services be {orchestrate services "stack"}

for each $service in $services:
    info "Ensuring {$service} is healthy"
    orchestrate "stack" health services [$service]
```

#### Show Endpoints

The `show endpoints` (alias: `endpoints`) action displays all service endpoints with health check URLs. This is useful for quickly accessing running services or sharing URLs with team members.

```drun
task "endpoints":
    orchestrate "my_stack" show endpoints
```

**Example Output:**
```text
 Service endpoints for orchestration: my_stack

 Running services with endpoints:
   • api: http://localhost:8080/health
   • frontend: http://localhost:3000/
   • admin: http://localhost:9000/admin

 Running services (no endpoints):
   • database
   • redis

  Stopped services:
   • worker
   • scheduler
```

**Behavior:**
- Shows services grouped by status: running with endpoints, running without, stopped
- Only displays URLs from HTTP health checks
- Services without health checks appear in "running without endpoints"
- Services that are not running appear in the stopped section

### Progress Display

The orchestration system features a BuildKit-inspired real-time progress display.

#### Status Indicators

-  **Pending** - Waiting to start
-  **Starting** - Service is starting
-  **Healthy** - Started and passed health checks
-  **Failed** - Failed to start or unhealthy
-  **Stopping** - Being stopped
-  **Stopped** - Successfully stopped

#### Example Output

```text
 Starting orchestration: full_stack
   4 services in dependency order

   database
   redis
   api
   frontend

   database     Starting service... [0s]
   database     Waiting for health check... [0s]
   database     Healthy [2s]
   redis        Starting service... [0s]
   redis        Healthy [1s]
   api          Starting service... [0s]
   api          Healthy [3s]
   frontend     Starting service... [0s]
   frontend     Healthy [2s]

 4/4 services completed successfully
```

### Circuit Breaker

When circuit breaker is enabled, any failure stops and rolls back all services.

#### Configuration

```drun
orchestrate "critical":
    services ["database", "api", "frontend"]
    circuit_breaker true  # Enable circuit breaker
```

#### Behavior

```text
 Starting orchestration: critical
    Circuit breaker: ENABLED - will stop all on failure

   database     Healthy [2s]
   api          Health check failed [5s]

 Circuit breaker triggered! Rolling back all services...

   database     Rolling back... [0s]
   database     Stopped (rollback) [0s]

 1/3 services failed
Error: circuit breaker: health check failed for 'api', all services stopped
```

### Resilient Mode

When circuit breaker is disabled, failures are tolerated and the system continues in degraded mode.

#### Configuration

```drun
orchestrate "resilient":
    services ["database", "api", "frontend"]
    circuit false  # Disable circuit breaker
```

#### Behavior

```text
 Starting orchestration: resilient

   database     Healthy [2s]
   api          Health check failed [5s]
   api            Unhealthy: health check failed [5s]
   frontend     Healthy [2s]

 2/3 services completed successfully
```

### Complete Example

```drun
version: 2.0

project "e-commerce" version "1.0":

# Infrastructure
service "database" in "./services/db":
    health check:
        type "tcp"
        endpoint "localhost:5432"
        timeout "10s"
        retries 10

service "cache" in "./services/redis":
    health check:
        type "tcp"
        endpoint "localhost:6379"
        timeout "5s"
        retries 5

# Application
service "api" in "./services/api":
    depends on ["database", "cache"]
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
        timeout "15s"
        interval "2s"
        retries 5

service "frontend" in "./services/web":
    depends on ["api"]
    health check:
        type "http"
        endpoint "http://localhost:3000/"
        timeout "10s"
        retries 3

# Orchestration
orchestrate "platform":
    services ["database", "cache", "api", "frontend"]
    strategy "dependency-based"
    circuit_breaker true

# Tasks
task "start":
    info " Starting platform..."
    orchestrate "platform" start
    success "Platform ready at http://localhost:3000"

task "stop":
    info " Stopping platform..."
    orchestrate "platform" stop

task "restart":
    orchestrate "platform" restart

task "status":
    orchestrate "platform" status

task "rebuild":
    orchestrate "platform" build
    orchestrate "platform" restart

task "cleanup":
    orchestrate "platform" down
```

### Docker Compose Integration

The orchestration system executes `docker compose` commands directly:

#### Start Service

```bash
cd /path/to/service
docker compose up -d
```

#### Stop Service

```bash
cd /path/to/service
docker compose stop
```

#### Service Status

```bash
cd /path/to/service
docker compose ps
```

#### Down (Remove)

```bash
cd /path/to/service
docker compose down
```

### Dependency Resolution

Services are started in topological order based on dependencies:

#### Example

```drun
service "database" in "./db":
service "cache" in "./cache":
service "api" in "./api":
    depends on ["database", "cache"]
service "frontend" in "./web":
    depends on ["api"]
```

**Resolution Order:**
1. `database` and `cache` (parallel - no dependencies)
2. `api` (after database and cache)
3. `frontend` (after api)

#### Shutdown Order

Shutdown occurs in reverse topological order:
1. `frontend`
2. `api`
3. `database` and `cache`

### Error Handling

#### Health Check Failures

```text
   api    Waiting for health check...: health check failed after 5 attempts [10s]
```

**Common causes:**
- Service not responding at endpoint
- Wrong port or URL configuration
- Service internal error
- Health check timeout too short

#### Docker Compose Errors

```text
   service  Starting service...: docker compose failed: exit status 1
Output: Error response from daemon: Bind for 0.0.0.0:8080 failed: port is already allocated
```

**Common causes:**
- Port conflict with another container
- Docker Compose file doesn't exist
- Permission issues
- Invalid Docker Compose configuration

#### Missing Service Path

```text
   service  Starting service...: docker compose failed: chdir /path: no such file or directory
```

**Common causes:**
- Incorrect path in service declaration
- Service directory doesn't exist
- Docker Compose file not in expected location

### Best Practices

1. **Always Use Health Checks**

```drun
service "api" in "./api":
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
        timeout "10s"
        retries 5
```

2. **Use Dependency-Based Strategy**

```drun
orchestrate "stack":
    services [...]
    strategy "dependency-based"  # Recommended
```

3. **Enable Circuit Breaker for Critical Systems**

```drun
orchestrate "production":
    circuit_breaker true  # Fail fast in production
```

4. **Group Related Services**

```drun
orchestrate "infra":
    services ["database", "cache"]

orchestrate "app":
    services ["api", "worker"]
```

5. **Use Meaningful Timeouts**

```drun
service "database" in "./db":
    health check:
        timeout "30s"  # Databases need more time
        retries 10

service "api" in "./api":
    health check:
        timeout "10s"  # APIs start faster
        retries 5
```

### Implementation Details

#### Components

1. **AST Nodes**
   - `ServiceStatement` - Service declarations
   - `OrchestrateStatement` - Orchestration groups
   - `OrchestrationActionStatement` - Actions in tasks

2. **Domain Models**
   - `Service` - Runtime service representation
   - `OrchestrationGroup` - Group configuration
   - `HealthCheck` - Health check configuration

3. **Execution Engine**
   - Dependency resolution (topological sort)
   - Docker Compose execution
   - Health check monitoring
   - Progress display

4. **Health Check System**
   - HTTP checker
   - TCP checker
   - Docker checker
   - DNS checker
   - Custom command checker

#### Data Flow

```text
Drunfile
   ↓
Parser (service & orchestrate)
   ↓
AST Nodes
   ↓
Domain Models
   ↓
Task Execution
   ↓
Orchestration Engine
   ├→ Dependency Resolution
   ├→ Docker Compose Commands
   ├→ Health Monitoring
   └→ Progress Display
```

### Implementation Details

The microservices orchestration system includes several sophisticated implementation details and edge case handling:

#### Parser Robustness

**Infinite Loop Prevention**
The orchestration parser includes comprehensive error handling to prevent infinite loops during parsing:

- String array parsing advances tokens on unexpected types
- Orchestration body parsing includes proper error recovery
- Service body parsing provides detailed error messages with graceful degradation

**Compose File Syntax Support**
The parser supports multiple Docker Compose file specification syntaxes:

- Inline syntax: `compose file "docker-compose.yml"`
- Block syntax: `compose:` with nested configuration
- Automatic detection and proper handling of both formats

#### Docker Compose Integration

**Working Directory Management**
Each service runs Docker Compose commands from its own directory with proper environment setup:

- Service directory resolution with absolute path conversion
- PWD environment variable setup for `${PWD}` references in compose files
- Proper working directory context for all Docker operations

**BuildKit Output Streaming**
Real-time Docker build output is streamed to provide visibility:

- Direct stdout/stderr streaming during build operations
- Integration with progress display system
- Build failure handling with circuit breaker integration

#### Circuit Breaker Implementation

**Smart Rollback Logic**
The circuit breaker uses intelligent dependency-aware rollback:

- Only stops services that were started after the failed service
- Index-based dependency tracking for efficient rollback
- Avoids stopping services that don't depend on the failed service

**Multi-Failure Type Handling**
Circuit breaker is triggered by various failure types:

- Docker Compose build failures
- Docker Compose startup failures
- Health check timeout failures
- Service dependency resolution failures

#### Progress Display System

**BuildKit-Style Visual Feedback**
Real-time progress display with comprehensive state tracking:

- Distinct pending, building, starting, healthy, and failed service states.
- Inline progress updates without output cluttering
- Build output integration with progress display
- Final summary with success/failure counts

**Thread-Safe State Management**
Comprehensive state tracking with concurrency safety:

- Mutex-protected state updates for concurrent access
- Timing information for performance monitoring
- Error tracking with detailed error information
- Clear state transitions throughout service lifecycle

#### Error Handling & Recovery

**Parser Error Recovery**
Robust error recovery mechanisms:

- Token synchronization on parse errors
- Detailed error messages with line/column information
- Graceful degradation when individual statements fail

**Docker Compose Error Handling**
Comprehensive error handling for Docker operations:

- Proper error capture and reporting for all Docker commands
- Output capture for both stdout and stderr debugging
- Configurable timeouts for long-running operations
- Exit code validation and error propagation

#### Performance Considerations

**Sequential vs Parallel Operations**
Balanced approach to safety and performance:

- Sequential service startup for proper dependency resolution
- Parallel health checks after startup completion
- Concurrent rollback operations for efficiency

**Memory Management**
Efficient resource usage patterns:

- Streaming output instead of buffering for large builds
- Proper goroutine cleanup and management
- Docker resource cleanup on completion

#### Configuration Flexibility

**Service Configuration Options**
Comprehensive service configuration:

- Multiple health check types (HTTP, TCP, Docker, DNS, Custom)
- Build configuration with timeout and parallel job settings
- Complex dependency graphs with topological sorting
- Environment variable management per service

**Orchestration Strategies**
Multiple orchestration approaches:

- Dependency-based startup (default)
- Sequential startup regardless of dependencies
- Future parallel startup capabilities

#### Testing & Validation

**Comprehensive Test Coverage**
Extensive testing implementation:

- Unit tests for all parsing scenarios
- Integration tests with real Docker containers
- Error case testing for various failure scenarios
- Circuit breaker behavior validation

**Real-World Validation**
Production validation with actual projects:

- POG programming language
- Eating our own dogfood at Phillarmonic
- Docker Compose file compatibility validation
- Health check testing with real services

### Future Enhancements

Planned features for future versions:

- Service discovery integration
- Metrics and monitoring
- Automatic rollback support
- Blue/green deployments
- Dynamic scaling operations

### Related Documentation

- [Orchestration Guide](../guides/orchestration.md) - Task-oriented orchestration documentation
- [Examples](./examples/) - Working examples
