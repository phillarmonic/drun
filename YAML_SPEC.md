# drun YAML Specification

This document provides a complete reference for the drun YAML configuration format.

## File Structure

drun automatically looks for configuration files in this order:

1. **Workspace default** (if configured in `.drun/workspace.yml`)
2. **Standard locations**:
   - `drun.yml`
   - `drun.yaml` 
   - `.drun.yml`
   - `.drun.yaml`
   - `.drun/drun.yml`
   - `.drun/drun.yaml`
   - `ops.drun.yml`
   - `ops.drun.yaml`

### Workspace Configuration

drun supports workspace-specific default configuration files. When you use `drun --init` with a custom file path, you can save it as the workspace default:

```yaml
# .drun/workspace.yml
default_config_file: .drun/drun.yml
```

This allows teams to use consistent configuration file locations across different projects.

## Complete Configuration Reference

```yaml
# Required: Configuration format version
version: 1.0

# Optional: Shell configuration per OS
shell:
  linux:
    cmd: "/bin/sh"
    args: ["-ceu"]
  darwin:
    cmd: "/bin/zsh" 
    args: ["-ceu"]
  windows:
    cmd: "pwsh"
    args: ["-NoLogo", "-Command"]

# Optional: Global environment variables
env:
  # Static values
  REGISTRY: "ghcr.io"
  ORG: "myorg"
  
  # Templated values (evaluated at runtime)
  BUILD_DATE: '{{ now "2006-01-02T15:04:05Z" }}'
  GIT_COMMIT: '{{ gitShortCommit }}'

# Optional: Template variables
vars:
  app_name: "myapp"
  version: 1.0.0
  # Can reference env and other vars
  image_tag: "{{ .version }}-{{ env \"BUILD_ENV\" }}"

# Optional: Global defaults for all recipes
defaults:
  working_dir: "."
  shell: "auto"              # or: linux/darwin/windows
  export_env: true
  timeout: "2h"
  inherit_env: true
  strict: true               # Fail on missing template variables

# Optional: Reusable code snippets
snippets:
  docker_login: |
    if ! docker login {{ .REGISTRY }}; then
      echo "Login failed"
      exit 1
    fi
  
  setup_node: |
    echo "Setting up Node.js environment"
    node --version
    npm --version
  
  setup_colors: |
    # ANSI color codes
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    PURPLE='\033[0;35m'
    CYAN='\033[0;36m'
    NC='\033[0m' # No Color
  
  setup_env: |
    echo "Setting up environment for {{ .project_name }}"
    export PROJECT_ROOT="$(pwd)"
    export PROJECT_VERSION="{{ .version }}"

# Optional: Snippets that run before EVERY recipe (RENAMED for clarity!)
recipe-prerun:
  # Can call existing snippets with full template support
  - '{{ snippet "setup_colors" }}'
  - '{{ snippet "setup_env" }}'
  # Or include inline code
  - |
    # Common shell settings
    set -euo pipefail
  - |
    # Helper functions available to all recipes
    log_info() {
      echo -e "${BLUE}ℹ️  $1${NC}"
    }
    log_success() {
      echo -e "${GREEN}✅ $1${NC}"
    }
    log_warning() {
      echo -e "${YELLOW}⚠️  $1${NC}"
    }
    log_error() {
      echo -e "${RED}❌ $1${NC}"
    }

# Optional: Snippets that run after EVERY recipe (NEW FEATURE!)
recipe-postrun:
  # Cleanup or logging that happens after each recipe
  - |
    # Log recipe completion
    if [ $? -eq 0 ]; then
      log_success "Recipe completed successfully"
    else
      log_error "Recipe failed with exit code $?"
    fi
  - |
    # Optional cleanup after each recipe
    # Clean up temporary files, reset state, etc.
    echo "Recipe execution finished at $(date)"

# Optional: Lifecycle blocks that run ONCE before any recipe execution (NEW FEATURE!)
before:
  - |
    # Setup that runs once before any recipe
    echo "🚀 Starting drun execution..."
    export DRUN_START_TIME=$(date +%s)
  - |
    # Initialize shared resources
    mkdir -p .drun/tmp
    echo "Initialized temporary directory"

# Optional: Lifecycle blocks that run ONCE after all recipe execution (NEW FEATURE!)
after:
  - |
    # Cleanup that runs once after all recipes
    echo "🧹 Cleaning up temporary files..."
    rm -rf .drun/tmp
  - |
    # Show execution summary
    if [ -n "${DRUN_START_TIME:-}" ]; then
      DURATION=$(($(date +%s) - DRUN_START_TIME))
      echo "✅ Execution completed in ${DURATION}s"
    fi

# Optional: Include other configuration files
include:
  # Local files (supports globs)
  - "shared/*.yml"
  - "tasks/docker.yml"
  
  # Remote HTTP/HTTPS includes
  - "https://raw.githubusercontent.com/company/drun-recipes/main/common.yml"
  
  # Git repositories
  - "git+https://github.com/company/recipes.git@main:docker/common.yml"
  - "git+https://github.com/company/recipes.git@v1.0.0"  # Uses drun.yml
  
  # Namespaced includes (NEW FEATURE!) - prevents recipe name collisions
  - "docker::shared/docker-tasks.yml"           # Recipes prefixed with "docker:"
  - "k8s::git+https://github.com/company/k8s-recipes.git@main"  # Recipes prefixed with "k8s:"
  - "common::https://example.com/common-tasks.yml"  # Recipes prefixed with "common:"

# Optional: Secrets management
secrets:
  api_key:
    source: "env://API_KEY"
    required: true
    description: "API key for external service"
  
  db_password:
    source: "file://~/.secrets/db-password"
    required: false
    description: "Database password"

# Required: Recipe definitions
recipes:
  # Simple recipe
  hello:
    help: "Say hello to the world"
    run: |
      echo "Hello, World!"

  # Recipe with positional arguments
  greet:
    help: "Greet someone by name"
    positionals:
      - name: name
        required: true
        pattern: "^[A-Za-z]+$"     # Optional regex validation
      - name: title
        default: "friend"
        one_of: ["Mr.", "Ms.", "Dr.", "friend"]  # Optional enum validation
      - name: extras
        variadic: true             # Accepts multiple values
    run: |
      log_info "Greeting {{ .title }} {{ .name }}"
      echo "Hello, {{ .title }} {{ .name }}!"
      {{ if .extras }}
      echo "Extra info: {{ range .extras }}{{ . }} {{ end }}"
      {{ end }}

  # Recipe with flags
  build:
    help: "Build the application"
    flags:
      push:
        type: bool
        default: false
        help: "Push to registry after build"
      
      tag:
        type: string
        default: "latest"
        help: "Docker tag to use"
      
      platforms:
        type: string[]
        default: ["linux/amd64"]
        help: "Target platforms for build"
      
      parallel:
        type: int
        default: 1
        help: "Number of parallel jobs"
    
    # Recipe-specific variables
    vars:
      dockerfile: "Dockerfile"
    
    # Recipe-specific environment
    env:
      DOCKER_BUILDKIT: "1"
    
    run: |
      log_info "Building {{ .app_name }} with tag {{ .flags.tag }}"
      
      docker build \
        -f {{ .dockerfile }} \
        -t {{ .app_name }}:{{ .flags.tag }} \
        {{ range .flags.platforms }}--platform {{ . }} {{ end }} \
        .
      
      {{ if .flags.push }}
      log_info "Pushing to registry"
      docker push {{ .app_name }}:{{ .flags.tag }}
      {{ end }}
      
      log_success "Build completed!"

  # Recipe with dependencies
  test:
    help: "Run tests"
    deps: ["build"]              # Run build first
    parallel_deps: false         # Run deps sequentially (default)
    run: |
      log_info "Running tests for {{ .app_name }}"
      go test ./...
      log_success "All tests passed!"

  # Recipe with multiple steps
  deploy:
    help: "Deploy the application"
    deps: ["build", "test"]
    parallel_deps: true          # Run deps in parallel
    
    # Recipe-specific settings
    working_dir: "./deploy"
    timeout: "10m"
    ignore_error: false
    shell: "bash"                # Override global shell
    
    # Matrix execution (run recipe multiple times with different values)
    matrix:
      environment: ["staging", "production"]
      region: ["us-east-1", "eu-west-1"]
    
    # Caching
    cache_key: "deploy-{{ .matrix.environment }}-{{ .matrix.region }}-{{ .version }}"
    
    run:
      - |
        # Step 1: Preparation
        log_info "Preparing deployment to {{ .matrix.environment }} in {{ .matrix.region }}"
        {{ snippet "docker_login" }}
      
      - |
        # Step 2: Deploy
        log_info "Deploying application"
        kubectl apply -f k8s/{{ .matrix.environment }}/
        
      - |
        # Step 3: Verify
        log_info "Verifying deployment"
        kubectl rollout status deployment/{{ .app_name }}
        log_success "Deployment to {{ .matrix.environment }} completed!"

  # Recipe with aliases
  ci:
    help: "Run CI pipeline"
    aliases: ["continuous-integration", "pipeline"]
    deps: ["test", "build"]
    run: |
      log_info "Running CI pipeline"
      # CI-specific commands here
      log_success "CI pipeline completed!"

# Optional: Caching configuration
cache:
  path: ".drun/cache"
  keys:
    - "build-{{ .version }}"
    - "test-{{ gitShortCommit }}"
```

