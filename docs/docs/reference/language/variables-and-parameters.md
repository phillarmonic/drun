# Variables and parameters

## Variable System

### Variable Declaration

All variables in drun v2 must be prefixed with `$` to distinguish them from keywords and improve syntax clarity.

### Variable Syntax Rules

#### Variable Naming Convention

1. **Declared Variables**: Must start with `$` prefix
   - `$name`, `$environment`, `$commit_hash`
   - Used in: parameter declarations, let/set statements, variable references

2. **Loop Variables**: Use `$` prefix for consistency with scoping system
   - `$item`, `$file`, `$i`, `$attempt`
   - Used in: `for each $item in items`, `for $i in range 1 to 10`

3. **Interpolation Syntax**:
   - Task variables: `{$variable_name}`
   - Project settings: `{$globals.setting_name}`
   - Built-in project vars: `{$globals.project}`, `{$globals.version}`
   - Loop variables: `{$variable_name}`
   - Built-in functions: `{now.format()}`, `{pwd}`
   - Conditional expressions: `{$var ? 'true_val' : 'false_val'}` or `{if $var then 'val1' else 'val2'}`

#### Examples

```drun
# Project settings (no $ prefix in declaration)
project "myapp" version "1.0.0":
  set registry to "ghcr.io/company"
  set api_url to "https://api.example.com"

# Parameter declarations
requires $environment from ["dev", "staging", "production"]
given $tag defaults to "latest"

# Task variable declarations
let $commit = current git commit
set $counter to 0

# Loop variables (with $ prefix)
for each $item in $items:
  info "Processing {$item}"  # Loop variable interpolation

for $i in range 1 to 5:
  info "Attempt {$i} of 5"   # Loop variable interpolation

# Mixed interpolation with different scopes
info "Deploying {$tag} to {$environment} from {$globals.registry}"
info "Project: {$globals.project} v{$globals.version}"
info "API: {$globals.api_url} - Processing item {$item}"
```

#### Let Bindings (Immutable)

```drun
let $name = "value"           # Simple assignment
let $result = compute_value() # Function result
let $config = {              # Object literal
  port: 8080,
  host: "localhost"
}
```

#### Set Statements (Mutable)

```drun
set $counter to 0
set $counter to {$counter} + 1  # Increment

set environment to:
  when running_locally: "development"
  else: "production"
```

#### Capture from Commands

```drun
# Expression capture (for functions and expressions)
capture git_branch from current git branch
capture start_time from now
capture calculated_result from {a} + {b}

# Shell command capture (for shell commands)
capture from shell "docker --version" as $docker_version
capture from shell "kubectl get pods --output=json" as $running_pods
capture from shell "whoami" as $current_user

# With error handling
try:
  capture from shell "systemctl status nginx" as $service_status
catch command_error:
  set $service_status to "unknown"
```

#### Conditional Interpolation

drun v2 supports conditional expressions within interpolation for dynamic value selection. This is particularly useful for optional command flags and environment-specific configuration.

##### Ternary Operator Syntax

The ternary operator provides a concise way to choose between two values based on a boolean condition:

```drun
# Basic ternary: condition ? true_value : false_value
{$var ? 'true_val' : 'false_val'}

# Examples
info "Debug mode: {$debug ? 'enabled' : 'disabled'}"
run "docker build {$no_cache ? '--no-cache' : ''} -t myapp ."
info "Log level: {$verbose ? 'debug' : 'info'}"

# Truthy values: 'true', 'yes', '1', 'on' (case-insensitive)
# All other values are considered falsy
```

##### If-Then-Else Syntax

The if-then-else syntax provides more readable conditional expressions with comparison operators:

```drun
# Simple boolean check
{if $var then 'val1' else 'val2'}

# With 'is' comparison
{if $var is 'value' then 'val1' else 'val2'}

# With 'is not' comparison
{if $var is not 'value' then 'val1' else 'val2'}

# Examples
info "Config: {if $env is 'production' then 'prod.yml' else 'dev.yml'}"
info "Replicas: {if $env is not 'dev' then '3' else '1'}"
run "npm test {if $coverage then '--coverage' : ''}"
```

##### Real-World Examples

**Docker Build with Optional Flags:**
```drun
task "docker-build":
  given $no_cache as boolean defaults to "false"
  given $push as boolean defaults to "false"
  given $platform defaults to "linux/amd64"

  run "docker build {$no_cache ? '--no-cache' : ''} {$push ? '--push' : ''} --platform {$platform} -t myapp:latest ."
```

