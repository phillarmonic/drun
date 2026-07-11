# Detection, execution, and errors

## Smart Detection

### Tool Detection

#### Tool Requirements (requires tools:) *New*

The most robust way to ensure required dependencies are available is using the declarative `requires tools:` block. This can be used at the project level (checked before any tasks run) or at the task level (checked before the task runs).

```drun
# Project-level tool requirements
project "myapp":
  requires tools:
    go >= "1.21"
    golangci-lint >= "1.50" <= "1.55"
    docker

# Task-level tool requirements
task "security":
  requires tools:
    gosec >= "2.27"
  info "Running gosec"
  run "gosec ./..."
```

To opt a requirement into automatic installation or version mutation, add a trailing `provision` marker to that tool line:

```drun
project "myapp":
  requires tools:
    golangci-lint >= "1.55" provision
    govulncheck provision

task "security":
  requires tools:
    gosec >= "2.27" provision
  info "Running gosec"
  run "gosec ./..."
```

**Key Features:**

- **Fail-Fast By Default**: Missing tools or unsatisfied versions cause execution to halt immediately with a clear error unless that specific line ends with `provision`.
- **Multiple Bounds**: Supports chaining comparison operators (`>=`, `<=`, `>`, `<`) on the same line to define a range.
- **Bare Tool Names**: Writing just the tool name (e.g., `docker`) asserts that the executable must be present on `PATH`, without version checking.
- **Unquoted Versions**: Supports both quoted (`"1.21"`) and unquoted (`1.21`) version numbers.
- **Per-Line Provisioning Intent**: `provision` applies only to the requirement on the same line. Other tools in the block keep their normal fail-fast behavior unless they also opt in.

**Provisioning Semantics:**

- **Missing Tool + `provision`**: If the executable is missing and the requirement line ends with `provision`, drun resolves a provisioning entry for that tool, runs it, and then re-checks the requirement before continuing.
- **Missing Tool Without `provision`**: If the executable is missing and the line does not end with `provision`, execution fails immediately.
- **Version Mismatch + `provision`**: If the tool exists but does not satisfy the declared version constraint, drun warns and refuses to mutate the installed version unless the run was started with `--allow-tool-version-changes`.
- **Version Mismatch + Flag**: With `--allow-tool-version-changes`, a provisionable requirement may upgrade or downgrade the installed tool to satisfy the declared constraint, then must re-run detection and version validation before execution continues.
- **Provisioning Failure**: If no provisioning entry exists, provisioning exits non-zero, or the post-provision re-check still fails, the enclosing project/task fails before any dependent work runs.

`requires tools:` validates installation and version only. Even with `provision`, it does not verify that background services or daemons are currently reachable. For runtime health checks, use detection conditions such as `if docker is running:`.

#### Provisioning Sources

When a requirement opts into provisioning, drun resolves installers from one or more catalog sources. Sources can be declared directly in the project:

```drun
project "myapp":
  provisioning sources:
    "./.drun/provisionings.yaml"
    "./tooling"
    "https://example.com/drun/provisionings.yaml"
    "github:acme/devx-catalog/catalog/provisionings.yaml@main"
    "ssh://git@github.com/acme/internal-tooling.git//catalog/provisionings.yaml?ref=main"

  requires tools:
    golangci-lint >= "1.64" provision
    gosec >= "2.22" <= "2.22" provision
```

`provisioning sources:` is a project-level setting. Each entry is a catalog source searched in declaration order. Supported source kinds are:

- Local manifest file path such as `./.drun/provisionings.yaml`
- Local directory path such as `./tooling`, which implies `./tooling/provisionings.yaml`
- HTTPS manifest URL
- GitHub shorthand in the form `github:owner/repo/path/provisionings.yaml@ref`
- SSH-backed Git manifest URL in the form `ssh://git@host/org/repo.git//path/provisionings.yaml?ref=<branch-or-tag>`

User-level fallback sources may also be declared in `~/.drun/config.yml`:

```yaml
provisioningSources:
  - "~/.drun/provisionings.yaml"
  - "github:acme/shared-tooling/catalog/provisionings.yaml@stable"
```

drun also ships a tiny embedded fallback catalog for smoke testing and last-resort fallback behavior. The substantive first-party tool catalog lives in the official `phillarmonic/drun-provisionings` repository and is consulted before the embedded fallback.

