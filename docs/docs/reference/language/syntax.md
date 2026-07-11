# Syntax specification

## Syntax Specification

### Project Declaration

```drun
project <name> [version <version>]:
  [project_settings]

# Examples:
project "myapp"
project "ecommerce" version "2.1.0"

project "microservices":
  set registry to "ghcr.io/company"
  set default_timeout to "5m"
  include "shared/common.drun"
```

### Shell Configuration

drun v2 supports cross-platform shell configuration with sensible defaults for each operating system. This allows you to specify different shell executables, startup arguments, and environment variables for different platforms.

```drun
project "my-app":
  shell config:
    mac:
      executable: "/bin/zsh"
      args:
        - "-l"
        - "-i"
      environment:
        TERM: "xterm-256color"
        SHELL_SESSION_HISTORY: "0"

    linux:
      executable: "/bin/bash"
      args:
        - "--login"
        - "--interactive"
      environment:
        TERM: "xterm-256color"
        HISTCONTROL: "ignoredups"

    windows:
      executable: "powershell.exe"
      args:
        - "-NoProfile"
        - "-ExecutionPolicy"
        - "Bypass"
      environment:
        PSModulePath: ""
```

#### Platform Detection

drun automatically detects the current platform using Go's `runtime.GOOS`:

- **mac**: macOS (legacy alias: `darwin`)
- **linux**: Linux distributions
- **windows**: Windows

`mac` is the canonical user-facing spelling in drun. Existing specs may continue to use `darwin`; drun normalises both spellings to the same platform internally.

### Declaration Annotations

drun v2 supports declaration decorators immediately before tasks, template tasks, and snippets:

```drun
@platform("linux", "mac")
task "shell" means "Open a Unix shell":
  run "bash" attached

@platform("windows")
task "shell" means "Open PowerShell":
  run "pwsh.exe" attached
```

#### `@platform(...)`

Platform allows you to have platform-bound specific implementation of tasks. This is especially useful when working with environments that require Windows, since they tend to have completely different workflows than UNIX-like systems, such as Linux and macOS. Directory slashes, for example, are different on Windows (except on Git Bash).

- Accepted values: `linux`, `mac`, `windows`
- Legacy alias: `darwin` is accepted and normalized to `mac`
- A declaration may list one or more platforms
- Unknown annotations are rejected

For **tasks** specifically, `@platform(...)` also enables platform-aware duplicate names. A task family may contain any number of disjoint platform-tagged variants plus at most one unannotated fallback variant. drun resolves the correct variant automatically when the task is invoked: exact platform match first, then the unannotated fallback if one exists.

If no variant matches the current platform, execution fails with a clear error listing the available variants.

#### Configuration Options

Each platform configuration supports:

- **executable**: Path to the shell executable (e.g., `/bin/zsh`, `/bin/bash`, `powershell.exe`)
- **args**: Array of startup arguments passed to the shell
- **environment**: Key-value pairs of environment variables set for all shell commands

#### Default Behavior

If no shell configuration is provided, drun uses sensible defaults:

- **Shell**: `/bin/sh` on Unix-like systems, system default on Windows
- **Args**: Basic shell invocation (`-c` for command execution)
- **Environment**: Inherits from parent process

#### Usage in Tasks

All shell commands (`run`, `exec`, `shell`, `capture`) automatically use the platform-specific configuration:

```drun
task "example":
  run "echo $SHELL"        # Uses configured shell
  run "echo $TERM"         # Uses configured environment
  capture "whoami" as $user # Uses configured shell and environment
```

### Lifecycle Hooks

drun v2 supports two types of lifecycle hooks that allow you to execute code at different points in the execution pipeline:

#### Task-Level Lifecycle Hooks

These hooks run around individual task execution:

```drun
project "myapp":
  before any task:
    info " Starting task: {$globals.current_task}"
    capture task_start_time from now

  after any task:
    capture task_end_time from now
    let task_duration be {task_end_time} - {task_start_time}
    info " Task completed in {task_duration}"
```

- **`before any task`**: Executes before each individual task runs
- **`after any task`**: Executes after each individual task completes

#### Tool-Level Lifecycle Hooks

These hooks run once per drun execution, providing tool-level startup and shutdown capabilities:

```drun
project "myapp":
  on drun setup:
    info " Starting drun execution pipeline"
    info " Tool version: {$globals.drun_version}"
    capture pipeline_start_time from now

  on drun teardown:
    capture pipeline_end_time from now
    let total_time be {pipeline_end_time} - {pipeline_start_time}
    info " Drun execution pipeline completed"
    info " Total execution time: {total_time}"
```

