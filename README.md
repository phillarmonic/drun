# drun (do run)

A **semantic, English-like** task automation language with intelligent execution, smart detection, and powerful built-in actions. Write automation tasks in natural language that compiles to efficient shell commands.

## Features

### 🚀 **Core Features**

- **Semantic Language**: Write tasks in English-like syntax that's intuitive and readable
- **Smart Parameters**: Type-safe parameters with constraints and defaults (`requires $env from ["dev", "prod"]`)
- **Variable System**: Powerful variable interpolation with `$variable` syntax, `$globals` namespace, and built-in functions
- **Control Flow**: Natural `if/else`, `for each`, `when` statements with intelligent conditions
- **Built-in Actions**: Docker, Kubernetes, Git, HTTP operations with semantic commands
- **Smart Detection**: Auto-detect project types, tools, and environments
- **Shell Integration**: Seamless shell command execution with output capture
- **Cross-Platform**: Works on Linux, macOS, and Windows with intelligent shell selection
- **Dry Run & Explain**: See what would be executed without running it
- **Type Safety**: Static analysis with runtime validation

### 🌟 **Advanced Features**

- **🔗 Project Declarations**: Define global project settings, includes, and lifecycle hooks
- **🔄 Dependency System**: Automatic task dependency resolution with parallel execution
- **🌐 HTTP Actions**: Built-in HTTP requests with authentication and response handling
- **🐳 Docker Integration**: Semantic Docker commands (`build docker image`, `run container`)
- **☸️ Kubernetes Support**: Native kubectl operations with intelligent resource management
- **📊 Error Handling**: Comprehensive `try/catch/finally` with custom error types
- **🔄 Parallel Execution**: True parallel loops with concurrency control and progress tracking
- **🎯 Smart Detection**: Auto-detect tools, frameworks, and environments intelligently
- **🔧 DRY Tool Detection**: Detect tool variants and capture working ones (`detect available "docker compose" or "docker-compose" as $compose_cmd`)
- **📁 File Operations**: Built-in file system operations with path interpolation
- **🎯 Pattern Macros**: Built-in validation patterns (`matching semver`, `matching uuid`, `matching url`) with descriptive error messages
- **🔄 Advanced Variable Operations**: Powerful data transformation (`{$files filtered by extension '.js' | sorted by name}`, `{$version without prefix 'v'}`, `{$path basename}`)

### 🛠️ **Developer Experience**

- **20+ Template Functions**: Docker detection, Git integration, HTTP calls, status messages, and more
- **Intelligent Caching**: HTTP and Git includes cached for performance
- **Rich Error Messages**: Helpful suggestions and context for debugging
- **Shell Completion**: Intelligent completion for bash, zsh, fish, and PowerShell
- **Self-Update**: Built-in update mechanism with backup management

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/phillarmonic/drun/releases):

| Platform    | Architecture  | Download                                  |
| ----------- | ------------- | ----------------------------------------- |
| **Linux**   | x86_64        | `drun-linux-amd64` (UPX compressed)       |
| **Linux**   | ARM64         | `drun-linux-arm64` (UPX compressed)       |
| **macOS**   | Intel         | `drun-darwin-amd64`                       |
| **macOS**   | Apple Silicon | `drun-darwin-arm64`                       |
| **Windows** | x86_64        | `drun-windows-amd64.exe` (UPX compressed) |
| **Windows** | ARM64         | `drun-windows-arm64.exe`                  |

All binaries are **statically linked** and have **no dependencies**.

### Install Script

```bash
# Install latest version (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash

# Install specific version
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s v1.0.0
```

## Quick Start

### 📁 **File Structure**

drun uses a simple, predictable file discovery system:

- **`.drun/spec.drun`** - Default task file location
- **Custom locations** - Use `--file` to specify any other location
- **Workspace configuration** - `.drun/.drun_workspace.yml` for custom defaults

**Moving your spec file:**
```bash
# Move your spec file anywhere
mv .drun/spec.drun ./my-project.drun

# Update workspace to point to new location
drun --set-workspace my-project.drun

# Now drun automatically uses your custom location
drun --list
```

### 🚀 **Getting Started**

1. **Create a simple task file** (`.drun/spec.drun`):
   
   ```drun
   project "my-app" version "1.0"
   
   task "hello" means "Say hello":
     info "Hello from drun v2! 🚀"
   ```

2. **List available tasks**:
   
   ```bash
   drun --list
   ```

3. **Run a task**:
   
   ```bash
   drun hello
   ```

4. **Use parameters**:
   
   ```bash
   drun deploy environment=production version=v1.0.0
   ```

5. **Explore examples**:
   
   ```bash
   drun -f examples/01-hello-world.drun hello
   ```

6. **Dry run to see what would execute**:
   
   ```bash
   drun build --dry-run
   ```

### 🔧 **Variable Scoping**

drun v2 uses a clear scoping system with explicit namespaces to prevent naming conflicts:

#### **Project Settings (Global)**
Declared without `$` prefix, accessed via `$globals` namespace:

```drun
project "myapp" version "1.0.0":
  set registry to "ghcr.io/company"
  set api_url to "https://api.example.com"

task "deploy":
  info "Project: {$globals.project}"        # → "myapp"
  info "Version: {$globals.version}"        # → "1.0.0" 
  info "Registry: {$globals.registry}"      # → "ghcr.io/company"
  info "API: {$globals.api_url}"           # → "https://api.example.com"
```