An end-to-end example that combines project overrides, exact-version forwarding, the implicit first-party catalog, and the embedded fallback lives at `examples/73-tool-provisioning.drun`.

#### `provisionings.yaml` v1

Provisioning catalogs are YAML manifests with schema version `1`. Like template catalogs, they may define provisionings as either a map or a sequence.

Map form:

```yaml
version: "1"
provisionings:
  golangci-lint:
    description: "Install golangci-lint"
    targets:
      - os: darwin
        arch: arm64
        install: "brew install golangci-lint"
        install_versioned: "brew install golangci-lint@{version}"
      - os: linux
        install: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        install_versioned: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@v{version}"
```

Equivalent sequence form:

```yaml
version: "1"
provisionings:
  - name: golangci-lint
    description: "Install golangci-lint"
    targets:
      - os: darwin
        arch: arm64
        install: "brew install golangci-lint"
        install_versioned: "brew install golangci-lint@{version}"
```

Each provisioning entry supports:

- `name`: required in sequence form; implied by the map key in map form
- `description`: optional human-facing explanation
- `aliases`: optional alternative executable names that resolve to the same entry
- `targets`: required list of installer targets

Each target supports:

- `os`: optional operating system selector such as `darwin`, `linux`, or `windows`
- `arch`: optional CPU selector such as `amd64` or `arm64`
- `install`: required command used when no exact version should be passed
- `install_versioned`: optional command template used when drun has one exact requested version to pass

Within a single manifest, provisioning names must be unique after alias expansion. Duplicate entries at the same name are invalid.

#### Catalog Resolution And Specificity

drun resolves provisioning entries using this precedence order:

1. Project `provisioning sources:` in declaration order
2. User `provisioningSources` from `~/.drun/config.yml` in declaration order
3. Official first-party `phillarmonic/drun-provisionings` catalog
4. Embedded drun default catalog

The first source that contains a matching provisioning entry wins. drun does not merge multiple catalogs for the same tool during a single lookup.

Inside the chosen source, drun picks the most specific installer target in this order:

1. Matching `name` or `alias` with exact `os` and exact `arch`
2. Matching `name` or `alias` with exact `os` and no `arch`
3. Matching `name` or `alias` with no `os` and no `arch`

If two targets in the same manifest have identical specificity for the same tool, the manifest is invalid and provisioning must fail with an ambiguity error.

#### Exact Version Forwarding

Provisioning lookups always use the tool name, but version arguments are forwarded only when the requirement requests one exact version and the selected target provides `install_versioned`.

- `gosec >= "2.22" <= "2.22" provision` forwards `2.22`
- `gosec >= "2.22" provision` does not forward a version because the requirement is open-ended
- `gosec provision` does not forward a version

Example:

```drun
project "quality":
  provisioning sources:
    "./.drun/provisionings.yaml"

  requires tools:
    gosec >= "2.22" <= "2.22" provision
    dummy-tool >= "1.2.3" <= "1.2.3" provision
```

In that example, `gosec` forwards `2.22` to `install_versioned` when the selected target supports it. If `dummy-tool` is not defined by the project or user catalogs, drun continues to the implicit first-party catalog and finally the embedded fallback catalog before failing.

If drun derives one exact version but the chosen target omits `install_versioned`, it falls back to `install`. If the requirement can only be satisfied by mutating to an exact version and the catalog has no version-aware installer path, provisioning fails before execution continues.

#### Dynamic Detection

The language also automatically detects available tools and uses appropriate commands when you prefer not to use strict requirements:

```drun
# Automatically uses "docker compose" or "docker-compose"
start docker compose services

# Automatically uses "docker buildx" or "docker build"
build multi-platform docker image

# Detects kubectl, helm, etc.
deploy to kubernetes
install helm chart "nginx-ingress"

# Check if tools are available or not available
if docker is available:
    info "Docker is ready"
else:
    error "Docker not found"

if docker is running:
    info "Docker daemon is reachable"
else:
    fail "Docker is installed, but the daemon is not reachable"

if kubectl is not available:
    warn "Kubernetes tools not installed"
    info "Skipping Kubernetes deployment"
```

`available` checks whether the command can be found and invoked. `running` is stricter and is intended for tools with a runtime component, such as Docker, where the CLI may exist even if the daemon/socket is unavailable.

#### Supported Tool Keywords

The following tools are recognized as built-in keywords and can be used without quotes:

**Package Managers & Runtimes:**

