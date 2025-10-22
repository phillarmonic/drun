# Microservices Orchestration in Drun

## Overview

Drun provides a comprehensive microservices orchestration system that manages the lifecycle of multi-service architectures with Docker Compose integration, health monitoring, dependency resolution, and visual progress feedback.

## Features at a Glance

- ✅ **Service Declarations**: Define services with paths, dependencies, and health checks
- ✅ **Orchestration Groups**: Group services with lifecycle management strategies
- ✅ **BuildKit-Style Progress**: Real-time visual feedback during operations
- ✅ **Health Monitoring**: HTTP, TCP, Docker, DNS, and custom health checks
- ✅ **Dependency Resolution**: Topological sort ensures correct startup/shutdown order
- ✅ **Circuit Breaker**: Stop all services on first failure (optional)
- ✅ **Docker Compose Integration**: Direct `docker compose` command execution
- ✅ **Graceful Shutdown**: Services stopped in reverse dependency order
- ✅ **Error Handling**: Detailed error messages with rollback support

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

task "stop":
    info "🛑 Stopping services..."
    orchestrate "full_stack" stop

task "restart":
    info "♻️ Restarting services..."
    orchestrate "full_stack" restart

task "status":
    info "📊 Service status:"
    orchestrate "full_stack" status

task "down":
    info "🗑️  Removing containers..."
    orchestrate "full_stack" down
```

### Running the Example

```bash
# Start all services with dependency resolution
drun start

# Check status
drun status

# Stop gracefully (reverse order)
drun stop

# Remove all containers
drun down
```

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

| Property | Description | Required |
|----------|-------------|----------|
| `in "path"` | Path to service directory containing docker-compose.yml | Yes |
| `depends on [...]` | List of service dependencies | No |
| `health check` | Health check configuration (see below) | No |

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

| Property | Description | Default |
|----------|-------------|---------|
| `services [...]` | List of services in this group | Required |
| `strategy "..."` | Startup strategy (see below) | "sequential" |
| `circuit true/false` | Enable circuit breaker | false |
| `health_check_interval` | How often to check health | "30s" |

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

| Action | Description |
|--------|-------------|
| `start` | Start all services in dependency order |
| `stop` | Stop all services in reverse order |
| `restart` | Stop then start services |
| `status` | Show status of all services |
| `build` | Build service images |
| `pull` | Pull latest images |
| `down` | Stop and remove containers |

### Action Examples

```drun
task "lifecycle-demo":
    # Start services
    orchestrate "my-stack" start
    
    # Check status
    orchestrate "my-stack" status
    
    # Restart specific services
    orchestrate "my-stack" restart services ["api"]
    
    # Stop everything
    orchestrate "my-stack" stop
```

## BuildKit-Style Progress Display

The orchestration system features a real-time progress display inspired by Docker BuildKit:

### Progress Indicators

Each service shows its current status with visual feedback:

| Icon | Status | Description |
|------|--------|-------------|
| ⏸️ | Pending | Waiting to start |
| 🔄 | Starting | Service is starting |
| ✅ | Healthy | Started and healthy |
| ❌ | Failed | Failed to start |
| 🛑 | Stopping | Being stopped |
| ⏹️ | Stopped | Successfully stopped |

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

## Support

For issues, feature requests, or questions:

- GitHub Issues: [github.com/phillarmonic/drun/issues](https://github.com/phillarmonic/drun/issues)
- Documentation: [github.com/phillarmonic/drun](https://github.com/phillarmonic/drun)

