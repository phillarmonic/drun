# drun (do run)

**drun** is a semantic, English-like task automation language with intelligent execution, smart detection, and powerful built-in actions. Write automation tasks in natural language that compiles to efficient shell commands.

**xdrun** (eXecute drun) is the CLI interpreter that runs the drun language and executes `.drun` files.

> **For Developers:** Want to contribute or understand the architecture? See the **[Developer Guide](./DEVELOPER_GUIDE.md)** for complete documentation on the codebase, architecture diagrams, and contribution guidelines.

## Quick Start

```bash
# Run a task
xdrun build

# Pass parameters using key=value syntax (NO dashes!)
xdrun deploy environment=production replicas=5

# List available tasks
xdrun --list

# Dry run to see what would execute
xdrun deploy --dry-run
```

**Important:** Task parameters use simple `key=value` syntax without `--` dashes. CLI flags (like `--dry-run`, `--list`) use `--` as they control xdrun behavior, not task parameters.

### Small example of a Drun language script:

```drun
# Add this file to .drun/spec.drun and see the magic happen
version: 2.0 # The drun spec version is required

# if you just run xdrun with nothing, this will run
task "default" means "This action will run by default":
  step "Welcome to drun v2!"
  success "Finished"

# Run 'xdrun hello' to run this task
task "hello":
  echo "Hello world!"

```

## Features

### Core Features

- **Semantic Language**: Write tasks in English-like syntax that's intuitive and readable
- **Smart Parameters**: Two types of parameters with clear semantics:
  - `requires` - Mandatory parameters (must be provided unless default specified)
  - `given` - Optional parameters (always have defaults)
  - Type-safe with constraints, enums, and validation