#### **Task Variables (Local)**
Declared with `$` prefix, accessed directly:

```drun
task "deploy":
  set $image_tag to "{$globals.registry}/myapp:{$globals.version}"
  set $replicas to 3
  
  info "Deploying {$image_tag} with {$replicas} replicas"
```

#### **Avoiding Conflicts**
The `$globals` namespace prevents naming conflicts:

```drun
project "myapp":
  set api_url to "https://project-level.com"

task "test":
  set $api_url to "https://task-level.com"    # Different variable!
  
  info "Global API: {$globals.api_url}"       # → "https://project-level.com"
  info "Task API: {$api_url}"                 # → "https://task-level.com"
```

## Configuration

drun uses a simple file discovery system:

1. **Workspace default** (if configured in `.drun/.drun_workspace.yml`)
2. **Default location**: `.drun/spec.drun`
3. **Explicit specification**: Use `--file` for any other location

### 🔧 **Workspace Configuration**

Create a workspace configuration file at `.drun/.drun_workspace.yml`:

```yaml
# Workspace settings
default_task_file: "custom-tasks.drun"
parallel_jobs: 4
shell: "/bin/bash"

# Global variables
variables:
  project_name: "my-app"
  environment: "development"

# Default parameters
defaults:
  environment: "dev"
  verbose: true
```

### Getting Started

Use `drun --init` to create a starter task file:

```bash
# Create default .drun/spec.drun
drun --init

# Create custom task file and save as workspace default
drun --init --file=my-project.drun --save-as-default

# Move existing file and update workspace
mv .drun/spec.drun ./tasks.drun
drun --set-workspace tasks.drun
```

See the included examples for comprehensive task configurations.

📖 **For complete v2 specification**: See [DRUN_V2_SPECIFICATION.md](DRUN_V2_SPECIFICATION.md) for detailed language reference and examples.

### Basic Task

```drun
version: 2.0

task "hello" means "Say hello":
  info "Hello, World! 👋"
```

### Task with Parameters

```drun
task "greet" means "Greet someone":
  requires $name
  given $title defaults to "friend"
  
  info "Hello, {$title} {$name}! 🎉"
```

**Usage examples:**

```bash
# Simple parameter passing
drun greet name=Alice
drun greet name=Bob title=Mr.

# All parameters
drun greet name=Alice title=Ms.
```

### Advanced Parameters with Control Flow

```drun
task "deploy" means "Deploy to environment with version":
  requires $environment from ["dev", "staging", "prod"]
  given $version defaults to "latest"
  given $features as list defaults to ""
  given $force as boolean defaults to false
  
  info "Deploying {$version} to {$environment}"
  
  if $features is not "":
    info "Features: {$features}"
  
  if $force is true:
    info "Force deployment enabled"
```

**Usage examples:**

```bash
# Basic deployment
drun deploy environment=prod

# With version and features
drun deploy environment=staging version=v1.1.0 features=auth,ui

# Force deployment
drun deploy environment=prod version=v1.2.0 force=true
```

### Task with Dependencies

```drun
task "test" means "Run tests":
  depends on build
  
  run "go test ./..."

task "build" means "Build the project":
  run "go build ./..."
```

### Prerun Snippets - DRY Common Setup

Define snippets that automatically run before every recipe - perfect for colors, helper functions, and common setup:

```yaml
version: 1.0

# Snippets that run before EVERY recipe
recipe-prerun:
  - |
    # ANSI color codes - available in all recipes
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
  - |
    # Helper functions available everywhere
    log_info() {
      echo -e "${BLUE}ℹ️  $1${NC}"
    }
    log_success() {
      echo -e "${GREEN}✅ $1${NC}"
    }
    log_error() {
      echo -e "${RED}❌ $1${NC}"
    }

recipes:
  setup:
    help: "Set up the project"
    run: |
      # Colors and functions automatically available!
      log_info "Setting up project..."
      echo -e "${GREEN}🚀 Initializing...${NC}"
      log_success "Setup completed!"

  test:
    help: "Run tests"  
    run: |
      # Same colors and functions available here too!
      log_info "Running test suite..."
      echo -e "${YELLOW}🧪 Testing...${NC}"
      log_success "All tests passed!"
```

**Benefits:**
- **DRY**: Define once, use everywhere
- **Automatic**: No need to manually include in each recipe
- **Templated**: Full template support with variables
- **Snippet Calls**: Can call existing snippets with `{{ snippet "name" }}`

## 🌟 Advanced Features Examples

### 🔗 Remote Includes

Share and reuse recipes across projects and teams:

```yaml
version: 1.0

# Include recipes from various sources
include:
  # Local files
  - "shared/docker-common.yml"

  # HTTP/HTTPS includes
  - "https://raw.githubusercontent.com/company/drun-recipes/main/docker/common.yml"
  - "https://raw.githubusercontent.com/company/drun-recipes/main/ci/github-actions.yml"

  # Git repositories with branch/tag references
  - "git+https://github.com/company/drun-recipes.git@main:docker/production.yml"
  - "git+https://github.com/company/drun-recipes.git@v1.0.0:ci/common.yml"
  - "git+https://github.com/company/base-recipes.git@stable"  # Uses default drun.yml

recipes:
  deploy:
    help: "Deploy using shared recipes"
    deps: [docker-build]  # From remote include
    run: |
      {{ step "Deploying with shared configuration" }}
      echo "Using recipes from remote sources!"
```

