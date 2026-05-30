# drun Language & xdrun CLI - Comprehensive Instructions

> **For AI/LLM Assistants**: This document provides complete instructions for using the drun semantic automation language and xdrun CLI. Embed this file in any repository where you want AI assistants to understand and write drun automation.

## Overview

**drun** is a semantic, English-like task automation language with intelligent execution, smart detection, and powerful built-in actions. Tasks are written in natural language that executes directly without compilation.

**xdrun** (eXecute drun) is the CLI interpreter that runs `.drun` files.

---

## Quick Reference

### File Location
- **Default**: `.drun/spec.drun`
- **Custom**: Use `xdrun -f path/to/file.drun`

### Running Tasks
```bash
xdrun taskname                           # Run a task
xdrun taskname param1=value param2=value # With parameters (NO dashes!)
xdrun --list                             # List available tasks
xdrun --dry-run taskname                 # Preview without executing
xdrun -f custom.drun taskname            # Use custom file
```

### Essential Syntax
```drun
version: 2.0

task "taskname" means "Description":
  info "Hello world!"
```

---

## Complete Language Reference

### File Structure

Every drun file requires a version declaration at the top:

```drun
version: 2.0

# Optional project declaration
project "project-name" version "1.0.0":
  set some_setting to "value"

# Task definitions
task "taskname" means "Description":
  # statements
```

### Comments

```drun
# Single-line comment

/*
  Multi-line comment
  spanning multiple lines
*/

task "example":  # End-of-line comment
  info "Hello"
```

### Indentation

drun uses Python-style indentation. Both tabs and spaces are supported (1 tab = 4 spaces). Be consistent within a file.

```drun
task "example":
  info "Level 1"
  if true:
    info "Level 2"
```

---

## Parameters

### Parameter Types

| Keyword | Purpose | Required? |
|---------|---------|-----------|
| `requires` | Mandatory parameter | Yes (unless has default) |
| `given` | Optional parameter | No (always has default) |
| `accepts` | List parameter | Optional |

### Basic Parameters

```drun
task "deploy":
  requires $environment                    # Must be provided
  given $replicas defaults to "3"          # Optional with default
  
  info "Deploying to {$environment} with {$replicas} replicas"
```

### Constrained Parameters

```drun
task "deploy":
  # From a list of allowed values
  requires $environment from ["dev", "staging", "production"]
  
  # With type and range
  requires $port as number between 1000 and 9999
  
  # With pattern validation
  requires $version as string matching semver
  requires $email as string matching email format
  requires $custom as string matching pattern "^v[0-9]+$"
  
  # Boolean parameters
  given $force as boolean defaults to "false"
  
  # List parameters
  accepts $features as list of strings
```

### Available Pattern Macros

| Macro | Description | Example |
|-------|-------------|---------|
| `semver` | Semantic versioning | `v1.2.3` |
| `semver_extended` | Extended semver | `v2.0.1-RC2` |
| `uuid` | UUID format | `550e8400-e29b-41d4-a716-446655440000` |
| `url` | HTTP/HTTPS URLs | `https://example.com` |
| `ipv4` | IPv4 addresses | `192.168.1.1` |
| `slug` | URL slugs | `my-project-name` |
| `docker_tag` | Docker image tags | `myapp:latest` |
| `git_branch` | Git branch names | `feature/my-branch` |
| `email format` | Email addresses | `user@example.com` |

---

## Variables

### Declaration

```drun
task "example":
  # Mutable variable
  set $name to "value"
  
  # Immutable binding
  let $const_value = "constant"
  
  # Capture shell output
  capture from shell "hostname" as $hostname
```

### Interpolation

Variables are interpolated using `{$variable}` syntax:

```drun
task "example":
  set $name to "Alice"
  info "Hello {$name}!"
```

### Project Globals

Access project-level settings via `$globals`:

```drun
project "myapp" version "1.0.0":
  set registry to "ghcr.io/company"

task "deploy":
  info "Project: {$globals.project}"     # → "myapp"
  info "Version: {$globals.version}"     # → "1.0.0"
  info "Registry: {$globals.registry}"   # → "ghcr.io/company"
```

### Built-in Functions

```drun
task "example":
  # System information
  info "Host: {hostname}"
  info "Directory: {pwd}"
  info "Directory name: {pwd('basename')}"
  info "Current file: {current file}"
  
  # Environment variables
  info "Home: {env('HOME')}"
  info "Custom: {env('CUSTOM_VAR', 'default')}"
  
  # Date/Time
  info "Date: {now.format('2006-01-02')}"
  info "Time: {now.format('15:04:05')}"
  
  # Git
  info "Commit: {current git commit}"
  info "Short: {current git commit('short')}"
  
  # File checks
  info "Exists: {file exists('path/to/file')}"
  info "Dir exists: {dir exists('path/to/dir')}"
```

### Variable Operations

Powerful transformations using pipe syntax:

```drun
task "example":
  set $version to "v2.1.0-beta"
  set $files to "src/app.js src/utils.js tests/test.js"
  set $path to "/etc/nginx/default.conf"
  
  # String operations
  info "{$version without prefix 'v'}"              # → 2.1.0-beta
  info "{$version without suffix '-beta'}"          # → v2.1.0
  info "{$version split by '.' | first}"            # → v2
  
  # Array operations  
  info "{$files filtered by extension '.js'}"       # → src/app.js src/utils.js tests/test.js
  info "{$files filtered by prefix 'src/'}"         # → src/app.js src/utils.js
  info "{$files sorted by name}"                    # alphabetically sorted
  info "{$files first}"                             # → src/app.js
  info "{$files last}"                              # → tests/test.js
  
  # Path operations
  info "{$path basename}"                           # → default.conf
  info "{$path dirname}"                            # → /etc/nginx
  info "{$path extension}"                          # → conf
  
  # Chaining operations
  info "{$files filtered by prefix 'src/' | filtered by extension '.js' | sorted by name}"
```

---

## Control Flow

### If/Else Statements

```drun
task "example":
  requires $environment from ["dev", "staging", "production"]
  
  if $environment is "production":
    warn "Deploying to production!"
  else if $environment is "staging":
    info "Deploying to staging"
  else:
    info "Deploying to development"
  
  # Negation
  if $environment is not "production":
    info "Not production"
  
  # Empty checks
  if $variable is empty:
    warn "Variable is empty"
  
  if $variable is not empty:
    info "Variable has value: {$variable}"
```

### When/Otherwise Statements

```drun
task "example":
  set $platform to "windows"
  
  when $platform is "windows":
    info "Windows detected"
    step "Using Windows commands"
  otherwise:
    info "Unix-like platform"
    step "Using Unix commands"
```

### For Each Loops

```drun
task "example":
  # Loop over list
  for each $item in ["a", "b", "c"]:
    info "Processing: {$item}"
  
  # Loop over variable
  set $services to "api web worker"
  for each $service in $services:
    info "Service: {$service}"
  
  # Parallel execution
  for each $region in ["us-east", "eu-west"] in parallel:
    info "Deploying to {$region}"
  
  # Range loop
  for $i in range 1 to 5:
    info "Iteration {$i}"
```

### Try/Catch/Finally

```drun
task "example":
  try:
    run "risky-command"
    success "Command succeeded"
  catch:
    warn "Command failed, handling error"
  finally:
    info "Cleanup always runs"
```

### Throw and Ignore

```drun
task "example":
  # Throw custom error
  if $environment is "production":
    if $approval is not "true":
      throw "Production deployment requires approval"
  
  # Ignore errors and continue
  ignore
  info "Continuing after ignore"
```

---

## Status Messages