- **Variable System**: Powerful variable interpolation with `$variable` syntax, `$globals` namespace, and built-in functions
- **Multi-line Strings**: Write complex shell commands across multiple lines with line continuation (`\`), escaped quotes (`\"`), and full interpolation support
- **Control Flow**: Natural `if/else`, `for each`, `when` statements with intelligent conditions
- **Built-in Actions**: Docker, Kubernetes (soon), Git, HTTP operations with semantic commands
- **Smart Detection**: Auto-detect project types, tools, and environments
- **Shell Integration**: Seamless shell command execution with output capture
- **Cross-Platform**: Works on Linux, macOS, and Windows with intelligent shell selection
- **Dry Run & Explain**: See what would be executed without running it
- **Type Safety**: Static analysis with runtime validation

### Advanced Features

- **Code Reuse**: Project-level parameters, reusable snippets, task templates, and namespaced includes for DRY automation
- **Project Declarations**: Define global project settings, includes, and lifecycle hooks
- **Dependency System**: Automatic task dependency resolution with parallel execution
- **Task Calling**: Call tasks from within other tasks with parameter passing (`call task "name" with param="value"`)
- **HTTP Actions**: Built-in HTTP requests with authentication and response handling
- **Docker Integration**: Semantic Docker commands (`build docker image`, `run container`)
- **Kubernetes Support**: Native kubectl operations with intelligent resource management (soon)
- **Error Handling**: Comprehensive `try/catch/finally` with custom error types
- **Parallel Execution**: True parallel loops with concurrency control and progress tracking
- **Progress & Timing**: Built-in progress indicators and timer functions for long-running operations
- **Smart Detection**: Auto-detect tools, frameworks, and environments intelligently
- **DRY Tool Detection**: Detect tool variants and capture working ones (`detect available "docker compose" or "docker-compose" as $compose_cmd`)
- **File Operations**: Built-in file system operations with path interpolation
- **Pattern Macros**: Built-in validation patterns (`matching semver`, `matching uuid`, `matching url`) with descriptive error messages
- **Advanced Variable Operations**: Powerful data transformation (`{$files filtered by extension '.js' | sorted by name}`, `{$version without prefix 'v'}`, `{$path basename}`)

### Developer Experience

- **20+ Template Functions**: Docker detection, Git integration, HTTP calls, status messages, and more
- **Intelligent Caching**: HTTP and Git includes cached for performance
- **Rich Error Messages**: Helpful suggestions and context for debugging
- **Shell Completion**: Intelligent completion for bash, zsh, fish, and PowerShell
- **Self-Update**: Built-in update mechanism with backup management

### Built-in Functions

drun includes powerful built-in functions for common operations:

#### **System Information**

- `{hostname}` - Get system hostname
- `{pwd}` - Get current working directory
- `{pwd('basename')}` - Get directory name only
- `{current file}` - Get path to the current drun file being executed
- `{env('VAR_NAME')}` - Get environment variable
- `{env('VAR_NAME', 'default')}` - Get environment variable with default

#### **Time & Date**

- `{now.format('2006-01-02 15:04:05')}` - Format current time
- `{now.format('Monday, January 2, 2006')}` - Custom date formats

#### **File System**

- `{file exists('path/to/file')}` - Check if file exists (returns "true"/"false")
- `{dir exists('path/to/dir')}` - Check if directory exists (returns "true"/"false")

#### **Git Integration**

- `{current git commit}` - Get full commit hash
- `{current git commit('short')}` - Get short commit hash

#### **Progress & Timing** *New*

- `{start progress('message')}` - Start a progress indicator
- `{update progress('50', 'message')}` - Update progress with percentage and message
- `{finish progress('message')}` - Complete progress indicator
- `{start timer('name')}` - Start a named timer
- `{stop timer('name')}` - Stop a timer and show elapsed time
- `{show elapsed time('name')}` - Show current elapsed time for a timer

#### **Usage Examples**

```drun
version: 2.0
project "my-app" version "1.0":

task "system info":
  info "Running on: {hostname}"
  info "Current directory: {pwd('basename')}"
  info "Current file: {current file}"
  info "Current time: {now.format('2006-01-02 15:04:05')}"
  info "Git commit: {current git commit('short')}"

task "progress demo":
  info "{start progress('Building application')}"
  info "{update progress('25', 'Compiling sources')}"
  info "{update progress('50', 'Running tests')}"
  info "{update progress('75', 'Creating package')}"
  info "{finish progress('Build completed successfully!')}"

task "timing demo":
  info "{start timer('build_time')}"
  # ... build operations ...
  info "{stop timer('build_time')}"
  info "Total build time: {show elapsed time('build_time')}"
```

## Why drun?

### The Problem: Make and Just aren't semantic enough

**Make** has confusing behaviors (`.PHONY` targets, `$$` for variables, cryptic errors) that make it unsuitable as a general task runner.

**Just** improved on Make but is still too technical - it's essentially parameterized shell scripts. Can your project manager understand `{{variable}}` syntax and bash operators?

### The Solution: Truly Semantic Automation

**drun is automation that everyone on your team can read and understand.**

Compare the same deployment task:

**Just version** (technical, shell-centric):

```just
deploy env version:
  #!/bin/bash
  set -euo pipefail
  docker build -t app:{{version}} . && \
  kubectl set image deploy/app app=app:{{version}} || \
  (kubectl rollout undo deploy/app && exit 1)
```

**drun version** (semantic, readable):

```drun
task "deploy" means "Deploy with automatic rollback":
  # Critical parameters (must provide)
  requires $environment from ["dev", "staging", "production"]
  requires $version as string matching semver
  
  # Optional configuration (has defaults)
  given $replicas defaults to "3"
  given $timeout defaults to "5m"

  docker build image "app:{$version}"
  kubectl set image deployment/app to "app:{$version}"

  try:
    kubectl wait for rollout deployment/app
    success "Deployment successful!"
  catch:
    warn "Deployment failed - rolling back"
    kubectl rollback deployment/app
    fail "Deployment rolled back"
```

### What makes drun different:

- **Readable by Everyone**: Managers, QA, DevOps - everyone understands the automation
- **Semantic Actions**: `docker build image`, `kubectl wait for rollout` - reads like English
- **Built-in Safety**: Type validation (`matching semver`), constraints (`from ["dev", "prod"]`)
- **Native Tool Support**: Docker, Git, Kubernetes, HTTP - drun speaks their language
- **Self-Documenting**: The `means` clause and clear syntax eliminate documentation drift
- **Smart Detection**: Auto-detect tool variants (`docker compose` vs `docker-compose`)

**When to use drun:**

- Your automation needs to be reviewed by non-technical stakeholders
- You want CI/CD pipelines that are actually readable
- You need built-in validation and type safety
- You're tired of debugging shell scripts
- Your team includes product, QA, and management who need to understand deployment flows

**[Read the full story: Why we created drun →](WHY_DRUN.md)**

---

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/phillarmonic/drun/releases):

| Platform    | Architecture  | Download                                   |
| ----------- | ------------- | ------------------------------------------ |
| **Linux**   | x86_64        | `xdrun-linux-amd64` (UPX compressed)       |
| **Linux**   | ARM64         | `xdrun-linux-arm64` (UPX compressed)       |
| **macOS**   | Intel         | `xdrun-darwin-amd64`                       |
| **macOS**   | Apple Silicon | `xdrun-darwin-arm64`                       |
| **Windows** | x86_64        | `xdrun-windows-amd64.exe` (UPX compressed) |
| **Windows** | ARM64         | `xdrun-windows-arm64.exe`                  |

All binaries are **statically linked** and have **no dependencies**.

### Install Script

```bash
# Install latest version (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash

# Install specific version
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s v2.10.0
```

### Troubleshooting Installation

#### macOS: "signal: killed" Error

If you encounter a "signal: killed" error when running `xdrun` on macOS (especially after download or update), this is caused by macOS Gatekeeper quarantine attributes. Fix it with:

```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/xdrun

# Or if installed elsewhere
xattr -d com.apple.quarantine /path/to/xdrun
```