- **`on drun setup`**: Executes once at the very beginning of drun execution (before any tasks)
- **`on drun teardown`**: Executes once at the very end of drun execution (after all tasks complete)

#### Execution Order

When both types of lifecycle hooks are present, they execute in this order:

1. **`on drun setup`** - Tool startup (once)
2. **`before any task`** - Before target task (once per task)
3. **Task execution** - The actual task(s)
4. **`after any task`** - After target task (once per task)
5. **`on drun teardown`** - Tool shutdown (once)

#### Use Cases

**Task-Level Hooks** are ideal for:
- Task-specific logging and timing
- Setting up task-specific environment
- Task cleanup operations

**Tool-Level Hooks** are ideal for:
- Global initialization and cleanup
- Pipeline-wide logging and metrics
- Tool version reporting
- Overall execution timing

### Task Definition

```drun
task <name> [means <description>]:
  [parameters]
  [dependencies]
  [lifecycle_hooks]
  [variables]
  <statements>

# Examples:
task "hello":
  info "Hello, world!"

task "deploy" means "Deploy application to environment":
  requires $environment from ["dev", "staging", "production"]
  depends on build and test

  deploy myapp to kubernetes namespace {$environment}
```

### Task Calling

Tasks can call other tasks directly using the `call task` statement. This allows for code reuse and modular task design.

#### Basic Syntax

```drun
call task "task_name"
```

Task names can be specified with or without quotes, depending on the naming pattern:

**Unquoted task names** (no quotes required):
- Single words: `call task test`, `call task build`
- Snake_case: `call task run_tests`, `call task hello_world`
- Keywords: `call task test`, `call task ci`, `call task build`

**Quoted task names** (quotes required):
- Kebab-case: `call task "hello-world"`, `call task "run-tests"`
- Multi-word: `call task "hello world"`, `call task "run tests"`
- Names with special characters or spaces

**Note**: Hyphens (`-`) are tokenized as operators, so kebab-case names like `my-task` must be quoted. Underscores (`_`) are part of identifiers, so snake_case names like `my_task` can be unquoted.

```drun
# Valid unquoted forms
call task test
call task build_app
call task hello_world

# Requires quotes
call task "hello-world"
call task "build app"
call task "my-special-task"
```

#### With Parameters

```drun
call task "task_name" with param1="value1" param2="value2"
call task task_name with param1="value1" param2="value2"  # Unquoted task name
call task fuzz with iterations=100                         # Numeric literals are allowed bare
```

Task call parameter values currently support:

- Quoted strings: `name="Alice"`
- Bare numbers: `iterations=100`, `replicas=3`

Bare numeric literals are passed through as string values to the called task, which keeps task-call syntax aligned with normal drun parameter handling while avoiding unnecessary quotes for numeric inputs.

#### Examples

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
  call task fuzz with iterations=100

  success "Full pipeline completed successfully!"
```

#### Key Features

- **Parameter Passing**: Pass parameters to called tasks using `with param="value"` or `with param=123` syntax
- **Variable Sharing**: Variables set in called tasks are available in the calling task
- **Error Handling**: If a called task fails, the calling task fails with an appropriate error message
- **Execution Flow**: Called tasks execute completely before returning control to the calling task
- **Dry Run Support**: Task calls are properly handled in dry-run mode

#### Parameter Handling

Parameters passed to called tasks override any default values defined in the called task:

```drun
task "greet":
  given $name defaults to "World"
  info "Hello, {$name}!"

task "main":
  call task "greet"                    # Uses default: "Hello, World!"
  call task "greet" with name="Alice"  # Uses passed value: "Hello, Alice!"
```

#### Error Handling

If a called task doesn't exist, the execution fails with a clear error message:

```drun
task "main":
  call task "nonexistent"  # Error: task 'nonexistent' not found
```

### Parameter Declarations

drun has two types of parameters with distinct semantic meanings:

#### `requires` - Mandatory Parameters

**Semantic Intent:** "This parameter is essential for the task to execute correctly."

Parameters declared with `requires` **MUST** be provided by the user (unless a default is specified).

```drun
requires <name> [constraints] [defaults to <value>]

# Examples:
requires $environment from ["dev", "staging", "production"]
requires $version matching pattern "v\d+\.\d+\.\d+"
requires $port as number between 1000 and 9999
requires $email matching email format
requires $files as list of strings

# Optional required parameter (validated with safe default):
requires $environment from ["dev", "staging", "production"] defaults to "dev"
```

**Key Characteristics:**
- Must be provided by user (if no default)
- Often used with validation constraints (enums, patterns, ranges)
- Emphasizes importance and criticality
- Can have defaults for convenience while maintaining validation

#### `given` - Optional Parameters (with optional defaults)

**Semantic Intent:** "This parameter is configurable and optional."

Parameters declared with `given` are optional. They _may_ specify a default value but are no longer required to do so. When no default is supplied, the parameter resolves to an empty string unless populated at runtime.

```drun
given <name> defaults to <value> [constraints]

