# drun (do run)

A **high-performance** YAML-based task runner with first-class arguments, powerful templating, and intelligent dependency management. Optimized for speed with microsecond-level operations and minimal memory usage.

## Features

### 🚀 **Core Features**

- **YAML Configuration**: Define tasks in a simple, readable YAML format
- **Positional Arguments**: First-class support for positional arguments with validation
- **Named Arguments**: Pass positional arguments by name for clarity (`--name=value` or `name=value`)
- **Prerun Snippets**: Common setup code that runs before every recipe (colors, functions, etc.)
- **Templating**: Powerful Go template engine with custom functions and caching
- **Dependencies**: Automatic dependency resolution and parallel execution
- **High Performance**: Microsecond-level operations with intelligent caching
- **Cross-Platform**: Works on Linux, macOS, and Windows with appropriate shell selection
- **Dry Run & Explain**: See what would be executed without running it
- **Recipe Flags**: Command-line flags specific to individual recipes

### 🌟 **Advanced Features**

- **🔗 Remote Includes**: Include recipes from HTTP/HTTPS URLs and Git repositories
- **🔄 Matrix Execution**: Run recipes across multiple configurations (OS, versions, architectures)
- **🔐 Secrets Management**: Secure handling of sensitive data with multiple sources
- **📊 Advanced Logging**: Structured logging with emoji status messages and metrics
- **🎯 Smart Detection**: Auto-detect Docker commands, Git info, package managers, and CI environments
- **📁 File Watching**: Auto-execution on file changes (coming soon)

### 🛠️ **Developer Experience**

- **15+ Template Functions**: Docker detection, Git integration, status messages, and more
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

1. **Initialize a new project**:
   
   ```bash
   drun --init
   ```

2. **List available recipes**:
   
   ```bash
   drun --list
   ```

3. **Run a recipe called build**:
   
   ```bash
   drun build
   ```

4. **Use positional arguments**:
   
   ```bash
   drun release v1.0.0 amd64
   ```

5. **Use named arguments for clarity**:
   
   ```bash
   # Flag-style named arguments
   drun release --version=v1.0.0 --arch=amd64
   
   # Assignment-style named arguments
   drun release version=v1.0.0 arch=amd64
   
   # Mix positional and named
   drun release v1.0.0 --arch=amd64
   ```

6. **Dry run to see what would execute**:
   
   ```bash
   drun build --dry-run
   ```

## Configuration

drun automatically looks for configuration files in this order:

- `drun.yml`
- `drun.yaml` 
- `.drun.yml`
- `.drun.yaml`
- `.drun/drun.yml`
- `.drun/drun.yaml`
- `ops.drun.yml`
- `ops.drun.yaml`

Use `drun --init` to create a starter configuration, or see the included examples for comprehensive configurations.

📖 **For complete YAML specification**: See [YAML_SPEC.md](YAML_SPEC.md) for detailed field reference and examples.

### Basic Recipe

```yaml
version: 0.1

recipes:
  hello:
    help: "Say hello"
    run: |
      echo "Hello, World!"
```

### Recipe with Positional Arguments

```yaml
recipes:
  greet:
    help: "Greet someone"
    positionals:
      - name: name
        required: true
      - name: title
        default: "friend"
    run: |
      echo "Hello, {{ .title }} {{ .name }}!"
```

**Usage examples:**

```bash
# Traditional positional arguments
drun greet Alice
drun greet Bob Mr.

# Named arguments (flag-style)
drun greet --name=Alice --title=Ms.

# Named arguments (assignment-style)  
drun greet name=Bob title=Dr.

# Mixed usage
drun greet Alice --title=Ms.
```

### Advanced Named Arguments

```yaml
recipes:
  deploy:
    help: "Deploy to environment with version"
    positionals:
      - name: environment
        required: true
        one_of: ["dev", "staging", "prod"]
      - name: version
        default: "latest"
      - name: features
        variadic: true
    flags:
      force:
        type: bool
        default: false
    run: |
      echo "Deploying {{ .version }} to {{ .environment }}"
      {{ if .features }}echo "Features: {{ range .features }}{{ . }} {{ end }}"{{ end }}
      {{ if .force }}echo "Force deployment enabled"{{ end }}
```

**Usage examples:**

```bash
# All positional
drun deploy prod v1.2.3 feature1 feature2 --force

# All named arguments
drun deploy --environment=prod --version=v1.2.3 --force

# Mixed style
drun deploy prod --version=v1.2.3 --force

# Assignment style with variadic
drun deploy environment=staging version=v1.1.0 features=auth,ui --force
```

### Recipe with Dependencies

```yaml
recipes:
  test:
    help: "Run tests"
    deps: [build]
    run: |
      go test ./...

  build:
    help: "Build the project"
    run: |
      go build ./...
```

### Prerun Snippets - DRY Common Setup

Define snippets that automatically run before every recipe - perfect for colors, helper functions, and common setup:

```yaml
version: 0.1

# Snippets that run before EVERY recipe
prerun:
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
version: 0.1

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

- `--init`: Initialize a new drun.yml configuration file
- `--list, -l`: List available recipes
- `--dry-run`: Show what would be executed without running
- `--explain`: Show rendered scripts and environment variables
- `--update`: Update drun to the latest version from GitHub releases
- `--file, -f`: Specify configuration file (default: auto-discover)
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

drun includes 15+ powerful built-in template functions plus all [Sprig](https://masterminds.github.io/sprig/) functions:

### 🐳 **Docker Integration**

- `{{ dockerCompose }}`: Auto-detect "docker compose" or "docker-compose"
- `{{ dockerBuildx }}`: Auto-detect "docker buildx" or "docker-buildx"
- `{{ hasCommand "kubectl" }}`: Check if command exists in PATH

### 🔗 **Git Integration**

- `{{ gitBranch }}`: Current Git branch name
- `{{ gitCommit }}`: Full commit hash (40 chars)
- `{{ gitShortCommit }}`: Short commit hash (7 chars)
- `{{ isDirty }}`: True if working directory has uncommitted changes

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

- **`examples/simple.yml`** - Basic recipes and patterns
- **`examples/docker-devops.yml`** - Docker workflows with auto-detection
- **`examples/includes-demo.yml`** - Local and remote includes
- **`examples/matrix-working-demo.yml`** - Matrix execution examples
- **`examples/secrets-demo.yml`** - Secrets management patterns
- **`examples/logging-demo.yml`** - Advanced logging and metrics
- **`examples/feature-showcase.yml`** - All features in one place
- **`examples/remote-includes-showcase.yml`** - Remote includes deep dive

### 🎯 **Quick Examples**

```bash
# Try the feature showcase
drun -f examples/feature-showcase.yml showcase-all

# Test matrix execution
drun -f examples/matrix-working-demo.yml test-matrix

# Explore remote includes
drun -f examples/remote-includes-showcase.yml show-remote-capabilities

# See smart template functions
drun -f examples/feature-showcase.yml smart-build
```

Each example includes comprehensive documentation and demonstrates best practices for different use cases.

## 🚀 Status & Roadmap

drun is **production-ready** with enterprise-grade features:

### ✅ **Implemented Features**

- **Core Functionality**: YAML config, positional args, templating, dependencies
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
./build.sh
```

### Testing

Run the comprehensive test suite (includes mandatory golangci-lint):

```bash
# Basic tests (includes linting, unit tests, build verification)
./test.sh

# With coverage report
./test.sh -c

# Verbose with race detection
./test.sh -v -r

# All options
./test.sh -v -c -r -b
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
./test-ci.sh
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
./test.sh -b  # Includes comprehensive performance benchmarks
```