**Benefits:**

- 🏢 **Enterprise**: Centralized governance and compliance
- 🌐 **Community**: Share recipes across open source projects  
- 🔄 **Versioning**: Pin to specific tags/commits for stability
- ⚡ **Performance**: Intelligent caching for fast execution

### 🌐 HTTP Integration

Integrate with APIs, send notifications, and fetch data using semantic HTTP actions:

```drun
task "notify slack" means "Send notification to Slack":
  requires $message
  given $channel defaults to "#general"
  
  post "https://hooks.slack.com/services/..." with body {
    "text": "{$message}",
    "channel": "{$channel}"
  } with header "Content-Type: application/json"

task "check api" means "Check API status":
  let $response = get "https://api.example.com/health"
  
  if $response contains "ok":
    success "API is healthy"
  else:
    fail "API is down"

```

**Key HTTP features:**

- 🌐 **Semantic HTTP Actions**: `get`, `post`, `put`, `delete` with natural syntax
- 🔗 **Authentication**: Built-in support for bearer tokens, basic auth, API keys
- 📊 **JSON Support**: Automatic JSON parsing and response handling
- 🔄 **Error Handling**: Intelligent retry and error management
- ⚡ **Response Capture**: Store responses in variables for processing

### 🔄 Matrix Execution

Run recipes across multiple configurations automatically:

```yaml
recipes:
  test-matrix:
    help: "Test across multiple environments"
    matrix:
      os: ["ubuntu", "macos", "windows"]
      node_version: ["16", "18", "20"]
      arch: ["amd64", "arm64"]
    run: |
      {{ step "Testing on {{ .matrix_os }}/{{ .matrix_node_version }}/{{ .matrix_arch }}" }}

      # OS-specific behavior
      {{ if eq .matrix_os "windows" }}
      echo "Running Windows-specific tests"
      {{ else if eq .matrix_os "macos" }}
      echo "Running macOS-specific tests"
      {{ else }}
      echo "Running Linux-specific tests"
      {{ end }}

      # Version-specific behavior
      {{ if eq .matrix_node_version "16" }}
      echo "Using legacy Node.js features"
      {{ else if eq .matrix_node_version "20" }}
      echo "Using latest Node.js features"
      {{ end }}

      {{ success "Test completed for {{ .matrix_os }}/{{ .matrix_node_version }}" }}

  build-matrix:
    help: "Build for multiple architectures"
    matrix:
      arch: ["amd64", "arm64"]
      variant: ["alpine", "debian"]
    deps: [setup]  # Runs once before all matrix jobs
    run: |
      {{ step "Building for {{ .matrix_arch }}/{{ .matrix_variant }}" }}

      IMAGE_TAG="myapp:{{ .matrix_arch }}-{{ .matrix_variant }}"
      docker build --platform linux/{{ .matrix_arch }} \
        -f Dockerfile.{{ .matrix_variant }} \
        -t $IMAGE_TAG .

      {{ success "Built: $IMAGE_TAG" }}
```

**Matrix expands to multiple jobs:**

- `test-matrix` → 18 jobs (3 OS × 3 versions × 2 arch)
- `build-matrix` → 4 jobs (2 arch × 2 variants)
- All jobs run in parallel with intelligent dependency management

### 🔐 Secrets Management

Handle sensitive data securely:

```yaml
# Define secrets with their sources
secrets:
  api_key:
    source: "env://API_KEY"
    required: true
    description: "API key for external service"

  db_password:
    source: "env://DATABASE_PASSWORD" 
    required: false
    description: "Database password"

  deploy_token:
    source: "file://~/.secrets/deploy-token"
    required: true
    description: "Deployment token"

recipes:
  deploy:
    help: "Secure deployment with secrets"
    run: |
      {{ step "Starting secure deployment" }}

      # Check required secrets
      {{ if not (hasSecret "api_key") }}
      {{ error "API_KEY environment variable is required" }}
      exit 1
      {{ end }}

      {{ info "All required secrets available" }}

      # Use secrets securely (not logged in plain text)
      curl -H "Authorization: Bearer {{ secret "api_key" }}" \
        -d '{"version": "{{ gitShortCommit }}"}' \
        https://api.company.com/deploy

      {{ if hasSecret "db_password" }}
      echo "Database configured with provided password"
      {{ else }}
      {{ warn "Using default database configuration" }}
      {{ end }}

      {{ success "Deployment completed securely" }}
```

**Supported sources:**

- `env://VAR_NAME` - Environment variables
- `file://path/to/secret` - File-based secrets
- `vault://path/to/secret` - HashiCorp Vault (planned)

### 🎯 Smart Template Functions

drun includes 15+ intelligent template functions:

```yaml
env:
  # Auto-detect commands and tools
  DOCKER_COMPOSE: "{{ dockerCompose }}"    # "docker compose" or "docker-compose"
  DOCKER_BUILDX: "{{ dockerBuildx }}"      # "docker buildx" or "docker-buildx"

  # Git information
  GIT_BRANCH: "{{ gitBranch }}"             # Current branch
  GIT_COMMIT: "{{ gitShortCommit }}"        # Short commit hash
  IS_DIRTY: "{{ isDirty }}"                 # Working directory dirty

  # Project detection
  PROJECT_TYPE: "{{ packageManager }}"      # npm, yarn, go, pip, etc.
  BUILD_ENV: "{{ if isCI }}ci{{ else }}local{{ end }}"

recipes:
  smart-build:
    help: "Intelligent build using auto-detection"
    run: |
      {{ step "Building {{ packageManager }} project" }}

      echo "🔍 Project Analysis:"
      echo "  Type: {{ packageManager }}"
      echo "  Git: {{ gitBranch }}@{{ gitShortCommit }}"
      echo "  Environment: {{ if isCI }}CI{{ else }}Local{{ end }}"

      # Docker integration
      {{ if hasFile "Dockerfile" }}
      {{ info "Docker configuration detected" }}
      $DOCKER_BUILDX build -t myapp:{{ gitShortCommit }} .
      {{ end }}

      # Package manager specific builds
      {{ if eq (packageManager) "npm" }}
      npm ci && npm run build
      {{ else if eq (packageManager) "go" }}
      go build ./...
      {{ else if eq (packageManager) "pip" }}
      pip install -r requirements.txt
      {{ end }}

      {{ success "Smart build completed!" }}

  status-demo:
    help: "Demonstrate status messages"
    run: |
      {{ step "Processing with status updates" }}

      {{ info "This is an informational message" }}
      {{ warn "This is a warning message" }}
      {{ error "This is an error message (non-fatal)" }}
      {{ success "This is a success message" }}

      # Conditional status based on detection
      {{ if hasFile "go.mod" }}
      {{ success "Go project detected" }}
      {{ else }}
      {{ warn "No Go project found" }}
      {{ end }}
```

**Available template functions:**

- **Docker**: `dockerCompose`, `dockerBuildx`, `hasCommand`
- **Git**: `gitBranch`, `gitCommit`, `gitShortCommit`, `isDirty`
- **Project**: `packageManager`, `hasFile`, `isCI`
- **Status**: `step`, `info`, `warn`, `error`, `success`
- **Secrets**: `secret`, `hasSecret`

### 🔧 DRY Tool Detection

Eliminate repetitive conditional logic with intelligent tool variant detection:

```drun
project "cross-platform-app" version "1.0":

task "setup-docker-tools" means "Setup Docker toolchain with DRY detection":
  info "🐳 Setting up Docker toolchain"
  
  # Detect which Docker Compose variant is available and capture it
  detect available "docker compose" or "docker-compose" as $compose_cmd
  
  # Detect which Docker Buildx variant is available and capture it
  detect available "docker buildx" or "docker-buildx" as $buildx_cmd
  
  info "✅ Detected tools:"
  info "  📦 Compose: {$compose_cmd}"
  info "  🔨 Buildx: {$buildx_cmd}"
  
  # Now use the captured variables consistently throughout the task
  run "{$compose_cmd} version"
  run "{$buildx_cmd} version"
  
  success "Docker toolchain ready!"

task "deploy-app" means "Deploy using detected tools":
  # Reuse the same detection pattern
  detect available "docker compose" or "docker-compose" as $compose_cmd
  
  info "🚀 Deploying with {$compose_cmd}"
  run "{$compose_cmd} up -d"
  run "{$compose_cmd} ps"
  
  success "Application deployed!"

task "multi-tool-example" means "Multiple tool alternatives":
  # Package managers
  detect available "npm" or "yarn" or "pnpm" as $package_manager
  run "{$package_manager} install"
  run "{$package_manager} run build"
  
  # Container runtimes
  detect available "docker" or "podman" as $container_runtime
  run "{$container_runtime} build -t myapp ."
  
  success "Built with {$package_manager} and {$container_runtime}!"
```

**Benefits:**

- **🎯 DRY Principle**: No repetitive `if/else` conditional logic
- **🌐 Cross-Platform**: Works across different tool installations automatically
- **🔧 Maintainable**: Single detection point, consistent usage throughout tasks
- **⚡ Flexible**: Supports any number of tool alternatives with `or` syntax
- **📝 Clear Intent**: Makes tool compatibility explicit and documented

### 🎯 Pattern Macro Validation

Built-in pattern macros provide common validation patterns without complex regex:

```drun
project "validation-demo" version "1.0":

task "deploy" means "Deploy with comprehensive validation":
  # Semantic versioning validation
  requires $version as string matching semver
  
  # Extended semantic versioning (with pre-release/build info)
  requires $release as string matching semver_extended
  
  # UUID validation for deployment tracking
  requires $deployment_id as string matching uuid
  
  # URL validation for endpoints
  requires $api_endpoint as string matching url
  
  # IPv4 address validation for servers
  requires $server_ip as string matching ipv4
  
  # Project slug validation (URL-safe names)
  requires $project_slug as string matching slug
  
  # Docker tag validation
  requires $image_tag as string matching docker_tag
  
  # Git branch validation
  requires $branch as string matching git_branch
  
  # Email validation (built-in)
  requires $admin_email as string matching email format
  
  # Custom regex patterns still supported
  requires $custom_id as string matching pattern "^DEPLOY-[0-9]{6}$"
  
  info "🚀 Deploying {version} to {server_ip}"
  info "📦 Project: {project_slug}, Branch: {branch}"
  info "🌐 API: {api_endpoint}"
  info "📧 Admin: {admin_email}"
  info "🆔 Deployment ID: {deployment_id}"
  
  success "Deployment validated and ready!"

task "validation-errors-demo" means "Show validation error messages":
  requires $version as string matching semver
  
  # This will show: Error: parameter 'version': value '1.2.3' does not match 
  # semver pattern (Basic semantic versioning (e.g., v1.2.3))
```