- `node` - Node.js runtime
- `npm` - Node Package Manager
- `yarn` - Yarn package manager
- `pnpm` - PNPM package manager
- `bun` - Bun JavaScript runtime and package manager
- `python` - Python interpreter
- `pip` - Python package installer
- `go` / `golang` - Go programming language
- `cargo` - Rust package manager
- `java` - Java runtime
- `maven` - Apache Maven build tool
- `gradle` - Gradle build tool
- `ruby` - Ruby interpreter
- `gem` - RubyGems package manager
- `php` - PHP interpreter
- `composer` - PHP dependency manager
- `rust` - Rust programming language
- `make` - GNU Make build tool

**Container & Orchestration:**

- `docker` - Docker container platform
- `kubectl` - Kubernetes command-line tool
- `helm` - Kubernetes package manager

**Infrastructure & Cloud:**

- `terraform` - Infrastructure as Code tool
- `aws` - AWS CLI
- `gcp` - Google Cloud CLI
- `azure` - Azure CLI

**Version Control:**

- `git` - Git version control system

**Note:** For tools with spaces or tools not in this list, use quoted strings:

```drun
if "docker compose" is available:
    info "Using Docker Compose v2"

if "docker-compose" is available:
    info "Using Docker Compose v1"
```

### Docker Compose Command Macro

Use the built-in Docker Compose macro when a task must work with either Compose V2 (`docker compose`) or the standalone Compose V1 command (`docker-compose`):

```drun
# Automatically resolves to "docker compose" or "docker-compose"
run "{docker compose command} up -d"
run "{docker compose command} ps"
run "{docker compose command} logs"
```

The shorter `{compose_cmd}` alias has the same behavior:

```drun
run "{compose_cmd} up -d"
run "{compose_cmd} logs --tail=100"
```

The macro prefers `docker compose` and falls back to `docker-compose`. Execution fails with a clear error if neither command is available.

### DRY Tool Detection

For other tools with multiple possible commands, detect the available variant and capture it in a variable:

```drun

# Multiple tool alternatives
detect available "npm" or "yarn" or "pnpm" as $package_manager
run "{$package_manager} install"
run "{$package_manager} run build"

# Docker Buildx variants
detect available "docker buildx" or "docker-buildx" as $buildx_cmd
run "{$buildx_cmd} build --platform linux/amd64,linux/arm64 ."
```

#### Benefits

- **DRY Principle**: No repetitive conditional logic
- **Cross-Platform**: Works across different tool installations
- **Maintainable**: Single detection point, consistent usage
- **Flexible**: Supports any number of tool alternatives

### Project Detection

```drun
# Detects package.json, yarn.lock, pnpm-lock.yaml
install dependencies                    # Uses npm, yarn, or pnpm

# Detects go.mod
build go application

# Detects requirements.txt, pyproject.toml
install python dependencies

# Detects Dockerfile, docker-compose.yml
build containerized application
```

### Environment Detection

```drun
# CI/CD detection
when running in CI:
  use non-interactive mode
  enable verbose logging

when running locally:
  enable development features
  use local configuration

# Platform detection
when running on macOS:
  use homebrew for dependencies

when running on Linux:
  use system package manager
```

### Environment Variable Interpolation  *New*

Drun supports shell-style environment variable interpolation using `${VAR}` syntax:

**Syntax:**

- `{$var}` - Drun variable (from parameters, captures, etc.)
- `${VAR:-default}` - Environment variable with default value
- `${VAR}` - Environment variable without default (fails if not set)

**Examples:**

```drun
task show-config:
	# With default values
	echo "User: ${USER:-unknown}"
	echo "Home: ${HOME:-/home/default}"
	echo "Shell: ${SHELL:-/bin/sh}"

	# Required environment variables (no default - will fail if not set)
	echo "API URL: ${API_URL}"
	echo "Database: ${DATABASE_URL}"

	# Combining with Drun variables
	capture from shell "date" as $timestamp
	echo "Timestamp: {$timestamp}"
	echo "User: ${USER:-unknown}"
```

**Key Features:**

- **Shell-style syntax**: Familiar `${VAR:-default}` pattern from bash/sh
- **Default values**: Use `:-` syntax to provide fallback values
- **Required variables**: Variables without defaults will fail if not set
- **OS environment**: Accesses environment variables from the shell
- **Safe defaults**: Prevents errors when optional config is missing
- **Integration**: Works seamlessly with `.env` file loading