```drun
task "example":
  step "Starting process"      # 📋 Step indicator
  info "Information message"   # ℹ️  Info
  warn "Warning message"       # ⚠️  Warning  
  error "Error message"        # ❌ Error (doesn't stop execution)
  success "Success message"    # ✅ Success
  fail "Fatal error"           # 💥 Stops execution with error
```

---

## Shell Commands

```drun
task "example":
  # Run shell command
  run "npm install"
  
  # Multi-line commands with line continuation
  run "docker run --rm \
      -v $(pwd):/app \
      -e NODE_ENV=production \
      node:18 npm test"
  
  # Multi-line commands (preserves line breaks)
  run "echo 'Line 1'
echo 'Line 2'
echo 'Line 3'"
  
  # Capture output
  capture from shell "date" as $current_date
  info "Current date: {$current_date}"
```

---

## Task Dependencies

```drun
task "test":
  depends on build
  run "npm test"

task "deploy":
  depends on build and test           # Both must complete
  run "deploy-script"

task "full-pipeline":
  depends on lint, test, build then deploy  # Sequential groups
  success "Pipeline complete"
```

---

## Task Calling

Call tasks from within other tasks:

```drun
task "setup":
  info "Setting up environment"

task "run-tests":
  given $test_type defaults to "unit"
  info "Running {$test_type} tests"

task "full-pipeline":
  # Call without parameters
  call task "setup"
  
  # Call with parameters
  call task "run-tests" with test_type="unit"
  call task "run-tests" with test_type="integration"
  
  success "Pipeline complete"
```

---

## Docker Actions

```drun
task "docker-workflow":
  # Build image
  docker build image "myapp:latest"
  docker build image "myapp:v1.0" from "Dockerfile.prod"
  
  # Tag and push
  docker tag image "myapp:latest" as "registry.io/myapp:latest"
  docker push image "myapp:latest" to "registry.io"
  
  # Container management
  docker run container "myapp-container" from "myapp:latest"
  docker stop container "myapp-container"
  docker remove container "myapp-container"
  docker remove image "myapp:latest"
  
  # Docker Compose
  docker compose up
  docker compose down
  docker compose build
```

---

## HTTP Actions

```drun
task "api-calls":
  # GET request
  get "https://api.example.com/health"
  get "https://api.example.com/users" accept json
  
  # POST request
  post "https://api.example.com/users" content type json with body "name=John&email=john@example.com"
  
  # PUT/PATCH/DELETE
  put "https://api.example.com/users/1" content type json with body "name=Updated"
  patch "https://api.example.com/users/1" content type json with body "status=active"
  delete "https://api.example.com/users/1"
  
  # With authentication
  get "https://api.example.com/data" with auth bearer "{$token}"
  
  # With custom headers
  get "https://api.example.com/data" with header "X-Custom: value"
  
  # With timeout
  get "https://api.example.com/slow" timeout "30s"
```

---

## Network Actions

```drun
task "network-checks":
  # Health checks
  check health of service at "https://api.example.com/health"
  check health of service at "https://api.example.com" timeout "10s"
  check health of service at "https://api.example.com" retry "3"
  
  # Wait for service
  wait for service at "https://api.example.com" to be ready
  wait for service at "https://api.example.com" to be ready timeout "60s"
  
  # Port checks
  check if port 80 is open on "localhost"
  check if port 443 is open on "github.com"
  test connection to "db.example.com" on port 5432 timeout "5s"
  
  # Ping
  ping host "google.com"
  ping host "8.8.8.8" timeout "3s"
```

---

## File Operations

```drun
task "file-ops":
  # Create directory
  create dir "build"
  create directory "dist/assets"
  
  # Copy and move
  copy "src/file.txt" to "dist/file.txt"
  move "temp/file.txt" to "final/file.txt"
  
  # Remove
  remove "build/"
  remove "temp.txt"
  
  # Backup
  backup "config.json" as "config-backup.json"
```

---

## Git Actions