## Field Reference

### Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | number | ✅ | Configuration format version (currently `1.0`) |
| `shell` | object | ❌ | Shell configuration per OS |
| `env` | object | ❌ | Global environment variables |
| `vars` | object | ❌ | Template variables |
| `defaults` | object | ❌ | Global defaults for all recipes |
| `snippets` | object | ❌ | Reusable code snippets |
| `recipe-prerun` | array | ❌ | **RENAMED:** Snippets that run before every recipe |
| `recipe-postrun` | array | ❌ | **NEW:** Snippets that run after every recipe |
| `before` | array | ❌ | **NEW:** Lifecycle blocks that run once before any recipe execution |
| `after` | array | ❌ | **NEW:** Lifecycle blocks that run once after all recipe execution |
| `include` | array | ❌ | Include other configuration files (supports namespacing with `namespace::path`) |
| `secrets` | object | ❌ | Secret definitions |
| `recipes` | object | ✅ | Recipe definitions |
| `cache` | object | ❌ | Caching configuration |

### Recipe Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `help` | string | ❌ | Help text for the recipe |
| `positionals` | array | ❌ | Positional argument definitions |
| `flags` | object | ❌ | Command-line flag definitions |
| `vars` | object | ❌ | Recipe-specific variables |
| `env` | object | ❌ | Recipe-specific environment variables |
| `deps` | array | ❌ | Recipe dependencies |
| `parallel_deps` | bool | ❌ | Run dependencies in parallel (default: false) |
| `run` | string/array | ✅ | Commands to execute |
| `working_dir` | string | ❌ | Working directory for execution |
| `shell` | string | ❌ | Shell override for this recipe |
| `timeout` | duration | ❌ | Execution timeout |
| `ignore_error` | bool | ❌ | Continue on command failure |
| `aliases` | array | ❌ | Alternative names for the recipe |
| `matrix` | object | ❌ | Matrix execution parameters |
| `cache_key` | string | ❌ | Cache key for this recipe |