**Available Pattern Macros:**

- **`semver`**: Basic semantic versioning (`v1.2.3`)
- **`semver_extended`**: Extended semver (`v2.0.1-RC2`, `v1.0.0-alpha.1+build.123`)
- **`uuid`**: UUID format (`550e8400-e29b-41d4-a716-446655440000`)
- **`url`**: HTTP/HTTPS URLs
- **`ipv4`**: IPv4 addresses (`192.168.1.1`)
- **`slug`**: URL slugs (`my-project-name`)
- **`docker_tag`**: Docker image tags
- **`git_branch`**: Git branch names

**Benefits:**

- **🎯 User-Friendly**: Simple, memorable names instead of complex regex
- **📚 Self-Documenting**: Built-in descriptions explain validation rules
- **🔒 Type-Safe**: Clear, descriptive error messages
- **⚡ Performance**: Efficient validation with minimal overhead
- **🔄 Extensible**: Easy to add new macros as needed

### 🔄 Advanced Variable Operations

Powerful data transformation operations with intuitive chaining syntax:

```drun
project "data-processing" version "1.0":

task "string_transformations" means "Demonstrate string operations":
  set $version to "v2.1.0-beta"
  set $filename to "my-app.tar.gz"
  set $docker_image to "nginx:1.21"
  
  info "🔤 String Operations:"
  info "  Clean version: {$version without prefix 'v' | without suffix '-beta'}"
  info "  App name: {$filename without suffix '.tar.gz'}"
  info "  Image name: {$docker_image split by ':' | first}"
  
  # Output:
  # Clean version: 2.1.0
  # App name: my-app
  # Image name: nginx

task "array_operations" means "Demonstrate array manipulation":
  set $files to "src/app.js src/utils.js tests/app.test.js docs/readme.md config.json"
  
  info "📋 Array Operations:"
  info "  JavaScript files: {$files filtered by extension '.js'}"
  info "  Source files (sorted): {$files filtered by prefix 'src/' | sorted by name}"
  info "  First file: {$files first}"
  info "  All files (sorted): {$files sorted by name}"
  
  # Output:
  # JavaScript files: src/app.js src/utils.js tests/app.test.js
  # Source files (sorted): src/app.js src/utils.js
  # First file: src/app.js
  # All files (sorted): config.json docs/readme.md src/app.js src/utils.js tests/app.test.js

task "path_operations" means "Demonstrate path manipulation":
  set $config_file to "/etc/nginx/sites-available/default.conf"
  
  info "📁 Path Operations:"
  info "  Filename: {$config_file basename}"
  info "  Directory: {$config_file dirname}"
  info "  Extension: {$config_file extension}"
  info "  Name without extension: {$config_file basename | without suffix '.conf'}"
  
  # Output:
  # Filename: default.conf
  # Directory: /etc/nginx/sites-available
  # Extension: conf
  # Name without extension: default

task "complex_chaining" means "Demonstrate operation chaining":
  set $project_files to "src/app.js src/utils.js tests/app.test.js tests/utils.test.js docs/readme.md"
  set $docker_images to "nginx:1.21 postgres:13 redis:6.2 node:16-alpine"
  
  info "⛓️ Complex Operation Chaining:"
  info "  Source JS files: {$project_files filtered by prefix 'src/' | filtered by extension '.js' | sorted by name}"
  info "  Test files: {$project_files filtered by prefix 'tests/' | sorted by name}"
  
  info "🐳 Processing Docker images:"
  for each img in $docker_images:
    info "    {img} -> {img split by ':' | first} (version: {img split by ':' | last})"
  
  # Output:
  # Source JS files: src/app.js src/utils.js
  # Test files: tests/app.test.js tests/utils.test.js
  # Processing Docker images:
  #   nginx:1.21 -> nginx (version: 1.21)
  #   postgres:13 -> postgres (version: 13)
  #   redis:6.2 -> redis (version: 6.2)
  #   node:16-alpine -> node (version: 16-alpine)
```

**Available Operations:**

**String Operations:**
- **`without prefix "text"`** - Remove prefix from string
- **`without suffix "text"`** - Remove suffix from string
- **`split by "delimiter"`** - Split string into space-separated parts

**Array Operations:**
- **`filtered by extension "ext"`** - Filter by file extension
- **`filtered by prefix "text"`** - Filter by prefix
- **`filtered by suffix "text"`** - Filter by suffix
- **`filtered by name "text"`** - Filter by name containing text
- **`sorted by name`** - Sort alphabetically
- **`sorted by length`** - Sort by string length
- **`reversed`** - Reverse order
- **`unique`** - Remove duplicates
- **`first`** - Get first item
- **`last`** - Get last item

