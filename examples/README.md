# drun v2 Examples

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

# Intelligent loops
for each service in ["api", "web", "worker"]:
  deploy service {service}

# Parallel execution
for each region in ["us-east", "eu-west"] in parallel:
  deploy to {region}
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

## 🔄 Migration from v1

The v2 language compiles to drun v1 YAML, so you can gradually migrate:

### v1 YAML:
```yaml
recipes:
  deploy:
    help: "Deploy to environment"
    positionals:
      - name: environment
        required: true
        one_of: ["dev", "staging", "production"]
    deps: [build]
    run: |
      kubectl set image deployment/myapp myapp=myapp:latest --namespace={{ .environment }}
```

### v2 Semantic:
```
task "deploy" means "Deploy to environment":
  requires environment from ["dev", "staging", "production"]
  depends on build
  
  deploy myapp:latest to kubernetes namespace {environment}
```

## 🛠️ Running Examples

*Note: drun v2 compiler is not yet implemented. These examples show the target syntax.*

Once the v2 compiler is ready, you'll run examples like this:

```bash
# Basic examples
drun -f 01-hello-world.drun hello
drun -f 02-parameters.drun greet --name=Alice --title=Ms.

# Docker examples  
drun -f 04-docker-basics.drun build --tag=v1.0.0
drun -f 04-docker-basics.drun "run local" --port=3000

# Kubernetes examples
drun -f 05-kubernetes.drun deploy --environment=staging
drun -f 05-kubernetes.drun scale --environment=production --replica_count=10

# CI/CD pipeline
drun -f 06-cicd-pipeline.drun "ci pipeline"
drun -f 06-cicd-pipeline.drun "deploy to staging"
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
capture result from "command"       # Capture command output

# Conditional assignment
let config be:
  when environment is "prod": production_config
  else: development_config
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
- **[Migration Guide](../docs/v2-migration.md)** - Migrating from v1 to v2
- **[Language Reference](../docs/v2-reference.md)** - Detailed syntax reference

---

**Ready to revolutionize your automation workflows?** Start with `01-hello-world.drun` and experience the future of task automation! 🚀