### Environment Variable Conditionals  *New*

Check environment variables with clean, semantic syntax for conditional logic:

#### Basic Existence Checks

```drun
# Check if environment variable exists
if env HOME exists:
  success "HOME is set"
  capture from shell "echo $HOME" as $home
  echo "Home directory: {$home}"
else:
  error "HOME is not set"

# Check multiple environment variables
if env USER exists:
  info "User: {env('USER')}"

if env PATH exists:
  info "PATH is configured"
```

#### Value Comparison

```drun
# Check if environment variable equals a specific value
if env APP_ENV is "production":
  warn "  Running in PRODUCTION environment"
  info "Extra caution advised!"
else:
  info " Not in production environment"

# Check if environment variable is NOT equal to a value
if env DEBUG_MODE is not "true":
  info "Debug mode is disabled"
```

#### Empty/Non-Empty Checks

```drun
# Check if environment variable is not empty
if env DATABASE_URL is not empty:
  success " DATABASE_URL is configured"
  run "python manage.py migrate"
else:
  warn "  DATABASE_URL is not set"
  info "Set it with: export DATABASE_URL=postgresql://..."
  fail "Missing required database configuration"
```

#### Compound Conditions  *New*

Combine multiple checks with `and` for more precise validation:

```drun
# Ensure variable exists AND is not empty (rejects empty strings)
task "secure-deploy":
  if env API_TOKEN exists and is not empty:
    success " API_TOKEN is properly configured"
    run "curl -H 'Authorization: Bearer ${API_TOKEN}' https://api.example.com/deploy"
  else:
    error " API_TOKEN must be set and not empty"
    fail "Missing required credentials"

# Ensure variable exists AND equals specific value
task "production-check":
  if env DEPLOY_ENV exists and is "production":
    warn "  Confirmed production deployment"
    info "Running extra validation..."
    run "npm run test:integration"
  else:
    info " Non-production environment"

# Build with optional build arguments
task "docker-build":
  if env BUILD_TOKEN exists and is not empty:
    info " Using authenticated build"
    run "docker build --build-arg TOKEN='${BUILD_TOKEN}' -t myapp ."
  else:
    info " Using public build (no authentication)"
    run "docker build -t myapp ."
```

#### Practical Examples

```drun
# Conditional deployment based on environment
task "deploy":
  if env DEPLOY_ENV is "production":
    warn "  Deploying to PRODUCTION"
    info "Running production pre-flight checks..."

    if env DATABASE_URL exists:
      success " Database configuration found"
    else:
      error " DATABASE_URL required for production"
      fail "Missing required environment variable"

    if env API_KEY exists:
      success " API key found"
    else:
      error " API_KEY required for production"
      fail "Missing required API credentials"

    success " All pre-flight checks passed"
    info "Deploying to production..."
  else:
    info " Deploying to development/staging"
    info "Skipping production pre-flight checks"

# CI/CD detection
task "build":
  if env CI exists:
    info " Running in CI environment"
    set $ci_mode to "true"
    run "npm run build --ci"
  else:
    info " Running locally"
    set $ci_mode to "false"
    run "npm run build"

# Configuration based on environment variables
task "configure":
  if env LOG_LEVEL is "debug":
    info " Using DEBUG log level"
    set $verbose to "true"
  else:
    if env LOG_LEVEL is "info":
      info "  Using INFO log level"
      set $verbose to "false"
    else:
      info " Using default log level"
      set $verbose to "false"

# Feature flags
task "start":
  if env ENABLE_EXPERIMENTAL is "true":
    info " Experimental features enabled"
    run "npm run start:experimental"
  else:
    info " Using stable version"
    run "npm run start"
```

#### Syntax Variants

```drun
# OLD SYNTAX (still supported via builtin functions)
set $var_exists to "{env exists(HOME)}"
when $var_exists is "true":
  success "HOME exists"

# NEW SYNTAX (recommended - cleaner and more readable)
if env HOME exists:
  success "HOME exists"
```

**Key Features:**

- **Clean syntax**: `if env VAR exists` is more readable than function-based checks
- **Value comparison**: Check if env var equals specific values
- **Empty checks**: Use `is not empty` to validate required configuration
- **OS environment**: Checks environment variables available when drun starts
- **Integration**: Works seamlessly with `.env` file loading (see `.env` loading section)

**Supported Conditions:**