**Path Operations:**
- **`basename`** - Extract filename from path
- **`dirname`** - Extract directory from path
- **`extension`** - Extract file extension (without dot)

**Benefits:**

- **🔥 Eliminates Shell Scripting**: No more complex `sed`, `awk`, or `cut` commands
- **⚡ Intuitive Syntax**: English-like operations that are self-documenting
- **🔗 Chainable**: Combine operations with pipe (`|`) for complex transformations
- **🎯 Type-Aware**: Works seamlessly with strings, arrays, and paths
- **🔄 Loop Integration**: Perfect integration with `for each` loops
- **📊 Performance**: Efficient operations with minimal overhead

### 📊 Advanced Logging & Metrics

Beautiful, structured logging with performance tracking:

```yaml
recipes:
  performance-demo:
    help: "Demonstrate advanced logging and metrics"
    run: |
      {{ step "Starting performance monitoring" }}

      START_TIME=$(date +%s)

      {{ info "Running performance tests" }}
      for i in {1..5}; do
        {{ info "Test $i/5 - Load testing..." }}
        # Simulate work
        sleep 0.5
      done

      {{ info "Running stress tests" }}
      for i in {1..3}; do
        {{ info "Stress test $i/3 - Memory testing..." }}
        sleep 0.3
      done

      END_TIME=$(date +%s)
      DURATION=$((END_TIME - START_TIME))

      {{ success "Performance tests completed in ${DURATION}s" }}

      echo "📊 Metrics Summary:"
      echo "  Duration: ${DURATION}s"
      echo "  Tests: 8"
      echo "  Success Rate: 100%"
      echo "  Throughput: $(echo "scale=2; 8 / $DURATION" | bc) tests/sec"

  comprehensive-workflow:
    help: "Comprehensive workflow with all features"
    matrix:
      environment: ["dev", "staging", "prod"]
    deps: [setup]
    run: |
      {{ step "Deploying to {{ .matrix_environment }}" }}

      # Smart detection
      {{ info "Project: {{ packageManager }} on {{ gitBranch }}" }}
      {{ info "Environment: {{ .matrix_environment }}" }}
      {{ info "CI Mode: {{ isCI }}" }}

      # Conditional logic
      {{ if eq .matrix_environment "prod" }}
      {{ warn "Production deployment - extra validation" }}
      {{ if isDirty }}
      {{ error "Cannot deploy dirty working directory to production" }}
      exit 1
      {{ end }}
      {{ end }}

      # Use auto-detected commands
      {{ if hasFile "docker-compose.yml" }}
      {{ info "Using Docker Compose: {{ dockerCompose }}" }}
      {{ dockerCompose }} -f docker-compose.{{ .matrix_environment }}.yml up -d
      {{ end }}

      # Secrets integration
      {{ if hasSecret "deploy_token" }}
      {{ info "Authenticating with deployment token" }}
      # Use secret securely
      {{ else }}
      {{ warn "No deployment token - using default auth" }}
      {{ end }}

      {{ success "Deployment to {{ .matrix_environment }} completed!" }}
```

## Command Line Options

- `--init`: Initialize a new .drun task file
- `--list, -l`: List available recipes
- `--dry-run`: Show what would be executed without running
- `--explain`: Show rendered scripts and environment variables
- `--update`: Update drun to the latest version from GitHub releases
- `--file, -f`: Specify task file (default: auto-discover .drun files)
- `--jobs, -j`: Number of parallel jobs for dependencies
- `--set`: Set variables (KEY=VALUE format)
- `--shell`: Override shell type (linux/darwin/windows)
- `completion [bash|zsh|fish|powershell]`: Generate shell completion scripts
- `cleanup-backups`: Clean up old backup files created during updates

## Shell Completion

drun supports intelligent shell completion for bash, zsh, fish, and PowerShell. The completion includes:

- **Recipe names** with descriptions
- **Positional arguments** with named syntax support (`--name=value` and `name=value`)
- **Recipe-specific flags** with type information and defaults
- **Value completion** for `one_of` constraints
- **Context-aware suggestions** based on what's already been typed

### Installation

#### Bash

```bash
# Load completion for current session
source <(drun completion bash)

# Install permanently (Linux)
drun completion bash > /etc/bash_completion.d/drun

# Install permanently (macOS with Homebrew)
drun completion bash > $(brew --prefix)/etc/bash_completion.d/drun
```

#### Zsh

```bash
# Enable completion system (if not already enabled)
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Load completion for current session
source <(drun completion zsh)

# Install permanently
drun completion zsh > "${fpath[1]}/_drun"

# Restart your shell or source ~/.zshrc
```

#### Fish

```bash
# Load completion for current session
drun completion fish | source

# Install permanently
drun completion fish > ~/.config/fish/completions/drun.fish
```

#### PowerShell

```powershell
# Load completion for current session
drun completion powershell | Out-String | Invoke-Expression

# Install permanently
drun completion powershell > drun.ps1
# Then source this file from your PowerShell profile
```

### Completion Examples

```bash
# Recipe completion
drun <TAB>                    # Shows all recipes with descriptions
drun rel<TAB>                 # Completes to "release"

# Named argument completion  
drun release <TAB>            # Shows: version= --version= --arch= --push
drun release --<TAB>          # Shows: --version= --arch= --push
drun release version=<TAB>    # Shows available values if one_of is defined

# Mixed completion
drun release v1.0.0 <TAB>     # Shows remaining arguments: --arch= --push
drun release --arch=<TAB>     # Shows: amd64, arm64, both
```

