# drun: LLM Usage Manual

**A comprehensive guide for Large Language Models to understand and generate drun automation tasks**

## Table of Contents

1. [Overview](#overview)
2. [Getting Started](#getting-started)
3. [Core Concepts](#core-concepts)
4. [Syntax Reference](#syntax-reference)
5. [Task Patterns](#task-patterns)
6. [Built-in Functions](#built-in-functions)
7. [Variable Operations](#variable-operations)
8. [Control Flow](#control-flow)
9. [Common Use Cases](#common-use-cases)
10. [Best Practices](#best-practices)
11. [Error Patterns](#error-patterns)

---

## Overview

**drun** is a semantic, English-like task automation language that compiles to efficient shell commands. It's designed for readability, maintainability, and intelligent execution.

### Key Characteristics
- **Natural Language Syntax**: Write tasks in English-like sentences
- **Type-Safe Parameters**: Strong parameter validation with constraints
- **Smart Detection**: Auto-detect tools, frameworks, and environments
- **Variable System**: Powerful interpolation with `$variable` syntax
- **Built-in Actions**: Docker, Kubernetes, Git, HTTP operations
- **Cross-Platform**: Works on Linux, macOS, and Windows

### File Structure
```
project/
├── .drun/
│   └── spec.drun          # Default task file
├── my-tasks.drun          # Custom task file
└── .drun/.drun_workspace  # Workspace configuration
```

### Project Initialization
To start a new drun project, use the `--init` command:

```bash
# Initialize with default .drun/spec.drun file
drun --init

# Initialize with custom file name
drun --init --file=my-project.drun

# Initialize and set as workspace default
drun --init --file=my-project.drun --save-as-default

# Set existing file as workspace default
drun --set-workspace my-project.drun
```

This creates a starter task file with basic examples and project structure.

---

## Getting Started

### 1. Initialize a New Project

Start by creating a new drun project in your directory:

```bash
# Create default task file at .drun/spec.drun
drun --init

# Or create with custom name
drun --init --file=my-tasks.drun

# Initialize and set as workspace default
drun --init --file=my-tasks.drun --save-as-default
```

### 2. Basic Project Structure

After initialization, you'll have a starter file like this:

```drun
project "my-app" version "1.0"

task "hello" means "Say hello":
  info "Hello from drun v2! 🚀"

task "greet" means "Greet someone by name":
  requires $name
  given $title defaults to "friend"
  
  info "Hello, {$title} {$name}!"
```

### 3. List Available Tasks

See what tasks are available:

```bash
drun --list
# or
drun -l
```

### 4. Run Tasks

Execute tasks with parameters:

```bash
# Simple task
drun hello

# Task with parameters
drun greet name=Alice
drun greet name=Bob title=Mr.

# Task with multiple parameters
drun deploy environment=staging version=v1.2.0
```

### 5. Dry Run and Debugging

Test what would be executed without running:

```bash
# See what would be executed
drun deploy --dry-run

# Show detailed execution plan
drun deploy --explain

# Debug parsing and AST
drun --debug -f my-tasks.drun
```

### 6. Workspace Management

Manage your drun workspace:

```bash
# Move task file and update workspace
mv .drun/spec.drun ./project-tasks.drun
drun --set-workspace project-tasks.drun

# Now drun automatically uses your custom location
drun --list
```

### 7. Example Workflow

Here's a typical workflow for creating a new automation project:

```bash
# 1. Initialize project
cd my-project
drun --init --file=automation.drun

# 2. Edit the file to add your tasks
# automation.drun:
```

```drun
project "my-project" version "1.0":
  set registry to "ghcr.io/mycompany"
  set environments as list to ["dev", "staging", "production"]

task "build" means "Build Docker image":
  given $tag defaults to "{current git commit}"
  
  step "Building Docker image"
  build docker image "{$globals.registry}/my-project:{$tag}"
  success "Built image: {$globals.registry}/my-project:{$tag}"

task "deploy" means "Deploy to environment":
  requires $environment from $globals.environments
  depends on build
  
  step "Deploying to {$environment}"
  deploy my-project:latest to kubernetes namespace {$environment}
  wait for rollout to complete
  success "Deployed to {$environment}"
```

```bash
# 3. List available tasks
drun --list

# 4. Test with dry run
drun deploy environment=dev --dry-run

# 5. Execute tasks
drun build
drun deploy environment=dev
```

---

## Core Concepts

### 1. Project Declaration
Every drun file starts with a project declaration:

```drun
project "my-app" version "1.0":
  set registry to "ghcr.io/company"
  set environments as list to ["dev", "staging", "production"]
```

### 2. Task Definition
Tasks are the fundamental unit of work:

```drun
task "deploy" means "Deploy application to environment":
  requires $environment from ["dev", "staging", "production"]
  given $replicas defaults to 3
  depends on build
  
  deploy myapp:latest to kubernetes namespace {$environment}
```

### 3. Variable Scoping
- **Project variables**: Accessed via `$globals.variable_name`
- **Task variables**: Declared with `$` prefix, accessed as `{$variable}`
- **Loop variables**: Use `$` prefix for consistency

### 4. Parameter Types
- **Required**: `requires $name`
- **Optional**: `given $name defaults to "value"`
- **Constrained**: `requires $env from ["dev", "prod"]`
- **Typed**: `requires $port as number between 1000 and 9999`
- **Lists**: `accepts $items as list of strings`

---

## Syntax Reference

### Basic Task Structure
```drun
task "task-name" [means "description"]:
  [parameters]
  [dependencies]
  [variables]
  [statements]
```

### Parameter Declaration
```drun
# Required parameters
requires $environment from ["dev", "staging", "production"]
requires $version matching pattern "v\d+\.\d+\.\d+"
requires $port as number between 1000 and 9999
requires $email matching email format

# Optional parameters
given $replicas defaults to 3
given $timeout defaults to "5m"
given $force defaults to false
given $features as list defaults to empty

# Variadic parameters
accepts $files as list of strings
accepts $configs as list
```

### Variable Declaration
```drun
# Simple assignment
let $name = "value"
set $counter to 0

# Capture from expressions
capture $start_time from now
capture $branch from current git branch

# Capture from shell commands
capture from shell "docker --version" as $docker_version
capture from shell as $build_info:
  echo "Build: $(date)"
  echo "User: $(whoami)"
  echo "Commit: $(git rev-parse HEAD)"

# Conditional assignment
let $config = when $environment is "production": prod_config else: dev_config
```

### Task Calling Syntax
```drun
# Basic task call
call task "task-name"

# Task call with parameters
call task "task-name" with param1="value1" param2="value2"

# Examples
call task "setup-environment"
call task "run-tests" with test_type="unit"
call task "deploy" with environment="production" replicas="3"
```

### Control Flow
```drun
# If statements
if $environment is "production":
  require manual approval
else if $environment is "staging":
  run integration tests
else:
  skip validation

# When statements (individual conditions)
when $package_manager is "npm":
  run "npm ci && npm run build"
when $package_manager is "yarn":
  run "yarn install && yarn build"
when $package_manager is "pnpm":
  run "pnpm install && pnpm build"

# When-otherwise (simplified conditional)
when $platform is "windows":
  run "build.bat"
otherwise:
  run "./build.sh"

# Loops
for each $service in ["api", "web", "worker"]:
  deploy {$service} to {$environment}

# Parallel loops
for each $region in ["us-east", "eu-west"] in parallel:
  deploy to {$region}

# Matrix execution
for each $os in ["ubuntu", "alpine"]:
  for each $version in ["16", "18", "20"]:
    test on {$os} with node {$version}

# Error handling
try:
  deploy to production
catch deployment_error:
  rollback deployment
  notify team
finally:
  cleanup resources
```

---

## Task Patterns

### 1. Simple Task
```drun
task "hello":
  info "Hello, World!"
```

### 2. Parameterized Task
```drun
task "greet" means "Greet someone by name":
  requires $name
  given $title defaults to "friend"
  
  info "Hello, {$title} {$name}!"
```

### 3. Task with Dependencies
```drun
task "deploy" means "Deploy application":
  requires $environment from ["dev", "staging", "production"]
  depends on build and test
  
  deploy myapp:latest to kubernetes namespace {$environment}
```

### 4. Complex Workflow
```drun
task "ci-pipeline" means "Complete CI/CD pipeline":
  given $skip_tests defaults to false
  
  step "Starting CI/CD pipeline"
  
  if not $skip_tests:
    run "lint and test" in parallel
  
  run "build docker image"
  run "security scan"
  run "deploy to staging"
  run "integration tests"
  
  if environment is "production":
    require manual approval "Deploy to production?"
    run "deploy to production"
    run "smoke tests"
  
  success "Pipeline completed successfully"
```

### 5. Task Calling Pattern
```drun
task "setup-environment":
  info "Setting up development environment"
  info "Installing dependencies..."

task "run-tests":
  given $test_type defaults to "unit"
  info "Running {$test_type} tests"
  info "All tests passed!"

task "build-application":
  given $target defaults to "production"
  info "Building application for {$target}"
  info "Build completed successfully"

task "full-pipeline":
  info "Starting full CI/CD pipeline"
  
  # Call tasks without parameters
  call task "setup-environment"
  
  # Call tasks with parameters
  call task "run-tests" with test_type="unit"
  call task "run-tests" with test_type="integration"
  call task "build-application" with target="production"
  
  success "Full pipeline completed successfully!"
```

### 6. Matrix Execution
```drun
task "cross-platform-build" means "Build for multiple platforms":
  for each $os in ["linux", "darwin", "windows"]:
    for each $arch in ["amd64", "arm64"]:
      step "Building for {$os}/{$arch}"
      run "GOOS={$os} GOARCH={$arch} go build -o bin/app-{$os}-{$arch}"
```

---

## Built-in Functions

### System Information
```drun
{hostname}                           # System hostname
{pwd}                               # Current working directory
{pwd('basename')}                   # Directory name only
{current file}                      # Path to current drun file
{env('VAR_NAME')}                   # Environment variable
{env('VAR_NAME', 'default')}        # Environment variable with default
```

### Time & Date
```drun
{now.format('2006-01-02 15:04:05')} # Formatted current time
{now.format('Monday, January 2')}   # Custom date format
```

### Git Integration
```drun
{current git commit}                # Full commit hash
{current git commit('short')}       # Short commit hash
{current git branch}                # Current branch name

# With pipe operations
{current git branch | replace "/" by "-"}           # Safe branch name
{current git branch | replace '/' by '-' | lowercase} # Docker-safe tag
```

### File System
```drun
{file exists('path/to/file')}       # Returns "true"/"false"
{dir exists('path/to/dir')}         # Returns "true"/"false"
```

### Progress & Timing
```drun
{start progress('Building application')}
{update progress('50', 'Compiling sources')}
{finish progress('Build completed!')}
{start timer('build_time')}
{stop timer('build_time')}
{show elapsed time('build_time')}
```

---

## Variable Operations

**⚠️ Implementation Status**: Variable operations are currently in development. Basic variable interpolation works, but advanced operations like `without prefix`, `filtered by`, etc. may display as literal text rather than being processed.

### String Operations
```drun
set $version to "v2.1.0-beta"
set $filename to "my-app.tar.gz"

# Remove prefix/suffix (Note: May not work in current implementation)
info "Clean version: {$version without prefix 'v' | without suffix '-beta'}"
info "App name: {$filename without suffix '.tar.gz'}"

# Split strings (Note: May not work in current implementation)
set $docker_image to "nginx:1.21"
info "Image name: {$docker_image split by ':' | first}"
info "Version: {$docker_image split by ':' | last}"
```

### Array Operations
```drun
set $files to "app.js test.js config.json readme.md"

# Filtering
info "JS files: {$files filtered by extension '.js'}"
info "Config files: {$files filtered by extension '.json'}"
info "Source files: {$files filtered by prefix 'src/'}"

# Sorting and manipulation
info "Sorted files: {$files sorted by name}"
info "Reversed: {$files reversed}"
info "First file: {$files first}"
info "Last file: {$files last}"
info "Unique items: {$files unique}"
```

### Path Operations
```drun
set $config_file to "/etc/nginx/sites-available/default.conf"

info "Filename: {$config_file basename}"           # default.conf
info "Directory: {$config_file dirname}"          # /etc/nginx/sites-available
info "Extension: {$config_file extension}"        # conf
info "Name only: {$config_file basename | without suffix '.conf'}" # default
```

### Operation Chaining
```drun
set $project_files to "src/app.js src/utils.js tests/app.test.js docs/readme.md"

# Complex chaining
info "Source JS files: {$project_files filtered by prefix 'src/' | filtered by extension '.js' | sorted by name}"

# Loop integration
for each img in $docker_images:
  info "Processing: {img split by ':' | first}"
```

---

## Control Flow

### Conditional Logic
```drun
# Basic conditions
if $environment is "production":
  enable monitoring
else:
  disable monitoring

# Compound conditions
if $environment is "production" and $force is not true:
  require manual approval

# Empty checks
if $features is empty:
  warn "No features specified"

if $features is not empty:
  info "Features: {$features}"

# Directory empty checks
if folder "build" is empty:
  info "Build directory is clean"

if directory "/tmp/cache" is not empty:
  run "rm -rf /tmp/cache"
```

### Smart Detection
```drun
# Tool availability
if docker is available:
  build container
else:
  error "Docker is required"

# Framework detection
when symfony is detected:
  run symfony console commands

when laravel is detected:
  run artisan commands

# Environment detection
when running in CI:
  use non-interactive mode

when running locally:
  enable development features
```

### Loop Patterns
```drun
# Simple iteration
for each $item in ["a", "b", "c"]:
  process {$item}

# Parallel execution
for each $region in ["us-east", "eu-west"] in parallel:
  deploy to {$region}

# Matrix execution
for each $os in ["ubuntu", "alpine"]:
  for each $version in ["16", "18", "20"]:
    test on {$os} with node {$version}

# With project arrays
for each $env in $globals.environments:
  deploy to {$env}
```

---

## Common Use Cases

### 1. Docker Workflows
```drun
task "docker-build" means "Build and push Docker image":
  given $tag defaults to "{current git commit}"
  given $push defaults to false
  
  step "Building Docker image"
  build docker image "myapp:{$tag}"
  
  if $push is true:
    step "Pushing to registry"
    push image "myapp:{$tag}" to "{$globals.registry}"
  
  success "Docker build completed: myapp:{$tag}"
```

### 2. Kubernetes Deployment
```drun
task "k8s-deploy" means "Deploy to Kubernetes":
  requires $environment from ["dev", "staging", "production"]
  given $replicas defaults to 3
  
  when $environment is "production":
    require manual approval "Deploy to production?"
    set $replicas to 5
  
  deploy myapp:latest to kubernetes namespace {$environment}
  scale deployment "myapp" to {$replicas} replicas
  wait for rollout to complete
  
  success "Deployed to {$environment} with {$replicas} replicas"
```

### 3. CI/CD Pipeline
```drun
task "ci-cd" means "Complete CI/CD pipeline":
  given $environment defaults to "staging"
  given $run_tests defaults to true
  
  step "Starting CI/CD pipeline"
  
  # Parallel testing
  if $run_tests is true:
    for each $suite in ["unit", "integration", "e2e"] in parallel:
      run "{$suite} tests"
  
  # Build and security
  run "build application" in parallel with "security scan"
  
  # Deploy
  deploy to {$environment}
  
  # Verification
  run "smoke tests"
  run "health checks"
  
  success "Pipeline completed for {$environment}"
```

### 4. Multi-Environment Setup
```drun
project "multi-env-app" version "1.0":
  set environments as list to ["dev", "staging", "production"]
  set services as list to ["api", "web", "worker"]

task "deploy-all" means "Deploy all services to environment":
  requires $target_env from $globals.environments
  
  for each $service in $globals.services:
    step "Deploying {$service} to {$target_env}"
    deploy {$service}:latest to kubernetes namespace {$target_env}
    
    # Environment-specific configuration
    when $target_env is "production":
      scale deployment "{$service}" to 3 replicas
      enable monitoring for {$service}
    
    when $target_env is "staging":
      scale deployment "{$service}" to 2 replicas
    
    when $target_env is "dev":
      scale deployment "{$service}" to 1 replicas
  
  success "All services deployed to {$target_env}"
```

### 5. Smart Framework Detection
```drun
task "smart-build" means "Build application using detected framework":
  step "Detecting project type"
  
  when symfony is detected:
    info "Symfony project detected"
    run "composer install --no-dev --optimize-autoloader"
    run "php bin/console cache:clear --env=prod"
  
  when laravel is detected:
    info "Laravel project detected"
    run "composer install --no-dev --optimize-autoloader"
    run "php artisan config:cache"
    run "php artisan route:cache"
  
  when node project exists:
    info "Node.js project detected"
    
    # Detect package manager
    detect available "npm" or "yarn" or "pnpm" as $package_manager
    run "{$package_manager} install"
    run "{$package_manager} run build"
  
  when file "go.mod" exists:
    info "Go project detected"
    run "go mod download"
    run "go build -o app"
  
  success "Build completed using detected framework"
```

---

## Best Practices

### 1. Task Naming and Documentation
```drun
# Good: Descriptive names with clear purpose
task "deploy-to-production" means "Deploy application to production environment with safety checks":

# Avoid: Vague or abbreviated names
task "deploy":
```

### 2. Parameter Design
```drun
# Good: Clear constraints and defaults
task "scale-service":
  requires $service_name matching pattern "[a-z-]+"
  requires $environment from ["dev", "staging", "production"]
  given $replica_count as number between 1 and 10 defaults to 3
  given $wait_for_rollout defaults to true

# Avoid: Unconstrained parameters
task "scale-service":
  requires $service
  requires $env
  given $replicas defaults to 1
```

### 3. Variable Scoping
```drun
# Good: Clear scoping with $globals for project settings
project "myapp":
  set registry to "ghcr.io/company"
  set default_timeout to "30s"

task "deploy":
  set $image_tag to "{$globals.registry}/myapp:latest"
  set $timeout to "{$globals.default_timeout}"

# Avoid: Ambiguous variable references
task "deploy":
  set $image_tag to "registry/myapp:latest"  # Hardcoded
```

### 4. Error Handling
```drun
# Good: Comprehensive error handling
task "deploy-with-rollback":
  try:
    deploy myapp:latest to kubernetes
    wait for rollout to complete
    run health checks
  catch deployment_error:
    warn "Deployment failed, rolling back"
    rollback deployment "myapp"
    notify team of failure
  catch health_check_error:
    warn "Health checks failed, investigating"
    get logs from deployment "myapp"
  finally:
    cleanup temporary resources

# Avoid: No error handling
task "deploy":
  deploy myapp:latest to kubernetes
  # What if this fails?
```

### 5. Status Messages
```drun
# Good: Informative status messages
task "build-and-deploy":
  step "Building Docker image for {$environment}"
  info "Using registry: {$globals.registry}"
  warn "Deployment will take approximately 5 minutes"
  success "Deployment completed successfully in {$duration}"

# Avoid: Generic or missing messages
task "build-and-deploy":
  info "Starting"
  # ... operations without status updates
  info "Done"
```

### 6. Matrix Execution Patterns
```drun
# Good: Logical matrix dimensions
task "comprehensive-testing":
  # Test across logical dimensions
  for each $browser in ["chrome", "firefox", "safari"]:
    for each $device in ["desktop", "tablet", "mobile"]:
      run ui tests on {$browser} for {$device}

# Good: Parallel where appropriate
task "multi-region-deployment":
  for each $region in ["us-east", "eu-west", "ap-south"] in parallel:
    deploy to {$region}

# Avoid: Unnecessary nesting
task "simple-build":
  for each $file in ["app.js"]:  # Single item, unnecessary loop
    build {$file}
```

---

## Error Patterns

### Common Syntax Errors
```drun
# ❌ Missing colon after task declaration
task "example"
  info "Hello"

# ✅ Correct syntax
task "example":
  info "Hello"

# ❌ Incorrect parameter syntax
requires environment from "dev", "staging", "production"

# ✅ Correct parameter syntax
requires $environment from ["dev", "staging", "production"]

# ❌ Missing $ prefix for variables
task "greet":
  requires name
  info "Hello {name}"

# ✅ Correct variable syntax
task "greet":
  requires $name
  info "Hello {$name}"
```

### Variable Scope Errors
```drun
# ❌ Undefined variable reference
task "deploy":
  info "Deploying to {$undefined_environment}"

# ✅ Proper variable definition
task "deploy":
  requires $environment from ["dev", "staging", "production"]
  info "Deploying to {$environment}"

# ❌ Incorrect global variable access
project "myapp":
  set registry to "ghcr.io/company"

task "build":
  info "Using registry: {registry}"  # Wrong

# ✅ Correct global variable access
task "build":
  info "Using registry: {$globals.registry}"
```

### Control Flow Errors
```drun
# ❌ Incorrect condition syntax
if environment == "production":  # Wrong operator

# ✅ Correct condition syntax
if $environment is "production":

# ❌ Incorrect when statement syntax
when $package_manager:
  is "npm": run "npm install"

# ✅ Correct when statement syntax
when $package_manager is "npm":
  run "npm install"
when $package_manager is "yarn":
  run "yarn install"
```

### Parameter Validation Errors
```drun
# ❌ Invalid constraint syntax
requires $port as number between "1000" and "9999"  # Strings instead of numbers

# ✅ Correct constraint syntax
requires $port as number between 1000 and 9999

# ❌ Invalid pattern syntax
requires $version matching "v\d+\.\d+\.\d+"  # Missing 'pattern' keyword

# ✅ Correct pattern syntax
requires $version matching pattern "v\d+\.\d+\.\d+"

# ❌ Global variables in constraints (not supported)
requires $environment from $globals.environments

# ✅ Use literal arrays in constraints
requires $environment from ["dev", "staging", "production"]
```

---

## Quick Reference Card

### Essential Syntax
```drun
# Project declaration
project "name" version "1.0":
  set key to "value"
  set list_key as list to ["item1", "item2"]

# Task definition
task "name" means "description":
  requires $param from ["option1", "option2"]
  given $optional defaults to "value"
  depends on other_task
  
  # Statements
  step "Doing something"
  info "Information: {$param}"
  success "Completed"

# Control flow
if condition:
  statements
else:
  statements

when $variable:
  is "value1": action1
  is "value2": action2
  else: default_action

for each $item in collection:
  process {$item}

try:
  risky_operation
catch error_type:
  handle_error
```

### Built-in Actions
```drun
# Docker
build docker image "name:tag"
push image "name:tag" to "registry"
run container "name:tag" on port 8080

# Kubernetes
deploy app:tag to kubernetes namespace env
scale deployment "app" to 5 replicas
rollback deployment "app"

# Git
commit changes with message "Update"
push to branch "main"
create tag "v1.0.0"

# Files
copy "src" to "dest"
backup "file" as "backup-{now.date}"
remove "old-files"

# Status
step "Starting process"
info "Information message"
warn "Warning message"
error "Error message"
success "Success message"
```

### Variable Operations
```drun
# String operations
{$var without prefix "pre"}
{$var without suffix "suf"}
{$var split by ":" | first}

# Array operations
{$list filtered by extension ".js"}
{$list sorted by name}
{$list first}
{$list last}

# Path operations
{$path basename}
{$path dirname}
{$path extension}
```

---

This manual provides comprehensive guidance for LLMs to understand and generate effective drun automation tasks. The semantic, English-like syntax makes it particularly suitable for AI-assisted development while maintaining the power and flexibility needed for complex automation workflows.
