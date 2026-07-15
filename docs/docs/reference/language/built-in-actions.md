# Built-in actions

## Built-in Actions

### Shell Commands

drun v2 supports both single-line and multiline shell command execution with consistent syntax patterns.

#### Single-Line Commands (Current)

```drun
# Execute and stream output
run "echo 'Hello World'"
run "command" attached
exec "date +%Y-%m-%d"
shell "pwd"

# Capture output to variable
capture from shell "git rev-parse --short HEAD" as $commit_hash
capture from shell "whoami" as $username
```

#### Task Modes

Tasks can opt into execution modes directly in the declaration:

```drun
task "ci" mode "ci" means "Run noisy checks quietly":
  step "Lint"
  run "golangci-lint run ./..."
  step "Security"
  run "gosec ./..."
```

Currently supported task execution supported modes:

- `normal` is the standard execution behavior. This is the implicit default when a task does not declare a mode. Shell command output streams normally as commands run.
- `ci` buffers shell command output for the task and only prints the buffered stdout/stderr if a command fails. Action statements like `step`, `info`, and `success` still print normally.

`ci` is the only mode currently declared inside task definitions. Use the normal default behavior by omitting the `mode` clause entirely.

The philosophy behind the CI execute is: 
> The output values of this execution only matter if something breaks

This is particularly useful in environments where having a lot of garbage in the logs can be costly, such as when monitoring logs with AI Large Language Models (input token cost) or ingesting the logs into tools that generate cost of ingestion, like DataDog, Loki, etc.

You can still see these values when debugging if your logs are accurate by overriding the runtime mode. Let's talk about that.

**Runtime Override**

`xdrun` can override the execution mode for a single invocation:

```bash
xdrun ci --task-mode=normal
xdrun build --task-mode=ci
```

- `--task-mode=normal` forces normal streaming behavior even when the task declaration uses `mode "ci"`.
- `--task-mode=ci` applies CI-style buffering for the invoked task and any called tasks during that run.
- Supported runtime override values are `normal` and `ci`.

#### Multiline Commands (Block Syntax)

For complex shell operations, use the block syntax with natural indentation:

```drun
# Multiline execution with streaming
run:
  echo "Starting deployment process..."
  git pull origin main
  npm install
  npm run build

# Multiline with output capture
capture from shell as $build_info:
  echo "Build Information:"
  echo "Commit: $(git rev-parse --short HEAD)"
  echo "Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "User: $(whoami)"

# Complex shell operations
shell:
  for file in *.log; do
    if [ -f "$file" ]; then
      echo "Processing $file"
      gzip "$file"
      mv "$file.gz" archive/
    fi
  done

# Multiline with different shell actions
exec:
  # Database backup
  pg_dump myapp_production > backup_$(date +%Y%m%d).sql

  # Compress backup
  gzip backup_$(date +%Y%m%d).sql

  # Upload to storage
  aws s3 cp backup_$(date +%Y%m%d).sql.gz s3://backups/
```

#### Execution Behavior

**Single-line commands**: Execute as individual shell commands

```drun
run "echo hello"  # Executes: /bin/sh -c "echo hello"
```

**Attached single-line `run` commands**: Stay connected to the current terminal for interactive programs

```drun
run "command" attached
run in service $servicename "npm run dev" attached
```

Use `attached` only with single-line `run` statements when the command needs stdin or terminal behavior. Plain `run` remains non-interactive by default.

**Multiline commands**: Execute as a single shell session

```drun
run:
  export VAR=value
  echo $VAR        # VAR is available from previous line
  cd /tmp
  pwd              # Shows /tmp (working directory persists)
```

#### Service-Scoped Shell Commands

When services are declared in the program, shell commands can target a service's working directory without manual `cd` operations:

```drun
task "inspect-service" given $servicename defaults to "some-service":
    run in service $servicename "ls -a"
    run in service $servicename "cat Dockerfile"
    run in service $servicename "npm run dev" attached
```

The runtime resolves the service name (from literals, task parameters, or captured variables), validates that the service exists, and executes the command inside the service directory. If no services are defined, or the service name cannot be resolved, execution fails fast with an explanatory error.

#### Changing Working Directory (`use workdir`)

For tasks that need to run commands in a different directory, `use workdir` provides a clean, readable way to temporarily change the working directory for all subsequent shell commands in the current task.

**Syntax:**

```drun
use workdir "path"
```

**Examples:**

```drun
# Basic: build a frontend project located in a subdirectory
task "front-dev" means "Builds the dev frontend of the app":
    use workdir "frontend"
    run "npm run build:dev"

# With variable interpolation
task "build-module" means "Build a specific module":
    given $module defaults to "frontend"
    use workdir "{$module}"
    run "npm run build"

# Switch between directories in one task
task "multi-build" means "Build both packages":
    use workdir "packages/frontend"
    run "npm run build"
    use workdir "packages/backend"
    run "go build ./..."

# Absolute path
task "deploy":
    use workdir "/var/www/app"
    run "git pull origin main"
```

**Key Behaviors:**

