# Collections and code reuse

## Array Literals and Matrix Execution

drun v2 supports array literals for defining lists of values directly in the code, enabling powerful matrix execution patterns for comprehensive testing and deployment scenarios.

### Array Literal Syntax

Array literals use square bracket notation with comma-separated values:

```
# Basic array literals
["item1", "item2", "item3"]
["linux", "mac", "windows"]
["dev", "staging", "production"]

# Numbers and mixed types
[1, 2, 3, 4, 5]
["port", 8080, "timeout", "30s"]
```

### Project-Level Array Settings

Arrays can be defined at the project level using two syntaxes:

```
project "myapp" version "1.0.0":
  # Simple string settings
  set registry to "ghcr.io/company"
  set api_url to "https://api.example.com"
  
  # Array settings using "as list to" syntax
  set platforms as list to ["linux", "mac", "windows"]
  set environments as list to ["dev", "staging", "production"]
  set node_versions as list to ["16", "18", "20"]
  set databases as list to ["postgres", "mysql", "mongodb"]
```

**Accessing Project Arrays:** Project-level arrays must be accessed using the consistent `$globals.` prefix:

```
# ✅ Correct: Use $globals prefix for consistency
for each $platform in $globals.platforms:
  info "Building for {$platform}"

# ❌ Deprecated: Direct access (will show deprecation warning)
for each $platform in $platforms:
  info "Building for {$platform}"
```

This maintains consistency with other global variable access patterns like `{$globals.project}`, `{$globals.version}`, and `{$globals.registry}`.

### Loop Variables with Array Literals

Loop variables use the `$variable` syntax for consistency with the scoping system. For readability in prose-style code, bare identifiers (`for service in [...]`) are also accepted and automatically normalised to `$service` within the loop body:

```
# Direct array literal in loops
for each platform in ["linux", "mac", "windows"]:
  info "Building for {$platform}"

# Using project-defined arrays
for each $env in $globals.environments:
  for each $service in ["api", "web", "worker"]:
    deploy {$service} to {$env}
```

### Matrix Execution Patterns

Matrix execution allows comprehensive testing across multiple dimensions:

#### Sequential Matrix Execution

```
# Cross-platform builds (OS × Architecture)
for each $platform in $globals.platforms:
  for each $arch in ["amd64", "arm64"]:
    build for {$platform}/{$arch}

# Database testing (Database × Version × Test Suite)
for each $db in $globals.databases:
  for each $version in ["latest", "lts", "stable"]:
    for each $suite in ["unit", "integration", "performance"]:
      test {$db}:{$version} with {$suite} tests
```

#### Parallel Matrix Execution

```
# Multi-region deployment (parallel regions, sequential services)
for each $region in ["us-east", "eu-west", "ap-south"] in parallel:
  for each $service in ["api", "web", "worker"]:
    deploy {$service} to {$region}

# CI/CD pipeline parallelization
for each $job in ["lint", "test", "security-scan", "build"] in parallel:
  when $job is "lint":
    for each $linter in ["eslint", "prettier", "golangci-lint"]:
      run {$linter}
  when $job is "test":
    for each $suite in ["unit", "integration"]:
      run {$suite} tests
```

#### Mixed Parallel/Sequential Execution

```
# Parallel environments, sequential deployment steps
for each $env in ["dev", "staging", "production"] in parallel:
  for each $step in ["build", "test", "deploy", "verify"]:
    execute {$step} in {$env}
```

### Real-World Matrix Use Cases

#### DevOps Scenarios
- **Multi-platform builds**: OS × Architecture × Compiler Version
- **Deployment strategies**: Environment × Service × Region
- **Testing matrices**: Browser × Device × Test Suite
- **Performance testing**: Load Level × Endpoint × Configuration

#### CI/CD Pipelines
- **Parallel job execution**: Lint, Test, Security, Build
- **Multi-environment deployment**: Dev, Staging, Production
- **Canary deployments**: Service × Traffic Percentage
- **Integration testing**: Service × Database × Version

