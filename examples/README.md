# drun v2 Examples (execute with xdrun CLI)

Welcome to drun v2! This directory contains examples showcasing the new **semantic, English-like language** for defining automation tasks. The v2 language compiles to shell commands while providing intuitive, readable syntax that anyone can understand.

## 🌟 What's New in v2

drun v2 introduces a revolutionary approach to task automation:

- **🗣️ Natural Language Syntax**: Write tasks in English-like sentences
- **🧠 Smart Detection**: Automatically detect tools, frameworks, and environments  
- **🔄 Intelligent Compilation**: Compiles to optimized shell commands
- **📚 Type Safety**: Parameters with validation and constraints
- **🎯 Intent-Focused**: Describe *what* you want, not *how* to do it

## 📁 Example Files

### Basic Examples
- **[01-hello-world.drun](01-hello-world.drun)** - Your first drun v2 tasks
- **[02-parameters.drun](02-parameters.drun)** - Parameters, defaults, and validation
- **[03-control-flow.drun](03-control-flow.drun)** - If statements, loops, and error handling

### Infrastructure Examples  
- **[04-docker-basics.drun](04-docker-basics.drun)** - Docker workflows and container management
- **[05-kubernetes.drun](05-kubernetes.drun)** - Kubernetes deployments and operations

### Advanced Examples
- **[06-cicd-pipeline.drun](06-cicd-pipeline.drun)** - Complete CI/CD pipeline with blue-green deployment
- **[07-smart-detection.drun](07-smart-detection.drun)** - Intelligent project and framework detection
- **[33-semantic-actions-showcase.drun](33-semantic-actions-showcase.drun)** - Comprehensive showcase of all semantic actions
- **[34-working-semantic-actions.drun](34-working-semantic-actions.drun)** - Working examples of implemented semantic actions
- **[35-advanced-parameter-validation.drun](35-advanced-parameter-validation.drun)** - Advanced parameter validation with pattern macros (`semver`, `uuid`, `url`)
- **[36-advanced-variable-operations.drun](36-advanced-variable-operations.drun)** - Comprehensive showcase of variable operations (`filtered`, `sorted`, `without`, `split`, chaining)

### 🔄 Matrix Execution & Array Literals
- **[42-matrix-sequential.drun](42-matrix-sequential.drun)** - Sequential matrix execution patterns (OS × Architecture, Database × Test Suite)
- **[43-matrix-parallel.drun](43-matrix-parallel.drun)** - Parallel matrix execution (Multi-region deployment, CI/CD parallelization)
- **[44-array-literals-showcase.drun](44-array-literals-showcase.drun)** - Comprehensive array literal examples and real-world use cases

## 🚀 Quick Start

### Hello World
```
task "hello":
  info "Hello from drun v2! 👋"
```

### With Parameters
```
task "greet" means "Greet someone by name":
  requires name
  given title defaults to "friend"
  
  info "Hello, {title} {name}!"
```

### Smart Docker Build
```
task "build" means "Build Docker image":
  given tag defaults to current git commit
  
  build docker image "myapp:{tag}"
  success "Built image: myapp:{tag}"
```

### Kubernetes Deployment
```
task "deploy" means "Deploy to Kubernetes":
  requires environment from ["dev", "staging", "production"]
  
  deploy myapp:latest to kubernetes namespace {environment}
  wait for rollout to complete
```

## 🎯 Key Language Features

### Natural Parameter Declaration
```
# Required parameters with validation
requires environment from ["dev", "staging", "production"]
requires port as number between 1000 and 9999
requires email matching email format

# Optional parameters with defaults
given replicas defaults to 3
given timeout defaults to "5m"
given force defaults to false

# Lists and arrays
accepts features as list of strings
accepts configs as list
```

### Smart Control Flow
```
# Natural conditionals
if docker is running:
  build container
else:
  error "Docker is not available"

# Pattern matching
when environment:
  is "production": require manual approval
  is "staging": run integration tests
  else: skip validation

# Array literals and loops
for each $service in ["api", "web", "worker"]:
  deploy service {$service}

# Parallel matrix execution
for each $region in ["us-east", "eu-west"] in parallel:
  for each $service in ["api", "web", "worker"]:
    deploy {$service} to {$region}
```

### Array Literals & Matrix Execution
```
# Project-level array definitions
project "MyApp" version "1.0":
  set platforms as list to ["linux", "darwin", "windows"]
  set architectures as list to ["amd64", "arm64"]

# Sequential matrix execution
for each $platform in platforms:
  for each $arch in architectures:
    build for {$platform}/{$arch}

# Parallel matrix execution  
for each $env in ["dev", "staging", "prod"] in parallel:
  for each $service in ["api", "web", "worker"]:
    deploy {$service} to {$env}
```

### Smart Detection
```
# Framework detection
when symfony is detected:
  run symfony console commands

when laravel is detected:
  run artisan commands

# Tool detection  
if docker is running:
  build containerized app

if kubernetes is available:
  deploy to cluster

# Project type detection
when package manager:
  is "npm": run "npm ci && npm run build"
  is "yarn": run "yarn install && yarn build"
  is "go": run "go build ./..."
```

### Built-in Actions
```
# Docker operations
build docker image "myapp:latest"
push image "myapp:latest" to "ghcr.io"
run container "myapp:latest" on port 8080

# Kubernetes operations
deploy myapp:latest to kubernetes namespace production
scale deployment "myapp" to 5 replicas
rollback deployment "myapp"

# Git operations
commit changes with message "Add new feature"
push to branch "main"
create tag "v1.2.3"

# File operations
copy "source.txt" to "destination.txt"
backup "important.txt" as "backup-{now.date}"
remove "old-files/"

# Status messages
step "Starting deployment"
info "Configuration loaded"
warn "Using default settings"
error "Connection failed"
success "Deployment completed"
```