### Positional Argument Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Argument name |
| `required` | bool | ❌ | Whether argument is required |
| `default` | string | ❌ | Default value if not provided |
| `one_of` | array | ❌ | List of allowed values |
| `pattern` | string | ❌ | Regex pattern for validation |
| `variadic` | bool | ❌ | Accept multiple values |

### Flag Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | ✅ | Flag type: `string`, `int`, `bool`, `string[]` |
| `default` | any | ❌ | Default value |
| `help` | string | ❌ | Help text for the flag |

### Secret Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source` | string | ✅ | Secret source: `env://VAR`, `file://path` |
| `required` | bool | ❌ | Whether secret is required |
| `description` | string | ❌ | Human-readable description |

## Execution Lifecycle (New Feature)

drun now supports different execution phases with dedicated lifecycle blocks:

### Execution Phases

1. **Startup**: Import and parse all configuration files
2. **Before**: Run `before` blocks once before any recipe execution
3. **Recipes**: Execute the requested recipes and their dependencies
4. **After**: Run `after` blocks once after all recipe execution (even on failure)

### Lifecycle Blocks

#### Before Blocks
The `before` section contains blocks that run **once** before any recipe execution:

```yaml
before:
  - |
    echo "🚀 Starting drun execution..."
    export DRUN_START_TIME=$(date +%s)
  - |
    mkdir -p .drun/tmp
    echo "Initialized shared resources"
```

#### After Blocks
The `after` section contains blocks that run **once** after all recipe execution:

```yaml
after:
  - |
    echo "🧹 Cleaning up..."
    rm -rf .drun/tmp
  - |
    if [ -n "${DRUN_START_TIME:-}" ]; then
      DURATION=$(($(date +%s) - DRUN_START_TIME))
      echo "✅ Completed in ${DURATION}s"
    fi
```

#### Recipe-Prerun Snippets
The `recipe-prerun` section contains snippets that run before **every** recipe:

```yaml
recipe-prerun:
  - |
    # Colors available in all recipes
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    NC='\033[0m'
  - |
    # Helper function available everywhere
    success() {
      echo -e "${GREEN}✅ $1${NC}"
    }
```

#### Recipe-Postrun Snippets
The `recipe-postrun` section contains snippets that run after **every** recipe:

```yaml
recipe-postrun:
  - |
    # Log completion status
    if [ $? -eq 0 ]; then
      echo -e "${GREEN}✅ Recipe completed successfully${NC}"
    else
      echo -e "${RED}❌ Recipe failed${NC}"
    fi
  - |
    # Cleanup after each recipe
    echo "Recipe finished at $(date)"
```

### Use Cases

- **Before**: Database setup, resource initialization, environment validation
- **After**: Cleanup, reporting, resource teardown, notifications
- **Recipe-Prerun**: Color definitions, helper functions, common shell settings
- **Recipe-Postrun**: Status logging, per-recipe cleanup, metrics collection