```drun
task "git-workflow":
  # Commit and push
  commit changes with message "Add new feature"
  push to branch "main"
  
  # Tags
  create tag "v1.0.0"
  
  # Branch operations
  checkout branch "feature/new-feature"
```

---

## Smart Detection

```drun
task "detect-tools":
  # Detect available tool variants (DRY pattern)
  detect available "docker compose" or "docker-compose" as $compose_cmd
  detect available "npm" or "yarn" or "pnpm" as $pkg_manager
  
  # Use detected tools
  run "{$compose_cmd} up -d"
  run "{$pkg_manager} install"
```

---

## Project Configuration

```drun
version: 2.0

project "myproject" version "1.0.0":
  # Project-level settings (accessed via $globals)
  set registry to "ghcr.io/company"
  set api_url to "https://api.example.com"
  
  # Project-level parameters (can be overridden at runtime)
  parameter $environment as string from ["dev", "staging", "prod"] defaults to "dev"
  parameter $no_cache as boolean defaults to "false"
  
  # Lifecycle hooks
  before any task:
    info "Starting task in {$globals.project}"
  
  after any task:
    info "Task completed"
  
  # Reusable snippets
  snippet "show-config":
    info "Environment: {$environment}"
    info "Registry: {$globals.registry}"
  
  # Shell configuration (optional)
  shell config:
    linux:
      executable: "/bin/bash"
      args:
        - "-c"
    darwin:
      executable: "/bin/zsh"
      args:
        - "-c"
    windows:
      executable: "powershell.exe"
      args:
        - "-Command"
```

---

## Code Reuse

### Snippets

```drun
project "myapp" version "1.0":
  snippet "header":
    info "═══════════════════════"
    info "  {$globals.project}"
    info "═══════════════════════"

task "build":
  use snippet "header"
  info "Building..."
```

### Task Templates

```drun
# Define template
template task "docker-build":
  given $target defaults to "prod"
  given $tag defaults to "latest"
  
  info "Building {$target} with tag {$tag}"

# Use template
task "build:web":
  call task "docker-build" with target="web" tag="myapp:web"
```

### Remote Includes

```drun
project "myapp":
  # From drunhub standard library
  include from drunhub "ops/docker" as ops
  
  # From GitHub
  include "github:myorg/drun-workflows/docker.drun@v1.0"
  
  # From HTTPS URL  
  include "https://raw.githubusercontent.com/org/repo/main/ci.drun"

task "deploy":
  use snippet "ops.docker-login"
  call task "ops.build"
```

---

## Secrets Management

### In Tasks

```drun
task "setup-secrets":
  # Store secrets
  secret set "api_key" to "secret_value"
  secret set "shared_token" to "team_token" in namespace "global"
  
  # Use secrets in interpolation
  info "API Key: {secret('api_key')}"
  info "With default: {secret('optional_key', 'default_value')}"
  info "From global: {secret('shared_token', '', 'global')}"
  
  # Check and delete
  secret exists "api_key"
  secret list
  secret delete "api_key"
```

### CLI Commands

```bash
# Add secrets
xdrun cmd:secret add api_key              # Prompts for value
xdrun cmd:secret add api_key --masked     # Masked input
xdrun cmd:secret add --global shared_key "value"
xdrun cmd:secret add --project db_pass "value"

# List secrets
xdrun cmd:secret list
xdrun cmd:secret list --global
xdrun cmd:secret list --show-values
xdrun cmd:secret list-all

# Remove secrets
xdrun cmd:secret remove api_key
xdrun cmd:secret remove --global shared_key
```

---

## xdrun CLI Reference

### Task Execution

```bash
xdrun taskname                    # Run task
xdrun taskname param=value        # With parameters (NO dashes!)
xdrun "task with spaces"          # Quoted task names
xdrun --dry-run taskname          # Preview without executing
xdrun --list                      # List available tasks
xdrun -f file.drun taskname       # Custom file
xdrun -v taskname                 # Verbose output
```