- `if env VAR exists` - Check if variable is set
- `if env VAR is "value"` - Check if variable equals value
- `if env VAR is not "value"` - Check if variable does not equal value
- `if env VAR is not empty` - Check if variable has a value
- `if env VAR exists and is not empty` - Check if variable is set AND has a non-empty value
- `if env VAR exists and is "value"` - Check if variable is set AND equals specific value

### Framework Detection

```drun
# Web frameworks
when symfony is detected:
  run symfony console commands
  use symfony-specific deployment

when laravel is detected:
  run artisan commands
  migrate database

when rails is detected:
  run rake tasks
  precompile assets

# Build tools
when webpack is detected:
  build with webpack

when vite is detected:
  build with vite
```

---

## Execution Model

### Execution Pipeline

1. **Lexical Analysis**: Tokenize source code into semantic tokens
2. **Parsing**: Build Abstract Syntax Tree (AST) from tokens
3. **Semantic Analysis**: Type checking, scope resolution, validation
4. **Smart Detection**: Analyze project structure and available tools
5. **Direct Execution**: Execute AST nodes directly through the v2 engine
6. **Runtime Integration**: Interface with shell, tools, and external systems

### Native Execution

The semantic language executes directly without intermediate compilation:

#### Source (Semantic v2):
```drun
task "deploy" means "Deploy to environment":
  requires environment from ["dev", "staging", "production"]
  depends on build and test

  deploy myapp:latest to kubernetes namespace {environment}
```

#### Execution Flow:
1. **Parse**: Convert source to AST with task dependencies and actions
2. **Validate**: Check parameter constraints and dependencies
3. **Execute**: Run dependency tasks first, then execute deployment actions
4. **Runtime**: Execute shell commands with parameter substitution

### Smart Execution

#### Docker Command Execution

```drun
# Source
build docker image "myapp:{version}"

# Runtime Detection & Execution
if dockerBuildx available:
  execute: docker buildx build -t myapp:${version} .
else:
  execute: docker build -t myapp:${version} .
```

#### Kubernetes Command Execution

```drun
# Source
deploy myapp:latest to kubernetes namespace production with 5 replicas

# Generated
kubectl set image deployment/myapp myapp=myapp:latest --namespace=production
kubectl scale deployment/myapp --replicas=5 --namespace=production
kubectl rollout status deployment/myapp --namespace=production
```

### Optimization Strategies

#### Command Batching

```drun
# Source
copy "file1.txt" to "dest/"
copy "file2.txt" to "dest/"
copy "file3.txt" to "dest/"

# Optimized
cp file1.txt file2.txt file3.txt dest/
```

#### Conditional Optimization

```drun
# Source
if docker is running:
  build docker image

# Optimized (check once, reuse result)
if docker info >/dev/null 2>&1; then
  docker build -t myapp .
fi
```

---

## Error Handling

### Compile-Time Errors

#### Syntax Errors

```drun
# Missing colon
task "example"
  info "Hello"
# Error: Expected ':' after task declaration

# Invalid parameter constraint
requires port as number between "low" and "high"
# Error: Range bounds must be numeric values
```

#### Type Errors

```drun
# Type mismatch
let count be "not a number"
for i from 1 to count:
  # Error: Range bounds must be numeric
```

#### Scope Errors

```drun
task "example":
  if condition:
    let local_var be "value"

  info local_var  # Error: Variable not in scope
```

### Runtime Errors

#### Command Failures

```drun
# Automatic error handling
try:
  deploy to production
catch deployment_error:
  rollback deployment
  notify team of failure
```

#### Resource Not Found

```drun
# File not found
if file "config.json" exists:
  load configuration from "config.json"
else:
  error "Configuration file not found"
  fail
```

#### Network Errors

```drun
# Network timeout
try:
  check health of service at "https://api.example.com"
catch timeout_error:
  warn "Service health check timed out"
  continue with deployment
```

### Error Recovery

#### Retry Logic

```drun
for attempt from 1 to 3:
  try:
    deploy to production
    break  # Success, exit retry loop
  catch deployment_error:
    if attempt == 3:
      fail "Deployment failed after 3 attempts"
    warn "Deployment attempt {attempt} failed, retrying..."
    wait {attempt * 5} seconds
```

#### Graceful Degradation

```drun
try:
  deploy with blue-green strategy
catch blue_green_error:
  warn "Blue-green deployment failed, falling back to rolling update"
  deploy with rolling update strategy
```

---