**Why this happens:** When binaries are downloaded from the internet, macOS automatically sets quarantine attributes for security. Since drun binaries aren't signed with an Apple Developer certificate, Gatekeeper blocks their execution.

**Note:** The install script automatically removes these attributes, but manual downloads or certain update scenarios may require this fix.

### Self-Update

Keep xdrun up to date with the built-in update command:

```bash
# Check for updates and install latest version
xdrun --self-update
```

The update process includes:
- Automatic backup of current version (kept in `~/.drun/`)
- Download and verification of new version
- Automatic restoration if update fails
- Keeps last 5 backups for safety

## Quick Start

### File Structure

drun uses a simple, predictable file discovery system:

- **`.drun/spec.drun`** - Default task file location
- **Custom locations** - Use `--file` to specify any other location
- **Workspace configuration** - `.drun/.drun_workspace` for custom defaults

**Moving your spec file:**

```bash
# Move your spec file anywhere
mv .drun/spec.drun ./my-project.drun

# Update workspace to point to new location
xdrun --set-workspace my-project.drun

# Now drun automatically uses your custom location
xdrun --list
```

### Getting Started

1. **Create a simple task file** (`.drun/spec.drun`):
   
   ```drun
   version: 2.0 # You need to specify the Drun file version
   project "my-app" version "1.0" # This is optional, except if you want to customize shell behavior
   
   task "hello" means "Say hello":
     info "Hello from drun v2!"
   ```

2. **List available tasks**:
   
   ```bash
   xdrun --list
   ```

3. **Run a task**:
   
   ```bash
   drun hello
   ```

4. **Use parameters**:
   
   ```bash
   xdrun deploy environment=production version=v1.0.0
   ```

5. **Explore examples**:
   
   ```bash
   xdrun -f examples/01-hello-world.drun hello
   ```

6. **Dry run to see what would execute**:
   
   ```bash
   xdrun build --dry-run
   ```

### Variable Scoping

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

1. **Workspace default** (if configured in `.drun/.drun_workspace`)
2. **Default location**: `.drun/spec.drun`
3. **Explicit specification**: Use `--file` for any other location

### Getting Started

Use `xdrun --init` to create a starter task file:

```bash
# Create default .drun/spec.drun
xdrun --init

# Create custom task file and save as workspace default
xdrun --init --file=my-project.drun --save-as-default

# Move existing file and update workspace
mv .drun/spec.drun ./tasks.drun
xdrun --set-workspace tasks.drun
```

See the included examples for comprehensive task configurations.

### Indentation

drun v2 supports both **tabs** and **spaces** for indentation, providing flexibility for different coding preferences:

```drun
# Using spaces (2 or 4 spaces per level)
task "spaces-example":
  info "Indented with spaces"
  if true:
    step "Nested with spaces"

# Using tabs
task "tabs-example":
    info "Indented with tabs"
    if true:
        step "Nested with tabs"
```

**Key points:**