- **Task-scoped**: The working directory change applies only within the current task. It does not affect called tasks (`call task`), dependent tasks (`depends on`), or any other task.
- **Relative paths from original cwd**: Relative paths are always resolved from the process's original working directory (the cwd when xdrun was invoked), not chained from a previous `use workdir`. This means:
  ```text
  use workdir "a"  # resolves to <original_cwd>/a
  use workdir "b"  # resolves to <original_cwd>/b  (NOT <original_cwd>/a/b)
  ```text
- **Absolute paths**: Absolute paths are used as-is.
- **Variable interpolation**: Full interpolation support — use `{$var}`, `{env('VAR')}`, etc.
- **Validation**: The resolved path must exist and be a directory. Non-existent paths fail immediately with a descriptive error.
- **Dry-run**: In `--dry-run` mode, logs `[DRY RUN] Would set working directory to: <path>` without resolving the path.

#### Variable Interpolation in Multiline Commands

Variables work seamlessly in multiline blocks:

```drun
let $environment = "production"
let $version = "v1.2.3"

run:
  echo "Deploying {$version} to {$environment}"
  docker build -t myapp:{$version} .
  docker tag myapp:{$version} myapp:latest
  docker push myapp:{$version}
  docker push myapp:latest
```

#### Error Handling in Multiline Commands

Multiline commands support the same error handling as single-line commands:

```drun
try:
  run:
    echo "Starting risky operation..."
    some_command_that_might_fail
    echo "Operation completed"
catch command_error:
  error "Multiline command failed: {command_error}"

  # Cleanup on failure
  shell:
    echo "Cleaning up..."
    rm -f temp_files/*
```

#### Best Practices

1. **Use multiline for related operations**: Group logically connected commands
2. **Preserve environment**: Variables and working directory persist across lines
3. **Error propagation**: Any failing command stops execution (unless using `|| true`)
4. **Readability**: Use multiline for complex operations, single-line for simple ones

#### Examples

```drun
task "deploy application":
  info "Starting deployment process"

  # Single-line for simple operations
  run "echo 'Deployment started at $(date)'"

  # Multiline for complex build process
  run:
    echo "Building application..."
    npm ci
    npm run build
    npm run test

  # Capture complex information
  capture from shell as $deployment_info:
    echo "=== Deployment Information ==="
    echo "Version: $(git describe --tags --always)"
    echo "Branch: $(git branch --show-current)"
    echo "Commit: $(git rev-parse HEAD)"
    echo "Built by: $(whoami) on $(hostname)"
    echo "Build time: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

  info "Build information: {$deployment_info}"

  # Multiline deployment commands
  shell:
    echo "Deploying to Kubernetes..."
    kubectl apply -f k8s/
    kubectl set image deployment/myapp app=myapp:latest
    kubectl rollout status deployment/myapp --timeout=300s

  success "Deployment completed successfully"
```

### Docker Actions

#### Image Operations

```drun
# Build image
build docker image                           # Infers name from project
build docker image "myapp:latest"          # Explicit name
build docker image "myapp:{version}" for ["linux/amd64", "linux/arm64"]

# Push image
push image "myapp:latest"                   # To default registry
push image "myapp:latest" to "ghcr.io"     # To specific registry

# Pull image
pull image "nginx:alpine"
```

#### Container Operations

```drun
# Run container
run container "myapp:latest"
run container "myapp:latest" on port 8080
run container "myapp:latest" with environment {DATABASE_URL: "postgres://..."}

# Container lifecycle
stop container "myapp"
remove container "myapp"
restart container "myapp"

# Container inspection
check health of container "myapp"
get logs from container "myapp"
```

#### Docker Compose

```drun
# Service management
start docker compose services
stop docker compose services
restart docker compose service "api"

# Scaling
scale docker compose service "worker" to 3 instances

# Execute commands within a service directory
docker compose in service "api" exec -it app bash
```

Service-scoped docker compose commands reuse the registered service paths. The service name can be dynamic—for example, using task parameters:

```drun
task "open-shell" given $servicename defaults to "some-service":
    docker compose in service $servicename exec app bash
```

### Kubernetes Actions

#### Deployment Operations

```drun
# Deploy application
deploy "myapp:latest" to kubernetes
deploy "myapp:latest" to kubernetes namespace "production"
deploy "myapp:latest" to kubernetes with 5 replicas

# Deployment management
scale deployment "myapp" to 10 replicas
rollback deployment "myapp"
restart deployment "myapp"

# Status checking
wait for rollout of deployment "myapp"
check status of deployment "myapp"
```

#### Service Operations

```drun
# Service management
expose deployment "myapp" on port 8080
create service "myapp-service" for deployment "myapp"

# Ingress
create ingress for service "myapp-service" with host "app.example.com"
```

#### Resource Management

```drun
# Apply manifests
apply kubernetes manifests from "k8s/"
apply kubernetes manifest "deployment.yaml"

# Resource inspection
get pods in namespace "production"
describe pod "myapp-pod-123"
get logs from pod "myapp-pod-123"
```

### Git Actions

#### Repository Operations

```drun
# Commit operations
commit changes
commit changes with message "Add new feature"
commit all changes with message "Update dependencies"

# Branch operations
create branch "feature/new-api"
checkout branch "main"
merge branch "feature/new-api"
delete branch "feature/old-feature"

# Remote operations
push to branch "main"
push tags to remote
pull from remote
fetch from remote
```

