# Microservices Orchestration in Drun

## Overview

Drun provides a comprehensive microservices orchestration system that manages the lifecycle of multi-service architectures with Docker Compose integration, health monitoring, dependency resolution, and visual progress feedback.

## Features at a Glance

- ✅ **Service Declarations**: Define services with paths, dependencies, and health checks
- ✅ **Orchestration Groups**: Group services with lifecycle management strategies
- ✅ **BuildKit-Style Progress**: Real-time visual feedback during operations
- ✅ **Health Monitoring**: HTTP, TCP, Docker, DNS, and custom health checks
- ✅ **Dependency Resolution**: Topological sort ensures correct startup/shutdown order
- ✅ **Smart Circuit Breaker**: Only stops dependent services on failure (not all services)
- ✅ **Docker Compose Integration**: Direct `docker compose` command execution
- ✅ **Automatic Building**: Build containers before starting with real-time output
- ✅ **PWD Environment**: Correct working directory for Docker Compose files
- ✅ **Graceful Shutdown**: Services stopped in reverse dependency order
- ✅ **Error Handling**: Detailed error messages with rollback support
- ✅ **BuildKit Output Streaming**: Real-time Docker build progress visibility

## Quick Start

### Basic Example

```drun
version: 2.0

project "my-microservices" version "1.0":

# Define services
service "database" in "./services/database":
    health check:
        type "tcp"
        endpoint "localhost:5432"
        timeout "5s"
        interval "2s"
        retries 5

service "api" in "./services/api":
    depends on ["database"]
    health check:
        type "http"
        endpoint "http://localhost:8080/"
        timeout "10s"
        interval "2s"
        retries 5

service "frontend" in "./services/frontend":
    depends on ["api"]
    health check:
        type "http"
        endpoint "http://localhost:3000/"

# Group services into an orchestration
orchestrate "full_stack":
    services ["database", "api", "frontend"]
    strategy "dependency-based"
    circuit true  # Stop all on failure

# Create tasks to manage the stack
task "start":
    info "🚀 Starting full stack..."
    orchestrate "full_stack" start
    success "All services running!"

task "up":
    info "📦 Bringing up full stack with fresh build..."
    orchestrate "full_stack" up
    success "All services rebuilt and running!"

task "stop":
    info "🛑 Stopping services..."
    orchestrate "full_stack" stop

task "restart":
    info "♻️ Restarting services..."
    orchestrate "full_stack" restart

task "status":
    info "📊 Service status:"
    orchestrate "full_stack" status

task "endpoints":
    info "🌐 Service endpoints:"
    orchestrate "full_stack" show-endpoints

task "down":
    info "🗑️  Removing containers..."
    orchestrate "full_stack" down
```

### Running the Example

```bash
# Start services (skip if already running and no updates detected)
drun start

# Bring up services (force rebuild and update repos on main/master)
drun up

# Check status
drun status

# View service endpoints
drun endpoints

# Stop gracefully (reverse order)
drun stop

# Remove all containers
drun down
```

### Difference Between `start` and `up`