- **Tab equivalence**: Each tab equals 4 spaces for indentation level calculation
- **Consistency**: Use consistent indentation within each file (don't mix tabs and spaces)
- **Flexibility**: Choose the style that works best for your team or editor
- **Generated files**: `xdrun --init` creates files with tab indentation by default

**For complete v2 specification**: See [DRUN_V2_SPECIFICATION.md](DRUN_V2_SPECIFICATION.md) for detailed language reference and examples.

### Basic Task

```drun
version: 2.0

project "my-app" version "1.0"

task "hello" means "Say hello":
  info "Hello, World! 👋"
```

### Task with Parameters

**Two parameter types with clear semantics:**
- **`requires`** - Mandatory parameters (user must provide)
- **`given`** - Optional parameters (always have defaults)

```drun
task "greet" means "Greet someone":
  requires $name                        # Must provide (mandatory)
  given $title defaults to "friend"     # Optional (has default)

  info "Hello, {$title} {$name}! "
```

**Usage examples:**

```bash
# Provide only required parameter (uses default for $title)
drun greet name=Alice

# Override optional parameter
drun greet name=Bob title=Mr.

# Provide all parameters
drun greet name=Alice title=Ms.
```

### Advanced Parameters with Control Flow

```drun
task "deploy" means "Deploy to environment with version":
  requires $environment from ["dev", "staging", "prod"]
  given $version defaults to "latest"
  given $features as list defaults to empty  # 🆕 empty keyword
  given $force as boolean defaults to false

  info "Deploying {$version} to {$environment}"

  if $features is not empty:  # 🆕 semantic empty conditions
    info "Features: {$features}"

  if $force is true:
    info "Force deployment enabled"
```

**Usage examples:**

```bash
# Basic deployment
xdrun deploy environment=prod

# With version and features
xdrun deploy environment=staging version=v1.1.0 features=auth,ui

# Force deployment
xdrun deploy environment=prod version=v1.2.0 force=true
```

### The `empty` Keyword

The `empty` keyword provides a semantic way to specify empty values and is completely interchangeable with empty strings (`""`):

```drun
task "example" means "Demonstrate empty keyword usage":
  # Default values - both are equivalent
  given $name defaults to empty      # semantic empty
  given $title defaults to ""        # traditional empty string
  given $features as list defaults to empty  # empty list

  # Conditions - semantic and readable
  if $name is empty:
    warn "Name is required"

  if $features is not empty:
    info "Enabled features: {$features}"

  # Works with all parameter types
  if $title is empty:
    info "Using default title"
```

**Key Benefits:**

- **Semantic**: More readable than empty quotes in automation contexts
- **Equivalent**: `empty` is exactly the same as `""` 
- **Flexible**: Works as default values and in conditions
- **Type-aware**: Creates appropriate empty values (empty string, empty list, etc.)

### Task with Dependencies

```drun
task "test" means "Run tests":
  depends on build

  run "go test ./..."

task "build" means "Build the project":
  run "go build ./..."
```

## Advanced Features Examples

### Multi-line Strings

Write complex shell commands across multiple lines with full support for line continuation, escaped quotes, and interpolation:

```drun
task "run tests" means "Execute test suite with coverage":
  let $app_env = "test"
  let $coverage_file = "coverage.xml"
  
  step "Running test suite with coverage"
  
  # Multi-line string with line continuation for readability
  run "docker compose exec \
      -e APP_ENV={$app_env} \
      -e XDEBUG_MODE=coverage \
      -u=www-data \
      php vendor/bin/phpunit --coverage-clover ./{$coverage_file}"
  
  # Multi-line string with natural line breaks
  run "echo \"Test Results:\"
echo \"Environment: {$app_env}\"
echo \"Coverage: {$coverage_file}\"
cat {$coverage_file}"
  
  success "Tests completed with coverage"

task "build script" means "Complex build with multiple commands":
  # Preserve line breaks for sequential commands
  run "echo 'Building frontend...'
cd frontend
npm install
npm run build
cd ..
echo 'Building backend...'
cd backend
go build -o app
cd ..
echo 'Build complete!'"
```

**Key Features:**

- **Line Continuation**: Use `\` before newline to join lines without a break
- **Natural Line Breaks**: Preserve newlines for multi-command scripts
- **Escaped Quotes**: Use `\"` for literal quotes in commands
- **Full Interpolation**: Variables and expressions work seamlessly
- **Readability**: Write maintainable, well-formatted shell scripts

### HTTP Integration

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

- **Semantic HTTP Actions**: `get`, `post`, `put`, `delete` with natural syntax
- **Authentication**: Built-in support for bearer tokens, basic auth, API keys
- **JSON Support**: Automatic JSON parsing and response handling
- **Error Handling**: Intelligent retry and error management
- **Response Capture**: Store responses in variables for processing

### DRY Tool Detection

Eliminate repetitive conditional logic with intelligent tool variant detection:

```drun
project "cross-platform-app" version "1.0":

task "setup-docker-tools" means "Setup Docker toolchain with DRY detection":
  info " Setting up Docker toolchain"

  # Detect which Docker Compose variant is available and capture it
  detect available "docker compose" or "docker-compose" as $compose_cmd

  # Detect which Docker Buildx variant is available and capture it
  detect available "docker buildx" or "docker-buildx" as $buildx_cmd

  info " Detected tools:"
  info "   Compose: {$compose_cmd}"
  info "  🔨 Buildx: {$buildx_cmd}"

  # Now use the captured variables consistently throughout the task
  run "{$compose_cmd} version"
  run "{$buildx_cmd} version"

  success "Docker toolchain ready!"

task "deploy-app" means "Deploy using detected tools":
  # Reuse the same detection pattern
  detect available "docker compose" or "docker-compose" as $compose_cmd

  info " Deploying with {$compose_cmd}"
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

- **DRY Principle**: No repetitive `if/else` conditional logic
- **Cross-Platform**: Works across different tool installations automatically
- **Maintainable**: Single detection point, consistent usage throughout tasks
- **Flexible**: Supports any number of tool alternatives with `or` syntax
- **Clear Intent**: Makes tool compatibility explicit and documented

### Pattern Macro Validation

Built-in pattern macros provide common validation patterns without complex regex:

```drun
version: 2.0
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

  info " Deploying {version} to {server_ip}"
  info " Project: {project_slug}, Branch: {branch}"
  info "API: {api_endpoint}"
  info " Admin: {admin_email}"
  info " Deployment ID: {deployment_id}"

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

- **User-Friendly**: Simple, memorable names instead of complex regex
- **Self-Documenting**: Built-in descriptions explain validation rules
- **Type-Safe**: Clear, descriptive error messages
- **Performance**: Efficient validation with minimal overhead
- **Extensible**: Easy to add new macros as needed

### Advanced Variable Operations

Powerful data transformation operations with intuitive chaining syntax:

```drun
version: 2.0

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

  info " Array Operations:"
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

  info " Path Operations:"
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

  info "Complex Operation Chaining:"
  info "  Source JS files: {$project_files filtered by prefix 'src/' | filtered by extension '.js' | sorted by name}"
  info "  Test files: {$project_files filtered by prefix 'tests/' | sorted by name}"

  info " Processing Docker images:"
  for each $img in $docker_images:
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

- ** Eliminates Shell Scripting**: No more complex `sed`, `awk`, or `cut` commands
- **Intuitive Syntax**: English-like operations that are self-documenting
- ** Chainable**: Combine operations with pipe (`|`) for complex transformations
- ** Type-Aware**: Works seamlessly with strings, arrays, and paths
- **Loop Integration**: Perfect integration with `for each` loops
- ** Performance**: Efficient operations with minimal overhead

## Command Line Options

- `--init`: Initialize a new .drun task file
- `--list, -l`: List available tasks
- `--dry-run`: Show what would be executed without running
- `--explain`: Show rendered scripts and environment variables
- `--update`: Update drun to the latest version from GitHub releases
- `--file, -f`: Specify task file (default: auto-discover .drun files)
- `--jobs, -j`: Number of parallel jobs for dependencies
- `--set`: Set variables (KEY=VALUE format)
- `--shell`: Override shell type (linux/darwin/windows)
- `--allow-undefined-variables`: Allow undefined variables in interpolation (default: strict mode)
- `completion [bash|zsh|fish|powershell]`: Generate shell completion scripts
- `cleanup-backups`: Clean up old backup files created during updates

### Debug Options

drun includes comprehensive debugging capabilities to help you understand how your tasks are parsed and executed:

- `--debug`: Enable debug mode (shows full debug output by default)
- `--debug-tokens`: Show lexer tokens from the parsing process
- `--debug-ast`: Show the Abstract Syntax Tree structure
- `--debug-json`: Show AST as JSON for detailed inspection
- `--debug-errors`: Show only parse errors
- `--debug-full`: Show complete debug information (tokens + AST + errors)
- `--debug-input "string"`: Debug input string directly instead of reading from file

**Debug Examples:**

```bash
# Show full debug output for a file
xdrun --debug -f my-tasks.drun

# Show only the lexer tokens
xdrun --debug --debug-tokens -f my-tasks.drun

# Debug inline input directly
xdrun --debug --debug-input 'task "test": info "hello"'

# Show AST structure only
xdrun --debug --debug-ast -f my-tasks.drun

# Show AST as JSON for tooling integration
xdrun --debug --debug-json -f my-tasks.drun
```

> **Note:** As of v2.0, debug functionality has been integrated into the main `drun` command. The separate debug tool has been removed for a more streamlined experience.

## Variable Checking

drun v2 includes **strict variable checking** by default to catch undefined variables early and prevent runtime errors.

### Strict Mode (Default)

By default, drun operates in strict mode and will fail with a clear error message if any undefined variables are encountered:

```bash
# This will fail if $undefined_var is not defined
drun my-task
# Error: task 'my-task' failed: in info statement: undefined variable: {$undefined_var}
```

### Allow Undefined Variables

Use the `--allow-undefined-variables` flag to allow undefined variables (legacy behavior):

```bash
# This will show {$undefined_var} as literal text
drun my-task --allow-undefined-variables
```

### Benefits of Strict Mode

- ** Early Error Detection**: Catch typos and missing variables before execution
- ** Clear Error Messages**: Precise location and context of undefined variables
- **Prevent Silent Failures**: Avoid unexpected behavior from missing variables
- ** Better Documentation**: Forces explicit variable definitions

### Examples

```drun
version: 2.0

task "strict example":
    let $name = "world"
    info "Hello {$name}"           #  Works: variable is defined
    info "Hello {$typo_name}"      # ❌ Fails: undefined variable (strict mode)

task "with parameters":
    accepts $target as string
    info "Deploying to {$target}"  #  Works: parameter is defined
    info "Version: {$version}"     # ❌ Fails: undefined variable (strict mode)
```

## Shell Completion

drun supports intelligent shell completion for bash, zsh, fish, and PowerShell with smart task and command detection. The completion system provides:

- ** Task Names**: Auto-complete available tasks from your drun file with `[task]` prefix
- **⚙ CLI Commands**: Complete drun CLI commands with `[drun CLI cmd]` prefix  
- ** Descriptions**: Show task descriptions alongside completions
- ** Dynamic Updates**: Completions automatically reflect your current drun file

### Quick Setup (Recommended)

For **persistent autocompletion** that stays up-to-date with your tasks, add this to your shell configuration:

#### Zsh (Most Common)

```bash
# Add to ~/.zshrc for persistent, always up-to-date completion
echo 'source <(xdrun completion zsh)' >> ~/.zshrc

# Reload your shell
source ~/.zshrc
```

#### Bash

```bash
# Add to ~/.bashrc for persistent, always up-to-date completion  
echo 'source <(xdrun completion bash)' >> ~/.bashrc

# Reload your shell
source ~/.bashrc
```

#### Fish

```bash
# Add to Fish config for persistent completion
echo 'xdrun completion fish | source' >> ~/.config/fish/config.fish

# Reload Fish
source ~/.config/fish/config.fish
```

### Alternative Installation Methods

If you prefer static completion files (updated less frequently):

#### Bash

```bash
# Load completion for current session
source <(xdrun completion bash)

# Install permanently (Linux)
xdrun completion bash > /etc/bash_completion.d/xdrun

# Install permanently (macOS with Homebrew)
xdrun completion bash > $(brew --prefix)/etc/bash_completion.d/xdrun
```

#### Zsh

```bash
# Enable completion system (if not already enabled)
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Load completion for current session
source <(xdrun completion zsh)

# Install permanently
xdrun completion zsh > "${fpath[1]}/_xdrun"

# Restart your shell or source ~/.zshrc
```

#### Fish

```bash
# Load completion for current session
xdrun completion fish | source

# Install permanently
xdrun completion fish > ~/.config/fish/completions/xdrun.fish
```

#### PowerShell

```powershell
# Load completion for current session
xdrun completion powershell | Out-String | Invoke-Expression

# Install permanently
xdrun completion powershell > xdrun.ps1
# Then source this file from your PowerShell profile
```

### Completion Features

The completion system intelligently distinguishes between:

- **`[task] Task Name`** - Tasks defined in your drun file
- **`[drun CLI cmd] Command`** - Built-in drun CLI commands

### Completion Examples

```bash
# Task and command completion with prefixes
xdrun <TAB>
# Shows:
#   completion    [xdrun CLI cmd] Generate completion script
#   help          Help about any command  
#   default       [task] Welcome to drun v2
#   hello         [task] Say hello
#   build         [task] Build the project
#   test          [task] Run tests
#   deploy        [task] Deploy application

# Task name completion
xdrun hel<TAB>                 # Completes to "hello"
xdrun dep<TAB>                 # Completes to "deploy"

# CLI command completion
xdrun comp<TAB>                # Completes to "completion"

# Flag completion
xdrun --<TAB>                  # Shows all available flags with descriptions
xdrun --list                   # Lists all tasks
```

### Why Use Dynamic Completion?

**Recommended approach**: `source <(xdrun completion zsh)` in your shell config

**Benefits:**

-  **Always Current**: Reflects your latest task definitions
-  **No Maintenance**: No need to regenerate completion files
-  **Project Aware**: Works with different drun files in different directories
-  **Fast**: Completion generation is highly optimized (microseconds)

**Static files** work but require manual updates when you add/remove tasks.

## Self-Update & Backup Management

drun includes built-in self-update functionality with intelligent backup management.

### Update Command

```bash
# Check for and install updates
xdrun --update
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

-  **User-writable backups** (no permission errors)
-  **Automatic rollback** on update failure
-  **Backup preservation** (not auto-deleted)
-  **Platform detection** (correct binary for your system)
-  **Version validation** (only update when newer version available)

## Examples

Explore comprehensive examples in the `examples/` directory:

###  **Example Files**

- **`examples/01-hello-world.drun`** - Basic introduction to drun v2
- **`examples/02-parameters.drun`** - Parameter handling and validation
- **`examples/03-interpolation.drun`** - Variable interpolation examples
- **`examples/04-docker-basics.drun`** - Docker operations and workflows
- **`examples/05-kubernetes.drun`** - Kubernetes deployment examples
- **`examples/06-cicd-pipeline.drun`** - CI/CD pipeline automation
- **`examples/07-final-showcase.drun`** - Comprehensive feature showcase
- **`examples/08-builtin-functions.drun`** - Built-in function examples
- **`examples/26-smart-detection.drun`** - Smart tool and environment detection
- **`examples/38-progress-and-timers.drun`** - Progress indicators and timing operations
- **`examples/46-task-calling.drun`** - Task calling and modular task design

### Quick Examples

```bash
# Try the hello world example
xdrun -f examples/01-hello-world.drun hello

# Test parameters and validation
xdrun -f examples/02-parameters.drun "deploy app" environment=dev

# Explore built-in functions
xdrun -f examples/08-builtin-functions.drun "system info"

# Try progress indicators and timers
xdrun -f examples/38-progress-and-timers.drun "progress demo"

# Explore smart detection
xdrun -f examples/26-smart-detection.drun "detect project"

# Try task calling and modular design
xdrun -f examples/46-task-calling.drun "quick-test"

# See comprehensive features
xdrun -f examples/07-final-showcase.drun showcase project_name=MyApp
```

Each example includes comprehensive documentation and demonstrates best practices for different use cases.

## Status & Roadmap

drun is **production-ready** with enterprise-grade features:

### Implemented Features

- **Core Functionality**: .drun semantic language, parameters, variables, control flow
- **Advanced Features**: Remote includes, matrix execution, secrets management, task calling
- **Developer Experience**: 15+ template functions, intelligent caching, rich errors
- **Performance**: Microsecond-level operations, high test coverage (71-83%)
- **Quality**: Zero linting issues, comprehensive test suite

### Coming Soon

- **File Watching**: Auto-execution on file changes
- **Plugin System**: Extensible architecture for custom functionality
- **Interactive TUI**: Beautiful terminal interface
- **Web UI**: Browser-based task management
- **AI Integration**: Natural language task generation

### Enterprise Ready

- **High Performance**: Microsecond-level operations
- **Scalability**: Handles 100+ tasks efficiently  
- **Security**: Secure secrets management
- **Reliability**: Comprehensive error handling
- **Maintainability**: Clean architecture with extensive tests

drun has evolved from a simple task runner into a **comprehensive automation platform** that's ready for production use at any scale! 

---

## Developer Guide

This section contains information for developers who want to build, test, or contribute to drun.

### Requirements

- **Go 1.25+** - drun requires Go 1.25 or later

### Build from Source

```bash
# Clone the repository
git clone https://github.com/phillarmonic/drun.git
cd drun

# Build drun
go build -o bin/xdrun ./cmd/drun

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

| Component              | Operation               | Time  | Memory  | Allocations |
| ---------------------- | ----------------------- | ----- | ------- | ----------- |
| **Spec Loading**       | Simple spec             | 2.5μs | 704 B   | 5 allocs    |
| **Spec Loading**       | Large spec (100 tasks)  | 8.6μs | 756 B   | 5 allocs    |
| **Template Rendering** | Basic template          | 29μs  | 3.9 KB  | 113 allocs  |
| **Template Rendering** | Complex template        | 51μs  | 7.0 KB  | 93 allocs   |
| **DAG Building**       | Simple dependency graph | 3.1μs | 10.7 KB | 109 allocs  |
| **DAG Building**       | Complex dependencies    | 3.9μs | 12.4 KB | 123 allocs  |
| **Topological Sort**   | 100 nodes               | 2.5μs | 8.0 KB  | 137 allocs  |

#### Optimization Impact

Our performance optimizations deliver significant improvements:

| Component              | Before       | After           | **Improvement**                   |
| ---------------------- | ------------ | --------------- | --------------------------------- |
| **Template Rendering** | 40μs, 60KB   | **29μs, 4KB**   | **1.4x faster, 15x less memory**  |
| **Spec Loading**       | 361μs, 42KB  | **2.5μs, 704B** | **144x faster, 59x less memory**  |
| **Large Spec Loading** | 3.4ms, 657KB | **8.6μs, 756B** | **396x faster, 869x less memory** |
| **DAG Building**       | 4.4μs, 14KB  | **3.1μs, 11KB** | **1.4x faster, 22% less memory**  |
| **Topological Sort**   | 4.7μs, 10KB  | **2.5μs, 8KB**  | **1.9x faster, 20% less memory**  |

#### Performance Features

- **Template Caching**: Compiled templates cached by hash for instant reuse
- ** Smart Pre-allocation**: Memory pools and capacity-aware data structures
- ** Spec Caching**: Task specs cached with file modification tracking
- **Optimized DAG**: Highly efficient dependency graph construction
- ** Memory Pools**: Reusable objects reduce GC pressure
- ** Lazy Evaluation**: Only compute what's needed when needed

#### Real-World Performance

- **Startup time**: Sub-millisecond for cached specs
- **Large projects**: 100+ tasks process in microseconds
- **Memory usage**: Minimal footprint with intelligent caching
- **Parallel execution**: Efficient DAG-based task scheduling
- **Template rendering**: Up to 20x faster than naive implementations

Run benchmarks yourself:

```bash
./scripts/test.sh -b  # Includes comprehensive performance benchmarks
```

### Code Reuse Features

drun v2 now supports powerful code reuse mechanisms to eliminate duplication and maintain DRY principles:

#### Project-Level Parameters

Define shared configuration once at the project level:

```drun
project "my-app" version "1.0.0":
  parameter $no_cache as boolean defaults to "false"
  parameter $environment as string from ["dev", "staging", "prod"] defaults to "dev"
  parameter $registry as string defaults to "docker.io"

task "build":
  info "Building for {$environment}"
  info "Registry: {$registry}"
```

```bash
# Override at runtime
xdrun build environment=prod registry=gcr.io
```

#### Reusable Snippets

Define common code blocks that can be reused across tasks:

```drun
project "my-app" version "1.0.0":
  snippet "log-header":
    info "═══════════════════════════════"
    info "  Starting Task"
    info "═══════════════════════════════"
  
  snippet "cleanup":
    info "Cleaning up..."
    info "Done"

task "build":
  use snippet "log-header"
  info "Building..."
  use snippet "cleanup"

task "deploy":
  use snippet "log-header"
  info "Deploying..."
  use snippet "cleanup"
```

#### Task Templates

Create parameterized task templates that act like functions:

```drun
# Define a template
template task "docker-build":
  given $target defaults to "prod"
  given $tag defaults to "latest"
  
  info "Building {$target} with tag {$tag}"
  # Build logic here

# Use the template with different parameters
task "build:web":
  call task "docker-build" with target="web" tag="myapp:web"

task "build:api":
  call task "docker-build" with target="api" tag="myapp:api"

task "build:worker":
  call task "docker-build" with target="worker" tag="myapp:worker"
```

#### Namespaced Includes

Share code across projects with automatic namespace resolution:

```drun
# shared/docker.drun
project "docker":
  snippet "login-check":
    if env DOCKER_AUTH exists:
      info "✓ Docker authenticated"
  
  template task "push":
    given $image defaults to "app:latest"
    use snippet "login-check"    # No namespace needed within same file!
    info "Pushing {$image}..."

# main.drun
project "myapp":
  include "shared/docker.drun"

task "deploy":
  use snippet "docker.login-check"    # ✨ Elegant dot notation
  call task "docker.push" with image="myapp:v1"
```

**Selective imports** for fine-grained control:

```drun
project "myapp":
  include snippets from "shared/utils.drun"
  include templates from "shared/k8s.drun"
  include snippets, templates from "shared/common.drun"
```

#### Remote Includes

Include workflows directly from GitHub, HTTPS URLs, or the **drunhub standard library**:

```drun
project "myapp":
  # From drunhub standard library (https://github.com/phillarmonic/drun-hub)
  include from drunhub "ops/docker" as ops
  include from drunhub "ops/kubernetes" as k8s
  
  # From GitHub (auto-detects default branch)
  include "github:myorg/drun-workflows/docker.drun@v1.2.0"
  
  # From HTTPS URL
  include "https://raw.githubusercontent.com/team/repo/main/ci.drun"

task "deploy":
  use snippet "ops.check-docker"         # From drunhub!
  use snippet "docker.security-scan"     # From remote include!
  call task "k8s.deploy" with replicas=3 # From drunhub!
```

**Features:**

-  **Drunhub Standard Library**: Official repository of reusable workflows at [github.com/phillarmonic/drun-hub](https://github.com/phillarmonic/drun-hub)
-  **Custom Namespaces**: Override project names with `as` clause for cleaner imports
- **GitHub & HTTPS**: Fetch from GitHub repos or any HTTPS source
-  **Version Control**: Pin to specific tags, branches, or commits
-  **Smart Caching**: 1-minute cache with stale fallback for offline use
- **Private Repos**: Supports `GITHUB_TOKEN` for authentication
- **Fast**: Cached includes load instantly

```bash
# Disable cache for fresh fetch
xdrun --no-drun-cache -f myfile.drun deploy
```

**Benefits:**

- **DRY Principle**: Eliminate duplication across tasks
- ** Maintainable**: Update logic in one place
- ** Type-Safe**: Full validation on all parameters
- ** Readable**: Clear, semantic names for reusable components
- ** Flexible**: Mix and match project parameters, snippets, and templates
- **Cross-Project Sharing**: Share workflows across multiple projects with includes
- **Namespace Safety**: Dot notation prevents naming collisions
- ** Community Workflows**: Leverage shared workflows from GitHub

**See it in action:**

```bash
# Try the comprehensive code reuse example
xdrun -f examples/code-reuse-demo.drun build:all

# With custom parameters
xdrun -f examples/code-reuse-demo.drun build:web environment=prod no_cache=true
```

---

##  Documentation

### For Users

- **[drun v2 Specification](DRUN_V2_SPECIFICATION.md)** - Complete v2 language specification with code reuse features
- **[Code Reuse Features](DRUN_V2_SPECIFICATION.md#code-reuse-features)** - Detailed documentation on project parameters, snippets, and templates
- **[Examples Directory](examples/)** - Real-world usage examples and patterns
- **[DRUN_LLM_USAGE_MANUAL.md](./DRUN_LLM_USAGE_MANUAL.md)** - Guide for LLMs to understand and write drun

### For Developers

Contributing to drun or want to understand how it works?

- **[DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md)** - **Start here!** Complete guide to the codebase
  - Architecture overview and diagrams
  - Package-by-package navigation
  - How to add new features
  - Testing strategies
  - Code style and best practices

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - System architecture with 10 Mermaid diagrams
  - Execution flow
  - Component interactions
  - Design patterns used
  - Extension points

- **[internal/README.md](./internal/README.md)** - Internal packages guide
  - Package organization
  - File structure
  - Common patterns
  - Navigation tips

- **[ROADMAP.md](./ROADMAP.md)** - Feature roadmap and implementation status

- **[CONTRIBUTING.md](./CONTRIBUTING.md)** - How to contribute to drun
  - Development setup
  - Code style guidelines
  - Pull request process