#### Tag Operations

```drun
# Tag management
create tag "v1.2.3"
create tag "v1.2.3" with message "Release version 1.2.3"
push tag "v1.2.3"
delete tag "v1.2.3"
```

### File System Actions

#### File Operations

```drun
# File management
copy "source.txt" to "destination.txt"
move "old-name.txt" to "new-name.txt"
remove "unwanted-file.txt"
backup "important-file.txt"
backup "important-file.txt" as "backup-{now.date}"
replace in "config/.env":
    "API_KEY=CHANGE_ME" with "API_KEY={$api_key}"
    "ENV=dev" with "ENV=production"

# Directory operations
create directory "new-folder"
remove directory "old-folder"
copy directory "src" to "backup/src"

The `replace` action accepts an indented list of `"old" with "new"` clauses, performing multiple replacements within the target file in a single operation.
```

#### Structured file values

Drun can read, validate, and update scalar values without delegating common
manifest edits to framework-specific shell commands:

```drun
get property "pluginVersion" from "gradle.properties" as $plugin_version
check property "pluginVersion" in "gradle.properties" equals "{$globals.version}"
check property "pluginVersion" in "gradle.properties" differs from "{$previous_version}"
update property "pluginVersion" in "gradle.properties" to "{$version}" or fail

get json "/version" from "package.json" as $package_version
update json "/version" in "package.json" to "{$version}" or add as string

get yaml "chart.appVersion" from "Chart.yaml" as $chart_version
get toml "package.version" from "Cargo.toml" as $crate_version

get match "(?m)^VERSION=(?P<value>[^\\r\\n]+)$" from "VERSION.txt" as $version
update match "(?m)^VERSION=(?P<value>[^\\r\\n]+)$" in "VERSION.txt" to "{$version}" or fail

check project version equals "{$version}"
update project version to "{$version}"
```

The grammar is:

```text
get <format> <selector> from <file> as <variable>
check <format> <selector> in <file> (equals <value> | differs from <value>)
update <format> <selector> in <file> to <value>
       (or fail | or add [as string|number|boolean])
check project version (equals <value> | differs from <value>)
update project version to <value>
```

Every format supports all three operations:

| Format | Selector | Read | Check | Update and missing-value behavior |
| --- | --- | --- | --- | --- |
| `property` | Exact, unescaped property key | `get property` | `check property` | `or fail`, or `or add`; additions are strings |
| `json` | RFC 6901 object-member pointer | `get json` | `check json` | `or fail`, or typed `or add as ...` |
| `yaml` | Dot-separated mapping path | `get yaml` | `check yaml` | `or fail`, or typed `or add as ...` |
| `toml` | TOML dotted-key path | `get toml` | `check toml` | `or fail`, or typed `or add as ...` |
| `match` | Go regular expression with one `value` capture | `get match` | `check match` | `or fail` only; `or add` is unsupported |