#### `orchestrate "stack" start`
- ✅ Checks if services are healthy
- ✅ Checks for repository updates (but doesn't force update)
- ✅ Only rebuilds if `build: required true` is set
- ✅ Skips services that are already running with no updates
- 🎯 **Use for**: Quick starts when nothing has changed

#### `orchestrate "stack" up`
- ✅ Updates all repositories on default branches (main/master)
- ✅ Forces rebuild of all services (even without `build: required`)
- ✅ Recreates containers with latest code
- ✅ Never skips services
- 🎯 **Use for**: Development workflow, pulling latest changes, ensuring fresh build

## Service Declaration

Services are the building blocks of your orchestration. Each service represents a Docker Compose project.

### Basic Service

```drun
service "my-service" in "./path/to/service":
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
```

### Service with Dependencies

```drun
service "api" in "./services/api":
    depends on ["database", "redis"]
    health check:
        type "http"
        endpoint "http://localhost:8080/"
        timeout "10s"
        interval "2s"
        retries 5
        condition "200"
```

### Service Properties

| Property                 | Description                                                                  | Required |
| ------------------------ | ---------------------------------------------------------------------------- | -------- |
| `in "path"`              | Path to service directory containing docker-compose.yml                      | Yes      |
| `depends on [...]`       | List of service dependencies                                                 | No       |
| `health check`           | Health check configuration (see below)                                       | No       |
| `repository`             | Git repository configuration (URL, branch/tag, clone/update behaviour)       | No       |
| `build`                  | Pre-start build configuration (shell command or Makefile, retries, fallback) | No       |
| `compose`                | Docker Compose overrides (`file`, `project`, advanced options)               | No       |
| `environment`            | Inline environment variables (`KEY "value"`)                                 | No       |
| `env_file`               | Automatically create/validate `.env` files (optional setup task)             | No       |
| `pre_task` / `post_task` | Task hook to run before start / after stop                                   | No       |
| `networks`               | Custom Docker network configuration                                          | No       |

## Health Checks

Health checks ensure services are ready before proceeding. Multiple types are supported:

### HTTP Health Check

```drun
service "api" in "./api":
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
        timeout "10s"
        interval "2s"
        retries 5
        condition "200"  # Expected HTTP status code
```

### TCP Health Check

```drun
service "database" in "./database":
    health check:
        type "tcp"
        endpoint "localhost:5432"
        timeout "5s"
        interval "1s"
        retries 10
```

### Docker Health Check

Uses Docker's built-in health check status:

```drun
service "redis" in "./redis":
    health check:
        type "docker"
        container "redis-container"
        timeout "30s"
        interval "2s"
```

### DNS Health Check

```drun
service "external-api" in "./api":
    health check:
        type "dns"
        endpoint "api.example.com"
        timeout "5s"
        retries 3
```

### Custom Health Check

Run a custom command:

```drun
service "custom" in "./custom":
    health check:
        type "custom"
        command "curl -f http://localhost:8080/ready || exit 1"
        timeout "5s"
        interval "2s"
        retries 5
```

## Repository Management

Services can automatically clone or update Git repositories before they start:

```drun
service "api" in "./services/api":
    repository:
        url "https://github.com/acme/api.git"
        branch "main"
        clone true  # default, can be omitted
        update_on_start false
```

Use `url`, optional `branch` / `tag`, `ssh_key`, and flags (`clone`, `update_on_start`) to control behaviour. The `clone` flag defaults to `true` (auto-clone missing repositories), set to `false` to disable. Orchestrations may define a cloning order across services with `clone_order ["service-a", ...]` and a `clone_timeout`.

### Updating Repositories

You can update all repositories or filter by branch name:

```drun
task "update_all":
    # Update all repositories in the orchestration
    orchestrate "my_stack" update repositories

task "update_main_branch":
    # Update only repositories on main/master branch
    orchestrate "my_stack" update repositories with branch "main"

task "update_specific_services":
    # Update specific services
    orchestrate "my_stack" update repositories services ["api", "frontend"]
```

The `update repositories` action will:
- Skip services without repository configuration
- Skip repositories that aren't cloned locally
- When a branch filter is specified, only update services currently on that branch (main/master are treated as equivalent)
- Pull the latest changes from the remote branch
- Provide a summary of updated, skipped, and failed repositories

## Build Configuration

The `build` block supports shell commands or Makefile targets with retries, timeouts, and fallbacks.

### Simple Build Command

```drun
service "api" in "./services/api":
    build:
        required true
        command "npm install && npm run build"
```

### Multiline Build Commands

**The `command` field fully supports multiline strings**, enabling complex multi-step build processes:

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

### Line Continuation

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

### Makefile-Based Builds

```drun
service "api" in "./services/api":
    build:
        required true
        makefile "Makefile"
        make_target "build"
        make_args ["ENV=development"]
        makefile_timeout "5m"
        retry_on_failure true
        max_retries 2
        retry_delay "5s"
        fallback_command "npm run build"
```

### Build with Variable Interpolation

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

### Complex Real-World Example

```drun
service "web-app" in "./webapp":
    repository:
        url "git@github.com:acme/webapp.git"
        branch "main"
        clone true
    build:
        required true
        command "echo 'Setting up monorepo build...'
# Install all dependencies
npm install --workspaces
# Build frontend
echo 'Building frontend...'
cd packages/frontend
npm run test
npm run build
cd ../..
# Build backend
echo 'Building backend...'
cd packages/backend
go mod download
go test ./...
go build -o ../../bin/server
cd ../..
echo 'Monorepo build complete!'"
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
```

### Build with TTY Allocation

For commands that require interactive terminal access (like `docker compose exec`):

```drun
service "gateway" in "./celesta-gateway":
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
- ✅ When your build uses `docker compose exec` to run commands inside containers
- ✅ When scripts require a TTY (resolves "input device is not a TTY" errors)
- ✅ When commands need interactive terminal features
- ❌ Not needed for regular shell commands, docker build, or make

### Build Configuration Options

When `required` is `true`, the engine honours the full configuration:

- **command**: Shell command(s) to execute (supports multiline strings)
- **allocate_tty**: Allocate a pseudo-TTY for the command (defaults to `false`)
- **makefile**: Path to Makefile (alternative to command)
- **make_target**: Specific Makefile target to execute
- **make_args**: Additional arguments to pass to make
- **makefile_timeout**: Maximum execution time for make command
- **retry_on_failure**: Automatically retry on build failure
- **max_retries**: Maximum number of retry attempts
- **retry_delay**: Delay between retry attempts
- **fallback_command**: Command to run if make fails

**Key Features:**
- ✅ **Multiline Support**: Write complex multi-step commands naturally
- ✅ **Line Continuation**: Use `\` to join long single commands  
- ✅ **Variable Interpolation**: Use `{$var}` syntax in build commands
- ✅ **Escaped Quotes**: Use `\"` for quotes within commands
- ✅ **TTY Allocation**: Enable for commands requiring terminal access
- ✅ **Make Integration**: Alternative Makefile-based builds with fallback
- ✅ **Retry Logic**: Automatic retries with configurable delays

## Environment File Management

Ensure `.env` files exist before a service starts by using `env_file`. You can run a task to generate missing files and take advantage of the new `replace` file action:

```drun
service "api" in "./services/api":
    env_file:
        required true
        task "setup_api_env"

task "setup_api_env":
    copy "./services/api/.env.example" to "./services/api/.env"
    replace in "./services/api/.env":
        "DB_PASSWORD=CHANGE_ME" with "DB_PASSWORD={$password}"
        "API_KEY=CHANGE_ME" with "API_KEY={$api_key}"
    run "chmod 600 ./services/api/.env"
```

## Advanced Orchestration Configuration

Orchestration groups now support additional lifecycle tuning:

```drun
orchestrate "all_services":
    services ["database", "api", "frontend"]
    strategy "dependency-based"
    circuit_breaker true
    stop_on_failure true
    health_check_interval "30s"
    startup_timeout "5m"
    shutdown_timeout "1m"
    makefile_order ["api", "frontend"]
    makefile_timeout "10m"
    clone_order ["api", "frontend"]
    clone_timeout "5m"
    pre_task "global_setup"
    post_task "global_cleanup"
```

- `health_check_interval` schedules background health monitoring after successful start.
- `startup_timeout` / `shutdown_timeout` apply orchestration-wide SLA bounds.
- `makefile_order`, `makefile_timeout`, `clone_order`, and `clone_timeout` coordinate builds and repository updates across services.
- `pre_task` / `post_task` run only once per orchestration start/stop cycle.

## Orchestration Actions in Tasks

The `orchestrate` action supports a growing list of verbs:

| Action                     | Description                                                                      |
| -------------------------- | -------------------------------------------------------------------------------- |
| `start`, `stop`, `restart` | Manage lifecycle while honouring dependencies and hooks                          |
| `recreate`                 | Force a fresh deployment by running `down → build → start` for targeted services |
| `status`                   | Print Docker Compose status for each service                                     |
| `health`, `health_check`   | Re-run service health checks and report any failures                             |
| `build`                    | Rebuild services based on their `build` configuration                            |
| `pull`                     | Pull images for all targeted services                                            |
| `down`                     | Tear down stacks in reverse dependency order                                     |
| `logs`                     | Tail logs for the selected services (`service` filter optional)                  |
| `clone repositories`       | Report repository cloning order (dry-run for execution)                          |
| `update repositories`      | Update repositories to latest version (optionally filter by branch)             |

You can filter orchestration actions to specific services directly or via CLI parameters:

```drun
task "show-api-logs":
    orchestrate "all_services" logs service "api"

task "rebuild-web-tier":
    orchestrate "all_services" build services ["frontend"]

task "bounce-api":
    orchestrate "all_services" recreate services ["api"] with cache "false"
```

Use the optional `with cache "false"` modifier with either `build` or `recreate` to pass `--no-cache` to underlying `docker compose build` runs when you need a completely fresh image.

## Service-Scoped Task Commands

When your project declares services, task steps can target their working directories without manual `cd` logic:

```drun
task "inspect-service":
    given $servicename defaults to "some-service"
    run in service $servicename "ls -a"
    docker compose in service $servicename exec -it app bash
```

- `run in service …` executes the shell command from the resolved service path (with the same path resolution used by orchestrations).
- `docker compose in service …` anchors Docker Compose commands to the service directory while preserving streamed output.
- Service names may come from literals, task parameters, or captured variables; the engine resolves them before execution.

Commands fail fast if no services are defined or the requested service cannot be found.

Need to fan out across several services? Capture the stack definition into a variable and iterate:

```drun
let $services be {orchestrate services "celesta-sb-stack"}

for each $service in $services:
    info "Checking {$service}"
    orchestrate "celesta-sb-stack" status services [$service]
```

The `{orchestrate services "…"}` builtin returns an array literal, so it plugs straight into `for each` loops or other list-aware features.

## Orchestration Groups

Group services together with shared lifecycle management:

```drun
orchestrate "my-stack":
    services ["database", "cache", "api", "frontend"]
    strategy "dependency-based"
    circuit true
    health_check_interval "30s"
```

### Orchestration Properties

| Property                | Description                    | Default      |
| ----------------------- | ------------------------------ | ------------ |
| `services [...]`        | List of services in this group | Required     |
| `strategy "..."`        | Startup strategy (see below)   | "sequential" |
| `circuit true/false`    | Enable circuit breaker         | false        |
| `health_check_interval` | How often to check health      | "30s"        |

### Startup Strategies

#### Sequential

Services start one by one in declaration order:

```drun
orchestrate "simple":
    services ["a", "b", "c"]
    strategy "sequential"
```

Order: a → b → c

#### Dependency-Based (Recommended)

Services start based on dependency graph:

```drun
orchestrate "smart":
    services ["frontend", "api", "database"]
    strategy "dependency-based"
```

Order: database → api → frontend (based on dependencies)

#### Parallel

Start all services simultaneously (use with caution):

```drun
orchestrate "fast":
    services ["worker1", "worker2", "worker3"]
    strategy "parallel"
```

## Orchestration Actions

Use orchestration actions within task bodies to manage services:

### Available Actions

| Action            | Description                            |
| ----------------- | -------------------------------------- |
| `start`           | Start all services in dependency order |
| `stop`            | Stop all services in reverse order     |
| `restart`         | Stop then start services               |
| `status`          | Show status of all services            |
| `show-endpoints`  | List all service endpoints             |
| `build`           | Build service images                   |
| `pull`            | Pull latest images                     |
| `down`            | Stop and remove containers             |

### Action Examples

```drun
task "lifecycle-demo":
    # Start services
    orchestrate "my-stack" start

    # Check status
    orchestrate "my-stack" status

    # View service endpoints
    orchestrate "my-stack" show-endpoints

    # Restart specific services
    orchestrate "my-stack" restart services ["api"]

    # Stop everything
    orchestrate "my-stack" stop
```

## BuildKit-Style Progress Display

The orchestration system features a real-time progress display inspired by Docker BuildKit:

### Progress Indicators

Each service shows its current status with visual feedback:

| Icon | Status   | Description          |
| ---- | -------- | -------------------- |
| ⏸️   | Pending  | Waiting to start     |
| 🔄   | Starting | Service is starting  |
| ✅    | Healthy  | Started and healthy  |
| ❌    | Failed   | Failed to start      |
| 🛑   | Stopping | Being stopped        |
| ⏹️   | Stopped  | Successfully stopped |

### Example Output

```
🚀 Starting orchestration: full_stack
   4 services in dependency order

  ⏸️ database     
  ⏸️ redis        
  ⏸️ api          
  ⏸️ frontend     

  🔄 database     Starting service... [0s]
  🔄 database     Waiting for health check... [0s]
  ✅ database     Healthy [2s]
  🔄 redis        Starting service... [0s]
  🔄 redis        Waiting for health check... [0s]
  ✅ redis        Healthy [1s]
  🔄 api          Starting service... [0s]
  🔄 api          Waiting for health check... [0s]
  ✅ api          Healthy [3s]
  🔄 frontend     Starting service... [0s]
  🔄 frontend     Waiting for health check... [0s]
  ✅ frontend     Healthy [2s]

✅ 4/4 services completed successfully
```

### Timing Information

Each service displays elapsed time for operations:

- Tracks time from start to completion
- Shows wait time during health checks
- Helps identify bottlenecks

## Error Handling

### Circuit Breaker Mode

When `circuit true` is enabled, any failure stops and rolls back all services:

```drun
orchestrate "critical_stack":
    services ["database", "api", "frontend"]
    strategy "dependency-based"
    circuit true  # Stop all on failure
```

**Example Failure:**

```
🚀 Starting orchestration: critical_stack
   3 services in dependency order
   🔴 Circuit breaker: ENABLED - will stop all on failure

  ⏸️ database     
  ⏸️ api          
  ⏸️ frontend     

  🔄 database     Starting service... [0s]
  🔄 database     Waiting for health check... [0s]
  ✅ database     Healthy [0s]
  🔄 api          Starting service... [0s]
  🔄 api          Waiting for health check... [0s]
  ❌ api          Waiting for health check...: health check failed after 5 attempts [10s]

🔴 Circuit breaker triggered! Rolling back all services...

  🛑 database     Rolling back... [0s]
  ⏹️ database     Stopped (rollback) [0s]

❌ 1/3 services failed
Error: circuit breaker: health check failed for 'api', all services stopped
```

### Resilient Mode

When `circuit false`, failures are tolerated and the system continues in degraded mode:

```drun
orchestrate "resilient_stack":
    services ["database", "api", "frontend"]
    strategy "dependency-based"
    circuit false  # Continue despite failures
```

**Example with Degraded Service:**

```
🚀 Starting orchestration: resilient_stack
   3 services in dependency order

  🔄 database     Starting service... [0s]
  ✅ database     Healthy [0s]
  🔄 api          Starting service... [0s]
  ❌ api          Waiting for health check...: health check failed [5s]
  🔄 api          ⚠️  Unhealthy: health check failed [5s]
  🔄 frontend     Starting service... [0s]
  ✅ frontend     Healthy [2s]

✅ 2/3 services completed successfully
```

### Common Error Scenarios

#### 1. Health Check Failure

```
  ❌ api          Waiting for health check...: health check failed after 5 attempts [10s]
```

**Causes:**

- Service starts but endpoint not responding
- Wrong port configuration
- Service internal error
- Health check timeout too short

#### 2. Missing Docker Compose File

```
  ❌ service      Starting service...: docker compose failed: chdir /path: no such file or directory
```

**Causes:**

- Incorrect service path
- Docker Compose file doesn't exist
- Permission issues

#### 3. Port Conflict

```
  ❌ redis        Starting service...: docker compose failed: exit status 1
Output: Error response from daemon: Bind for 0.0.0.0:6379 failed: port is already allocated
```

**Causes:**

- Another container using the same port
- Previous containers not cleaned up
- Port already in use by host process

## Complete Example

Here's a full example demonstrating all features:

```drun
version: 2.0

project "e-commerce-platform" version "2.0":

# Infrastructure services
service "database" in "./services/database":
    health check:
        type "tcp"
        endpoint "localhost:5432"
        timeout "10s"
        interval "2s"
        retries 10

service "cache" in "./services/redis":
    health check:
        type "tcp"
        endpoint "localhost:6379"
        timeout "5s"
        interval "1s"
        retries 5

service "message_queue" in "./services/rabbitmq":
    health check:
        type "http"
        endpoint "http://localhost:15672/"
        timeout "10s"
        interval "2s"
        retries 5

# Application services
service "auth_service" in "./services/auth":
    depends on ["database", "cache"]
    health check:
        type "http"
        endpoint "http://localhost:8001/health"
        timeout "15s"
        interval "2s"
        retries 5

service "product_service" in "./services/products":
    depends on ["database", "cache", "message_queue"]
    health check:
        type "http"
        endpoint "http://localhost:8002/health"
        timeout "15s"
        interval "2s"
        retries 5

service "order_service" in "./services/orders":
    depends on ["database", "message_queue", "auth_service"]
    health check:
        type "http"
        endpoint "http://localhost:8003/health"
        timeout "15s"
        interval "2s"
        retries 5

service "api_gateway" in "./services/gateway":
    depends on ["auth_service", "product_service", "order_service"]
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
        timeout "10s"
        interval "2s"
        retries 5

service "frontend" in "./services/web":
    depends on ["api_gateway"]
    health check:
        type "http"
        endpoint "http://localhost:3000/"
        timeout "10s"
        interval "2s"
        retries 3

# Orchestration groups
orchestrate "infrastructure":
    services ["database", "cache", "message_queue"]
    strategy "parallel"
    circuit true

orchestrate "backend":
    services ["auth_service", "product_service", "order_service"]
    strategy "dependency-based"
    circuit true

orchestrate "frontend_stack":
    services ["api_gateway", "frontend"]
    strategy "dependency-based"
    circuit true

orchestrate "full_platform":
    services [
        "database", 
        "cache", 
        "message_queue",
        "auth_service",
        "product_service",
        "order_service",
        "api_gateway",
        "frontend"
    ]
    strategy "dependency-based"
    circuit true
    health_check_interval "60s"

# Management tasks
task "start":
    info "╔═══════════════════════════════════════╗"
    info "║  Starting E-Commerce Platform         ║"
    info "╚═══════════════════════════════════════╝"
    info ""
    orchestrate "full_platform" start
    info ""
    success "Platform is ready!"
    info "  Frontend: http://localhost:3000"
    info "  API: http://localhost:8080"

task "start:infra":
    info "Starting infrastructure services..."
    orchestrate "infrastructure" start

task "start:backend":
    info "Starting backend services..."
    orchestrate "backend" start

task "start:frontend":
    info "Starting frontend..."
    orchestrate "frontend_stack" start

task "stop":
    info "Stopping all services..."
    orchestrate "full_platform" stop
    success "All services stopped"

task "restart":
    info "Restarting platform..."
    orchestrate "full_platform" restart
    success "Platform restarted"

task "status":
    info "Platform Status:"
    orchestrate "full_platform" status

task "health":
    info "Running health checks..."
    orchestrate "full_platform" status

task "rebuild":
    info "Rebuilding all services..."
    orchestrate "full_platform" build
    orchestrate "full_platform" restart
    success "Platform rebuilt and restarted"

task "cleanup":
    info "Removing all containers and networks..."
    orchestrate "full_platform" down
    success "Cleanup complete"

task "logs":
    requires $service as string
    info "Showing logs for {$service}..."
    orchestrate "full_platform" status
```

## Best Practices

### 1. Use Descriptive Service Names

```drun
# Good
service "user_authentication_service" in "./auth"

# Avoid
service "svc1" in "./s1"
```

### 2. Always Configure Health Checks

```drun
# Always specify health checks for reliable orchestration
service "api" in "./api":
    health check:
        type "http"
        endpoint "http://localhost:8080/health"
        timeout "10s"
        retries 5
```

### 3. Use Dependency-Based Strategy

```drun
# Recommended for most use cases
orchestrate "my-stack":
    services [...]
    strategy "dependency-based"  # Ensures correct order
```

### 4. Enable Circuit Breaker for Critical Systems

```drun
orchestrate "production":
    services [...]
    circuit true  # Fail fast in production
```

### 5. Group Related Services

```drun
# Infrastructure
orchestrate "infra":
    services ["database", "cache", "queue"]

# Application
orchestrate "app":
    services ["api", "worker", "frontend"]
```

### 6. Use Meaningful Timeouts

```drun
service "database" in "./db":
    health check:
        timeout "30s"    # Databases can be slow to start
        retries 10       # Give it multiple chances

service "api" in "./api":
    health check:
        timeout "10s"    # APIs should start faster
        retries 5
```

## Troubleshooting

### Services Not Starting

1. Check Docker Compose files exist
2. Verify paths in service declarations
3. Check port conflicts: `docker ps -a`
4. Review health check configuration
5. Increase timeout/retries if needed

### Health Checks Failing

1. Verify endpoints are correct
2. Check service logs: `docker compose logs`
3. Test endpoint manually: `curl http://localhost:8080/health`
4. Increase health check timeout
5. Verify service is fully initialized

### Dependency Issues

1. Review dependency declarations
2. Check for circular dependencies
3. Verify dependency names match service names
4. Use `drun status` to see service states

### Circuit Breaker Triggering Unexpectedly

1. Check health check timeouts (may be too aggressive)
2. Review service startup times
3. Consider using `circuit false` for development
4. Add logging to identify failing service

## Architecture

The orchestration system consists of several components:

### Components

1. **AST (Abstract Syntax Tree)**
   
   - `ServiceStatement`: Service declarations
   - `OrchestrateStatement`: Orchestration group declarations
   - `OrchestrationActionStatement`: Actions in task bodies

2. **Parser**
   
   - Parses service and orchestrate blocks
   - Parses orchestration actions within tasks
   - Validates syntax and structure

3. **Domain Models**
   
   - `Service`: Runtime service representation
   - `OrchestrationGroup`: Group of services
   - `HealthCheck`: Health check configuration

4. **Execution Engine**
   
   - `orchestrateStart`: Start services with dependency resolution
   - `orchestrateStop`: Stop services in reverse order
   - `ProgressDisplay`: BuildKit-style visual feedback
   - `waitForHealth`: Health check monitoring

5. **Docker Integration**
   
   - `runDockerCompose`: Execute docker compose commands
   - `buildDockerComposeCmd`: Build command with environment
   - Service status queries

6. **Health Check System**
   
   - HTTP health checks
   - TCP health checks
   - Docker native health checks
   - DNS resolution checks
   - Custom command health checks

### Data Flow

```
Drunfile
   ↓
Parser (service & orchestrate declarations)
   ↓
AST Nodes
   ↓
Domain Models (Service, OrchestrationGroup)
   ↓
Task Execution (orchestrate "group" action)
   ↓
Orchestration Engine
   ├→ Dependency Resolution (Topological Sort)
   ├→ Docker Compose Execution
   ├→ Health Check Monitoring
   └→ Progress Display
```

## Future Enhancements

Planned features for future releases:

- [ ] **Pre/Post Tasks**: Run tasks before/after service lifecycle events
- [ ] **Environment File Management**: Automatic .env file setup
- [ ] **Makefile Integration**: Execute Makefiles during build
- [ ] **Repository Management**: Auto-clone Git repositories
- [ ] **Service Discovery**: DNS and service mesh integration
- [ ] **Monitoring Integration**: Metrics collection and reporting
- [ ] **Rollback Support**: Automatic rollback on deployment failure
- [ ] **Blue/Green Deployments**: Zero-downtime deployments
- [ ] **Scaling Operations**: Dynamic service scaling
- [ ] **Load Balancing**: Automatic load balancer configuration

## Related Documentation

- [Drun v2 Specification](../DRUN_V2_SPECIFICATION.md)
- [Microservices Orchestration Spec](../spec/microservices-orchestration.md)
- [Examples](../examples/)

## Implementation Details & Minutia

This section covers the technical implementation details, edge cases, and minutia that have been addressed in the orchestration system.

### Parser Implementation

#### Infinite Loop Prevention

The orchestration parser includes robust error handling to prevent infinite loops:

- **String Array Parsing**: Added `continue` statements to advance tokens when encountering unexpected types
- **Orchestration Body Parsing**: Proper error handling with token advancement in default cases
- **Service Body Parsing**: Comprehensive error recovery with detailed error messages

```go
// Example: String array parsing with loop prevention
for p.curToken.Type != lexer.RBRACKET && p.curToken.Type != lexer.EOF {
    if p.curToken.Type == lexer.STRING {
        result = append(result, p.curToken.Literal)
    } else {
        p.addError(fmt.Sprintf("expected string in array, got %s", p.curToken.Type))
        // Advance to avoid infinite loop
        p.nextToken()
        continue
    }
    p.nextToken()
    // ... rest of parsing logic
}
```

#### Compose File Syntax Support

The parser supports both syntaxes for Docker Compose file specification:

- **Block syntax**: `compose:` with nested configuration
- **Inline syntax**: `compose file "docker-compose.yml"`

```drun
# Both syntaxes are supported:
service "api" in "./api":
    compose file "docker-compose.dev.yml"  # Inline syntax

service "db" in "./database":
    compose:                               # Block syntax
        file "docker-compose.yml"
        project "myproject"
```

### Docker Compose Integration

#### Working Directory Management

The orchestration system ensures Docker Compose commands run with the correct working directory:

- **Service Directory**: Each service runs `docker compose` from its own directory
- **PWD Environment**: Sets `PWD` environment variable to service directory for `${PWD}` references
- **Absolute Path Resolution**: Converts relative service paths to absolute paths

```go
// PWD environment setup for Docker Compose
env := os.Environ()
for i, envVar := range env {
    if strings.HasPrefix(envVar, "PWD=") {
        env[i] = "PWD=" + servicePath
        break
    }
}
if !strings.Contains(strings.Join(env, " "), "PWD=") {
    env = append(env, "PWD="+servicePath)
}
cmd.Env = env
```

#### BuildKit Output Streaming

Real-time Docker build output is streamed to provide visibility into the build process:

- **Real-time Streaming**: Build output is streamed directly to the user
- **Progress Integration**: Build status is integrated with the progress display
- **Error Handling**: Build failures trigger circuit breaker if enabled

### Circuit Breaker Implementation

#### Smart Rollback Logic

The circuit breaker uses intelligent rollback that only stops dependent services:

- **Dependency-Aware**: Only stops services that were started after the failed service
- **Index-Based**: Uses service order to determine which services to rollback
- **Selective Stopping**: Avoids stopping services that don't depend on the failed service

```go
// Smart rollback - only stop dependent services
failedServiceIndex := -1
for i, svcName := range orderedServices {
    if svcName == serviceName {
        failedServiceIndex = i
        break
    }
}

// Only stop services that were started after the failed service
for i := failedServiceIndex + 1; i < len(orderedServices); i++ {
    // ... rollback logic
}
```

#### Build Failure Handling

Circuit breaker is triggered by both startup failures and build failures:

- **Build Failures**: `docker compose build` failures trigger circuit breaker
- **Startup Failures**: `docker compose up` failures trigger circuit breaker
- **Health Check Failures**: Health check timeouts trigger circuit breaker

### Progress Display System

#### BuildKit-Style Visual Feedback

The progress display provides real-time visual feedback similar to Docker BuildKit:

- **Status Icons**: Different icons for each service state (⏸️ pending, 🔨 building, 🔄 starting, ✅ healthy, ❌ failed)
- **Real-time Updates**: Progress updates are rendered inline without cluttering output
- **Build Output**: Docker build output is streamed and integrated with progress display
- **Summary Display**: Final summary shows success/failure counts

#### Service State Management

Comprehensive state tracking for each service:

- **Thread-Safe**: Uses mutex locks for concurrent access
- **Timing Information**: Tracks start/end times for performance monitoring
- **Error Tracking**: Captures and displays detailed error information
- **Status Transitions**: Clear state transitions (pending → building → starting → healthy)

### Error Handling & Recovery

#### Parser Error Recovery

Robust error recovery mechanisms in the parser:

- **Token Synchronization**: Skips to next valid token on parse errors
- **Detailed Error Messages**: Specific error messages with line/column information
- **Graceful Degradation**: Continues parsing other statements when one fails

#### Docker Compose Error Handling

Comprehensive error handling for Docker operations:

- **Command Execution**: Proper error capture and reporting
- **Output Capture**: Captures both stdout and stderr for debugging
- **Timeout Handling**: Configurable timeouts for long-running operations
- **Exit Code Checking**: Proper exit code validation

### Performance Considerations

#### Parallel vs Sequential Operations

The orchestration system balances safety and performance:

- **Sequential Startup**: Services start one at a time to ensure proper dependency resolution
- **Parallel Health Checks**: Health checks can run concurrently after startup
- **Concurrent Rollback**: Multiple services can be stopped simultaneously during rollback

#### Memory Management

Efficient memory usage patterns:

- **Streaming Output**: Build output is streamed rather than buffered
- **Goroutine Management**: Proper cleanup of background goroutines
- **Resource Cleanup**: Docker containers and networks are properly cleaned up

### Configuration Flexibility

#### Service Configuration

Comprehensive service configuration options:

- **Health Check Types**: HTTP, TCP, Docker, DNS, and custom health checks
- **Build Configuration**: Optional building with timeout and parallel job settings
- **Dependency Management**: Complex dependency graphs with topological sorting
- **Environment Variables**: Service-specific environment variable management

#### Orchestration Strategies

Multiple orchestration strategies supported:

- **Dependency-Based**: Services start in dependency order (default)
- **Sequential**: Services start one after another regardless of dependencies
- **Parallel**: All services start simultaneously (future enhancement)

### Testing & Validation

#### Comprehensive Test Coverage

The implementation includes extensive testing:

- **Parser Tests**: Unit tests for all parsing scenarios
- **Integration Tests**: End-to-end tests with real Docker containers
- **Error Case Tests**: Tests for various failure scenarios
- **Circuit Breaker Tests**: Specific tests for rollback behavior

#### Real-World Validation

The system has been validated with real projects:

- **Celesta Project**: Production validation with complex microservices
- **Docker Compose Integration**: Real Docker Compose file compatibility
- **Health Check Validation**: Actual HTTP/TCP health check testing

## Support

For issues, feature requests, or questions:

- GitHub Issues: [github.com/phillarmonic/drun/issues](https://github.com/phillarmonic/drun/issues)
- Documentation: [github.com/phillarmonic/drun](https://github.com/phillarmonic/drun)