#### Infrastructure Management
- **Multi-cloud deployment**: Provider × Region × Service
- **Monitoring setup**: Service × Metric × Alert Rule
- **Security scanning**: Tool × Target × Severity Level
- **Backup strategies**: Database × Schedule × Retention Policy

### Variable Scoping in Matrix Execution

Loop variables follow the established scoping rules:

```
project "matrix-demo":
  set platforms as list to ["linux", "mac", "windows"]
  set registry to "ghcr.io/company"

task "matrix-build":
  # Project arrays accessed via $globals
  for each $platform in $globals.platforms:
    # Loop variable uses $variable syntax
    for each $arch in ["amd64", "arm64"]:
      # Both loop variables available in nested scope
      info "Building {$globals.registry}/app:{$platform}-{$arch}"
      build for {$platform}/{$arch}
```

This matrix execution system enables comprehensive automation workflows while maintaining drun's natural language philosophy and clear variable scoping.

---

## Code Reuse Features

drun v2 provides powerful mechanisms for code reuse and eliminating duplication in automation workflows. These features enable you to write DRY (Don't Repeat Yourself) task definitions while maintaining readability and maintainability.

### Project-Level Parameters

Project-level parameters are defined once at the project level and shared across all tasks. They can be overridden via CLI, making them perfect for global configuration values.

#### Syntax

```drun
project "my-app" version "1.0.0":
  parameter $name as type [from [values]] defaults to "value"
```

#### Examples

```drun
version: 2.0

project "docker-automation" version "1.0.0":
  # Boolean parameter with default
  parameter $no_cache as boolean defaults to "false"
  
  # String parameter with constraint list
  parameter $environment as string from ["dev", "staging", "prod"] defaults to "dev"
  
  # String parameter with pattern validation
  parameter $registry as string defaults to "docker.io"
  
  # Number parameter with range
  parameter $timeout as number defaults to 300

task "build" means "Build with project-level configuration":
  info "Environment: {$environment}"
  info "Registry: {$registry}"
  info "No cache: {$no_cache}"
  info "Timeout: {$timeout}s"
```

#### Usage

```bash
# Use defaults
xdrun build

# Override parameters via CLI
xdrun build environment=prod no_cache=true registry=gcr.io
```

#### Key Features

- **Shared Configuration**: Define once, use everywhere
- **Type Safety**: Full type validation and constraints
- **CLI Overrides**: Can be overridden at runtime
- **Default Values**: Always have sensible defaults
- **Validation**: Same validation rules as task parameters

### Snippets

Snippets are reusable blocks of statements that can be included in any task. They're perfect for common sequences of actions that appear across multiple tasks.

#### Syntax

```drun
project "my-app" version "1.0.0":
  snippet "name":
    # Statements that can be reused
    statement1
    statement2
```

#### Examples

```drun
version: 2.0

project "my-app" version "1.0.0":
  # Common logging snippet
  snippet "log-start":
    info "═══════════════════════════════════"
    info "  Starting task execution"
    info "═══════════════════════════════════"
  
  # Environment check snippet
  snippet "check-env":
    if env DOCKER_HOST exists:
      info "Docker: Remote host at ${DOCKER_HOST}"
    else:
      info "Docker: Local daemon"
  
  # Cleanup snippet
  snippet "cleanup-temp":
    info "Cleaning up temporary files..."
    info "Done"

task "build" means "Build application":
  use snippet "log-start"
  use snippet "check-env"
  
  info "Building application..."
  # Build logic here
  
  use snippet "cleanup-temp"
  success "Build complete"

task "deploy" means "Deploy application":
  use snippet "log-start"
  use snippet "check-env"
  
  info "Deploying application..."
  # Deploy logic here
  
  success "Deploy complete"
```

#### Key Features

- **Reusability**: Define once, use multiple times
- **Scoped Access**: Snippets can access project parameters and task variables
- **Variable Interpolation**: Full support for variable interpolation
- **Control Flow**: Can contain any valid drun statements including conditionals

### Task Templates

Task templates allow you to define parameterized task structures that can be called like functions. They're perfect for tasks that follow the same pattern but with different parameters.

#### Syntax

```drun
template task "name":
  given $param defaults to "value"
  # Template body
```

#### Examples

```drun
version: 2.0

project "docker-builds" version "1.0.0":
  parameter $no_cache as boolean defaults to "false"
  parameter $registry as string defaults to "docker.io"
  
  snippet "show-config":
    info "Registry: {$registry}"
    info "Cache: {$no_cache ? 'disabled' : 'enabled'}"

# Define a reusable template
template task "docker-build":
  given $target defaults to "prod"
  given $tag defaults to "latest"
  given $platform defaults to "linux/amd64"
  
  step "Building Docker image"
  use snippet "show-config"
  
  info "Target: {$target}"
  info "Tag: {$registry}/{$tag}"
  info "Platform: {$platform}"
  info "Building: docker build {$no_cache ? '--no-cache' : ''} --target={$target} --platform={$platform} -t {$registry}/{$tag} ."
  
  success "Built {$tag}"

# Use the template with different parameters
task "build:web" means "Build web application":
  call task "docker-build" with target="web" tag="myapp:web"

task "build:api" means "Build API server":
  call task "docker-build" with target="api" tag="myapp:api"

task "build:worker" means "Build background worker":
  call task "docker-build" with target="worker" tag="myapp:worker" platform="linux/arm64"

# Use the template with all defaults
task "build:base" means "Build base image":
  call task "docker-build" with target="base"

# Complex task that calls template multiple times
task "build:all" means "Build all images":
  info "Building complete application stack..."
  
  call task "build:web"
  call task "build:api"
  call task "build:worker"
  call task "build:base"
  
  success "All images built successfully!"
```

#### Calling Templates

```bash
# Call regular tasks (which may call templates internally)
xdrun build:web

# Can override project parameters too
xdrun build:all no_cache=true registry=ghcr.io
```

#### Key Features

- **Parameterization**: Accept parameters with defaults
- **Reusability**: Call the same template with different parameters
- **Composition**: Templates can call other tasks and use snippets
- **Type Safety**: Template parameters support all standard validations
- **Variable Access**: Templates have access to project parameters

### Complete Example: Docker Build System

Here's a comprehensive example combining all code reuse features:

```drun
version: 2.0

project "microservices" version "1.0.0":
  # Global configuration
  parameter $no_cache as boolean defaults to "false"
  parameter $environment as string from ["dev", "staging", "prod"] defaults to "dev"
  parameter $registry as string defaults to "docker.io"
  parameter $push as boolean defaults to "false"
  
  # Reusable configuration display
  snippet "show-build-config":
    info "╔════════════════════════════════╗"
    info "║     Build Configuration        ║"
    info "╚════════════════════════════════╝"
    info "Environment: {$environment}"
    info "Registry: {$registry}"
    info "Cache: {$no_cache ? 'disabled' : 'enabled'}"
    info "Push: {$push ? 'yes' : 'no'}"
    info ""
  
  # Reusable Docker login check
  snippet "check-registry-auth":
    if $push is true:
      info "Checking registry authentication..."
      if env DOCKER_AUTH exists:
        info "✓ Registry authentication configured"
      else:
        warn "⚠ No registry authentication found"
  
  # Cleanup snippet
  snippet "cleanup":
    info "Cleaning up build artifacts..."
    info "Done"

# Template for Docker builds
template task "docker-build":
  given $service defaults to "app"
  given $target defaults to "prod"
  given $tag defaults to "latest"
  
  step "Building {$service} image"
  use snippet "show-build-config"
  use snippet "check-registry-auth"
  
  info "Service: {$service}"
  info "Target: {$target}"
  info "Full tag: {$registry}/{$service}:{$tag}"
  
  info "Building image..."
  # Actual Docker build would go here
  
  if $push is true:
    info "Pushing to registry..."
    # Actual Docker push would go here
  
  success "✓ Built {$service}:{$tag}"

# Template for testing services
template task "test-service":
  given $service defaults to "app"
  given $test_suite defaults to "all"
  
  step "Testing {$service}"
  info "Test suite: {$test_suite}"
  info "Running tests..."
  # Actual test commands would go here
  success "✓ Tests passed for {$service}"

# Concrete tasks using templates
task "build:frontend" means "Build frontend service":
  call task "docker-build" with service="frontend" target="web" tag="v1.0.0"
  use snippet "cleanup"

task "build:backend" means "Build backend API":
  call task "docker-build" with service="backend" target="api" tag="v1.0.0"
  use snippet "cleanup"

task "build:worker" means "Build background worker":
  call task "docker-build" with service="worker" target="worker" tag="v1.0.0"
  use snippet "cleanup"

task "test:frontend" means "Test frontend":
  call task "test-service" with service="frontend" test_suite="e2e"

task "test:backend" means "Test backend":
  call task "test-service" with service="backend" test_suite="integration"

# Orchestration tasks
task "build:all" means "Build all services":
  info "═══════════════════════════════════════"
  info "  Building Complete Microservices Stack"
  info "═══════════════════════════════════════"
  info ""
  
  call task "build:frontend"
  call task "build:backend"
  call task "build:worker"
  
  success "✨ All services built successfully!"

task "test:all" means "Test all services":
  call task "test:frontend"
  call task "test:backend"
  success "✨ All tests passed!"

task "ci" means "Complete CI pipeline":
  call task "build:all"
  call task "test:all"
  success "✨ CI pipeline completed!"
```

#### Usage Examples

```bash
# Build individual services
xdrun build:frontend
xdrun build:backend

# Build with custom parameters
xdrun build:frontend no_cache=true environment=prod push=true

# Build everything
xdrun build:all

# Test services
xdrun test:frontend
xdrun test:all

# Complete CI pipeline
xdrun ci environment=staging registry=gcr.io
```

### Benefits

The code reuse features provide several key benefits:

1. **DRY Principle**: Eliminate duplication across tasks
2. **Maintainability**: Update logic in one place
3. **Consistency**: Ensure consistent behavior across tasks
4. **Readability**: Templates and snippets have clear, semantic names
5. **Flexibility**: Override parameters as needed
6. **Type Safety**: Full validation on all parameters
7. **Composition**: Combine features for powerful workflows

### Namespaced Includes

Namespaced includes allow you to import snippets, templates, and tasks from external `.drun` files, enabling true code sharing across projects. Each included file gets its own namespace (derived from its project name) to prevent naming collisions.

#### Basic Syntax

```drun
project "myapp":
    # Include everything from a file
    include "shared/docker.drun"
    
    # Selective includes
    include snippets from "shared/utils.drun"
    include templates from "shared/k8s.drun"
    include tasks from "shared/common.drun"
    
    # Multiple selectors
    include snippets, templates from "shared/helpers.drun"
```

#### Namespace Resolution

The namespace is automatically derived from the `project` declaration in the included file:

```drun
# shared/docker.drun
project "docker":
    snippet "login-check":
        if env DOCKER_AUTH exists:
            info "✓ Docker authenticated"
        else:
            warn "⚠ No Docker authentication"
    
    template task "build":
        given $image defaults to "app:latest"
        info "Building {$image}..."

# main.drun
project "myapp":
    include "shared/docker.drun"

task "deploy":
    use snippet "docker.login-check"    # namespace.element
    call task "docker.build"             # namespace.task
```

#### Transitive Resolution

When an included element references another element from the same file, it's automatically resolved within that namespace:

```drun
# shared/docker.drun
project "docker":
    snippet "login-check":
        info "Checking auth..."
    
    template task "push":
        given $image
        use snippet "login-check"    # No namespace needed within same file
        info "Pushing {$image}..."

# main.drun
project "myapp":
    include "shared/docker.drun"

task "deploy":
    call task "docker.push" with image="myapp:v1"
    # ✓ Works! docker.push automatically finds docker.login-check
```

#### Path Resolution

Include paths are resolved in the following order:

1. **Relative to current file**: `../shared/docker.drun` 
2. **Relative to workspace root**: `shared/docker.drun`
3. **Absolute path**: `/absolute/path/docker.drun`

#### Circular Include Detection

drun automatically detects and prevents circular includes:

```drun
# main.drun includes docker.drun
# docker.drun includes utils.drun
# utils.drun includes docker.drun  ← Circular! Will be skipped
```

#### Complete Example

```drun
# shared/docker.drun
version: 2.0

project "docker":
    parameter $registry as string defaults to "docker.io"
    
    snippet "login-check":
        if env DOCKER_AUTH exists:
            info "✓ Authenticated with {$registry}"
        else:
            warn "⚠ No authentication for {$registry}"
    
    snippet "cleanup":
        info "Cleaning up Docker resources..."
    
    template task "build":
        given $target defaults to "prod"
        given $image defaults to "app:latest"
        
        step "Building Docker image"
        use snippet "login-check"
        info "docker build --target={$target} -t {$image} ."
        use snippet "cleanup"
        success "Built {$image}"
    
    template task "push":
        given $image defaults to "app:latest"
        
        step "Pushing to registry"
        use snippet "login-check"
        info "docker push {$registry}/{$image}"
        success "Pushed {$image}"

# main.drun
version: 2.0

project "myapp":
    include "shared/docker.drun"
    
    parameter $version as string defaults to "1.0.0"

task "build:web":
    call task "docker.build" with target="web" image="myapp:web-{$version}"

task "build:api":
    call task "docker.build" with target="api" image="myapp:api-{$version}"

task "deploy":
    call task "build:web"
    call task "build:api"
    call task "docker.push" with image="myapp:web-{$version}"
    call task "docker.push" with image="myapp:api-{$version}"
```

#### Key Features

- **Namespace Safety**: No naming collisions between included files
- **Dot Notation**: Clean, familiar syntax for referencing included elements
- **Selective Imports**: Import only what you need (`snippets`, `templates`, `tasks`)
- **Transitive Resolution**: Included elements automatically resolve their dependencies
- **Path Flexibility**: Relative, workspace, and absolute path support
- **Circular Detection**: Automatic prevention of circular includes
- **Verbose Logging**: Use `-v` flag to see what's being included

#### Benefits

1. **Code Sharing**: Share common workflows across multiple projects
2. **Library Pattern**: Create reusable "library" files for different domains (docker, k8s, git, etc.)
3. **Team Standards**: Enforce consistent patterns across team projects
4. **DRY at Scale**: Eliminate duplication not just within a project, but across all projects
5. **Maintainability**: Update shared logic once, affects all users
6. **Namespace Safety**: Clear ownership and no conflicts

### Remote Includes

Remote includes extend the include system to fetch `.drun` files from external sources like GitHub repositories and HTTPS URLs. This enables sharing workflows across teams, organizations, and the entire community.

#### GitHub Includes

Include files directly from GitHub repositories using the `github:` protocol:

```drun
project "myapp":
    # Include from GitHub with auto branch detection
    include "github:owner/repo/path/to/file.drun"
    
    # Include from specific branch
    include "github:owner/repo/path/to/file.drun@main"
    
    # Include from specific tag
    include "github:owner/repo/path/to/file.drun@v1.0.0"
    
    # Include from specific commit
    include "github:owner/repo/path/to/file.drun@abc123"
```

**Smart Default Branch Detection**: If no branch/ref is specified, drun automatically detects the repository's default branch (`main` or `master`).

#### HTTPS Includes

Include files from any HTTPS URL:

```drun
project "myapp":
    # Include from raw GitHub URL
    include "https://raw.githubusercontent.com/owner/repo/main/shared/workflow.drun"
    
    # Include from any HTTPS source
    include "https://example.com/shared/tasks.drun"
```

#### Drunhub Standard Library

Drunhub is the official standard library repository at `https://github.com/phillarmonic/drun-hub` containing reusable templates, snippets, and tasks organized by category. Import from drunhub using the `drunhub:` protocol:

```drun
project "myapp":
    # Import from drunhub - uses project name as namespace
    include from drunhub "ops/docker"
    
    # Import with custom namespace (overrides project name)
    include from drunhub "ops/kubernetes" as k8s
    
    # Import from nested folders
    include from drunhub "utils/logging/advanced" as log
    
    # Import from specific branch/tag
    include from drunhub "ops/docker@v1.0" as ops
```

**Key Features**:

- **Automatic `.drun` extension**: No need to add `.drun` extension
- **Custom namespaces**: Override default project names with `as` clause
- **Folder protection**: Certain folders like `docs` and `.github` are blocked for security
- **Same caching**: Uses the same smart caching as other remote includes

**Example Usage**:

```drun
version: 2.0

project "deploy-app":
    # Import Docker utilities as "ops" namespace
    include from drunhub "ops/docker" as ops
    
    # Import Kubernetes helpers
    include from drunhub "ops/kubernetes" as k8s

task "deploy":
    # Use snippet from ops namespace
    use snippet "ops.check-docker"
    
    # Call task from k8s namespace
    call task "k8s.deploy" with namespace="production" replicas=3
    
    success "✓ Deployed successfully!"
```

**Custom Namespaces with Traditional Includes**:

The `as` clause also works with regular includes:

```drun
project "myapp":
    # Override namespace from included file
    include "shared/docker-utils.drun" as docker
    
    # Now use docker.* instead of the original project name
    use snippet "docker.build"
```

#### Smart Caching

Remote includes are automatically cached to `~/.drun/cache.solo` with:

- **1-minute expiration** by default
- **Automatic refresh** when cache expires
- **Stale cache fallback** for offline resilience (if network fails, uses expired cache)
- **Content-based keys** (hash of URL + ref)

**Disable caching** when needed:

```bash
# Bypass cache and always fetch fresh
xdrun --no-drun-cache -f myfile.drun mytask
```

#### Example: Community Workflows

```drun
# my-project.drun
version: 2.0

project "my-awesome-app":
    # Include Docker utilities from your organization
    include "github:myorg/drun-workflows/docker.drun@v1.2.0"
    
    # Include Kubernetes helpers from community
    include "github:drun-community/k8s-workflows/deployment.drun"
    
    # Include CI/CD patterns from team repo
    include "https://raw.githubusercontent.com/myteam/workflows/main/ci.drun"

task "deploy":
    # Use included snippets and templates
    use snippet "docker.security-scan"
    call task "k8s.deploy" with namespace="production"
    call task "ci.notify-slack" with message="Deployed!"
```

#### Authentication

For private repositories, set a GitHub token:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
xdrun -f myfile.drun deploy
```

#### Benefits

1. **Community Sharing**: Leverage workflows from the broader drun community
2. **Organization Libraries**: Share standardized workflows across your organization
3. **Version Control**: Pin to specific tags/commits for reproducibility
4. **Offline Resilience**: Stale cache fallback ensures workflows work offline
5. **Performance**: Smart caching reduces network requests
6. **Flexibility**: Works with both GitHub and any HTTPS source

### Best Practices

1. **Use Project Parameters for Global Config**: Things like registry URLs, environment, cache settings
2. **Use Snippets for Common Sequences**: Logging, cleanup, environment checks
3. **Use Templates for Repeated Patterns**: Build tasks, test tasks, deployment tasks
4. **Use Includes for Cross-Project Sharing**: Create shared library files for common domains (Docker, Kubernetes, Git)
5. **Meaningful Names**: Choose descriptive names for snippets and templates
6. **Namespace Organization**: Group related snippets/templates in a single file with a clear project name
7. **Selective Imports**: Use `include snippets from` when you only need specific types of elements
8. **Documentation**: Add comments explaining what each reusable component does
9. **Defaults**: Always provide sensible defaults for template parameters
10. **Library Structure**: Organize shared files in a `shared/` directory for clarity

---