**Environment-Specific Configuration:**
```drun
task "deploy":
  requires $env from ["dev", "staging", "production"]

  set $replicas to "{if $env is 'production' then '3' else '1'}"
  set $cpu to "{if $env is 'production' then '2000m' else '500m'}"
  set $log_level to "{if $env is 'production' then 'error' else 'debug'}"

  info "Deploying with {$replicas} replicas, {$cpu} CPU, {$log_level} logging"
```

**Build Optimization Flags:**
```drun
task "compile":
  given $optimize as boolean defaults to "true"
  given $debug as boolean defaults to "false"

  run "gcc {$optimize ? '-O2' : '-O0'} {$debug ? '-g' : ''} -o app main.c"
```

**CI/CD Pipeline Flags:**
```drun
task "ci-pipeline":
  given $run_tests as boolean defaults to "true"
  given $coverage as boolean defaults to "false"

  info "Running tests: {$run_tests ? 'YES' : 'SKIP'}"
  run "npm test {if $coverage then '--coverage' else ''}"
```

### Variable Scoping

drun v2 uses a clear scoping system with explicit namespaces to avoid naming conflicts and improve code clarity.

#### Project Scope (Global Variables)

Project-level settings are declared without the `$` prefix and accessed via the `$globals` namespace:

```drun
project "myapp" version "1.0.0":
  set registry to "ghcr.io/company"    # Project setting
  set api_url to "https://api.example.com"
  set timeout to "30s"
  set platforms as list to ["linux", "mac", "windows"]  # Array setting
  set environments as list to ["dev", "staging", "production"]
```

**Accessing Project Settings:**
```drun
task "deploy":
  info "Project: {$globals.project}"        # → "myapp"
  info "Version: {$globals.version}"        # → "1.0.0"
  info "Registry: {$globals.registry}"      # → "ghcr.io/company"
  info "API URL: {$globals.api_url}"        # → "https://api.example.com"
  info "Timeout: {$globals.timeout}"        # → "30s"
```

#### Task Scope (Local Variables)

Task-level variables are declared with the `$` prefix and accessed directly:

```drun
task "deploy":
  set $image_tag to "{$globals.registry}/myapp:{$globals.version}"  # Task-local
  set $replicas to 3

  info "Deploying {$image_tag} with {$replicas} replicas"
```

#### Scoping Rules and Precedence

1. **Project Settings**: Declared without `$`, accessed via `$globals.key`
2. **Task Variables**: Declared with `$`, accessed with `$variable`
3. **Loop Variables**: Use `$` prefix, accessed with `{$variable}`
4. **Built-in Variables**: Special project variables via `$globals.project` and `$globals.version`

**Variable Resolution Order:**
1. Parameters (`$param`)
2. Task variables (`$variable`)
3. Loop variables (`$variable`)
4. Project settings (`$globals.key`)
5. Built-in functions

#### Avoiding Naming Conflicts

The `$globals` namespace prevents conflicts between project settings and task variables:

```drun
project "myapp":
  set api_url to "https://project-level.com"

task "test":
  set $api_url to "https://task-level.com"    # Different variable

  info "Global API: {$globals.api_url}"       # → "https://project-level.com"
  info "Task API: {$api_url}"                 # → "https://task-level.com"
```

#### Nested Scope in Control Structures

```drun
task "deploy":
  set $base_replicas to 3

  if environment is "production":
    set $replicas to {$base_replicas} * 2     # Block-local, shadows outer scope
    info "Production replicas: {$replicas}"   # → 6
  else:
    info "Default replicas: {$base_replicas}" # → 3
```

#### Parameter Scope

```drun
task "greet":
  requires name
  given title defaults to "friend"

  # Parameters are available as variables
  info "Hello, {title} {name}!"
```

### Variable Interpolation

#### String Interpolation

```drun
let name be "world"
let greeting be "Hello, {name}!"

# Complex expressions
let message be "Deployment {version} to {environment} at {now.format('HH:mm')}"

# Nested interpolation
let docker_tag be "{registry}/{app_name}:{version}-{git_commit.short}"
```

#### Command Interpolation

```drun
let image_name be "myapp:latest"
run "docker push {image_name}"

# Multiple interpolations
run "kubectl set image deployment/{app_name} {app_name}={image_name}"
```

#### Strict Variable Checking

**New in v2.0**: drun now enforces strict variable checking by default to prevent runtime errors from undefined variables.

**Default Behavior (Strict Mode)**:
```drun
task "example":
    let $name = "world"
    info "Hello {$name}"        #  Works: variable defined
    info "Hello {$undefined}"   #  Error: undefined variable: {$undefined}
```

