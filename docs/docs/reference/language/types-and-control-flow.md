# Types and control flow

## Type System

### Primitive Types

- **string**: Text values, support interpolation
- **number**: Integer and floating-point numbers
- **boolean**: `true` or `false`
- **duration**: Time durations (e.g., "5m", "2h", "30s")

### Collection Types

- **array**: Ordered list of values `[1, 2, 3]`
- **object**: Key-value pairs `{name: "value", count: 42}`

### Special Types

- **command**: Shell command that can be executed
- **path**: File system path with validation
- **url**: URL with protocol validation
- **regex**: Regular expression pattern
- **secret**: Secure value (not logged in plain text)

### Type Inference

The compiler infers types based on context:

```drun
let count be 42                    # number
let name be "hello"                # string
let enabled be true                # boolean
let timeout be "5m"                # duration
let files be ["a.txt", "b.txt"]    # array of strings
let config be {port: 8080}         # object
```

### Type Constraints

Parameters can specify type constraints:

```drun
requires port as number between 1000 and 9999
requires timeout as duration
requires files as list of paths
requires config as object
```

---

## Control Flow

### Conditional Execution

#### Simple Conditions

```drun
# Boolean conditions
if enabled:
  start service

if not maintenance_mode:
  accept traffic

# Comparison conditions
if replicas > 0:
  scale deployment

if version >= "2.0.0":
  use new features

# Empty/non-empty conditions
if $features is empty:
  info "No features specified"

if $features is not empty:
  info "Features: {$features}"

if $name is "":
  warn "Name is required"

# Folder/directory empty conditions
if folder "build" is empty:
  info "Build directory is empty"

if folder "dist" is not empty:
  info "Distribution files exist"

if directory "/tmp/cache" is empty:
  run "rm -rf /tmp/cache"

if dir "{$output_path}" is not empty:
  warn "Output directory contains files"
```

#### When-Otherwise Conditions

The `when-otherwise` syntax provides a clean alternative to `if-else` for simple conditional logic:

```drun
# Basic when-otherwise
when $platform is "windows":
  step "Building Windows binary with .exe extension"
otherwise:
  step "Building Unix binary without extension"

# When without otherwise (optional else clause)
when $environment is "production":
  step "Deploy with production settings"
  step "Enable monitoring"

# Nested when-otherwise
when $platform is "windows":
  info "Windows platform detected"
  when $arch is "amd64":
    step "Building for Windows x64"
  otherwise:
    step "Building for Windows ARM"
otherwise:
  info "Unix-like platform detected"
  when $platform is "mac":
    step "Building for macOS"
  otherwise:
    step "Building for Linux"

# When-otherwise in loops (matrix execution)
for each $platform in ["windows", "linux", "mac"]:
  when $platform is "windows":
    run "GOOS={$platform} go build -o app.exe"
  otherwise:
    run "GOOS={$platform} go build -o app"
```

**Supported Condition Types:**

- String equality: `$var is "value"`
- String inequality: `$var is not "value"`
- Empty checks: `$var is empty`, `$var is not empty`
- All condition types supported by `if` statements

**Key Features:**

- Clean, readable syntax for simple conditions
- Optional `otherwise` clause (equivalent to `else`)
- Full nesting support
- Works seamlessly with loops and matrix execution
- Consistent variable scoping rules

#### Smart Detection Conditions

```drun
# Tool availability detection
if docker is available:
  build container
else:
  error "Docker is required"

if docker is not available:
  error "Docker is required for this task"
  fail "Missing dependency"

if kubernetes is available:
  deploy to cluster

# Multiple tool availability check
# For "is available": ALL tools must be available (AND logic)
if docker,"docker-compose" is available:
  info "Docker and Docker Compose are both available"

# Alternative: use 'are' for better readability with multiple tools
if docker,"docker-compose" are available:
  info "Docker and Docker Compose are both available"

# For "is not available": ANY tool must be unavailable (OR logic)
if docker,"docker-compose",kubectl is not available:
  error "One or more required tools are missing"
else:
  info "All required tools are available"

# Alternative: use 'are not' for better readability
if docker,"docker-compose",kubectl are not available:
  error "One or more required tools are missing"

# For "is running": ALL tools must be running (AND logic)
if docker,"docker compose" are running:
  info "Docker CLI and daemon are ready"

# For "is not running": ANY tool may be down (OR logic)
if docker,"docker compose" are not running:
  fail "Docker is installed, but the runtime is not reachable"

# Availability can be chained with a version check
if "golangci-lint" is available and version >= "2.12":
  info "golangci-lint is installed and new enough"
else:
  fail "golangci-lint >= 2.12 is required"

# File/directory detection
if file "package.json" exists:
  install npm dependencies

if directory ".git" exists:
  commit changes

# Service detection
when symfony is detected:
  run symfony console commands

when node project exists:
  use npm or yarn
```

#### Compound Conditions

```drun
# Logical operators
if docker is running and kubernetes is available:
  deploy containerized application

if environment is "production" or environment is "staging":
  require approval

# Parentheses for grouping
if (environment is "production" and git repo is clean) or force_deploy:
  proceed with deployment
```

### Iteration

#### Simple Iteration

```drun
for each $item in $collection:
  process item

# With index
for each item at index in collection:
  info "Processing item {index}: {item}"
```

#### Parallel Execution

```drun
for each region in ["us-east", "eu-west"] in parallel:
  deploy to {region}

# Parallel with synchronization
run in parallel:
  - unit_tests -> test_results.unit
  - integration_tests -> test_results.integration
  - security_scan -> test_results.security

wait for all to complete
```

#### Range Iteration

```drun
for port from 3000 to 3005:
  check if port {port} is available

for i from 1 to retry_count:
  try:
    perform operation
    break
  catch:
    if i == retry_count:
      fail "Max retries exceeded"
    wait {i} seconds
```

#### Filtered Iteration

```drun
for each file in "src/**/*.js" where file is modified:
  lint {file}

for each container in docker containers where status is "running":
  check health of {container}
```

### Loop Control

```drun
for each $service in $services:
  if service is healthy:
    continue

  try:
    restart service
  catch:
    error "Failed to restart {service}"
    break  # Exit loop on critical failure
```

---

