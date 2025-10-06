# drun v2 Semantic Language Specification (execute with xdrun CLI)

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
10. [Code Reuse Features](#code-reuse-features)
11. [Built-in Actions](#built-in-actions)
12. [Smart Detection](#smart-detection)
13. [Compilation Model](#compilation-model)
14. [Error Handling](#error-handling)
15. [Examples](#examples)
16. [Migration Path](#migration-path)
17. [Implementation Notes](#implementation-notes)

---

## Overview

drun v2 introduces a semantic, English-like domain-specific language (DSL) for defining automation tasks. It features a **completely new execution engine** that directly interprets and executes the semantic language without compilation to intermediate formats.

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
                | "set" identifier "as" "list" "to" array_literal
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
lifecycle_hook = "before" "any" "task" ":" statement_block
               | "after" "any" "task" ":" statement_block
               | "on" "drun" "setup" ":" statement_block
               | "on" "drun" "teardown" ":" statement_block ;

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
                 | detection_statement
                 | task_call_statement ;

(* Control flow *)
if_statement = "if" condition ":" statement_block
              [ "else" "if" condition ":" statement_block ]
              [ "else" ":" statement_block ] ;

when_statement = "when" expression ":" statement_block
                [ "otherwise" ":" statement_block ] ;

for_statement = "for" "each" variable "in" ( expression | array_literal ) [ "in" "parallel" ] ":"
               statement_block ;

try_statement = "try" ":" statement_block
               { "catch" identifier ":" statement_block }
               [ "finally" ":" statement_block ] ;

(* Task calls *)
task_call_statement = "call" "task" string_literal [ "with" parameter_list ] ;

parameter_list = parameter_assignment { parameter_assignment } ;

parameter_assignment = identifier "=" string_literal ;

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
                     | "capture" identifier "from" expression
                     | "capture" "from" "shell" string "as" variable ;

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
                    | "if" tool_list "is" "available" ":" statement_block [ "else" ":" statement_block ]
                    | "if" tool_list "is" "not" "available" ":" statement_block [ "else" ":" statement_block ]
                    | "if" ( tool_name | string_literal ) "version" comparison_operator string_literal ":" statement_block [ "else" ":" statement_block ]
                    | "when" "in" environment_name "environment" ":" statement_block [ "else" ":" statement_block ] ;

tool_list = ( tool_name | string_literal ) { "," ( tool_name | string_literal ) } ;

(* Note: Both "is" and "are" are accepted for tool availability checks *)

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
interpolation = "{" expression [ pipe_operations ] "}" ;
pipe_operations = { "|" pipe_operation } ;
pipe_operation = "replace" string_literal ( "by" | "with" ) string_literal
               | "without" ( "prefix" | "suffix" ) string_literal
               | "uppercase" | "lowercase" | "trim" ;

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

(* Comments *)
single_line_comment = "#" { any_character_except_newline } ;
multiline_comment = "/*" { any_character } "*/" ;
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
depends, on, before, after, running, then, parallel, drun, setup, teardown

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

# Built-in functions
current git commit, current git branch, now.format, pwd, hostname, env
```

### Comments

drun v2 supports both single-line and multiline comments for documenting your automation workflows.

#### Single-line Comments

Single-line comments start with `#` and continue to the end of the line:

```
# This is a single-line comment
task "example":  # End-of-line comment
  info "Hello"
```

#### Multiline Comments

Multiline comments use C-style `/* */` syntax and can span multiple lines:

```
/*
    This is a multiline comment
    that can span several lines
    and is useful for detailed documentation
*/

version: 2.0

/*
    Project configuration and setup
    Author: Development Team
    Last updated: 2025-09-22
*/
project "my-app" version "1.0":
    info "Starting application setup"
```

**Key Features:**
- Multiline comments preserve formatting and indentation
- They can appear anywhere in the file where whitespace is allowed
- Unterminated multiline comments are handled gracefully (consume to end of file)
- Comments are completely ignored during parsing and execution
- Useful for file headers, detailed explanations, and temporary code disabling

**Best Practices:**
- Use single-line comments for brief explanations
- Use multiline comments for file headers, detailed documentation, and block commenting
- Consider using multiline comments to temporarily disable sections of code during development

### Indentation

drun v2 uses **Python-style indentation** to define code blocks and supports both **tabs** and **spaces**:

#### Supported Indentation Styles

```drun
# Spaces (2 or 4 spaces per level)
task "spaces-example":
  info "Level 1 with spaces"
  if true:
    step "Level 2 with spaces"
    for each item in ["a", "b"]:
      info "Level 3: {item}"

# Tabs
task "tabs-example":
	info "Level 1 with tabs"
	if true:
		step "Level 2 with tabs"
		for each item in ["a", "b"]:
			info "Level 3: {item}"
```

#### Indentation Rules

1. **Tab Equivalence**: Each tab character equals 4 spaces for indentation level calculation
2. **Consistency**: Maintain consistent indentation style within each file
3. **Block Structure**: Indentation defines code blocks (similar to Python)
4. **Nesting**: Deeper indentation creates nested blocks
5. **Dedentation**: Returning to a previous indentation level closes blocks

#### Mixed Indentation

While both tabs and spaces are supported, **mixing them is discouraged** but technically allowed:

```drun
task "mixed-example":
    info "4 spaces"
	info "1 tab (equivalent to 4 spaces)"
        info "8 spaces"
		info "2 tabs (equivalent to 8 spaces)"
```

#### Error Handling

Invalid indentation patterns will result in parse errors:

```drun
task "invalid":
  info "Level 1"
   info "Invalid: 3 spaces doesn't match any previous level"
```

#### Best Practices

- **Choose one style**: Use either tabs or spaces consistently throughout your project
- **Editor configuration**: Configure your editor to show whitespace characters
- **Team standards**: Establish indentation standards for your team
- **Generated files**: `xdrun --init` uses tabs by default

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

### Lifecycle Hooks

drun v2 supports two types of lifecycle hooks that allow you to execute code at different points in the execution pipeline:

#### Task-Level Lifecycle Hooks

These hooks run around individual task execution:

```
project "myapp":
  before any task:
    info "🚀 Starting task: {$globals.current_task}"
    capture task_start_time from now
  
  after any task:
    capture task_end_time from now
    let task_duration be {task_end_time} - {task_start_time}
    info "✅ Task completed in {task_duration}"
```

- **`before any task`**: Executes before each individual task runs
- **`after any task`**: Executes after each individual task completes

#### Tool-Level Lifecycle Hooks

These hooks run once per drun execution, providing tool-level startup and shutdown capabilities:

```
project "myapp":
  on drun setup:
    info "🚀 Starting drun execution pipeline"
    info "📊 Tool version: {$globals.drun_version}"
    capture pipeline_start_time from now
  
  on drun teardown:
    capture pipeline_end_time from now
    let total_time be {pipeline_end_time} - {pipeline_start_time}
    info "🏁 Drun execution pipeline completed"
    info "📊 Total execution time: {total_time}"
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

### Task Calling

Tasks can call other tasks directly using the `call task` statement. This allows for code reuse and modular task design.

#### Basic Syntax

```
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

```
call task "task_name" with param1="value1" param2="value2"
call task task_name with param1="value1" param2="value2"  # Unquoted task name
```

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
  
  success "Full pipeline completed successfully!"
```

#### Key Features

- **Parameter Passing**: Pass parameters to called tasks using `with param="value"` syntax
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
given features defaults to empty  # equivalent to ""

# Built-in function defaults
given version defaults to "{current git commit}"
given branch defaults to "{current git branch}"
given safe_branch defaults to "{current git branch | replace '/' by '-'}"
given timestamp defaults to "{now.format('2006-01-02-15-04-05')}"
```

#### The `empty` Keyword

The `empty` keyword provides a semantic way to specify empty values and is completely interchangeable with empty strings (`""`):

```
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

drun v2 supports two types of capture operations:

#### Expression Capture
Captures values from expressions, functions, and built-in operations:

```
capture <name> from <expression>

# Examples:
capture start_time from now
capture branch_name from current git branch
capture calculated_value from {a} + {b}
```

#### Shell Command Capture
Captures output from shell commands:

```
capture from shell "<command>" as $<variable>

# Examples:
capture from shell "docker ps --format json" as $running_containers
capture from shell "df -h /" as $disk_usage
capture from shell "whoami" as $current_user
```

#### Multiline Shell Command Capture

For complex shell operations that span multiple commands, use the multiline syntax:

```
capture from shell as $<variable>:
  <command1>
  <command2>
  <command3>
```

**Examples:**

```
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

#### Enhanced If-Else Chains ⭐ *New*

drun v2 supports natural `else if` syntax for cleaner conditional logic:

```drun
task "deployment strategy":
  requires $environment from ["dev", "staging", "production"]
  
  if $environment == "production":
    info "🚀 Production deployment"
    set $replicas to 5
    set $timeout to "300s"
  else if $environment == "staging":
    info "🧪 Staging deployment"  
    set $replicas to 3
    set $timeout to "180s"
  else if $environment == "dev":
    info "🔧 Development deployment"
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

```
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
  when $platform is "darwin":
    step "Building for macOS"
  otherwise:
    step "Building for Linux"

# When-otherwise in loops (matrix execution)
for each $platform in ["windows", "linux", "darwin"]:
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

```
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

# Loop variables (with $ prefix)
for each $item in items:
  info "Processing {$item}"  # Loop variable interpolation

for $i in range 1 to 5:
  info "Attempt {$i} of 5"   # Loop variable interpolation

# Mixed interpolation with different scopes
info "Deploying {$tag} to {$environment} from {$globals.registry}"
info "Project: {$globals.project} v{$globals.version}"
info "API: {$globals.api_url} - Processing item {$item}"
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

```
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

```
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
```
task "docker-build":
  given $no_cache as boolean defaults to "false"
  given $push as boolean defaults to "false"
  given $platform defaults to "linux/amd64"
  
  run "docker build {$no_cache ? '--no-cache' : ''} {$push ? '--push' : ''} --platform {$platform} -t myapp:latest ."
```

**Environment-Specific Configuration:**
```
task "deploy":
  requires $env from ["dev", "staging", "production"]
  
  set $replicas to "{if $env is 'production' then '3' else '1'}"
  set $cpu to "{if $env is 'production' then '2000m' else '500m'}"
  set $log_level to "{if $env is 'production' then 'error' else 'debug'}"
  
  info "Deploying with {$replicas} replicas, {$cpu} CPU, {$log_level} logging"
```

**Build Optimization Flags:**
```
task "compile":
  given $optimize as boolean defaults to "true"
  given $debug as boolean defaults to "false"
  
  run "gcc {$optimize ? '-O2' : '-O0'} {$debug ? '-g' : ''} -o app main.c"
```

**CI/CD Pipeline Flags:**
```
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

```
project "myapp" version "1.0.0":
  set registry to "ghcr.io/company"    # Project setting
  set api_url to "https://api.example.com"
  set timeout to "30s"
  set platforms as list to ["linux", "darwin", "windows"]  # Array setting
  set environments as list to ["dev", "staging", "production"]
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

#### Strict Variable Checking

**New in v2.0**: drun now enforces strict variable checking by default to prevent runtime errors from undefined variables.

**Default Behavior (Strict Mode)**:
```
task "example":
    let $name = "world"
    info "Hello {$name}"        # ✅ Works: variable defined
    info "Hello {$undefined}"   # ❌ Error: undefined variable: {$undefined}
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
  set $platforms as list to ["linux", "darwin", "windows"]
  
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
  
  # Usage: xdrun deploy environment=production version=v1.2.3
```

#### Required Parameters with Defaults

Required parameters can have default values. When a default is provided, the parameter becomes optional at the CLI level but still benefits from the validation constraints:

```
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

```
# Valid:
requires $env from ["dev", "staging", "prod"] defaults to "dev"

# Invalid - will cause parse error:
requires $env from ["dev", "staging", "prod"] defaults to "production"
# Error: default value 'production' must be one of the allowed values: [dev, staging, prod]
```

#### CLI Argument Syntax

Parameters are passed to tasks using simple `key=value` syntax (no `--` prefix required):

```bash
# Parameter passing examples
xdrun deploy environment=production
xdrun build tag=v1.2.3 push=true
xdrun test suites=unit,integration verbose=true

# Multiple parameters
xdrun deploy environment=staging replicas=5 timeout=300
```

#### Optional Parameters

```
task "build":
  given $tag defaults to current git commit
  given $push defaults to false
  given $platforms defaults to ["linux/amd64"]
  
  # Usage: xdrun build
  # Usage: xdrun build tag=custom push=true
```

#### Variadic Parameters

```
task "test":
  accepts $suites as list of strings
  accepts flags as list
  
  # Usage: xdrun test --suites=unit,integration --flags=--verbose,--coverage
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

## Array Literals and Matrix Execution

drun v2 supports array literals for defining lists of values directly in the code, enabling powerful matrix execution patterns for comprehensive testing and deployment scenarios.

### Array Literal Syntax

Array literals use square bracket notation with comma-separated values:

```
# Basic array literals
["item1", "item2", "item3"]
["linux", "darwin", "windows"]
["dev", "staging", "production"]

# Numbers and mixed types
[1, 2, 3, 4, 5]
["port", 8080, "timeout", "30s"]
```

### Project-Level Array Settings

Arrays can be defined at the project level using two syntaxes:

```
project "myapp" version "1.0.0":
  # Simple string settings
  set registry to "ghcr.io/company"
  set api_url to "https://api.example.com"
  
  # Array settings using "as list to" syntax
  set platforms as list to ["linux", "darwin", "windows"]
  set environments as list to ["dev", "staging", "production"]
  set node_versions as list to ["16", "18", "20"]
  set databases as list to ["postgres", "mysql", "mongodb"]
```

**Accessing Project Arrays:** Project-level arrays must be accessed using the consistent `$globals.` prefix:

```
# ✅ Correct: Use $globals prefix for consistency
for each $platform in $globals.platforms:
  info "Building for {$platform}"

# ❌ Deprecated: Direct access (will show deprecation warning)
for each $platform in platforms:
  info "Building for {$platform}"
```

This maintains consistency with other global variable access patterns like `{$globals.project}`, `{$globals.version}`, and `{$globals.registry}`.

### Loop Variables with Array Literals

Loop variables use the `$variable` syntax for consistency with the scoping system:

```
# Direct array literal in loops
for each $platform in ["linux", "darwin", "windows"]:
  info "Building for {$platform}"

# Using project-defined arrays
for each $env in $globals.environments:
  for each $service in ["api", "web", "worker"]:
    deploy {$service} to {$env}
```

### Matrix Execution Patterns

Matrix execution allows comprehensive testing across multiple dimensions:

#### Sequential Matrix Execution

```
# Cross-platform builds (OS × Architecture)
for each $platform in $globals.platforms:
  for each $arch in ["amd64", "arm64"]:
    build for {$platform}/{$arch}

# Database testing (Database × Version × Test Suite)
for each $db in $globals.databases:
  for each $version in ["latest", "lts", "stable"]:
    for each $suite in ["unit", "integration", "performance"]:
      test {$db}:{$version} with {$suite} tests
```

#### Parallel Matrix Execution

```
# Multi-region deployment (parallel regions, sequential services)
for each $region in ["us-east", "eu-west", "ap-south"] in parallel:
  for each $service in ["api", "web", "worker"]:
    deploy {$service} to {$region}

# CI/CD pipeline parallelization
for each $job in ["lint", "test", "security-scan", "build"] in parallel:
  when $job is "lint":
    for each $linter in ["eslint", "prettier", "golangci-lint"]:
      run {$linter}
  when $job is "test":
    for each $suite in ["unit", "integration"]:
      run {$suite} tests
```

#### Mixed Parallel/Sequential Execution

```
# Parallel environments, sequential deployment steps
for each $env in ["dev", "staging", "production"] in parallel:
  for each $step in ["build", "test", "deploy", "verify"]:
    execute {$step} in {$env}
```

### Real-World Matrix Use Cases

#### DevOps Scenarios
- **Multi-platform builds**: OS × Architecture × Compiler Version
- **Deployment strategies**: Environment × Service × Region
- **Testing matrices**: Browser × Device × Test Suite
- **Performance testing**: Load Level × Endpoint × Configuration

#### CI/CD Pipelines
- **Parallel job execution**: Lint, Test, Security, Build
- **Multi-environment deployment**: Dev, Staging, Production
- **Canary deployments**: Service × Traffic Percentage
- **Integration testing**: Service × Database × Version

#### Infrastructure Management
- **Multi-cloud deployment**: Provider × Region × Service
- **Monitoring setup**: Service × Metric × Alert Rule
- **Security scanning**: Tool × Target × Severity Level
- **Backup strategies**: Database × Schedule × Retention Policy

### Variable Scoping in Matrix Execution

Loop variables follow the established scoping rules:

```
project "matrix-demo":
  set platforms as list to ["linux", "darwin", "windows"]
  set registry to "ghcr.io/company"

task "matrix-build":
  # Project arrays accessed via $globals
  for each $platform in $globals.platforms:
    # Loop variable uses $variable syntax
    for each $arch in ["amd64", "arm64"]:
      # Both loop variables available in nested scope
      info "Building {$globals.registry}/app:{$platform}-{$arch}"
      build for {$platform}/{$arch}
```

This matrix execution system enables comprehensive automation workflows while maintaining drun's natural language philosophy and clear variable scoping.

---

## Code Reuse Features

drun v2 provides powerful mechanisms for code reuse and eliminating duplication in automation workflows. These features enable you to write DRY (Don't Repeat Yourself) task definitions while maintaining readability and maintainability.

### Project-Level Parameters

Project-level parameters are defined once at the project level and shared across all tasks. They can be overridden via CLI, making them perfect for global configuration values.

#### Syntax

```drun
project "my-app" version "1.0.0":
  parameter $name as type [from [values]] defaults to "value"
```

#### Examples

```drun
version: 2.0

project "docker-automation" version "1.0.0":
  # Boolean parameter with default
  parameter $no_cache as boolean defaults to "false"
  
  # String parameter with constraint list
  parameter $environment as string from ["dev", "staging", "prod"] defaults to "dev"
  
  # String parameter with pattern validation
  parameter $registry as string defaults to "docker.io"
  
  # Number parameter with range
  parameter $timeout as number defaults to 300

task "build" means "Build with project-level configuration":
  info "Environment: {$environment}"
  info "Registry: {$registry}"
  info "No cache: {$no_cache}"
  info "Timeout: {$timeout}s"
```

#### Usage

```bash
# Use defaults
xdrun build

# Override parameters via CLI
xdrun build environment=prod no_cache=true registry=gcr.io
```

#### Key Features

- **Shared Configuration**: Define once, use everywhere
- **Type Safety**: Full type validation and constraints
- **CLI Overrides**: Can be overridden at runtime
- **Default Values**: Always have sensible defaults
- **Validation**: Same validation rules as task parameters

### Snippets

Snippets are reusable blocks of statements that can be included in any task. They're perfect for common sequences of actions that appear across multiple tasks.

#### Syntax

```drun
project "my-app" version "1.0.0":
  snippet "name":
    # Statements that can be reused
    statement1
    statement2
```

#### Examples

```drun
version: 2.0

project "my-app" version "1.0.0":
  # Common logging snippet
  snippet "log-start":
    info "═══════════════════════════════════"
    info "  Starting task execution"
    info "═══════════════════════════════════"
  
  # Environment check snippet
  snippet "check-env":
    if env DOCKER_HOST exists:
      info "Docker: Remote host at ${DOCKER_HOST}"
    else:
      info "Docker: Local daemon"
  
  # Cleanup snippet
  snippet "cleanup-temp":
    info "Cleaning up temporary files..."
    info "Done"

task "build" means "Build application":
  use snippet "log-start"
  use snippet "check-env"
  
  info "Building application..."
  # Build logic here
  
  use snippet "cleanup-temp"
  success "Build complete"

task "deploy" means "Deploy application":
  use snippet "log-start"
  use snippet "check-env"
  
  info "Deploying application..."
  # Deploy logic here
  
  success "Deploy complete"
```

#### Key Features

- **Reusability**: Define once, use multiple times
- **Scoped Access**: Snippets can access project parameters and task variables
- **Variable Interpolation**: Full support for variable interpolation
- **Control Flow**: Can contain any valid drun statements including conditionals

### Task Templates

Task templates allow you to define parameterized task structures that can be called like functions. They're perfect for tasks that follow the same pattern but with different parameters.

#### Syntax

```drun
template task "name":
  given $param defaults to "value"
  # Template body
```

#### Examples

```drun
version: 2.0

project "docker-builds" version "1.0.0":
  parameter $no_cache as boolean defaults to "false"
  parameter $registry as string defaults to "docker.io"
  
  snippet "show-config":
    info "Registry: {$registry}"
    info "Cache: {$no_cache ? 'disabled' : 'enabled'}"

# Define a reusable template
template task "docker-build":
  given $target defaults to "prod"
  given $tag defaults to "latest"
  given $platform defaults to "linux/amd64"
  
  step "Building Docker image"
  use snippet "show-config"
  
  info "Target: {$target}"
  info "Tag: {$registry}/{$tag}"
  info "Platform: {$platform}"
  info "Building: docker build {$no_cache ? '--no-cache' : ''} --target={$target} --platform={$platform} -t {$registry}/{$tag} ."
  
  success "Built {$tag}"

# Use the template with different parameters
task "build:web" means "Build web application":
  call task "docker-build" with target="web" tag="myapp:web"

task "build:api" means "Build API server":
  call task "docker-build" with target="api" tag="myapp:api"

task "build:worker" means "Build background worker":
  call task "docker-build" with target="worker" tag="myapp:worker" platform="linux/arm64"

# Use the template with all defaults
task "build:base" means "Build base image":
  call task "docker-build" with target="base"

# Complex task that calls template multiple times
task "build:all" means "Build all images":
  info "Building complete application stack..."
  
  call task "build:web"
  call task "build:api"
  call task "build:worker"
  call task "build:base"
  
  success "All images built successfully!"
```

#### Calling Templates

```bash
# Call regular tasks (which may call templates internally)
xdrun build:web

# Can override project parameters too
xdrun build:all no_cache=true registry=ghcr.io
```

#### Key Features

- **Parameterization**: Accept parameters with defaults
- **Reusability**: Call the same template with different parameters
- **Composition**: Templates can call other tasks and use snippets
- **Type Safety**: Template parameters support all standard validations
- **Variable Access**: Templates have access to project parameters

### Complete Example: Docker Build System

Here's a comprehensive example combining all code reuse features:

```drun
version: 2.0

project "microservices" version "1.0.0":
  # Global configuration
  parameter $no_cache as boolean defaults to "false"
  parameter $environment as string from ["dev", "staging", "prod"] defaults to "dev"
  parameter $registry as string defaults to "docker.io"
  parameter $push as boolean defaults to "false"
  
  # Reusable configuration display
  snippet "show-build-config":
    info "╔════════════════════════════════╗"
    info "║     Build Configuration        ║"
    info "╚════════════════════════════════╝"
    info "Environment: {$environment}"
    info "Registry: {$registry}"
    info "Cache: {$no_cache ? 'disabled' : 'enabled'}"
    info "Push: {$push ? 'yes' : 'no'}"
    info ""
  
  # Reusable Docker login check
  snippet "check-registry-auth":
    if $push is true:
      info "Checking registry authentication..."
      if env DOCKER_AUTH exists:
        info "✓ Registry authentication configured"
      else:
        warn "⚠ No registry authentication found"
  
  # Cleanup snippet
  snippet "cleanup":
    info "Cleaning up build artifacts..."
    info "Done"

# Template for Docker builds
template task "docker-build":
  given $service defaults to "app"
  given $target defaults to "prod"
  given $tag defaults to "latest"
  
  step "Building {$service} image"
  use snippet "show-build-config"
  use snippet "check-registry-auth"
  
  info "Service: {$service}"
  info "Target: {$target}"
  info "Full tag: {$registry}/{$service}:{$tag}"
  
  info "Building image..."
  # Actual Docker build would go here
  
  if $push is true:
    info "Pushing to registry..."
    # Actual Docker push would go here
  
  success "✓ Built {$service}:{$tag}"

# Template for testing services
template task "test-service":
  given $service defaults to "app"
  given $test_suite defaults to "all"
  
  step "Testing {$service}"
  info "Test suite: {$test_suite}"
  info "Running tests..."
  # Actual test commands would go here
  success "✓ Tests passed for {$service}"

# Concrete tasks using templates
task "build:frontend" means "Build frontend service":
  call task "docker-build" with service="frontend" target="web" tag="v1.0.0"
  use snippet "cleanup"

task "build:backend" means "Build backend API":
  call task "docker-build" with service="backend" target="api" tag="v1.0.0"
  use snippet "cleanup"

task "build:worker" means "Build background worker":
  call task "docker-build" with service="worker" target="worker" tag="v1.0.0"
  use snippet "cleanup"

task "test:frontend" means "Test frontend":
  call task "test-service" with service="frontend" test_suite="e2e"

task "test:backend" means "Test backend":
  call task "test-service" with service="backend" test_suite="integration"

# Orchestration tasks
task "build:all" means "Build all services":
  info "═══════════════════════════════════════"
  info "  Building Complete Microservices Stack"
  info "═══════════════════════════════════════"
  info ""
  
  call task "build:frontend"
  call task "build:backend"
  call task "build:worker"
  
  success "✨ All services built successfully!"

task "test:all" means "Test all services":
  call task "test:frontend"
  call task "test:backend"
  success "✨ All tests passed!"

task "ci" means "Complete CI pipeline":
  call task "build:all"
  call task "test:all"
  success "✨ CI pipeline completed!"
```

#### Usage Examples

```bash
# Build individual services
xdrun build:frontend
xdrun build:backend

# Build with custom parameters
xdrun build:frontend no_cache=true environment=prod push=true

# Build everything
xdrun build:all

# Test services
xdrun test:frontend
xdrun test:all

# Complete CI pipeline
xdrun ci environment=staging registry=gcr.io
```

### Benefits

The code reuse features provide several key benefits:

1. **DRY Principle**: Eliminate duplication across tasks
2. **Maintainability**: Update logic in one place
3. **Consistency**: Ensure consistent behavior across tasks
4. **Readability**: Templates and snippets have clear, semantic names
5. **Flexibility**: Override parameters as needed
6. **Type Safety**: Full validation on all parameters
7. **Composition**: Combine features for powerful workflows

### Namespaced Includes

Namespaced includes allow you to import snippets, templates, and tasks from external `.drun` files, enabling true code sharing across projects. Each included file gets its own namespace (derived from its project name) to prevent naming collisions.

#### Basic Syntax

```drun
project "myapp":
    # Include everything from a file
    include "shared/docker.drun"
    
    # Selective includes
    include snippets from "shared/utils.drun"
    include templates from "shared/k8s.drun"
    include tasks from "shared/common.drun"
    
    # Multiple selectors
    include snippets, templates from "shared/helpers.drun"
```

#### Namespace Resolution

The namespace is automatically derived from the `project` declaration in the included file:

```drun
# shared/docker.drun
project "docker":
    snippet "login-check":
        if env DOCKER_AUTH exists:
            info "✓ Docker authenticated"
        else:
            warn "⚠ No Docker authentication"
    
    template task "build":
        given $image defaults to "app:latest"
        info "Building {$image}..."

# main.drun
project "myapp":
    include "shared/docker.drun"

task "deploy":
    use snippet "docker.login-check"    # namespace.element
    call task "docker.build"             # namespace.task
```

#### Transitive Resolution

When an included element references another element from the same file, it's automatically resolved within that namespace:

```drun
# shared/docker.drun
project "docker":
    snippet "login-check":
        info "Checking auth..."
    
    template task "push":
        given $image
        use snippet "login-check"    # No namespace needed within same file
        info "Pushing {$image}..."

# main.drun
project "myapp":
    include "shared/docker.drun"

task "deploy":
    call task "docker.push" with image="myapp:v1"
    # ✓ Works! docker.push automatically finds docker.login-check
```

#### Path Resolution

Include paths are resolved in the following order:

1. **Relative to current file**: `../shared/docker.drun` 
2. **Relative to workspace root**: `shared/docker.drun`
3. **Absolute path**: `/absolute/path/docker.drun`

#### Circular Include Detection

drun automatically detects and prevents circular includes:

```drun
# main.drun includes docker.drun
# docker.drun includes utils.drun
# utils.drun includes docker.drun  ← Circular! Will be skipped
```

#### Complete Example

```drun
# shared/docker.drun
version: 2.0

project "docker":
    parameter $registry as string defaults to "docker.io"
    
    snippet "login-check":
        if env DOCKER_AUTH exists:
            info "✓ Authenticated with {$registry}"
        else:
            warn "⚠ No authentication for {$registry}"
    
    snippet "cleanup":
        info "Cleaning up Docker resources..."
    
    template task "build":
        given $target defaults to "prod"
        given $image defaults to "app:latest"
        
        step "Building Docker image"
        use snippet "login-check"
        info "docker build --target={$target} -t {$image} ."
        use snippet "cleanup"
        success "Built {$image}"
    
    template task "push":
        given $image defaults to "app:latest"
        
        step "Pushing to registry"
        use snippet "login-check"
        info "docker push {$registry}/{$image}"
        success "Pushed {$image}"

# main.drun
version: 2.0

project "myapp":
    include "shared/docker.drun"
    
    parameter $version as string defaults to "1.0.0"

task "build:web":
    call task "docker.build" with target="web" image="myapp:web-{$version}"

task "build:api":
    call task "docker.build" with target="api" image="myapp:api-{$version}"

task "deploy":
    call task "build:web"
    call task "build:api"
    call task "docker.push" with image="myapp:web-{$version}"
    call task "docker.push" with image="myapp:api-{$version}"
```

#### Key Features

- **Namespace Safety**: No naming collisions between included files
- **Dot Notation**: Clean, familiar syntax for referencing included elements
- **Selective Imports**: Import only what you need (`snippets`, `templates`, `tasks`)
- **Transitive Resolution**: Included elements automatically resolve their dependencies
- **Path Flexibility**: Relative, workspace, and absolute path support
- **Circular Detection**: Automatic prevention of circular includes
- **Verbose Logging**: Use `-v` flag to see what's being included

#### Benefits

1. **Code Sharing**: Share common workflows across multiple projects
2. **Library Pattern**: Create reusable "library" files for different domains (docker, k8s, git, etc.)
3. **Team Standards**: Enforce consistent patterns across team projects
4. **DRY at Scale**: Eliminate duplication not just within a project, but across all projects
5. **Maintainability**: Update shared logic once, affects all users
6. **Namespace Safety**: Clear ownership and no conflicts

### Remote Includes

Remote includes extend the include system to fetch `.drun` files from external sources like GitHub repositories and HTTPS URLs. This enables sharing workflows across teams, organizations, and the entire community.

#### GitHub Includes

Include files directly from GitHub repositories using the `github:` protocol:

```drun
project "myapp":
    # Include from GitHub with auto branch detection
    include "github:owner/repo/path/to/file.drun"
    
    # Include from specific branch
    include "github:owner/repo/path/to/file.drun@main"
    
    # Include from specific tag
    include "github:owner/repo/path/to/file.drun@v1.0.0"
    
    # Include from specific commit
    include "github:owner/repo/path/to/file.drun@abc123"
```

**Smart Default Branch Detection**: If no branch/ref is specified, drun automatically detects the repository's default branch (`main` or `master`).

#### HTTPS Includes

Include files from any HTTPS URL:

```drun
project "myapp":
    # Include from raw GitHub URL
    include "https://raw.githubusercontent.com/owner/repo/main/shared/workflow.drun"
    
    # Include from any HTTPS source
    include "https://example.com/shared/tasks.drun"
```

#### Drunhub Standard Library

Drunhub is the official standard library repository at `https://github.com/phillarmonic/drun-hub` containing reusable templates, snippets, and tasks organized by category. Import from drunhub using the `drunhub:` protocol:

```drun
project "myapp":
    # Import from drunhub - uses project name as namespace
    include from drunhub "ops/docker"
    
    # Import with custom namespace (overrides project name)
    include from drunhub "ops/kubernetes" as k8s
    
    # Import from nested folders
    include from drunhub "utils/logging/advanced" as log
    
    # Import from specific branch/tag
    include from drunhub "ops/docker@v1.0" as ops
```

**Key Features**:

- **Automatic `.drun` extension**: No need to add `.drun` extension
- **Custom namespaces**: Override default project names with `as` clause
- **Folder protection**: Certain folders like `docs` and `.github` are blocked for security
- **Same caching**: Uses the same smart caching as other remote includes

**Example Usage**:

```drun
version: 2.0

project "deploy-app":
    # Import Docker utilities as "ops" namespace
    include from drunhub "ops/docker" as ops
    
    # Import Kubernetes helpers
    include from drunhub "ops/kubernetes" as k8s

task "deploy":
    # Use snippet from ops namespace
    use snippet "ops.check-docker"
    
    # Call task from k8s namespace
    call task "k8s.deploy" with namespace="production" replicas=3
    
    success "✓ Deployed successfully!"
```

**Custom Namespaces with Traditional Includes**:

The `as` clause also works with regular includes:

```drun
project "myapp":
    # Override namespace from included file
    include "shared/docker-utils.drun" as docker
    
    # Now use docker.* instead of the original project name
    use snippet "docker.build"
```

#### Smart Caching

Remote includes are automatically cached to `~/.drun/cache.solo` with:

- **1-minute expiration** by default
- **Automatic refresh** when cache expires
- **Stale cache fallback** for offline resilience (if network fails, uses expired cache)
- **Content-based keys** (hash of URL + ref)

**Disable caching** when needed:

```bash
# Bypass cache and always fetch fresh
xdrun --no-drun-cache -f myfile.drun mytask
```

#### Example: Community Workflows

```drun
# my-project.drun
version: 2.0

project "my-awesome-app":
    # Include Docker utilities from your organization
    include "github:myorg/drun-workflows/docker.drun@v1.2.0"
    
    # Include Kubernetes helpers from community
    include "github:drun-community/k8s-workflows/deployment.drun"
    
    # Include CI/CD patterns from team repo
    include "https://raw.githubusercontent.com/myteam/workflows/main/ci.drun"

task "deploy":
    # Use included snippets and templates
    use snippet "docker.security-scan"
    call task "k8s.deploy" with namespace="production"
    call task "ci.notify-slack" with message="Deployed!"
```

#### Authentication

For private repositories, set a GitHub token:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
xdrun -f myfile.drun deploy
```

#### Benefits

1. **Community Sharing**: Leverage workflows from the broader drun community
2. **Organization Libraries**: Share standardized workflows across your organization
3. **Version Control**: Pin to specific tags/commits for reproducibility
4. **Offline Resilience**: Stale cache fallback ensures workflows work offline
5. **Performance**: Smart caching reduces network requests
6. **Flexibility**: Works with both GitHub and any HTTPS source

### Best Practices

1. **Use Project Parameters for Global Config**: Things like registry URLs, environment, cache settings
2. **Use Snippets for Common Sequences**: Logging, cleanup, environment checks
3. **Use Templates for Repeated Patterns**: Build tasks, test tasks, deployment tasks
4. **Use Includes for Cross-Project Sharing**: Create shared library files for common domains (Docker, Kubernetes, Git)
5. **Meaningful Names**: Choose descriptive names for snippets and templates
6. **Namespace Organization**: Group related snippets/templates in a single file with a clear project name
7. **Selective Imports**: Use `include snippets from` when you only need specific types of elements
8. **Documentation**: Add comments explaining what each reusable component does
9. **Defaults**: Always provide sensible defaults for template parameters
10. **Library Structure**: Organize shared files in a `shared/` directory for clarity

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
capture from shell "git rev-parse --short HEAD" as $commit_hash
capture from shell "whoami" as $username
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

#### Directory Empty Checks ⭐ *New*

Check if directories are empty or contain files using semantic conditions:

```
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

```
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
```
download "<url>" to "<path>"
```

**Advanced Options:**
```
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

```
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
```
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
```
⬇️  Downloading: https://example.com/large-file.zip
   → downloads/large-file.zip
   📥 [████████████████░░░░░░░░░░░░░░] 55.2% | 2.3 GB/4.2 GB | 15.3 MB/s | ETA: 2m15s
   📊 4.2 GB in 4m32s (15.45 MB/s)
✅ Downloaded successfully to: downloads/large-file.zip
   🔒 Set permissions: -rwxr--r--
```

**Error Handling:**

```
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

```
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

```
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

```
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

```
# Service waiting with timeout and retry
wait for service at "https://app.example.com/health" to be ready
wait for service at "https://app.example.com" to be ready timeout "60s"
wait for service at "https://api.example.com" to be ready timeout "30s" retry "5s"

# Health checks with status validation
# Note: Health checks are implemented via HTTP GET requests with curl
# They automatically validate HTTP status codes and provide retry logic
```

#### Network Testing

```
# Port connectivity testing
test connection to "database.example.com" on port 5432
test connection to "localhost" on port 8080 timeout "10s"

# Ping testing
ping host "example.com"
ping host "8.8.8.8" timeout "3s"
```

#### Advanced Network Operations

```
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

```
step "Starting deployment process"
info "Configuration loaded successfully"
warn "Using default configuration"
error "Failed to connect to database"
success "Deployment completed successfully"
```

**Output Formatting:**

- `step` - Displays message in a box (no line breaks by default):
  ```
  ┌────────────────────────────────┐
  │ Starting deployment process    │
  └────────────────────────────────┘
  ```
- `info` - Displays with ℹ️ emoji prefix: `ℹ️  Configuration loaded successfully`
- `warn` - Displays with ⚠️ emoji prefix: `⚠️  Using default configuration`
- `error` - Displays with ❌ emoji prefix: `❌ Failed to connect to database`
- `success` - Displays with ✅ emoji prefix: `✅ Deployment completed successfully`
- `fail` - Displays with 💥 emoji prefix and exits with error

**Optional Line Breaks for `step`:**

By default, step boxes have no extra spacing. Add line breaks when you need visual separation:

```
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

```
fail                                    # Exit with error code 1
fail with "Custom error message"        # Exit with custom message
exit with code 0                        # Exit with specific code
```

#### Progress Tracking

drun v2 provides built-in progress indicators and timer functions for tracking long-running operations:

##### Progress Indicators

```
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

```
# Start a named timer
info "{start timer('deployment_timer')}"

# Show elapsed time for a running timer
info "{show elapsed time('deployment_timer')}"

# Stop a timer and show final elapsed time
info "{stop timer('deployment_timer')}"
```

##### Advanced Progress and Timer Usage

```
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

```
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

#### Built-in Function Pipe Operations ⭐ *New*

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

# Check if tools are available or not available
if docker is available:
    info "Docker is ready"
else:
    error "Docker not found"

if kubectl is not available:
    warn "Kubernetes tools not installed"
    info "Skipping Kubernetes deployment"
```

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
```
if "docker compose" is available:
    info "Using Docker Compose v2"

if "docker-compose" is available:
    info "Using Docker Compose v1"
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

### Environment Variable Interpolation ⭐ *New*

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

### Environment Variable Conditionals ⭐ *New*

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
  warn "⚠️  Running in PRODUCTION environment"
  info "Extra caution advised!"
else:
  info "✅ Not in production environment"

# Check if environment variable is NOT equal to a value
if env DEBUG_MODE is not "true":
  info "Debug mode is disabled"
```

#### Empty/Non-Empty Checks

```drun
# Check if environment variable is not empty
if env DATABASE_URL is not empty:
  success "✅ DATABASE_URL is configured"
  run "python manage.py migrate"
else:
  warn "⚠️  DATABASE_URL is not set"
  info "Set it with: export DATABASE_URL=postgresql://..."
  fail "Missing required database configuration"
```

#### Compound Conditions ⭐ *New*

Combine multiple checks with `and` for more precise validation:

```drun
# Ensure variable exists AND is not empty (rejects empty strings)
task "secure-deploy":
  if env API_TOKEN exists and is not empty:
    success "✅ API_TOKEN is properly configured"
    run "curl -H 'Authorization: Bearer ${API_TOKEN}' https://api.example.com/deploy"
  else:
    error "❌ API_TOKEN must be set and not empty"
    fail "Missing required credentials"

# Ensure variable exists AND equals specific value
task "production-check":
  if env DEPLOY_ENV exists and is "production":
    warn "⚠️  Confirmed production deployment"
    info "Running extra validation..."
    run "npm run test:integration"
  else:
    info "✅ Non-production environment"

# Build with optional build arguments
task "docker-build":
  if env BUILD_TOKEN exists and is not empty:
    info "🔑 Using authenticated build"
    run "docker build --build-arg TOKEN='${BUILD_TOKEN}' -t myapp ."
  else:
    info "🔓 Using public build (no authentication)"
    run "docker build -t myapp ."
```

#### Practical Examples

```drun
# Conditional deployment based on environment
task "deploy":
  if env DEPLOY_ENV is "production":
    warn "⚠️  Deploying to PRODUCTION"
    info "Running production pre-flight checks..."
    
    if env DATABASE_URL exists:
      success "✅ Database configuration found"
    else:
      error "❌ DATABASE_URL required for production"
      fail "Missing required environment variable"
    
    if env API_KEY exists:
      success "✅ API key found"
    else:
      error "❌ API_KEY required for production"
      fail "Missing required API credentials"
    
    success "✅ All pre-flight checks passed"
    info "Deploying to production..."
  else:
    info "✅ Deploying to development/staging"
    info "Skipping production pre-flight checks"

# CI/CD detection
task "build":
  if env CI exists:
    info "🤖 Running in CI environment"
    set $ci_mode to "true"
    run "npm run build --ci"
  else:
    info "💻 Running locally"
    set $ci_mode to "false"
    run "npm run build"

# Configuration based on environment variables
task "configure":
  if env LOG_LEVEL is "debug":
    info "🔧 Using DEBUG log level"
    set $verbose to "true"
  else:
    if env LOG_LEVEL is "info":
      info "ℹ️  Using INFO log level"
      set $verbose to "false"
    else:
      info "✅ Using default log level"
      set $verbose to "false"

# Feature flags
task "start":
  if env ENABLE_EXPERIMENTAL is "true":
    info "🧪 Experimental features enabled"
    run "npm run start:experimental"
  else:
    info "✅ Using stable version"
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
  given tag defaults to "{current git commit}"
  
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

### Git Branch Operations ⭐ *New*

```
version: 2.0

task "git branch operations" means "Demonstrate git branch builtin and pipe operations":
  info "🌿 Testing git branch operations..."
  
  # Basic git branch builtin
  set $branch to {current git branch}
  info "Current branch: {$branch}"
  
  # Git branch with pipe operations
  set $safe_branch to {current git branch | replace "/" by "-"}
  info "Safe branch name: {$safe_branch}"
  
  # Use in parameter defaults
  given $deployment_branch defaults to "{current git branch | replace '/' by '-' | lowercase}"
  info "Deployment branch: {$deployment_branch}"
  
  success "✅ Git branch operations completed!"

task "parameter defaults with builtins" means "Demonstrate builtin functions in parameter defaults":
  given $commit defaults to "{current git commit}"
  given $branch defaults to "{current git branch}"
  given $safe_branch defaults to "{current git branch | replace '/' by '-'}"
  
  info "📋 Parameter values:"
  info "  Commit: {$commit}"
  info "  Branch: {$branch}" 
  info "  Safe branch: {$safe_branch}"
  
  success "✅ Parameter defaults test completed!"
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
  
  # Tool-level lifecycle hooks (run once per drun execution)
  on drun setup:
    info "🚀 Starting drun execution pipeline"
    info "📊 Tool version: {$globals.drun_version}"
    capture pipeline_start_time from now
  
  on drun teardown:
    capture pipeline_end_time from now
    let total_time be {pipeline_end_time} - {pipeline_start_time}
    info "🏁 Drun execution pipeline completed"
    info "📊 Total execution time: {total_time}"

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