**Error Messages**:
```bash
# Single undefined variable
Error: task 'example' failed: in info statement: undefined variable: {$undefined}

# Multiple undefined variables
Error: task 'example' failed: in info statement: undefined variables: {$var1}, {$var2}

# In shell commands
Error: task 'example' failed: in shell command: undefined variable: {$missing}

# In conditions
Error: task 'example' failed: in when condition: undefined variable: {$undefined_var}
```

**Allow Undefined Variables**:
Use the `--allow-undefined-variables` CLI flag to revert to legacy behavior:

```bash
drun my-task --allow-undefined-variables
# Output: Hello {$undefined}  (literal text)
```

**Benefits**:
- **Early Error Detection**: Catch typos and missing variables before execution
- **Clear Error Context**: Precise location (statement type) and variable name
- **Prevent Silent Failures**: Avoid unexpected behavior from undefined variables
- **Better Developer Experience**: Forces explicit variable definitions

**Variable Resolution Order**:
1. Task parameters (`accepts $param`)
2. Local variables (`let $var = "value"`)
3. Project settings (`$globals.setting`)
4. Built-in variables (`$globals.version`, `$globals.project`)

### Advanced Variable Operations

drun v2 provides powerful variable transformation operations that can be chained together for complex data manipulation.

#### Variable Assignment

Both `let` and `set` support variable assignment with optional type declarations:

```drun
task "variable_assignment":
  # Simple assignment with let
  let $name = "value"

  # Simple assignment with set
  set $variable to "value"

  # Array assignment with let
  let $items as list to ["value1", "value2", "value3"]

  # Array assignment with set
  set $platforms as list to ["linux", "mac", "windows"]

  # Arrays are stored as comma-separated strings
  # and can be used in loops
  for each $platform in $platforms:
    info "Platform: {$platform}"
```

#### String Operations

Transform string values with intuitive operations:

```drun
task "string_operations":
  set $version to "v2.1.0-beta"
  set $filename to "my-app.tar.gz"
  set $url to "https://api.example.com/v1/users"

  info "Clean version: {$version without prefix 'v' | without suffix '-beta'}"
  # Output: 2.1.0

  info "App name: {$filename without suffix '.tar.gz'}"
  # Output: my-app

  info "Domain: {$url without prefix 'https://' | without suffix '/v1/users'}"
  # Output: api.example.com
```

#### Array Operations

Manipulate space-separated lists with filtering, sorting, and selection:

```drun
task "array_operations":
  set $files to "app.js test.js config.json package.json readme.md"

  info "JavaScript files: {$files filtered by extension '.js'}"
  # Output: app.js test.js

  info "Sorted files: {$files sorted by name}"
  # Output: app.js config.json package.json readme.md test.js

  info "First file: {$files first}"
  # Output: app.js

  info "Unique items: {$files unique}"
  # Removes duplicates if any exist
```

#### Path Operations

Extract components from file paths:

```drun
task "path_operations":
  set $source_file to "/home/user/projects/myapp/src/main.js"

  info "Filename: {$source_file basename}"
  # Output: main.js

  info "Directory: {$source_file dirname}"
  # Output: /home/user/projects/myapp/src

  info "Extension: {$source_file extension}"
  # Output: js

  info "Name without extension: {$source_file basename | without suffix '.js'}"
  # Output: main
```

#### Advanced String Operations

Split strings and extract parts:

```drun
task "advanced_string_ops":
  set $docker_image to "nginx:1.21"
  set $csv_data to "name,age,city"

  info "Image name: {$docker_image split by ':' | first}"
  # Output: nginx

  info "CSV headers: {$csv_data split by ','}"
  # Output: name age city (space-separated for further processing)
```

#### Operation Chaining

Combine multiple operations with the pipe (`|`) operator:

```drun
task "complex_chaining":
  set $project_files to "src/app.js src/utils.js tests/app.test.js docs/readme.md"

  # Complex filtering and sorting chain
  info "Source JS files: {$project_files filtered by prefix 'src/' | filtered by extension '.js' | sorted by name}"
  # Output: src/app.js src/utils.js

  # Path manipulation chain
  set $config_path to "/etc/nginx/sites-available/default.conf"
  info "Config name: {$config_path basename | without suffix '.conf'}"
  # Output: default
```

#### For Each Loop Integration

Variable operations work seamlessly with for each loops:

```drun
task "loop_with_operations":
  set $docker_images to "nginx:1.21 postgres:13 redis:6.2"

  for each $img in $docker_images:
    info "Processing: {img}"
    info "Image name: {img split by ':' | first}"
    info "Version: {img split by ':' | last}"
```

