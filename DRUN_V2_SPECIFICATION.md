# drun v2 Semantic Language Specification

**Version**: 2.0.0-draft  
**Date**: 2025-09-17  
**Status**: Design Specification  

## Table of Contents

1. [Overview](#overview)
2. [Design Philosophy](#design-philosophy)
3. [Language Grammar](#language-grammar)
4. [Lexical Structure](#lexical-structure)
5. [Syntax Specification](#syntax-specification)
6. [Type System](#type-system)
7. [Control Flow](#control-flow)
8. [Variable System](#variable-system)
9. [Parameter System](#parameter-system)
10. [Built-in Actions](#built-in-actions)
11. [Smart Detection](#smart-detection)
12. [Compilation Model](#compilation-model)
13. [Error Handling](#error-handling)
14. [Examples](#examples)
15. [Migration Path](#migration-path)
16. [Implementation Notes](#implementation-notes)

---

## Overview

drun v2 introduces a semantic, English-like domain-specific language (DSL) for defining automation tasks. Unlike v1 which uses YAML configuration, v2 features a **completely new execution engine** that directly interprets and executes the semantic language without compilation to intermediate formats.

### Key Features

- **Natural Language Syntax**: Write automation in English-like sentences
- **Native Execution Engine**: Direct interpretation and execution of v2 syntax
- **Shell Backend**: All constructs execute as shell commands when needed
- **Smart Inference**: Automatic detection of tools, environments, and patterns
- **Type Safety**: Static analysis with runtime validation

### Architecture

drun v2 uses a **new execution engine** with the following components:

1. **Lexer**: Tokenizes the semantic language source code
2. **Parser**: Builds an Abstract Syntax Tree (AST) from tokens
3. **Engine**: Directly executes the AST without intermediate compilation
4. **Runtime**: Provides built-in actions, smart detection, and shell integration

### Design Goals

1. **Readability**: Non-technical stakeholders can understand automation workflows
2. **Maintainability**: Reduce boilerplate, focus on intent
3. **Composability**: Natural language enables intuitive composition
4. **Performance**: Direct execution without compilation overhead
5. **Extensibility**: Plugin system for domain-specific actions

---

## Design Philosophy

### Natural Language First

The language prioritizes human readability over machine optimization. Every construct should read like natural English while maintaining precise semantics.

```
# Good: Natural and clear
deploy myapp to production with 3 replicas

# Avoid: Technical but unclear intent
kubectl.apply(deployment.spec(replicas=3, namespace="production"))
```

### Declarative Intent

Focus on *what* should happen, not *how* to do it. The compiler handles implementation details.

```
# Declarative: What should happen
ensure database is running

# Imperative: How to do it (handled by compiler)
# if ! docker ps | grep -q postgres; then
#   docker run -d postgres:13
# fi
```

### Smart Defaults

The language should infer sensible defaults based on context and project structure.

```
# Infers image name from project context
build docker image

# Explicit when needed
build docker image "custom-name:v1.0"
```

---

## Language Grammar

### EBNF Grammar

```ebnf
(* Top-level constructs *)
program = { project_declaration | global_declaration | task_definition } ;

(* Project declaration *)
project_declaration = "project" string_literal [ "version" string_literal ] ":" 
                     { project_setting } ;

project_setting = "set" identifier "to" expression
                | "include" string_literal
                | "before" "any" "task" ":" statement_block
                | "after" "any" "task" ":" statement_block
                | shell_config ;

(* Task definition *)
task_definition = "task" string_literal [ "means" string_literal ] ":"
                 { task_property }
                 statement_block ;

task_property = parameter_declaration
              | dependency_declaration
              | lifecycle_hook
              | variable_declaration ;

(* Parameters *)
parameter_declaration = "requires" parameter_spec
                      | "given" parameter_spec
                      | "accepts" parameter_spec ;

parameter_spec = identifier [ parameter_constraint ] [ parameter_default ] ;

parameter_constraint = "from" array_literal
                     | "matching" "pattern" string_literal
                     | "matching" "email" "format"
                     | "as" type_name [ range_constraint ]
                     | "as" "list" [ "of" type_name ] ;

parameter_default = "defaults" "to" expression ;

range_constraint = "between" number "and" number ;

(* Dependencies *)
dependency_declaration = "depends" "on" dependency_list ;

dependency_list = dependency_item { ( "," | "and" ) dependency_item } [ "then" dependency_item ] ;

dependency_item = identifier [ "in" "parallel" ] ;

(* Lifecycle hooks *)
lifecycle_hook = "before" "running" ":" statement_block
               | "after" "running" ":" statement_block ;

(* Statements *)
statement_block = { statement } ;

statement = expression_statement
          | control_statement
          | declaration_statement
          | action_statement ;

expression_statement = expression ;

control_statement = if_statement
                  | when_statement
                  | for_statement
                  | try_statement ;

declaration_statement = variable_declaration
                      | constant_declaration ;

action_statement = built_in_action
                 | shell_command
                 | detection_statement ;

(* Control flow *)
if_statement = "if" condition ":" statement_block
              [ "else" "if" condition ":" statement_block ]
              [ "else" ":" statement_block ] ;

when_statement = "when" expression ":"
                { "is" expression ":" statement_block }
                [ "else" ":" statement_block ] ;

for_statement = "for" "each" identifier "in" expression [ "in" "parallel" ] ":"
               statement_block ;

try_statement = "try" ":" statement_block
               { "catch" identifier ":" statement_block }
               [ "finally" ":" statement_block ] ;

(* Conditions *)
condition = logical_expression ;

logical_expression = comparison_expression
                   { ( "and" | "or" ) comparison_expression } ;

comparison_expression = additive_expression
                      [ comparison_operator additive_expression ] ;

comparison_operator = "is" | "is" "not" | "==" | "!=" | "<" | ">" | "<=" | ">=" 
                    | "contains" | "matches" | "exists" ;

(* Expressions *)
expression = logical_expression ;

additive_expression = multiplicative_expression
                    { ( "+" | "-" ) multiplicative_expression } ;

multiplicative_expression = unary_expression
                          { ( "*" | "/" | "%" ) unary_expression } ;

unary_expression = [ "not" ] primary_expression ;

primary_expression = identifier
                   | literal
                   | function_call
                   | member_access
                   | interpolated_string
                   | "(" expression ")" ;

function_call = identifier "(" [ argument_list ] ")" ;

member_access = primary_expression "." identifier ;

argument_list = expression { "," expression } ;

(* Variables *)
variable_declaration = "let" identifier "be" expression
                     | "set" identifier "to" expression
                     | "capture" identifier "from" expression ;

constant_declaration = "define" identifier "as" expression ;

(* Built-in actions *)
built_in_action = docker_action
                | kubernetes_action
                | git_action
                | file_action
                | network_action
                | status_action
                | deployment_action ;

docker_action = "build" "docker" "image" [ string_literal ]
              | "push" "image" string_literal [ "to" string_literal ]
              | "run" "container" string_literal [ container_options ]
              | "stop" "container" string_literal
              | "remove" "container" string_literal ;

kubernetes_action = "deploy" string_literal "to" "kubernetes" [ kubernetes_options ]
                  | "scale" string_literal "to" number "replicas"
                  | "rollback" string_literal
                  | "wait" "for" "rollout" [ "of" string_literal ] ;

git_action = "commit" "changes" [ "with" "message" string_literal ]
           | "push" "to" "branch" string_literal
           | "create" "tag" string_literal
           | "checkout" "branch" string_literal ;

file_action = "copy" string_literal "to" string_literal
            | "move" string_literal "to" string_literal
            | "remove" string_literal
            | "create" "directory" string_literal
            | "backup" string_literal [ "as" string_literal ] ;

status_action = "step" string_literal
              | "info" string_literal
              | "warn" string_literal
              | "error" string_literal
              | "success" string_literal
              | "fail" [ "with" string_literal ] ;

(* Shell commands *)
shell_command = shell_action ( string_literal | ":" statement_block )
              | "capture" ( string_literal | ":" statement_block ) "as" identifier ;

shell_action = "run" | "exec" | "shell" ;

(* Detection statements *)
detection_statement = "detect" detection_target
                    | "detect" "available" tool_alternatives [ "as" variable_name ]
                    | "if" ( tool_name | string_literal ) "is" "available" ":" statement_block [ "else" ":" statement_block ]
                    | "if" ( tool_name | string_literal ) "version" comparison_operator string_literal ":" statement_block [ "else" ":" statement_block ]
                    | "when" "in" environment_name "environment" ":" statement_block [ "else" ":" statement_block ] ;

detection_target = "project" "type"
                 | tool_name [ "version" ] ;

tool_alternatives = ( tool_name | string_literal ) { "or" ( tool_name | string_literal ) } ;

tool_name = identifier ;

environment_name = "ci" | "local" | "production" | "staging" | "development" | identifier ;

variable_name = "$" identifier ;

(* Shell configuration *)
shell_config = "shell" "config" ":" { platform_config } ;

platform_config = identifier ":" platform_settings ;

platform_settings = { platform_setting } ;

platform_setting = "executable" ":" string_literal
                  | "args" ":" string_array
                  | "environment" ":" key_value_pairs ;

string_array = { "-" string_literal } ;

key_value_pairs = { identifier ":" string_literal } ;

(* Literals *)
literal = string_literal
        | number_literal
        | boolean_literal
        | array_literal
        | object_literal ;

string_literal = '"' { string_character } '"' ;
interpolated_string = '"' { string_character | interpolation } '"' ;
interpolation = "{" expression "}" ;

(* Variable syntax: declared variables use $prefix, loop variables are bare identifiers *)
variable = "$" identifier ;  (* Declared variables: $name, $environment *)
loop_variable = identifier ; (* Loop variables: item, i, file *)

number_literal = integer_literal | float_literal ;
integer_literal = digit { digit } ;
float_literal = integer_literal "." integer_literal ;

boolean_literal = "true" | "false" ;

array_literal = "[" [ expression { "," expression } ] "]" ;

object_literal = "{" [ object_member { "," object_member } ] "}" ;
object_member = ( identifier | string_literal ) ":" expression ;

(* Identifiers and keywords *)
identifier = letter { letter | digit | "_" } ;
letter = "a" | "b" | ... | "z" | "A" | "B" | ... | "Z" ;
digit = "0" | "1" | ... | "9" ;
```

---

## Lexical Structure

### Tokens

The language consists of the following token types:

1. **Keywords**: Reserved words with special meaning
2. **Identifiers**: User-defined names for variables, tasks, etc.
3. **Literals**: String, number, boolean, array, and object literals
4. **Operators**: Arithmetic, comparison, and logical operators
5. **Punctuation**: Colons, commas, parentheses, brackets, braces
6. **Comments**: Single-line and multi-line comments

### Keywords

```
# Control flow
if, else, when, for, each, in, try, catch, finally, break, continue

# Declarations
task, project, let, set, capture, define, given, requires, accepts

# Dependencies and lifecycle
depends, on, before, after, running, then, parallel

# Types and constraints
from, matching, pattern, format, as, list, of, between, and, defaults, to

# Logical operators
and, or, not, is

# Built-in actions
build, deploy, push, run, stop, remove, scale, rollback, wait, commit
copy, move, create, backup, step, info, warn, error, success, fail

# Smart detection
docker, kubernetes, git, image, container, replicas, branch, tag
directory, file, exists, running, healthy, available

# Special values
true, false, now, current, secret, env
```

### Comments

```
# Single-line comment
task "example":  # End-of-line comment
  info "Hello"

/*
Multi-line comment
Can span multiple lines
*/
```

### String Interpolation

Strings support variable interpolation using `{$variable}` syntax for declared variables and `{variable}` for loop variables:

```
let $name = "world"
info "Hello, {$name}!"  # Outputs: Hello, world!

# Loop variables use bare identifiers
for each item in items:
  info "Processing {item}"  # Loop variable without $

# Complex expressions in interpolation
info "Current time: {now.format('HH:mm:ss')}"
```

---

## Syntax Specification

### Project Declaration

```
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

```
project "my-app":
  shell config:
    darwin:
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
- **darwin**: macOS
- **linux**: Linux distributions
- **windows**: Windows

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

```
task "example":
  run "echo $SHELL"        # Uses configured shell
  run "echo $TERM"         # Uses configured environment
  capture "whoami" as $user # Uses configured shell and environment
```

### Task Definition

```
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

### Parameter Declarations

#### Required Parameters

```
requires <name> [constraints]

# Examples:
requires $environment from ["dev", "staging", "production"]
requires $version matching pattern "v\d+\.\d+\.\d+"
requires $port as number between 1000 and 9999
requires $email matching email format
requires files as list of strings
```

#### Optional Parameters with Defaults

```
given <name> defaults to <value> [constraints]

# Examples:
given replicas defaults to 3
given timeout defaults to "5m"
given force defaults to false
given tags defaults to [] as list of strings
```

#### Variadic Parameters

```
accepts <name> as list [of <type>]

# Examples:
accepts features as list
accepts ports as list of numbers
accepts configs as list of strings
```

### Dependencies

```
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

```
let <name> be <expression>
set <name> to <expression>

# Examples:
let image_name be "myapp:latest"
set build_time to now
let git_hash be current git commit
```

#### Capture from Commands

```
capture <name> from <expression>

# Examples:
capture running_containers from "docker ps --format json"
capture disk_usage from "df -h /"
capture branch_name from current git branch
```

#### Conditional Assignment

```
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

```
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

#### When Statements (Pattern Matching)

```
when <expression>:
  is <value>: <statements>
  is <value>: <statements>
  else: <statements>

# Example:
when package_manager:
  is "npm": run "npm ci && npm run build"
  is "yarn": run "yarn install && yarn build"
  is "pnpm": run "pnpm install && pnpm build"
  else: error "Unknown package manager: {package_manager}"
```

#### For Loops

```
for each <variable> in <expression> [in parallel]:
  <statements>

# Examples:
for each env in ["dev", "staging", "prod"]:
  deploy to {env}

for each service in microservices in parallel:
  test service {service}

# Nested loops (matrix execution)
for each os in ["ubuntu", "alpine"]:
  for each version in ["16", "18", "20"]:
    test on {os} with node {version}
```

#### Exception Handling

```
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

```
let count be 42                    # number
let name be "hello"                # string
let enabled be true                # boolean
let timeout be "5m"                # duration
let files be ["a.txt", "b.txt"]    # array of strings
let config be {port: 8080}         # object
```

### Type Constraints

Parameters can specify type constraints:

```
requires port as number between 1000 and 9999
requires timeout as duration
requires files as list of paths
requires config as object
```

---

## Control Flow

### Conditional Execution

#### Simple Conditions

```
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
```

#### Smart Detection Conditions

```
# Tool detection
if docker is running:
  build container

if kubernetes is available:
  deploy to cluster

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

```
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

```
for each item in collection:
  process item

# With index
for each item at index in collection:
  info "Processing item {index}: {item}"
```

#### Parallel Execution

```
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

```
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

```
for each file in "src/**/*.js" where file is modified:
  lint {file}

for each container in docker containers where status is "running":
  check health of {container}
```

### Loop Control

```
for each service in services:
  if service is healthy:
    continue
  
  try:
    restart service
  catch:
    error "Failed to restart {service}"
    break  # Exit loop on critical failure
```

---

## Variable System

### Variable Declaration

All variables in drun v2 must be prefixed with `$` to distinguish them from keywords and improve syntax clarity.

### Variable Syntax Rules

#### Variable Naming Convention

1. **Declared Variables**: Must start with `$` prefix
   - `$name`, `$environment`, `$commit_hash`
   - Used in: parameter declarations, let/set statements, variable references

2. **Loop Variables**: Use bare identifiers (no `$` prefix)
   - `item`, `file`, `i`, `attempt`
   - Used in: `for each item in items`, `for i in range 1 to 10`

3. **Interpolation Syntax**:
   - Task variables: `{$variable_name}`
   - Project settings: `{$globals.setting_name}`
   - Built-in project vars: `{$globals.project}`, `{$globals.version}`
   - Loop variables: `{variable_name}`
   - Built-in functions: `{now.format()}`, `{pwd}`

#### Examples

```
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

# Loop variables (bare identifiers)
for each item in items:
  info "Processing {item}"  # Loop variable interpolation

for i in range 1 to 5:
  info "Attempt {i} of 5"   # Loop variable interpolation

# Mixed interpolation with different scopes
info "Deploying {$tag} to {$environment} from {$globals.registry}"
info "Project: {$globals.project} v{$globals.version}"
info "API: {$globals.api_url} - Processing item {item}"
```

#### Let Bindings (Immutable)

```
let $name = "value"           # Simple assignment
let $result = compute_value() # Function result
let $config = {              # Object literal
  port: 8080,
  host: "localhost"
}
```

#### Set Statements (Mutable)

```
set $counter to 0
set $counter to {$counter} + 1  # Increment

set environment to:
  when running_locally: "development"
  else: "production"
```

#### Capture from Commands

```
capture git_branch from current git branch
capture docker_version from "docker --version"
capture running_pods from "kubectl get pods --output=json"

# With error handling
try:
  capture service_status from "systemctl status nginx"
catch command_error:
  set service_status to "unknown"
```

### Variable Scoping

drun v2 uses a clear scoping system with explicit namespaces to avoid naming conflicts and improve code clarity.

#### Project Scope (Global Variables)

Project-level settings are declared without the `$` prefix and accessed via the `$globals` namespace:

```
project "myapp" version "1.0.0":
  set registry to "ghcr.io/company"    # Project setting
  set api_url to "https://api.example.com"
  set timeout to "30s"
```

**Accessing Project Settings:**
```
task "deploy":
  info "Project: {$globals.project}"        # → "myapp"
  info "Version: {$globals.version}"        # → "1.0.0"
  info "Registry: {$globals.registry}"      # → "ghcr.io/company"
  info "API URL: {$globals.api_url}"        # → "https://api.example.com"
  info "Timeout: {$globals.timeout}"        # → "30s"
```

#### Task Scope (Local Variables)

Task-level variables are declared with the `$` prefix and accessed directly:

```
task "deploy":
  set $image_tag to "{$globals.registry}/myapp:{$globals.version}"  # Task-local
  set $replicas to 3
  
  info "Deploying {$image_tag} with {$replicas} replicas"
```

#### Scoping Rules and Precedence

1. **Project Settings**: Declared without `$`, accessed via `$globals.key`
2. **Task Variables**: Declared with `$`, accessed with `$variable`
3. **Loop Variables**: Bare identifiers, accessed with `{variable}`
4. **Built-in Variables**: Special project variables via `$globals.project` and `$globals.version`

**Variable Resolution Order:**
1. Parameters (`$param`)
2. Task variables (`$variable`)
3. Loop variables (`variable`)
4. Project settings (`$globals.key`)
5. Built-in functions

#### Avoiding Naming Conflicts

The `$globals` namespace prevents conflicts between project settings and task variables:

```
project "myapp":
  set api_url to "https://project-level.com"

task "test":
  set $api_url to "https://task-level.com"    # Different variable
  
  info "Global API: {$globals.api_url}"       # → "https://project-level.com"
  info "Task API: {$api_url}"                 # → "https://task-level.com"
```

#### Nested Scope in Control Structures

```
task "deploy":
  set $base_replicas to 3
  
  if environment is "production":
    set $replicas to {$base_replicas} * 2     # Block-local, shadows outer scope
    info "Production replicas: {$replicas}"   # → 6
  else:
    info "Default replicas: {$base_replicas}" # → 3
```

#### Parameter Scope

```
task "greet":
  requires name
  given title defaults to "friend"
  
  # Parameters are available as variables
  info "Hello, {title} {name}!"
```

### Variable Interpolation

#### String Interpolation

```
let name be "world"
let greeting be "Hello, {name}!"

# Complex expressions
let message be "Deployment {version} to {environment} at {now.format('HH:mm')}"

# Nested interpolation
let docker_tag be "{registry}/{app_name}:{version}-{git_commit.short}"
```

#### Command Interpolation

```
let image_name be "myapp:latest"
run "docker push {image_name}"

# Multiple interpolations
run "kubectl set image deployment/{app_name} {app_name}={image_name}"
```

### Advanced Variable Operations

drun v2 provides powerful variable transformation operations that can be chained together for complex data manipulation.

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
  
  for each img in $docker_images:
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

```
task "deploy":
  requires $environment from ["dev", "staging", "production"]
  requires $version matching pattern "v\d+\.\d+\.\d+"
  
  # Usage: drun deploy environment=production version=v1.2.3
```

#### CLI Argument Syntax

Parameters are passed to tasks using simple `key=value` syntax (no `--` prefix required):

```bash
# Parameter passing examples
drun deploy environment=production
drun build tag=v1.2.3 push=true
drun test suites=unit,integration verbose=true

# Multiple parameters
drun deploy environment=staging replicas=5 timeout=300
```

#### Optional Parameters

```
task "build":
  given $tag defaults to current git commit
  given $push defaults to false
  given $platforms defaults to ["linux/amd64"]
  
  # Usage: drun build
  # Usage: drun build tag=custom push=true
```

#### Variadic Parameters

```
task "test":
  accepts $suites as list of strings
  accepts flags as list
  
  # Usage: drun test --suites=unit,integration --flags=--verbose,--coverage
```

### Parameter Validation

#### Type Validation

```
requires port as number between 1000 and 65535
requires timeout as duration
requires config_file as path that exists
requires webhook_url as url with https protocol
```

#### Pattern Validation

```
requires version matching pattern "v\d+\.\d+\.\d+"
requires email matching email format
requires branch_name matching pattern "[a-zA-Z0-9-_/]+"
```

#### Enum Validation

```
requires log_level from ["debug", "info", "warn", "error"]
requires deployment_strategy from ["rolling", "blue-green", "canary"]
```

#### Custom Validation

```
requires replicas as number where value > 0 and value <= 100
requires memory as string where value matches pattern "\d+[MGT]i?"
```

### Parameter Usage

#### Direct Access

```
task "greet":
  requires name
  given title defaults to "friend"
  
  info "Hello, {title} {name}!"
```

#### Conditional Parameters

```
task "deploy":
  requires environment from ["dev", "staging", "production"]
  
  when environment is "production":
    requires approval_token
    requires backup_confirmation defaults to true
  
  when environment is "dev":
    given debug_mode defaults to true
```

#### Parameter Transformation

```
task "build":
  requires version
  
  let clean_version be {version} without prefix "v"
  let image_tag be "myapp:{clean_version}"
```

---

## Built-in Actions

### Shell Commands

drun v2 supports both single-line and multiline shell command execution with consistent syntax patterns.

#### Single-Line Commands (Current)

```
# Execute and stream output
run "echo 'Hello World'"
exec "date +%Y-%m-%d"
shell "pwd"

# Capture output to variable
capture "git rev-parse --short HEAD" as $commit_hash
capture "whoami" as $username
```

#### Multiline Commands (Block Syntax)

For complex shell operations, use the block syntax with natural indentation:

```
# Multiline execution with streaming
run:
  echo "Starting deployment process..."
  git pull origin main
  npm install
  npm run build

# Multiline with output capture
capture as $build_info:
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
```
run "echo hello"  # Executes: /bin/sh -c "echo hello"
```

**Multiline commands**: Execute as a single shell session
```
run:
  export VAR=value
  echo $VAR        # VAR is available from previous line
  cd /tmp
  pwd              # Shows /tmp (working directory persists)
```

#### Variable Interpolation in Multiline Commands

Variables work seamlessly in multiline blocks:

```
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

```
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

```
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
  capture as $deployment_info:
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

```
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

```
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

```
# Service management
start docker compose services
stop docker compose services
restart docker compose service "api"

# Scaling
scale docker compose service "worker" to 3 instances
```

### Kubernetes Actions

#### Deployment Operations

```
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

```
# Service management
expose deployment "myapp" on port 8080
create service "myapp-service" for deployment "myapp"

# Ingress
create ingress for service "myapp-service" with host "app.example.com"
```

#### Resource Management

```
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

```
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

```
# Tag management
create tag "v1.2.3"
create tag "v1.2.3" with message "Release version 1.2.3"
push tag "v1.2.3"
delete tag "v1.2.3"
```

### File System Actions

#### File Operations

```
# File management
copy "source.txt" to "destination.txt"
move "old-name.txt" to "new-name.txt"
remove "unwanted-file.txt"
backup "important-file.txt"
backup "important-file.txt" as "backup-{now.date}"

# Directory operations
create directory "new-folder"
remove directory "old-folder"
copy directory "src" to "backup/src"
```

#### File Inspection

```
# File checking
check if file "config.json" exists
check if directory ".git" exists
get size of file "large-file.dat"
get modification time of file "config.json"
```

### Network Actions

#### HTTP Operations

```
# HTTP requests
send GET request to "https://api.example.com/status"
send POST request to "https://api.example.com/deploy" with data {version: "1.2.3"}
download "https://example.com/file.zip" to "downloads/"

# Health checks
check health of service at "https://app.example.com/health"
wait for service at "https://app.example.com" to be ready
```

#### Network Testing

```
# Connectivity testing
check if port 8080 is open on "localhost"
test connection to "database.example.com" on port 5432
ping host "example.com"
```

### Status and Logging Actions

#### Status Messages

```
step "Starting deployment process"
info "Configuration loaded successfully"
warn "Using default configuration"
error "Failed to connect to database"
success "Deployment completed successfully"
```

#### Process Control

```
fail                                    # Exit with error code 1
fail with "Custom error message"        # Exit with custom message
exit with code 0                        # Exit with specific code
```

#### Progress Tracking

```
# Progress indicators
start progress "Downloading dependencies"
update progress to 50% with message "Half complete"
finish progress with "Download completed"

# Timing
start timer "deployment"
stop timer "deployment"
show elapsed time for "deployment"
```

---

## Smart Detection

### Tool Detection

The language automatically detects available tools and uses appropriate commands:

```
# Automatically uses "docker compose" or "docker-compose"
start docker compose services

# Automatically uses "docker buildx" or "docker build"
build multi-platform docker image

# Detects kubectl, helm, etc.
deploy to kubernetes
install helm chart "nginx-ingress"
```

### DRY Tool Detection

For maximum flexibility and maintainability, drun supports detecting tool variants and capturing the working one in a variable:

```
# Detect which Docker Compose variant is available and capture it
detect available "docker compose" or "docker-compose" as $compose_cmd

# Use the captured variable consistently throughout the task
run "{$compose_cmd} up -d"
run "{$compose_cmd} ps"
run "{$compose_cmd} logs"

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

```
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

```
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

### Framework Detection

```
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
```
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

```
# Source
build docker image "myapp:{version}"

# Runtime Detection & Execution
if dockerBuildx available:
  execute: docker buildx build -t myapp:${version} .
else:
  execute: docker build -t myapp:${version} .
```

#### Kubernetes Command Execution

```
# Source
deploy myapp:latest to kubernetes namespace production with 5 replicas

# Generated
kubectl set image deployment/myapp myapp=myapp:latest --namespace=production
kubectl scale deployment/myapp --replicas=5 --namespace=production
kubectl rollout status deployment/myapp --namespace=production
```

### Optimization Strategies

#### Command Batching

```
# Source
copy "file1.txt" to "dest/"
copy "file2.txt" to "dest/"
copy "file3.txt" to "dest/"

# Optimized
cp file1.txt file2.txt file3.txt dest/
```

#### Conditional Optimization

```
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

```
# Missing colon
task "example"
  info "Hello"
# Error: Expected ':' after task declaration

# Invalid parameter constraint
requires port as number between "low" and "high"
# Error: Range bounds must be numeric values
```

#### Type Errors

```
# Type mismatch
let count be "not a number"
for i from 1 to count:
  # Error: Range bounds must be numeric
```

#### Scope Errors

```
task "example":
  if condition:
    let local_var be "value"
  
  info local_var  # Error: Variable not in scope
```

### Runtime Errors

#### Command Failures

```
# Automatic error handling
try:
  deploy to production
catch deployment_error:
  rollback deployment
  notify team of failure
```

#### Resource Not Found

```
# File not found
if file "config.json" exists:
  load configuration from "config.json"
else:
  error "Configuration file not found"
  fail
```

#### Network Errors

```
# Network timeout
try:
  check health of service at "https://api.example.com"
catch timeout_error:
  warn "Service health check timed out"
  continue with deployment
```

### Error Recovery

#### Retry Logic

```
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

```
try:
  deploy with blue-green strategy
catch blue_green_error:
  warn "Blue-green deployment failed, falling back to rolling update"
  deploy with rolling update strategy
```

---

## Examples

### Simple Task

```
task "hello":
  info "Hello, drun v2!"
```

### Task with Parameters

```
task "greet" means "Greet someone by name":
  requires name
  given title defaults to "friend"
  
  info "Hello, {title} {name}!"
```

### Docker Build and Deploy

```
project "webapp" version "1.0.0":
  set registry to "ghcr.io/company"

task "build" means "Build Docker image":
  given tag defaults to current git commit
  
  step "Building application image"
  build docker image "{registry}/webapp:{tag}"
  success "Build completed: {registry}/webapp:{tag}"

task "deploy" means "Deploy to Kubernetes":
  requires environment from ["dev", "staging", "production"]
  given replicas defaults to 3
  depends on build
  
  step "Deploying to {environment}"
  
  when environment is "production":
    require manual approval "Deploy to production?"
    ensure git repo is clean
  
  deploy webapp:latest to kubernetes namespace {environment} with {replicas} replicas
  wait for rollout to complete
  
  success "Deployment to {environment} completed"
```

### Complex CI/CD Pipeline

```
project "microservices":
  set registry to "ghcr.io/company"
  set environments to ["dev", "staging", "production"]
  set services to ["api", "web", "worker"]

task "test matrix" means "Run tests across multiple configurations":
  for each service in {services}:
    for each env in ["test", "integration"]:
      step "Testing {service} in {env} environment"
      run tests for {service} in {env} mode

task "build all" means "Build all service images":
  for each service in {services} in parallel:
    step "Building {service}"
    build docker image "{registry}/{service}:latest"
    push image "{registry}/{service}:latest"

task "deploy pipeline" means "Full deployment pipeline":
  requires target_env from {environments}
  depends on test_matrix and build_all
  
  let deployment_id be "deploy-{now.unix}"
  let failed_services be empty list
  
  step "Starting deployment {deployment_id} to {target_env}"
  
  for each service in {services}:
    try:
      deploy {service}:latest to kubernetes namespace {target_env}
      wait for {service} rollout to complete
      check health of {service} in {target_env}
      success "{service} deployed successfully"
    catch deployment_error:
      error "{service} deployment failed: {deployment_error}"
      add {service} to {failed_services}
  
  if {failed_services} is not empty:
    error "Deployment failed for services: {failed_services}"
    
    for each service in {failed_services}:
      rollback {service} in {target_env}
    
    fail "Deployment {deployment_id} failed"
  else:
    success "Deployment {deployment_id} completed successfully"
    notify slack "✅ All services deployed to {target_env}"
```

### Advanced Features Example

```
project "advanced-example":
  set notification_webhook to secret "slack_webhook"
  
  before any task:
    capture start_time from now
    info "Starting task execution at {start_time}"
  
  after any task:
    capture end_time from now
    let duration be {end_time} - {start_time}
    info "Task completed in {duration}"

task "smart deployment" means "Intelligent deployment with auto-detection":
  requires environment from ["dev", "staging", "production"]
  given force_deploy defaults to false
  
  # Smart project detection
  when symfony is detected:
    let app_type be "symfony"
    let health_endpoint be "/health"
  when laravel is detected:
    let app_type be "laravel"  
    let health_endpoint be "/api/health"
  when node project exists:
    let app_type be "node"
    let health_endpoint be "/healthz"
  else:
    let app_type be "generic"
    let health_endpoint be "/"
  
  step "Detected {app_type} application"
  
  # Environment-specific validation
  when environment is "production":
    if not force_deploy and git repo is dirty:
      error "Cannot deploy dirty repository to production"
      fail
    
    if not git tag exists for current commit:
      warn "No git tag found for current commit"
      require manual approval "Deploy untagged commit to production?"
  
  # Smart build detection
  if file "Dockerfile" exists:
    step "Building containerized application"
    build docker image "myapp:latest"
    
    if kubernetes is available:
      deploy myapp:latest to kubernetes namespace {environment}
    else:
      run container "myapp:latest" on port 8080
  else:
    step "Deploying non-containerized application"
    
    when app_type is "symfony":
      run "composer install --no-dev --optimize-autoloader"
      run "php bin/console cache:clear --env=prod"
    when app_type is "laravel":
      run "composer install --no-dev --optimize-autoloader"
      run "php artisan config:cache"
      run "php artisan route:cache"
    when app_type is "node":
      install dependencies
      run "npm run build"
  
  # Health check
  step "Performing health check"
  
  for attempt from 1 to 5:
    try:
      check health of service at "https://app-{environment}.example.com{health_endpoint}"
      success "Health check passed"
      break
    catch health_check_error:
      if attempt == 5:
        error "Health check failed after 5 attempts"
        fail
      warn "Health check attempt {attempt} failed, retrying in {attempt * 2} seconds..."
      wait {attempt * 2} seconds
  
  # Notification
  let message be "✅ {app_type} application deployed to {environment}"
  send POST request to {notification_webhook} with data {
    text: message,
    username: "drun-bot"
  }
  
  success "Deployment completed successfully"
```

---

## Migration Path

### From drun v1 to v2

#### Automatic Migration Tool

A migration tool would convert existing YAML configurations to semantic v2 syntax:

```bash
drun migrate drun.yml --output drun.v2 --format semantic
```

#### Migration Examples

##### Simple Recipe Migration

**v1 YAML:**
```yaml
recipes:
  hello:
    help: "Say hello"
    run: echo "Hello, World!"
```

**v2 Semantic:**
```
task "hello" means "Say hello":
  info "Hello, World!"
```

##### Complex Recipe Migration

**v1 YAML:**
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
      kubectl rollout status deployment/myapp --namespace={{ .environment }}
```

**v2 Semantic:**
```
task "deploy" means "Deploy to environment":
  requires environment from ["dev", "staging", "production"]
  depends on build
  
  deploy myapp:latest to kubernetes namespace {environment}
  wait for rollout to complete
```

### Gradual Migration Strategy

#### Phase 1: Coexistence
- v2 compiler generates v1 YAML
- Existing v1 configurations continue to work
- Teams can migrate individual tasks

#### Phase 2: Enhanced Features
- v2-specific features (enhanced smart detection)
- Improved error messages and debugging
- Advanced IDE support

#### Phase 3: Full Migration
- v1 format deprecated but supported
- New features only available in v2
- Migration tooling and documentation

### Backward Compatibility

#### v1 Include Support

```
# v2 can include v1 YAML files
project "mixed-project":
  include "legacy-tasks.yml"  # v1 YAML file

task "new-task":  # v2 semantic syntax
  depends on legacy_build  # Task from v1 file
  deploy with modern syntax
```

#### Escape Hatch to Shell

```
# When semantic syntax isn't sufficient
task "custom-operation":
  run shell command: |
    # Raw shell script for complex operations
    for file in $(find . -name "*.log" -mtime +7); do
      gzip "$file"
      mv "$file.gz" archive/
    done
```

---

## Implementation Notes

### Architecture Overview

drun v2 uses a **completely new execution engine** separate from v1:

```
drun/
├── internal/
│   ├── v1/           # Legacy v1 components (YAML-based)
│   │   ├── model/    # v1 data structures
│   │   ├── spec/     # v1 YAML loader
│   │   ├── runner/   # v1 task execution
│   │   ├── cache/    # v1 caching system
│   │   ├── dag/      # v1 dependency graph
│   │   ├── http/     # v1 HTTP integration
│   │   ├── pool/     # v1 worker pools
│   │   ├── shell/    # v1 shell execution
│   │   └── tmpl/     # v1 template engine
│   └── v2/           # New v2 components (semantic language)
│       ├── lexer/    # Lexical analysis domain
│       │   ├── token.go    # Token definitions
│       │   ├── lexer.go    # Tokenizer implementation
│       │   └── lexer_test.go
│       ├── parser/   # Syntax parsing domain
│       │   ├── parser.go   # Parser implementation
│       │   └── parser_test.go
│       ├── ast/      # Abstract Syntax Tree domain
│       │   └── ast.go      # AST node definitions
│       └── engine/   # Execution engine domain
│           ├── engine.go   # Direct execution engine
│           └── engine_test.go
└── cmd/drun/
    └── main.go       # CLI integration for both v1 and v2
```

### v2 Engine Components

1. **Lexer** (`internal/v2/lexer/`): Tokenizes semantic language into tokens
2. **Parser** (`internal/v2/parser/`): Builds Abstract Syntax Tree from tokens
3. **AST** (`internal/v2/ast/`): Defines semantic language node structures
4. **Engine** (`internal/v2/engine/`): Directly executes AST nodes
5. **Runtime**: Built-in actions, smart detection, shell integration

### Domain Separation

Each v2 component is organized into its own domain package:

- **`lexer/`**: Handles tokenization of source code
- **`parser/`**: Converts tokens into structured AST
- **`ast/`**: Defines the semantic language's syntax tree nodes
- **`engine/`**: Executes the parsed AST directly

### Parser Implementation

#### Lexer Design

```go
type TokenType int

const (
    // Literals
    STRING TokenType = iota
    NUMBER
    BOOLEAN
    
    // Keywords
    TASK
    PROJECT
    REQUIRES
    GIVEN
    DEPENDS
    IF
    WHEN
    FOR
    
    // Operators
    ASSIGN      // "be", "to"
    EQUALS      // "is", "=="
    NOT_EQUALS  // "is not", "!="
    
    // Punctuation
    COLON
    COMMA
    LPAREN
    RPAREN
    LBRACE
    RBRACE
    LBRACKET
    RBRACKET
)

type Token struct {
    Type     TokenType
    Value    string
    Position Position
}
```

#### AST Nodes

```go
type Node interface {
    Accept(visitor Visitor) error
}

type TaskDefinition struct {
    Name         string
    Description  string
    Parameters   []Parameter
    Dependencies []Dependency
    Body         []Statement
}

type Parameter struct {
    Name        string
    Type        ParameterType
    Required    bool
    Default     Expression
    Constraints []Constraint
}

type Statement interface {
    Node
    Execute(context ExecutionContext) error
}
```

### Code Generation

#### Template System

```go
type CodeGenerator struct {
    templates map[string]*template.Template
}

func (g *CodeGenerator) GenerateYAML(task *TaskDefinition) (string, error) {
    tmpl := g.templates["task"]
    var buf bytes.Buffer
    err := tmpl.Execute(&buf, task)
    return buf.String(), err
}
```

#### Smart Detection Engine

```go
type DetectionEngine struct {
    detectors []Detector
}

type Detector interface {
    Detect(projectPath string) (DetectionResult, error)
}

type DockerDetector struct{}

func (d *DockerDetector) Detect(projectPath string) (DetectionResult, error) {
    if fileExists(filepath.Join(projectPath, "Dockerfile")) {
        return DetectionResult{
            Type: "docker",
            Commands: map[string]string{
                "build": "docker build",
                "run":   "docker run",
            },
        }, nil
    }
    return DetectionResult{}, nil
}
```

### Runtime Integration

#### Execution Engine

```go
type ExecutionEngine struct {
    compiler *Compiler
    runner   *drun.Runner  // Existing drun v1 runner
}

func (e *ExecutionEngine) Execute(source string, args []string) error {
    // Compile semantic v2 to v1 YAML
    yamlConfig, err := e.compiler.Compile(source)
    if err != nil {
        return err
    }
    
    // Use existing drun v1 execution engine
    return e.runner.Execute(yamlConfig, args)
}
```

#### Error Reporting

```go
type CompileError struct {
    Message  string
    Position Position
    Suggestions []string
}

func (e *CompileError) Error() string {
    return fmt.Sprintf("%s at line %d, column %d", 
        e.Message, e.Position.Line, e.Position.Column)
}
```

### IDE Integration

#### Language Server Protocol

```go
type LanguageServer struct {
    compiler *Compiler
    detector *DetectionEngine
}

func (ls *LanguageServer) HandleCompletion(params CompletionParams) ([]CompletionItem, error) {
    // Provide intelligent completions based on context
    context := ls.analyzeContext(params.Position)
    
    switch context.Type {
    case "action":
        return ls.getActionCompletions(context)
    case "parameter":
        return ls.getParameterCompletions(context)
    default:
        return ls.getGeneralCompletions(context)
    }
}
```

#### Syntax Highlighting

```json
{
  "name": "drun-v2",
  "scopeName": "source.drun",
  "patterns": [
    {
      "name": "keyword.control.drun",
      "match": "\\b(task|project|if|when|for|try|catch)\\b"
    },
    {
      "name": "keyword.declaration.drun", 
      "match": "\\b(requires|given|depends|let|set)\\b"
    },
    {
      "name": "support.function.builtin.drun",
      "match": "\\b(build|deploy|push|run|info|error|success)\\b"
    }
  ]
}
```

### Performance Considerations

#### Compilation Caching

```go
type CompilationCache struct {
    cache map[string]CachedResult
    mutex sync.RWMutex
}

type CachedResult struct {
    YAML     string
    ModTime  time.Time
    Checksum string
}

func (c *CompilationCache) Get(source string, modTime time.Time) (string, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    if result, exists := c.cache[source]; exists {
        if result.ModTime.Equal(modTime) {
            return result.YAML, true
        }
    }
    return "", false
}
```

#### Incremental Compilation

```go
type IncrementalCompiler struct {
    ast    *AST
    dirty  map[string]bool
    cache  *CompilationCache
}

func (ic *IncrementalCompiler) CompileChanged(changes []Change) error {
    // Only recompile affected nodes
    for _, change := range changes {
        ic.markDirty(change.AffectedNodes...)
    }
    
    return ic.compileMarkedNodes()
}
```

---

This specification provides a comprehensive foundation for implementing drun v2's semantic language. The design prioritizes readability and maintainability while leveraging the existing drun infrastructure for performance and compatibility.


## Pattern Macro System

### Built-in Pattern Macros

drun v2 includes a comprehensive set of built-in pattern macros that provide common validation patterns without requiring complex regular expressions:

#### Available Pattern Macros

- **`semver`**: Basic semantic versioning (e.g., `v1.2.3`)
- **`semver_extended`**: Extended semantic versioning with pre-release and build metadata (e.g., `v2.0.1-RC2`, `v1.0.0-alpha.1+build.123`)
- **`uuid`**: UUID format (e.g., `550e8400-e29b-41d4-a716-446655440000`)
- **`url`**: HTTP/HTTPS URL format
- **`ipv4`**: IPv4 address format (e.g., `192.168.1.1`)
- **`slug`**: URL slug format (lowercase, hyphens only, e.g., `my-project-name`)
- **`docker_tag`**: Docker image tag format
- **`git_branch`**: Git branch name format

#### Usage Examples

```drun
task "deploy" means "Deploy with validation":
  # Basic semantic versioning
  requires $version as string matching semver
  
  # Extended semantic versioning
  requires $release as string matching semver_extended
  
  # UUID validation
  requires $deployment_id as string matching uuid
  
  # URL validation
  requires $api_endpoint as string matching url
  
  # IPv4 address validation
  requires $server_ip as string matching ipv4
  
  # Slug validation for project names
  requires $project_slug as string matching slug
  
  # Docker tag validation
  requires $image_tag as string matching docker_tag
  
  # Git branch validation
  requires $branch as string matching git_branch
  
  info "Deploying {version} to {server_ip}"
```

#### Pattern Macros vs Raw Patterns

Pattern macros can be used alongside raw regex patterns:

```drun
task "validation_examples":
  # Using pattern macros (recommended)
  requires $version as string matching semver
  requires $id as string matching uuid
  
  # Using raw patterns (for custom validation)
  requires $custom as string matching pattern "^custom-[0-9]+$"
  
  # Email validation (built-in)
  requires $email as string matching email format
```

#### Error Messages

Pattern macros provide descriptive error messages:

```bash
# Semver validation error
Error: parameter 'version': value '1.2.3' does not match semver pattern (Basic semantic versioning (e.g., v1.2.3))

# UUID validation error  
Error: parameter 'id': value 'not-a-uuid' does not match uuid pattern (UUID format (e.g., 550e8400-e29b-41d4-a716-446655440000))
```