# Examples:
given $replicas defaults to "3"
given $timeout defaults to "5m"
given $force defaults to "false"
given $tags defaults to [] as list of strings
given $features defaults to empty  # equivalent to ""
given $service_name  # optional without explicit default

# Optional with enum validation (NEW!):
given $log_level from ["error", "warn", "info", "debug"] defaults to "info"

# Built-in function defaults:
given $version defaults to "{current git commit}"
given $branch defaults to "{current git branch}"
given $safe_branch defaults to "{current git branch | replace '/' by '-'}"
given $timestamp defaults to "{now.format('2006-01-02-15-04-05')}"
```

**Key Characteristics:**
- Default value is optional (defaults to empty string when omitted)
- User can override but doesn't have to
- Used for configuration, feature flags, optional overrides
- Can also have validation constraints (enums, types)

#### Comparison Table

| Feature | `requires` | `given` |
|---------|------------|---------|
| **Must provide value?** | Yes (unless has default) | No (always optional) |
| **Default value** | Optional | Optional (defaults to empty string) |
| **Semantic meaning** | Essential/Critical | Configurable/Optional |
| **Validation** | Recommended | Optional |
| **Use case** | Core parameters | Configuration options |

#### Usage Examples

```drun
task "deploy":
  # Critical parameter - must be provided
  requires $name

  # Validated required parameter with safe default
  requires $environment from ["dev", "staging", "production"] defaults to "dev"

  # Optional configuration with default
  given $replicas defaults to "3"
  given $timeout defaults to "30s"

  info "Deploying {$name} to {$environment} with {$replicas} replicas"
```

**CLI Usage:**
```bash
# Must provide 'name', others use defaults
xdrun deploy name=myapp
# Output: Deploying myapp to dev with 3 replicas

# Override defaults
xdrun deploy name=myapp environment=production replicas=5
# Output: Deploying myapp to production with 5 replicas
```

#### The `empty` Keyword

The `empty` keyword provides a semantic way to specify empty values and is completely interchangeable with empty strings (`""`):

```drun
# Default value usage
given $name defaults to empty
given $features as list defaults to empty
given $config defaults to ""  # equivalent to empty

# Condition usage
if $features is empty:
  info "No features specified"

if $features is not empty:
  info "Features: {$features}"

# The empty keyword works with all parameter types
given $message defaults to empty     # string parameter
given $items as list defaults to empty  # list parameter (empty list)
given $enabled defaults to false    # boolean parameter (use false, not empty)
```

**Key Features:**
- `empty` is semantically equivalent to `""` (empty string)
- Works as default values for any parameter type
- Works in conditional expressions (`is empty`, `is not empty`)
- For list parameters, `empty` creates an empty list `[]`
- More readable than empty quotes in semantic contexts

#### Variadic Parameters

```drun
accepts <name> as list [of <type>]

# Examples:
accepts features as list
accepts ports as list of numbers
accepts configs as list of strings
```

### Dependencies

```drun
depends on <dependency_list>

# Sequential dependencies
depends on build and test then deploy

# Parallel dependencies
depends on lint, test, security_scan

# Mixed dependencies
depends on build then test, integration_test then deploy
```

### Variable Declarations

#### Simple Assignment

```drun
let <name> be <expression>
set <name> to <expression>

# Examples:
let image_name be "myapp:latest"
set build_time to now
let git_hash be current git commit
```

#### Capture from Commands

drun v2 supports two types of capture operations:

#### Expression Capture
Captures values from expressions, functions, and built-in operations:

```drun
capture <name> from <expression>

# Examples:
capture start_time from now
capture branch_name from current git branch
capture calculated_value from {a} + {b}
```

#### Shell Command Capture
Captures output from shell commands:

```drun
capture from shell "<command>" as $<variable>

# Examples:
capture from shell "docker ps --format json" as $running_containers
capture from shell "df -h /" as $disk_usage
capture from shell "whoami" as $current_user
```

#### Multiline Shell Command Capture

For complex shell operations that span multiple commands, use the multiline syntax:

```drun
capture from shell as $<variable>:
  <command1>
  <command2>
  <command3>
```

**Examples:**

```drun
# Capture system information
capture from shell as $system_info:
  echo "System Information:"
  echo "User: $(whoami)"
  echo "Date: $(date)"
  echo "Hostname: $(hostname)"
  echo "Working Directory: $(pwd)"