## 🛠️ Running Examples

*Note: drun v2 compiler is not yet implemented. These examples show the target syntax.*

Once the v2 compiler is ready, you'll run examples like this:

```bash
# Basic examples
xdrun -f 01-hello-world.drun hello
xdrun -f 02-parameters.drun greet --name=Alice --title=Ms.
xdrun -f 02-parameters.drun "build docker" image=base dest=local
xdrun -f 02-parameters.drun "deploy service"  # Uses all defaults (dev, replicas=1)

# Docker examples  
xdrun -f 04-docker-basics.drun build --tag=v1.0.0
xdrun -f 04-docker-basics.drun "run local" --port=3000

# Kubernetes examples
xdrun -f 05-kubernetes.drun deploy --environment=staging
xdrun -f 05-kubernetes.drun scale --environment=production --replica_count=10

# CI/CD pipeline
xdrun -f 06-cicd-pipeline.drun "ci pipeline"
xdrun -f 06-cicd-pipeline.drun "deploy to staging"
```

## 📖 Language Reference

### Task Definition
```
task <name> [means <description>]:
  [parameters]
  [dependencies] 
  [lifecycle_hooks]
  [variables]
  <statements>
```

### Parameter Types
- **String**: `requires name`
- **Number**: `requires port as number`
- **Boolean**: `given force defaults to false`
- **List**: `accepts items as list of strings`
- **Constrained**: `requires env from ["dev", "prod"]`
- **Pattern**: `requires version matching pattern "v\d+\.\d+\.\d+"`

### Dependencies
```
depends on build                    # Single dependency
depends on build and test          # Multiple dependencies  
depends on build then deploy       # Sequential dependencies
depends on lint, test, scan        # Parallel dependencies
```

### Variables
```
let name be "value"                 # Immutable binding
set counter to 0                    # Mutable variable
capture from shell "command" as $variable       # Capture command output

# Conditional assignment
let config be:
  when environment is "prod": production_config
  else: development_config
```

### Variable Operations
```drun
# String operations
set $version to "v2.1.0-beta"
info "Clean version: {$version without prefix 'v' | without suffix '-beta'}"
# Output: 2.1.0

# Array operations
set $files to "app.js test.js config.json readme.md"
info "JS files: {$files filtered by extension '.js'}"
# Output: app.js test.js

info "Sorted files: {$files sorted by name}"
# Output: app.js config.json readme.md test.js

# Path operations
set $config_path to "/etc/nginx/default.conf"
info "Filename: {$config_path basename}"
# Output: default.conf

# Complex chaining
set $source_files to "src/app.js src/utils.js tests/app.test.js"
info "Source JS files: {$source_files filtered by prefix 'src/' | filtered by extension '.js' | sorted by name}"
# Output: src/app.js src/utils.js

# Loop integration
for each img in $docker_images:
  info "Processing: {img split by ':' | first}"
```

### Control Flow
```
# Conditionals
if condition:
  statements
else if other_condition:
  statements  
else:
  statements

# Pattern matching
when expression:
  is value1: statements
  is value2: statements
  else: statements

# Loops
for each item in collection:
  process item

for i from 1 to 10:
  step "Iteration {i}"

# Error handling
try:
  risky_operation
catch error_type:
  handle_error
finally:
  cleanup
```

## 🎨 Best Practices

### 1. Use Descriptive Task Names
```
# Good
task "deploy to production" means "Deploy application to production environment"

# Avoid
task "deploy"
```

### 2. Leverage Smart Detection
```
# Good - let drun detect the right approach
when symfony is detected:
  run symfony console commands

# Avoid - hardcoding specific commands
run "php bin/console cache:clear"
```

### 3. Use Natural Parameter Names
```
# Good
requires target_environment from ["dev", "staging", "production"]
given replica_count defaults to 3

# Avoid
requires env from ["dev", "staging", "production"]  
given replicas defaults to 3
```

### 4. Structure Complex Workflows
```
# Break down complex operations
task "full deployment":
  depends on "run tests" and "build image" then "deploy to staging"
  
  step "Starting production deployment"
  run "deploy to production"
  run "verify deployment"
  run "notify team"
```

### 5. Use Meaningful Status Messages
```
step "Building Docker image for {environment}"
info "Using configuration: {config_file}"
warn "No SSL certificate found, using HTTP"
error "Database connection failed: {error_message}"
success "Deployment completed in {duration}"
```

## 🔮 Future Features

The v2 language is designed for extensibility. Planned features include:

- **Plugin System**: Custom actions and detectors
- **Advanced Templates**: More sophisticated templating
- **IDE Integration**: Syntax highlighting, completion, debugging
- **Visual Editor**: Drag-and-drop task builder
- **AI Assistant**: Natural language to drun conversion

## 🤝 Contributing

These examples represent the target syntax for drun v2. As we implement the compiler, examples may evolve. Contributions and feedback are welcome!

## 📚 Learn More

- **[drun v2 Specification](../DRUN_V2_SPECIFICATION.md)** - Complete language specification
- **[Language Reference](../docs/v2-reference.md)** - Detailed syntax reference

---

**Ready to revolutionize your automation workflows?** Start with `01-hello-world.drun` and experience the future of task automation! 🚀