## Self-Update & Backup Management

drun includes built-in self-update functionality with intelligent backup management.

### Update Command

```bash
# Check for and install updates
drun --update
```

The update process:

1. **Checks GitHub releases** for the latest version
2. **Creates a backup** in `~/.drun/backups/` (user-writable location)
3. **Downloads** the appropriate binary for your platform
4. **Replaces** the current binary safely
5. **Preserves the backup** for safety

### Backup Management

Backups are automatically created during updates and stored in user-writable locations:

- **Primary location**: `~/.drun/backups/`
- **Fallback location**: System temp directory
- **Naming format**: `drun.YYYY-MM-DD_HH-MM-SS.backup`

#### Cleanup Command

```bash
# List and clean up old backups (interactive)
drun cleanup-backups

# Keep only the 3 most recent backups
drun cleanup-backups --keep=3

# Remove all backup files
drun cleanup-backups --all
```

The cleanup command provides:

- **Interactive cleanup** with file listing and sizes
- **Selective retention** (keep N most recent backups)
- **Safety confirmation** before deletion
- **Size reporting** (freed disk space)

### Update Safety Features

- ✅ **User-writable backups** (no permission errors)
- ✅ **Automatic rollback** on update failure
- ✅ **Backup preservation** (not auto-deleted)
- ✅ **Platform detection** (correct binary for your system)
- ✅ **Version validation** (only update when newer version available)

## Template Functions