### Built-in Commands (cmd: prefix)

```bash
xdrun cmd:completion bash|zsh|fish|powershell  # Shell completion
xdrun cmd:from makefile                         # Convert Makefile
xdrun cmd:secret add|remove|list               # Manage secrets
xdrun cmd:dump-env                             # Show environment variables
xdrun cmd:link services/api                    # Link directories
```

### Management Commands

```bash
xdrun --init                      # Create new .drun/spec.drun
xdrun --init -f custom.drun       # Create custom file
xdrun --self-update               # Update xdrun
xdrun --set-workspace file.drun   # Set workspace default
```

### Debug Options

```bash
xdrun --debug taskname            # Enable debug mode
xdrun --debug --debug-tokens      # Show lexer tokens
xdrun --debug --debug-ast         # Show AST structure
xdrun --debug --debug-json        # Show AST as JSON
xdrun --debug --debug-full        # Full debug output
```

---

## Complete Example

```drun
version: 2.0

project "web-app" version "2.0.0":
  set registry to "ghcr.io/myorg"
  set compose_file to "docker-compose.yml"
  
  parameter $environment from ["dev", "staging", "prod"] defaults to "dev"
  
  snippet "show-env":
    info "Environment: {$environment}"
    info "Project: {$globals.project} v{$globals.version}"

task "default" means "Show available commands":
  info "Available tasks:"
  info "  build    - Build the application"
  info "  test     - Run tests"
  info "  deploy   - Deploy to environment"
  info ""
  info "Usage: xdrun <task> [environment=dev|staging|prod]"

task "build" means "Build Docker images":
  use snippet "show-env"
  
  detect available "docker compose" or "docker-compose" as $compose_cmd
  
  step "Building images"
  run "{$compose_cmd} -f {$globals.compose_file} build"
  
  docker tag image "{$globals.project}:latest" as "{$globals.registry}/{$globals.project}:{$globals.version}"
  
  success "Build complete!"

task "test" means "Run test suite":
  depends on build
  
  step "Running tests"
  run "npm test"
  
  if $environment is "staging":
    step "Running integration tests"
    run "npm run test:integration"
  
  success "All tests passed!"

task "deploy" means "Deploy application":
  depends on build and test
  requires $environment from ["dev", "staging", "prod"]
  given $replicas defaults to "2"
  
  use snippet "show-env"
  
  if $environment is "prod":
    warn "⚠️  Production deployment!"
    info "Replicas: {$replicas}"
  
  step "Pushing images"
  docker push image "{$globals.registry}/{$globals.project}:{$globals.version}"
  
  step "Deploying to {$environment}"
  run "kubectl apply -f k8s/{$environment}/"
  
  check health of service at "https://{$environment}.example.com/health" timeout "60s"
  
  success "Deployment to {$environment} complete!"

task "cleanup" means "Clean up resources":
  step "Removing containers"
  docker compose down
  
  step "Removing images"
  docker remove image "{$globals.project}:latest"
  
  success "Cleanup complete!"
```

**Run this example:**
```bash
xdrun --list                           # List tasks
xdrun build                            # Build for dev
xdrun test environment=staging         # Test for staging
xdrun deploy environment=prod replicas=5  # Deploy to prod
```

---

## Best Practices

1. **Always start with `version: 2.0`**
2. **Use descriptive task names with `means` clauses**
3. **Validate inputs with `requires` constraints**
4. **Use `$globals` for project-wide settings**
5. **Leverage snippets for reusable code blocks**
6. **Use `detect available` for cross-platform compatibility**
7. **Add status messages (`step`, `info`, `success`) for clarity**
8. **Use `--dry-run` to preview before executing**

---

*This document covers drun v2.x syntax. For the most up-to-date information, see the [drun repository](https://github.com/phillarmonic/drun).*