# Capture build information
capture from shell as $build_details:
  echo "Build Details:"
  echo "Commit: $(git rev-parse --short HEAD)"
  echo "Branch: $(git branch --show-current)"
  echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "Built by: $(whoami)"

# Capture file analysis
capture from shell as $file_report:
  echo "File Analysis Report:"
  echo "Text files: $(find . -name '*.txt' | wc -l)"
  echo "Markdown files: $(find . -name '*.md' | wc -l)"
  echo "Total files: $(find . -type f | wc -l)"
```

**Key Features:**
- All commands are executed as a single shell script
- Output from all commands is captured together
- Commands can use shell features like pipes, redirections, and command substitution
- Variable interpolation works within the commands: `echo "Hello {$username}"`
- Each command runs in the same shell session, so environment variables persist

**Key Differences:**
- **Expression capture** uses plain identifiers and supports complex expressions with arithmetic operations
- **Shell capture** uses `$variable` syntax and executes commands in the system shell
- **Expression capture** can reference other variables: `capture result from {a} - {b}`
- **Shell capture** supports variable interpolation in commands: `capture from shell "echo 'Hello {name}'" as $greeting`
- **Multiline shell capture** executes multiple commands as a single script and captures all output

#### Conditional Assignment

```drun
let <name> be:
  when <condition>: <value>
  when <condition>: <value>
  else: <value>

# Example:
let database_url be:
  when environment is "production": secret "prod_db_url"
  when environment is "staging": secret "staging_db_url"
  else: "sqlite:///local.db"
```

### Control Flow

#### If Statements

```drun
if <condition>:
  <statements>
[else if <condition>:
  <statements>]
[else:
  <statements>]

# Examples:
if docker is running:
  build image "myapp"
else:
  error "Docker is not running"

if environment is "production" and git repo is clean:
  deploy to production
else if environment is "staging":
  deploy to staging
else:
  error "Invalid deployment conditions"
```

#### Enhanced If-Else Chains  *New*

drun v2 supports natural `else if` syntax for cleaner conditional logic:

```drun
task "deployment strategy":
  requires $environment from ["dev", "staging", "production"]

  if $environment == "production":
    info " Production deployment"
    set $replicas to 5
    set $timeout to "300s"
  else if $environment == "staging":
    info " Staging deployment"
    set $replicas to 3
    set $timeout to "180s"
  else if $environment == "dev":
    info " Development deployment"
    set $replicas to 1
    set $timeout to "60s"
  else:
    error "Unknown environment: {$environment}"
    fail

# Multiple else if chains
task "build strategy":
  if file "Dockerfile" exists:
    info "Building with Docker"
    build docker image
  else if file "package.json" exists:
    info "Building Node.js application"
    run "npm ci && npm run build"
  else if file "go.mod" exists:
    info "Building Go application"
    run "go build -o app"
  else if file "requirements.txt" exists:
    info "Building Python application"
    run "pip install -r requirements.txt"
  else:
    warn "No recognized build configuration found"
    info "Skipping build step"
```

**Key Features:**
- **Natural syntax**: `else if` reads like natural English
- **Unlimited chaining**: Support for multiple `else if` conditions
- **Proper precedence**: Conditions evaluated in order, first match wins
- **Optional else**: Final `else` clause is optional

#### When Statements

The current parser-backed `when` form is a conditional block with an optional `otherwise` branch:

```drun
when <condition>:
  <statements>
otherwise:
  <statements>

# Example:
when $package_manager is "npm":
  run "npm ci && npm run build"
otherwise:
  error "Unsupported package manager: {$package_manager}"
```

Use `when in <environment> environment:` for environment-targeted detection blocks:

```drun
when in ci environment:
  info "Running in CI"
else:
  info "Running locally"
```

#### For Loops

```drun
for each <variable> in <expression> [in parallel]:
  <statements>

# Examples with array literals:
for each $env in ["dev", "staging", "prod"]:
  deploy to {$env}

for each $service in microservices in parallel:
  test service {$service}

# Matrix execution (nested loops)
for each $os in ["ubuntu", "alpine", "debian"]:
  for each $version in ["16", "18", "20"]:
    test on {$os} with node {$version}

# Parallel matrix execution
for each $region in ["us-east", "eu-west"] in parallel:
  for each $service in ["api", "web", "worker"]:
    deploy {$service} to {$region}
```

#### Exception Handling

```drun
try:
  <statements>
[catch <error_type>:
  <statements>]
[finally:
  <statements>]

# Example:
try:
  deploy to production
catch timeout_error:
  warn "Deployment timed out"
  rollback deployment
catch permission_error:
  error "Insufficient permissions"
  fail
finally:
  cleanup temporary resources
```

---