## Recipe Namespacing (New Feature)

Prevent recipe name collisions when including multiple configuration files by using namespaces.

### Namespace Syntax

Use the `namespace::path` syntax in include statements:

```yaml
include:
  - "docker::shared/docker-tasks.yml"     # Recipes become "docker:build", "docker:push", etc.
  - "k8s::tasks/kubernetes.yml"           # Recipes become "k8s:deploy", "k8s:rollback", etc.
  - "common::https://example.com/common.yml"  # Remote includes also support namespacing
```

### Using Namespaced Recipes

Execute namespaced recipes using the full name:

```bash
# Execute a namespaced recipe
drun docker:build

# List all recipes (shows both local and namespaced)
drun --list
```

### Benefits

- **Collision Prevention**: Multiple files can have recipes with the same name
- **Organization**: Clear separation of recipe sources
- **Flexibility**: Mix local and remote includes without conflicts
- **Clarity**: Easy to see which file a recipe comes from

### Example

```yaml
# main drun.yml
include:
  - "docker::shared/docker.yml"
  - "k8s::shared/kubernetes.yml"

recipes:
  # Local recipe
  build:
    run: echo "Local build"

# shared/docker.yml contains:
# recipes:
#   build:  # Becomes "docker:build"
#     run: docker build .
#   push:   # Becomes "docker:push" 
#     run: docker push

# shared/kubernetes.yml contains:
# recipes:
#   deploy: # Becomes "k8s:deploy"
#     run: kubectl apply -f k8s/
```

Result: You can run `drun build`, `drun docker:build`, or `drun k8s:deploy` without conflicts.

## Template Functions

drun provides 15+ built-in template functions:

### Variable Access

Recipe flags, positional arguments, variables, and environment can be accessed in templates:

```yaml
recipes:
  example:
    flags:
      verbose: { type: bool, default: false }
      output: { type: string, default: "json" }
    positionals:
      - name: target
        required: true
    vars:
      app_name: "myapp"
    env:
      BUILD_ENV: "production"
    run: |
      # Flag access (both syntaxes work)
      {{ if .flags.verbose }}echo "Verbose mode"{{ end }}
      {{ if .verbose }}echo "Also verbose mode"{{ end }}
      
      # Other variable access
      echo "Building {{ .app_name }} for {{ .target }}"
      echo "Environment: {{ .BUILD_ENV }}"
```

**Flag Access Patterns:**
- `{{ .flags.flagname }}` - Explicit namespaced access (recommended)
- `{{ .flagname }}` - Direct access (also works)

### Standard Functions
- `{{ now "2006-01-02" }}` - Current time formatting
- `{{ env "HOME" }}` - Environment variables  
- `{{ snippet "name" }}` - Include reusable snippets
- `{{ shellquote .arg }}` - Shell-safe quoting

### Detection Functions
- `{{ dockerCompose }}` - Auto-detect `docker compose` vs `docker-compose`
- `{{ dockerBuildx }}` - Auto-detect `docker buildx` vs `docker-buildx`
- `{{ hasCommand "git" }}` - Check if command exists
- `{{ packageManager }}` - Detect npm, yarn, pnpm, etc.

### Git Functions
- `{{ gitBranch }}` - Current Git branch
- `{{ gitCommit }}` - Full commit hash
- `{{ gitShortCommit }}` - Short commit hash
- `{{ isDirty }}` - Working directory has changes

### Status Functions
- `{{ info "message" }}` - Info message with emoji
- `{{ warn "message" }}` - Warning message with emoji
- `{{ error "message" }}` - Error message with emoji
- `{{ success "message" }}` - Success message with emoji
- `{{ step "message" }}` - Step message with emoji

### System Functions
- `{{ os }}` - Operating system (linux/darwin/windows)
- `{{ arch }}` - Architecture (amd64/arm64)
- `{{ hostname }}` - System hostname
- `{{ isCI }}` - Detect CI environment

Plus all [Sprig functions](https://masterminds.github.io/sprig/) (150+ additional functions).

## Examples

See the `examples/` directory for comprehensive configuration examples:

- `examples/simple.yml` - Basic recipes
- `examples/prerun-demo.yml` - Recipe-prerun and recipe-postrun snippets showcase
- `examples/feature-showcase.yml` - Advanced features
- `examples/matrix-demo.yml` - Matrix execution
- `examples/remote-includes-demo.yml` - Remote includes
- `examples/secrets-demo.yml` - Secrets management

## Migration Guide

### From v0.0.x to v0.1

- Add `version: 1.0` to your configuration
- `recipe-prerun` and `recipe-postrun` fields are new and optional
- **Enhanced flag access**: Both `{{ .flagname }}` and `{{ .flags.flagname }}` syntaxes now work
- All existing configurations remain compatible