`check` supports both `equals` and `differs from` for every format. The
complete executable example is
[`examples/74-file-values.drun`](https://github.com/phillarmonic/drun/blob/master/examples/74-file-values.drun).

The project-version forms target the project declaration in the Drun file that
is currently executing, including a custom path supplied through `--file`.
They require exactly one versioned project declaration, preserve its surrounding
layout and comments, and never need `or fail`: a missing or ambiguous declaration
is already an error. `update project version` participates in dry runs and uses
the same permission-preserving atomic replacement as other structured updates.

`<format>` is `property`, `json`, `yaml`, `toml`, or `match`. Property
selectors are exact keys. JSON selectors are RFC 6901 pointers and select
object members only. YAML selectors are dot-separated mapping paths. TOML
selectors use TOML dotted-key syntax. A `match` selector is a Go regular
expression containing exactly one named capture called `value`; the expression
itself must match exactly once.

Structured operations accept scalar strings, numbers, and booleans. Reads
capture the scalar's textual value. Updates preserve the existing scalar type,
and an added JSON, YAML, or TOML value must state its type. `or add` creates only
the missing leaf below an existing parent. It is invalid for `match`.

Missing, duplicate, ambiguous, collection-valued, or type-invalid selections
fail before a write. Checks read real files during dry runs. Updates interpolate
their file, selector, and value, but only report the prospective change during a
dry run. Successful writes preserve file permissions and use an atomic
same-directory replacement.

Property, Drun project-version, JSON, and regex updates preserve surrounding source layout. YAML and
TOML updates use deterministic parser serialization and can normalize formatting
and comments. For source shapes outside these v1 rules, use the regex adapter.

These are Drun language-version 2 statements. Version 1 specs do not recognize
them. The initial format adapters intentionally do not support JSON array
elements, YAML sequences, TOML arrays/tables as selected values, escaped dots in
YAML paths, or dotted property-key traversal. Selectors must resolve to a scalar;
use `match` or an explicit shell command when a file falls outside these rules.

The existing literal `replace in` action remains unchanged and independent of
structured file-value operations.

#### File Inspection

```drun
# File checking
check if file "config.json" exists
check if directory ".git" exists
get size of file "large-file.dat"
get modification time of file "config.json"
```

#### Directory Empty Checks  *New*

Check if directories are empty or contain files using semantic conditions:

```drun
# Basic directory empty checks
if folder "build" is empty:
  info "Build directory is clean"

if folder "dist" is not empty:
  info "Distribution files exist"
  run "rm -rf dist/*"

# Alternative keywords
if directory "/tmp/cache" is empty:
  info "Cache is empty"

if dir "logs" is not empty:
  info "Log files found"
  run "gzip logs/*.log"

# With variable interpolation
if folder "{$output_dir}" is empty:
  warn "Output directory is empty"

if directory "{$project_root}/node_modules" is not empty:
  info "Dependencies are installed"

# Practical examples
if folder "migrations/pending" is not empty:
  run "php artisan migrate"

if directory "tests/coverage" is empty:
  run "npm run test:coverage"
```

**Key Features:**

- **Multiple keywords**: Use `folder`, `directory`, or `dir` interchangeably
- **Path interpolation**: Support for variable interpolation in paths
- **Non-existent handling**: Non-existent directories are treated as empty
- **Semantic conditions**: Natural `is empty` and `is not empty` syntax

### Network Actions

#### HTTP Operations

```drun
# HTTP requests
get "https://api.example.com/status"
post "https://api.example.com/deploy" content type json with body "version=1.2.3"
put "https://api.example.com/users/1" content type json with body "name=John"
delete "https://api.example.com/users/1"
patch "https://api.example.com/users/1" content type json with body "email=john@example.com"

# HTTP with authentication
get "https://api.example.com/secure" with auth bearer "token123"
post "https://api.example.com/data" with auth basic "user:pass"

# HTTP with headers and options
get "https://api.example.com/data" with header "X-Custom: value" timeout "30s"
post "https://api.example.com/upload" content type json with body "data" retry "3"

# File operations
get "https://example.com/file.zip" download "downloads/file.zip"
post "https://api.example.com/upload" upload "local-file.txt"
```

#### Download Operations

The `download` statement provides a native Go HTTP client with advanced features including progress tracking, permission management, and authentication.

**Features:**

- Native Go HTTP client (no external dependencies)
- Real-time progress bar with speed and ETA
- Matrix-based permission system
- Authentication support (Bearer, Basic, Token)
- Timeout and retry configuration
- Automatic redirect following

**Basic Syntax:**

```drun
download "<url>" to "<path>"
```

**Advanced Options:**

```drun
# Simple download with progress tracking
download "https://example.com/file.zip" to "downloads/file.zip"

# Allow overwriting existing files
download "https://example.com/data.json" to "data.json" allow overwrite

# Download with authentication
download "https://api.github.com/user" to "user.json" with auth bearer "token123"
download "https://private.example.com/file" to "file.dat" with auth basic "user:pass"

# Download with timeout and retry
download "https://example.com/large-file.zip" to "file.zip" timeout "60s" retry "3"

# Download with custom headers
download "https://api.example.com/data" to "data.json" with header "Accept: application/json"
```

**Permission Matrix System:**

The download statement supports granular Unix file permissions using a matrix notation:

```drun
# Make downloaded binary executable by user
download "https://github.com/cli/cli/releases/download/v2.40.0/gh_linux_amd64" to "gh"
  allow overwrite
  allow permissions ["execute"] to ["user"]

# Read/write for user, read-only for group/others
download "https://example.com/config.json" to "config.json"
  allow overwrite
  allow permissions ["read","write"] to ["user"]
  allow permissions ["read"] to ["group","others"]

# Multiple permission specifications
download "https://example.com/script.sh" to "script.sh"
  allow overwrite
  allow permissions ["read"] to ["user","group","others"]
  allow permissions ["write","execute"] to ["user"]

# Download and set complete permissions
download "https://example.com/tool" to "bin/tool"
  allow permissions ["execute","read"] to ["user"]
  allow permissions ["read"] to ["group","others"]
```

**Permission Types:**

- `read` - Read permission
- `write` - Write permission
- `execute` - Execute permission

**Permission Targets:**

- `user` - File owner
- `group` - Group members
- `others` - All other users

**Complete Example:**

```drun
task "download_and_install_binary":
  info "Downloading binary with full configuration"

  # Download with progress bar, auth, timeout, and permissions
  download "https://github.com/user/tool/releases/download/v1.0/tool-linux-amd64"
    to "bin/tool"
    allow overwrite
    timeout "120s"
    retry "5"
    with auth bearer "github-token"
    allow permissions ["execute","read"] to ["user"]
    allow permissions ["read"] to ["group","others"]

  success "Binary installed and configured!"
```

**Progress Display:**

The download statement shows real-time progress with:

- Progress bar (visual indicator)
- Percentage complete
- Downloaded / Total size
- Download speed (MB/s)
- Estimated time remaining (ETA)

Example output:

```drun
  Downloading: https://example.com/large-file.zip
   → downloads/large-file.zip
    [████████████████░░░░░░░░░░░░░░] 55.2% | 2.3 GB/4.2 GB | 15.3 MB/s | ETA: 2m15s
    4.2 GB in 4m32s (15.45 MB/s)
 Downloaded successfully to: downloads/large-file.zip
    Set permissions: -rwxr--r--
```

**Error Handling:**

```drun
# Prevent accidental overwrites
try:
  download "https://example.com/file.zip" to "existing-file.zip"
catch:
  warn "File already exists, use 'allow overwrite' to replace"

# With overwrite allowed
download "https://example.com/file.zip" to "file.zip" allow overwrite
```

**Archive Extraction:**

The download statement supports automatic extraction of archives using the pure-Go [github.com/mholt/archives](https://github.com/mholt/archives) library (no external dependencies):

**Supported Formats:**

- **Archives:** ZIP, TAR, TAR.GZ, TAR.BZ2, TAR.XZ, 7Z, RAR
- **Compression:** GZ, BZ2, XZ, ZSTD, BROTLI, LZ4, SNAPPY, LZW

```drun
# Download and extract archive
download "https://example.com/release.zip" to "release.zip" extract to "release/"

# Download, extract, and remove archive
download "https://example.com/release.tar.gz" to "release.tar.gz" extract to "bin/" remove archive

# With all options combined
download "https://github.com/user/tool/releases/download/v1.0/tool.zip"
  to "tool.zip"
  extract to "tools/"
  remove archive
  timeout "120s"
  with auth bearer "token"
```

**Extraction Examples:**

```drun
# Extract ZIP archive
task "install_from_zip":
  download "https://releases.example.com/app-v1.0.0.zip"
    to "app-v1.0.0.zip"
    extract to "app/"
    remove archive

# Extract tarball with compression
task "install_from_tarball":
  download "https://releases.example.com/tool-linux-amd64.tar.gz"
    to "tool.tar.gz"
    extract to "/usr/local/bin/"
    remove archive

# Keep archive for backup
task "extract_but_keep":
  download "https://releases.example.com/source.tar.gz"
    to "source.tar.gz"
    extract to "source/"
  # Archive stays as source.tar.gz

# Download and extract in parallel
task "parallel_installs":
  for each $version in ["v1.0", "v2.0", "v3.0"] in parallel:
    download "https://releases.example.com/tool-{$version}.zip"
      to ".downloads/tool-{$version}.zip"
      extract to "tools/{$version}/"
      remove archive
```

**Cross-Platform Benefits:**

- Pure Go implementation (no external tools like `tar`, `unzip`, `7z` required)
- Works identically on Windows, Linux, and macOS
- Automatic format detection from file extension and header
- Preserves file permissions and directory structure

**Real-World Examples:**

```drun
# Download GitHub release binary
task "install_gh_cli":
  download "https://github.com/cli/cli/releases/download/v2.40.0/gh_2.40.0_linux_amd64.tar.gz"
    to "gh.tar.gz"
    allow overwrite
    timeout "120s"
    allow permissions ["read","write"] to ["user"]

# Download multiple files in parallel
task "download_data":
  for each $file in ["users","posts","comments"] in parallel:
    download "https://api.example.com/{$file}.json"
      to "data/{$file}.json"
      allow overwrite
      allow permissions ["read","write"] to ["user"]
      allow permissions ["read"] to ["group"]

# Download with environment-specific permissions
task "download_config":
  requires $env from ["dev","prod"]

  when $env == "prod":
    download "https://config.example.com/prod.json" to "config.json"
      allow overwrite
      allow permissions ["read"] to ["user","group","others"]
  otherwise:
    download "https://config.example.com/dev.json" to "config.json"
      allow overwrite
      allow permissions ["read","write"] to ["user","group","others"]
```

#### Network Health Checks and Service Waiting

```drun
# Service waiting with timeout and retry
wait for service at "https://app.example.com/health" to be ready
wait for service at "https://app.example.com" to be ready timeout "60s"
wait for service at "https://api.example.com" to be ready timeout "30s" retry "5s"

# Health checks with status validation
# Note: Health checks are implemented via HTTP GET requests with curl
# They automatically validate HTTP status codes and provide retry logic
```

#### Network Testing

```drun
# Port connectivity testing
test connection to "database.example.com" on port 5432
test connection to "localhost" on port 8080 timeout "10s"

# Ping testing
ping host "example.com"
ping host "8.8.8.8" timeout "3s"
```

#### Advanced Network Operations

```drun
# Service waiting with detailed configuration
wait for service at "https://microservice.local/health" to be ready timeout "120s" retry "10s"

# Port testing with timeout
test connection to "redis.local" on port 6379 timeout "5s"

# Network diagnostics
ping host "gateway.local" timeout "2s"

# Combined network validation
task "validate_infrastructure":
  info "Validating network infrastructure"

  # Check external connectivity
  ping host "8.8.8.8" timeout "3s"
  ping host "1.1.1.1" timeout "3s"

  # Validate service dependencies
  test connection to "database.local" on port 5432 timeout "10s"
  test connection to "redis.local" on port 6379 timeout "5s"

  # Wait for application services
  wait for service at "https://api.local/health" to be ready timeout "60s"
  wait for service at "https://web.local/health" to be ready timeout "30s"

  success "Infrastructure validation completed!"
```

### Status and Logging Actions

#### Status Messages

```drun
step "Starting deployment process"
info "Configuration loaded successfully"
warn "Using default configuration"
error "Failed to connect to database"
success "Deployment completed successfully"
```

**Output Formatting:**

- `step` - Displays message in a box (no line breaks by default):
  ```drun
  ┌────────────────────────────────┐
  │ Starting deployment process    │
  └────────────────────────────────┘
  ```
  Multiline strings are supported and each line is rendered inside the same box:

  ```drun
  step "Executing semantic fuzz tests against example-based inputs
  Iterations: 50"
  ```
  Produces:

  ```text
  ┌─────────────────────────────────────────────────────────────┐
  │ Executing semantic fuzz tests against example-based inputs │
  │ Iterations: 50                                             │
  └─────────────────────────────────────────────────────────────┘
  ```
- `info` - Displays an informational message.
- `warn` - Displays a warning message.
- `error` - Displays an error message.
- `success` - Displays a success message.
- `fail` - Displays a failure message and exits with an error.

**Optional Line Breaks for `step`:**

By default, step boxes have no extra spacing. Add line breaks when you need visual separation:

```drun
# Default: no line breaks (compact)
step "Build phase"

# Line break before only
step "Build phase" add line break before

# Line break after only
step "Build phase" add line break after

# Line breaks both before and after
step "Build phase" add line break before add line break after
```

**Example Usage:**

```drun
task "compact":
  info "Starting deployment"

  # Compact steps - default behavior
  step "Phase 1: Build"
  info "Building application..."

  step "Phase 2: Test"
  info "Running tests..."

  step "Phase 3: Deploy"
  info "Deploying to production..."

  success "Deployment complete!"

task "spaced":
  info "Starting deployment"

  # Well-spaced sections with line breaks
  step "Phase 1: Build" add line break before add line break after
  info "Building application..."

  step "Phase 2: Test" add line break before add line break after
  info "Running tests..."

  step "Phase 3: Deploy" add line break before add line break after
  info "Deploying to production..."

  success "Deployment complete!"
```

#### Process Control

```drun
fail                                    # Exit with error code 1
fail with "Custom error message"        # Exit with custom message
exit with code 0                        # Exit with specific code
```

#### Progress Tracking

drun v2 provides built-in progress indicators and timer functions for tracking long-running operations:

##### Progress Indicators

```drun
# Start a progress indicator
info "{start progress('Initializing system')}"

# Update progress with percentage and message
info "{update progress('25', 'Loading configuration')}"
info "{update progress('50', 'Processing data')}"
info "{update progress('75', 'Finalizing setup')}"

# Complete the progress indicator
info "{finish progress('System ready!')}"
```

##### Timer Functions

```drun
# Start a named timer
info "{start timer('deployment_timer')}"

# Show elapsed time for a running timer
info "{show elapsed time('deployment_timer')}"

# Stop a timer and show final elapsed time
info "{stop timer('deployment_timer')}"
```

##### Advanced Progress and Timer Usage

```drun
task "deployment with progress":
  # Start both progress and timer
  info "{start progress('Starting deployment')}"
  info "{start timer('deploy')}"

  # Simulate deployment steps with progress updates
  info "{update progress('20', 'Building application')}"
  shell "sleep 1"  # Simulate build time

  info "{update progress('40', 'Pushing to registry')}"
  shell "sleep 1"  # Simulate push time

  info "{update progress('60', 'Deploying to cluster')}"
  shell "sleep 1"  # Simulate deploy time

  info "{update progress('80', 'Running health checks')}"
  shell "sleep 1"  # Simulate health check time

  info "{update progress('100', 'Deployment verification')}"

  # Show final timing and complete progress
  info "{show elapsed time('deploy')}"
  info "{finish progress('Deployment completed successfully!')}"
  info "{stop timer('deploy')}"
```

##### Multiple Named Progress Indicators and Timers

```drun
task "parallel operations":
  # Multiple progress indicators
  info "{start progress('Database migration', 'db_progress')}"
  info "{start progress('Asset compilation', 'asset_progress')}"

  # Multiple timers
  info "{start timer('db_timer')}"
  info "{start timer('asset_timer')}"

  # Update different progress indicators
  info "{update progress('30', 'Migrating users table', 'db_progress')}"
  info "{update progress('50', 'Compiling CSS', 'asset_progress')}"

  # Complete operations
  info "{finish progress('Database migration complete', 'db_progress')}"
  info "{stop timer('db_timer')}"

  info "{finish progress('Asset compilation complete', 'asset_progress')}"
  info "{stop timer('asset_timer')}"
```

**Built-in Function Reference:**

- `{start progress('message')}` - Start default progress indicator
- `{start progress('message', 'name')}` - Start named progress indicator
- `{update progress('percentage', 'message')}` - Update default progress (0-100)
- `{update progress('percentage', 'message', 'name')}` - Update named progress
- `{finish progress('message')}` - Complete default progress indicator
- `{finish progress('message', 'name')}` - Complete named progress indicator
- `{start timer('name')}` - Start a named timer
- `{stop timer('name')}` - Stop timer and show elapsed time
- `{show elapsed time('name')}` - Show elapsed time for running timer

### Built-in Functions

drun v2 provides a comprehensive set of built-in functions that can be used in expressions, variable assignments, and parameter defaults. These functions are called using the `{function name}` syntax and support pipe operations for data transformation.

#### Git Functions

```drun
# Get current git commit hash (short form)
set $commit to {current git commit}
info "Deploying commit: {$commit}"

# Get current git branch name
set $branch to {current git branch}
info "Building from branch: {$branch}"

# Use in parameter defaults
task "deploy":
  given $version defaults to "{current git commit}"
  given $branch_name defaults to "{current git branch}"
```

#### System Functions

```drun
# Get current working directory
set $project_dir to {pwd}

# Get hostname
set $host to {hostname}

# Get environment variable
set $api_key to {env('API_KEY')}

# Format current time
set $timestamp to {now.format('2006-01-02 15:04:05')}
```

#### Built-in Function Pipe Operations  *New*

Built-in functions support pipe operations for data transformation, allowing you to chain operations together:

```drun
# Replace characters in git branch names
set $safe_branch to {current git branch | replace "/" by "-"}
info "Safe branch name: {$safe_branch}"

# Chain multiple operations
set $clean_branch to {current git branch | replace "/" by "-" | lowercase}

# Use in parameter defaults with pipes
task "build":
  given $image_tag defaults to "{current git branch | replace '/' by '-' | lowercase}"
  given $commit_short defaults to "{current git commit}"

  info "Building image: myapp:{$image_tag}"
  info "From commit: {$commit_short}"
```

#### Available Pipe Operations

**String Operations:**

- `replace "from" by "to"` - Replace all occurrences of "from" with "to"
- `replace "from" with "to"` - Alternative syntax for replace
- `without prefix "text"` - Remove prefix from string
- `without suffix "text"` - Remove suffix from string
- `uppercase` - Convert to uppercase
- `lowercase` - Convert to lowercase
- `trim` - Remove leading and trailing whitespace

#### Practical Examples

```drun
task "git branch operations":
  # Basic git branch usage
  set $current_branch to {current git branch}
  info "Current branch: {$current_branch}"

  # Transform branch name for Docker tags (no slashes allowed)
  set $docker_tag to {current git branch | replace "/" by "-"}
  info "Docker tag: myapp:{$docker_tag}"

  # Create deployment-safe branch names
  set $deploy_name to {current git branch | replace "/" by "-" | lowercase}
  info "Deployment name: {$deploy_name}"

  # Use in complex expressions
  set $image_name to "registry.example.com/myapp:{current git branch | replace '/' by '-'}"
  info "Full image name: {$image_name}"

task "parameter defaults with pipes":
  # Parameter defaults can use piped builtin functions
  given $deployment_branch defaults to "{current git branch | replace '/' by '-' | lowercase}"
  given $build_tag defaults to "{current git commit}"
  given $timestamp defaults to "{now.format('2006-01-02-15-04-05')}"

  info "Deployment config:"
  info "  Branch: {$deployment_branch}"
  info "  Tag: {$build_tag}"
  info "  Timestamp: {$timestamp}"
```

#### Built-in Function Reference

| Function | Description | Example Output |
|----------|-------------|----------------|
| `{current git commit}` | Current git commit hash (short) | `a72091f` |
| `{current git branch}` | Current git branch name | `feature/new-api` |
| `{pwd}` | Current working directory | `/home/user/project` |
| `{hostname}` | System hostname | `dev-machine` |
| `{env('VAR')}` | Environment variable | `production` |
| `{now.format('layout')}` | Formatted current time | `2025-09-22 14:30:00` |

**Key Features:**

- **Interpolation**: All built-in functions use `{function}` syntax
- **Pipe Operations**: Chain transformations with `|` operator
- **Parameter Defaults**: Use in parameter default values with full pipe support
- **Variable Assignment**: Assign results to variables for reuse
- **Expression Context**: Work in any expression context (info messages, conditions, etc.)

---
## SCM registries and Git version queries

Projects may register source-control repositories once and refer to them by a
readable alias from tasks. Registries are grouped by SCM technology, then by
provider, so future technologies can be added without changing Git sources:

```drun
project "my-project":
  scm:
    git:
      github:
        app:
          default: https
          https: "https://github.com/example/app.git"
          ssh: "git@github.com:example/app.git"
          cli:
            repository: "example/app"
            host: "github.com"

      generic:
        local-library:
          filesystem: "../library"
```

Git source aliases must be unique within the `git` technology. GitHub and
GitLab sources support `https`, `ssh`, and `cli`; generic sources support
`https`, `ssh`, an opaque `remote`, and `filesystem` paths to worktrees or bare
repositories. One declared access method is the implicit default. Sources with
multiple methods must declare `default`, and Drun never silently falls back to
a different method.

Expanded HTTPS profiles accept `url` and optional `authentication: ambient`.
Expanded SSH profiles accept `url` and an optional `key` path. CLI profiles
accept `repository` and an optional `host`. Values are interpolated at runtime;
filesystem and key paths expand `~`. Credentials and private-key contents are
never stored in the Drunfile.

Generic remote sources normally read refs without fetching objects. A source
may declare `metadata: fetch`, but a task must still say `allow fetch` before a
date-ordered query may create temporary bare storage and fetch tag objects.
Both permissions are required. Filesystem sources inspect local objects
directly, and GitHub/GitLab CLI profiles use their authenticated provider APIs.

The registry describes reusable repository access rather than one operation.
The same source contract can support future branch, release, clone, archive,
mirror, and inspection statements.

### Version tags

Conventional stable tags (`1.2.3` and `v1.2.3`) need no configuration. For a
readable custom convention, use a format template:

```drun
version tags: "php-{version}"
```

`{version}` matches and captures exactly `MAJOR.MINOR.PATCH`; surrounding text
is treated literally and the complete tag is matched. `{{` and `}}` escape
literal braces. The presets `semver` and `semver_optional_v` mean
`"v{version}"` and the pair `"{version}"`, `"v{version}"` respectively.
Repositories that changed conventions can declare several templates:

```drun
version tags:
  formats:
    "php-{version}"
    "legacy-php-{version}"
```

Raw regular expressions are the advanced escape hatch. They must contain
exactly one Go-style named capture called `version`:

```drun
version tags:
  pattern: "^php-(?P<version>[0-9]+\\.[0-9]+\\.[0-9]+)$"
```

### Latest tag and version queries

```drun
git get latest tag from app as $latest_tag
git get latest version from app as $latest_version
git get latest version from php in series "8.4" as $php_version
git get latest version from php matching version ">=8.4.0 <8.5.0" as $php_version
```

`latest tag` returns the original tag; `latest version` returns its extracted
stable version. `in series "8"` selects `>=8.0.0 <9.0.0`, while `in series
"8.4"` selects `>=8.4.0 <8.5.0`. General constraints accept whitespace-joined
`>`, `>=`, `<`, `<=`, and `=` clauses. Series and general constraints are
mutually exclusive.

An inline contract overrides the source contract for one query. Prefer a
template, using a raw pattern only when necessary:

```drun
git get latest version from monorepo matching tags "runtime-{version}" in series "8.4" as $runtime_version
git get latest version from monorepo matching tags pattern "^runtime-(?P<version>[0-9]+\\.[0-9]+\\.[0-9]+)$" in series "8.4" as $runtime_version
```

Queries order numerically by version unless they say `ordered by date`.
Annotated tags use tagger time and lightweight tags use their target commit
time. Access can be overridden with `using https|ssh|cli|remote|filesystem`.
Remote generic date queries must also say `allow fetch` and use a source that
declares `metadata: fetch`.

For PHP tags, `"php-{version}"` excludes `php-8.4.24RC` and
`php-8.6.0-alpha2`. A query `in series "8.4"` also excludes stable releases in
other series, selecting `php-8.4.23` as tag or `8.4.23` as version. Dry-run
performs no network, filesystem, CLI, or temporary-fetch work; it reports the
resolved source, method, ordering, and capture variable.

### Ensuring a release version is newer

Use the composite guard when a release may proceed only if its candidate is
strictly newer than the latest stable version published by a registered source:

```drun
git ensure $release_version is newer than latest version from drun-vscode

git ensure $release_version is newer than latest version from drun-vscode
  as $latest_version
```

The optional capture receives the extracted latest version after the guard
succeeds. The statement is atomic: source resolution, stable-tag selection,
numeric comparison, and capture are one operation, so a failed guard never
writes the capture variable.

The guard accepts the query modifiers that affect repository access and the
version-tag contract. They appear after the source in the order shown, with the
optional capture last:

```drun
git ensure $release_version is newer than latest version from runtime
  using ssh
  matching tags "runtime-{version}"
  as $latest_version
```

`using https|ssh|cli|remote|filesystem` overrides the source's default access
method. `matching tags` accepts the same preset, format, and advanced `pattern`
forms as a latest-version query and overrides the source's reusable version-tag
contract for this statement. Without an inline contract, the source contract or
the conventional `semver_optional_v` default is used.

The candidate is interpolated at execution time and must be exactly a stable
`MAJOR.MINOR.PATCH` value. A leading `v`, a partial version, a prerelease, and
build metadata are rejected. Matching tags are reduced to extracted stable
versions and ordered numerically; tag spelling and enumeration order do not
affect the result.

The outcomes are deliberately distinct:

- If the source, selected access method, or stable-tag set cannot be resolved,
  the statement fails with a credential-safe source-resolution error. A source
  with no tags matching the effective stable version contract is in this case.
- If the candidate equals the latest version, it fails as already tagged and
  reports that version.
- If the candidate is older, it fails and reports the latest version.
- If the candidate is newer, it succeeds and then assigns the optional capture.

Diagnostics may identify the provider, source alias, selected access method,
and effective tag contract, but never include token, password, private-key, or
credential-bearing URL contents.

Dry-run interpolates and validates the candidate, then reports the planned
source, access method, effective tag contract, comparison, and capture. It
performs no network, filesystem, provider-CLI, fetch, or temporary-repository
work, does not claim that the invariant passed, and does not assign the capture.

The guard intentionally does not accept `latest tag`, `in series`, `matching
version`, `ordered by date`, or `allow fetch`. Those change the meaning away
from the latest stable numeric release or are unnecessary for numeric remote-ref
queries. Use `git get latest version` followed by the existing `older than
version` and `newer than version` conditions for custom workflows. Those
primitive statements and conditions remain supported and unchanged.