drun includes 20+ powerful built-in template functions plus all [Sprig](https://masterminds.github.io/sprig/) functions:

### 🐳 **Docker Integration**

- `{{ dockerCompose }}`: Auto-detect "docker compose" or "docker-compose"
- `{{ dockerBuildx }}`: Auto-detect "docker buildx" or "docker-buildx"
- `{{ hasCommand "kubectl" }}`: Check if command exists in PATH

### 🔗 **Git Integration**

- `{{ gitBranch }}`: Current Git branch name
- `{{ gitCommit }}`: Full commit hash (40 chars)
- `{{ gitShortCommit }}`: Short commit hash (7 chars)
- `{{ isDirty }}`: True if working directory has uncommitted changes

### 🌐 **HTTP Actions**

- `get "url"`: HTTP GET request with response capture
- `post "url" with body "data"`: HTTP POST with JSON body
- `put "url" with header "key: value"`: HTTP PUT with custom headers
- `delete "url" with auth bearer "token"`: HTTP DELETE with authentication

### 📦 **Project Detection**

- `{{ packageManager }}`: Auto-detect npm, yarn, pnpm, go, pip, etc.
- `{{ hasFile "go.mod" }}`: Check if file exists
- `{{ isCI }}`: Detect CI environment (GitHub Actions, GitLab CI, etc.)

### 📊 **Status Messages**

- `{{ step "message" }}`: 🚀 Step indicator
- `{{ info "message" }}`: ℹ️ Information message
- `{{ warn "message" }}`: ⚠️ Warning message
- `{{ error "message" }}`: ❌ Error message (non-fatal)
- `{{ success "message" }}`: ✅ Success message

### 🔐 **Secrets Management**

- `{{ secret "name" }}`: Access secret value securely
- `{{ hasSecret "name" }}`: Check if secret is available

### 🛠️ **Standard Functions**

- `{{ now "2006-01-02" }}`: Current time formatting
- `{{ .version }}`: Access positional arguments and variables
- `{{ env "HOME" }}`: Environment variables
- `{{ snippet "name" }}`: Include reusable snippets
- `{{ shellquote .arg }}`: Shell-safe quoting
- Plus all [Sprig](https://masterminds.github.io/sprig/) functions (150+ additional functions)

## Examples

Explore comprehensive examples in the `examples/` directory:

### 📚 **Example Files**

- **`examples/01-hello-world.drun`** - Basic introduction to drun v2
- **`examples/02-parameters.drun`** - Parameter handling and validation
- **`examples/03-interpolation.drun`** - Variable interpolation examples
- **`examples/04-docker-basics.drun`** - Docker operations and workflows
- **`examples/05-kubernetes.drun`** - Kubernetes deployment examples
- **`examples/06-cicd-pipeline.drun`** - CI/CD pipeline automation
- **`examples/07-final-showcase.drun`** - Comprehensive feature showcase
- **`examples/26-smart-detection.drun`** - Smart tool and environment detection

### 🎯 **Quick Examples**

```bash
# Try the hello world example
drun -f examples/01-hello-world.drun hello

# Test parameters and validation
drun -f examples/02-parameters.drun "deploy app" environment=dev

# Explore smart detection
drun -f examples/26-smart-detection.drun "detect project"

# See comprehensive features
drun -f examples/07-final-showcase.drun showcase project_name=MyApp
```

Each example includes comprehensive documentation and demonstrates best practices for different use cases.

## 🚀 Status & Roadmap

drun is **production-ready** with enterprise-grade features:

### ✅ **Implemented Features**

- **Core Functionality**: .drun semantic language, parameters, variables, control flow
- **Advanced Features**: Remote includes, matrix execution, secrets management
- **Developer Experience**: 15+ template functions, intelligent caching, rich errors
- **Performance**: Microsecond-level operations, high test coverage (71-83%)
- **Quality**: Zero linting issues, comprehensive test suite

### 🚧 **Coming Soon**

- **📁 File Watching**: Auto-execution on file changes
- **🔌 Plugin System**: Extensible architecture for custom functionality
- **🎮 Interactive TUI**: Beautiful terminal interface
- **🌐 Web UI**: Browser-based recipe management
- **🤖 AI Integration**: Natural language recipe generation

### 🎯 **Enterprise Ready**

- **High Performance**: Microsecond-level operations
- **Scalability**: Handles 100+ recipes efficiently  
- **Security**: Secure secrets management
- **Reliability**: Comprehensive error handling
- **Maintainability**: Clean architecture with extensive tests

drun has evolved from a simple task runner into a **comprehensive automation platform** that's ready for production use at any scale! 🏆

---

## 🛠️ Developer Guide

This section contains information for developers who want to build, test, or contribute to drun.

### Requirements

- **Go 1.25+** - drun requires Go 1.25 or later

### Build from Source

```bash
# Clone the repository
git clone https://github.com/phillarmonic/drun.git
cd drun

# Build drun
go build -o bin/drun ./cmd/drun

# Or use the build script for all platforms
./scripts/build.sh
```

### Testing

Run the comprehensive test suite (includes mandatory golangci-lint):

```bash
# Basic tests (includes linting, unit tests, build verification)
./scripts/test.sh

# With coverage report
./scripts/test.sh -c

# Verbose with race detection
./scripts/test.sh -v -r

# All options
./scripts/test.sh -v -c -r -b
```

Or run components manually:

```bash
# Linting (required - auto-installs golangci-lint if needed)
golangci-lint run ./...

# Unit tests only
go test ./internal/...

# With coverage
go test -cover ./internal/...

# CI-optimized test suite
./scripts/test-ci.sh
```

### Performance Benchmarks

drun is engineered for **high performance** and **low resource usage**. Extensive optimizations ensure fast execution even for large projects with complex dependency graphs.

#### Benchmark Results

Performance benchmarks on Apple M4 (your results may vary):

| Component              | Operation                | Time  | Memory  | Allocations |
| ---------------------- | ------------------------ | ----- | ------- | ----------- |
| **YAML Loading**       | Simple spec              | 2.5μs | 704 B   | 5 allocs    |
| **YAML Loading**       | Large spec (100 recipes) | 8.6μs | 756 B   | 5 allocs    |
| **Template Rendering** | Basic template           | 29μs  | 3.9 KB  | 113 allocs  |
| **Template Rendering** | Complex template         | 51μs  | 7.0 KB  | 93 allocs   |
| **DAG Building**       | Simple dependency graph  | 3.1μs | 10.7 KB | 109 allocs  |
| **DAG Building**       | Complex dependencies     | 3.9μs | 12.4 KB | 123 allocs  |
| **Topological Sort**   | 100 nodes                | 2.5μs | 8.0 KB  | 137 allocs  |

#### Optimization Impact

Our performance optimizations deliver significant improvements:

| Component              | Before       | After           | **Improvement**                   |
| ---------------------- | ------------ | --------------- | --------------------------------- |
| **Template Rendering** | 40μs, 60KB   | **29μs, 4KB**   | **1.4x faster, 15x less memory**  |
| **YAML Loading**       | 361μs, 42KB  | **2.5μs, 704B** | **144x faster, 59x less memory**  |
| **Large Spec Loading** | 3.4ms, 657KB | **8.6μs, 756B** | **396x faster, 869x less memory** |
| **DAG Building**       | 4.4μs, 14KB  | **3.1μs, 11KB** | **1.4x faster, 22% less memory**  |
| **Topological Sort**   | 4.7μs, 10KB  | **2.5μs, 8KB**  | **1.9x faster, 20% less memory**  |

#### Performance Features

- **⚡ Template Caching**: Compiled templates cached by hash for instant reuse
- **🧠 Smart Pre-allocation**: Memory pools and capacity-aware data structures
- **📊 Spec Caching**: YAML specs cached with file modification tracking
- **🔄 Optimized DAG**: Highly efficient dependency graph construction
- **💾 Memory Pools**: Reusable objects reduce GC pressure
- **🎯 Lazy Evaluation**: Only compute what's needed when needed

#### Real-World Performance

- **Startup time**: Sub-millisecond for cached specs
- **Large projects**: 100+ recipes process in microseconds
- **Memory usage**: Minimal footprint with intelligent caching
- **Parallel execution**: Efficient DAG-based task scheduling
- **Template rendering**: Up to 20x faster than naive implementations

Run benchmarks yourself:

```bash
./scripts/test.sh -b  # Includes comprehensive performance benchmarks
```

## 📚 Documentation

- **[Template Functions Reference](TEMPLATE_FUNCTIONS.md)** - Complete guide to all built-in template functions
- **[drun v2 Specification](DRUN_V2_SPECIFICATION.md)** - Complete v2 language specification with HTTP actions
- **[YAML Specification](YAML_SPEC.md)** - Legacy v1 YAML reference (deprecated)
- **[Examples Directory](examples/)** - Real-world usage examples and patterns