#### Available Operations Reference

**String Operations:**
- `without prefix "text"` - Remove prefix from string
- `without suffix "text"` - Remove suffix from string
- `split by "delimiter"` - Split string into space-separated parts

**Array Operations:**
- `filtered by extension "ext"` - Filter by file extension
- `filtered by prefix "text"` - Filter by prefix
- `filtered by suffix "text"` - Filter by suffix
- `filtered by name "text"` - Filter by name containing text
- `sorted by name` - Sort alphabetically
- `sorted by length` - Sort by string length
- `reversed` - Reverse order
- `unique` - Remove duplicates
- `first` - Get first item
- `last` - Get last item

**Path Operations:**
- `basename` - Extract filename from path
- `dirname` - Extract directory from path
- `extension` - Extract file extension (without dot)

---

## Parameter System

### Parameter Types

#### Required Parameters

```drun
task "deploy":
  requires $environment from ["dev", "staging", "production"]
  requires $version matching pattern "v\d+\.\d+\.\d+"

  # Usage: xdrun deploy environment=production version=v1.2.3
```

#### Required Parameters with Defaults

Required parameters can have default values. When a default is provided, the parameter becomes optional at the CLI level but still benefits from the validation constraints:

```drun
task "build":
  requires $image from ["base", "worker", "dev", "all"]
  requires $cache from ["yes", "no"] defaults to "no"

  # Usage without cache parameter (uses default "no"):
  # xdrun build image=base

  # Usage with cache parameter override:
  # xdrun build image=base cache=yes
```

**Important validation rules:**
- The default value MUST be one of the values in the constraint list (if constraints are specified)
- The parser will validate this at parse time and emit an error if the default value is not in the allowed values

```drun
# Valid:
requires $env from ["dev", "staging", "prod"] defaults to "dev"

# Invalid - will cause parse error:
requires $env from ["dev", "staging", "prod"] defaults to "production"
# Error: default value 'production' must be one of the allowed values: [dev, staging, prod]
```

#### CLI Argument Syntax

**Important:** Task parameters use simple `key=value` syntax **without `--` dashes**. This is different from typical CLI tools!

```bash
# CORRECT: Task parameters (no dashes)
xdrun deploy environment=production replicas=5
xdrun build tag=v1.2.3 push=true
xdrun test suites=unit,integration verbose=true

# WRONG: Do NOT use dashes for task parameters
xdrun deploy --environment=production --replicas=5  # This won't work!

# CLI flags use dashes (they control xdrun behavior)
xdrun deploy environment=prod --dry-run --verbose
xdrun build --list
```

**Why no dashes?**
- Task parameters are part of the drun language (semantic parameters)
- CLI flags control the xdrun interpreter itself (operational flags)
- This distinction keeps the language clean and consistent

#### Optional Parameters

```drun
task "build":
  given $tag defaults to current git commit
  given $push defaults to false
  given $platforms defaults to ["linux/amd64"]

  # Usage: xdrun build
  # Usage: xdrun build tag=custom push=true
```

#### Variadic Parameters

```drun
task "test":
  accepts $suites as list of strings
  accepts flags as list

  # Usage: xdrun test --suites=unit,integration --flags=--verbose,--coverage
```

### Parameter Validation

#### Type Validation

```drun
requires port as number between 1000 and 65535
requires timeout as duration
requires config_file as path that exists
requires webhook_url as url with https protocol
```

#### Pattern Validation

```drun
requires version matching pattern "v\d+\.\d+\.\d+"
requires email matching email format
requires branch_name matching pattern "[a-zA-Z0-9-_/]+"
```

#### Enum Validation

```drun
requires log_level from ["debug", "info", "warn", "error"]
requires deployment_strategy from ["rolling", "blue-green", "canary"]
```

#### Custom Validation

```drun
requires replicas as number where value > 0 and value <= 100
requires memory as string where value matches pattern "\d+[MGT]i?"
```

### Parameter Usage

#### Direct Access

```drun
task "greet":
  requires name
  given title defaults to "friend"

  info "Hello, {title} {name}!"
```

#### Conditional Parameters

```drun
task "deploy":
  requires environment from ["dev", "staging", "production"]

  when environment is "production":
    requires approval_token
    requires backup_confirmation defaults to true

  when environment is "dev":
    given debug_mode defaults to true
```

#### Parameter Transformation

```drun
task "build":
  requires version

  let clean_version be {version} without prefix "v"
  let image_tag be "myapp:{clean_version}"
```

---

