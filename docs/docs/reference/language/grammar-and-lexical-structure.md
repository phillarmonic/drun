# Grammar and lexical structure

## Language Grammar

### File-value statements

```ebnf
file-value-statement = file-value-get | file-value-check | file-value-update ;
file-value-get       = "get", file-value-format, string, "from", string, "as", variable ;
file-value-check     = "check", file-value-format, string, "in", string,
                       ("equals" | "differs", "from"), string ;
file-value-update    = "update", file-value-format, string, "in", string, "to", string,
                       "or", ("fail" | "add", ["as", scalar-type]) ;
file-value-format    = "property" | "json" | "yaml" | "toml" | "match" ;
scalar-type          = "string" | "number" | "boolean" ;
```

The first string is the format-specific selector and the second is the file
path. `match` updates cannot use `or add`; JSON, YAML, and TOML additions require
an explicit scalar type. See [Structured file values](built-in-actions.md#structured-file-values)
for selector rules and runtime guarantees.

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

```drun
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
get, check, update

# File-value formats and comparisons
property, json, yaml, toml, match, equals, differs

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

```drun
# This is a single-line comment
task "example":  # End-of-line comment
  info "Hello"
```

#### Multiline Comments

Multiline comments use C-style `/* */` syntax and can span multiple lines:

```drun
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
6. **Non-code lines**: Blank lines and comment-only lines do not establish or change indentation, regardless of their leading spaces or tabs

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

### `xdrun --init` Template Manifests

`xdrun --init` and `xdrun --init-minimal` can bootstrap a spec from a template catalog instead of the built-in starter config:

```bash
xdrun --list-templates --templates-repo ../drun-templates
xdrun --init --template go-cli --templates-repo ../drun-templates
xdrun --init --from-template ../drun-templates --template go-cli

# Or target a specific manifest:
xdrun --init \
  --from-template github:owner/repo/templates/drun-init.yaml@main \
  --template go-cli
```

#### CLI Rules

- `--templates-repo <path>` points at a local template repository root that contains `templates.yaml`
- `--list-templates --templates-repo <path>` lists templates from that local repository
- `--list-templates --from-template <manifest>` lists templates from a specific manifest
- `--template <name> --templates-repo <path>` resolves through that local repository
- `--from-template` accepts `github:...`, `drunhub:...`, `https://...`, a local manifest path, or a local directory root containing `templates.yaml`
- `--template` is required when `--from-template` is used for initialization
- If neither `--templates-repo` nor `--from-template` is provided, `xdrun` falls back to configured catalog sources such as `DRUN_TEMPLATES_MANIFEST`, `DRUN_TEMPLATES_REPO`, or the remote official catalog
- `--file` and `--save-as-default` behave the same as regular `xdrun --init`

For local template development, a directory target lets you test without publishing a remote manifest. If `--from-template` points at a directory, `xdrun` reads `templates.yaml` from that directory root automatically.

#### Manifest Format

The manifest referenced by `--from-template`, or by the official catalog, is not the final `.drun` spec. The manifest can define templates as a map or a sequence:

```yaml
version: "1"
templates:
  go-cli:
    kind: go-cli
    description: "Drun starter for Go CLI projects"
    source: "templates/go-cli.drun"
```

Equivalent sequence form:

```yaml
version: "1"
templates:
  - name: go-cli
    kind: go-cli
    description: "Drun starter for Go CLI projects"
    source: "templates/go-cli.drun"
```

Each template entry supports:

- `name`: the template selector passed to `--template`
- `source`: the `.drun` file to fetch; it can be remote, local, or relative to the manifest
- `kind`: optional specialization such as `go-cli`
- `description`: optional human-facing description

#### Placeholder Contract

Template-authored `.drun` files can use a small placeholder vocabulary that `xdrun --init` rewrites safely:

- `{{project_name}}`: inferred from the current working directory
- `{{binary_name}}`: defaults to the inferred project name
- `{{cmd_path}}`: defaults to `./cmd/{{binary_name}}`
- `{{module_name}}`: inferred from local `go.mod` when present, otherwise falls back to the project name

#### Go CLI Template Example

For `kind: go-cli`, `xdrun --init` also applies narrow Go-specific rewrites for common command shapes such as `go build -o ./bin/<name>` and `./cmd/<name>`.

Example remote template:

```drun
# drun (do-run) CLI is a fast, semantic task runner with
# its own powerful automation language. Effortless tasks, serious speed.
# Learn more at https://github.com/phillarmonic/drun

version: 2.0

project "{{project_name}}" version "1.0":
task "default" means "Welcome":
	info "{{project_name}} Drun Spec"

task "build" means "Build {{binary_name}}":
	step "Building {{binary_name}}..."
	run "go build -ldflags=\"-X 'main.version=v0.0.1 (dev build)'\" -o ./bin/{{binary_name}} {{cmd_path}}"
	success "Build completed for {{binary_name}}"

task "install" means "Install {{binary_name}}":
	step "Installing {{binary_name}}..."
	run "go install {{cmd_path}}"
	success "Install completed for {{module_name}}"
```

This keeps the rewrite contract explicit: init only rewrites known placeholders and a few well-known Go command patterns. It does not attempt free-form semantic rewriting of arbitrary shell commands.

### String Interpolation

Strings support variable interpolation using `{$variable}` syntax for declared variables and `{variable}` for loop variables:

```drun
let $name = "world"
info "Hello, {$name}!"  # Outputs: Hello, world!

# Loop variables use $ prefix
for each $item in $items:
  info "Processing {$item}"  # Loop variable with $

# Complex expressions in interpolation
info "Current time: {now.format('HH:mm:ss')}"
```

### Multi-line Strings

drun v2 supports multi-line strings enclosed in quotes, allowing you to write commands that span multiple lines while maintaining readability and supporting all string features like interpolation and escape sequences.

#### Basic Multi-line Strings

Strings can span multiple lines, preserving line breaks:

```drun
task "example":
  run "echo Line 1
echo Line 2
echo Line 3"
```

This will execute as three separate echo commands, preserving the newlines.

#### Line Continuation

Use a backslash `\` before a newline to continue a command on the next line without inserting a line break:

```drun
task "docker build":
  run "docker run --rm \
      -v $(pwd):/workspace \
      -e ENV=production \
      -e DEBUG=false \
      myimage:latest"
```

The backslash-newline combination is removed, joining the lines together.

#### Escaped Quotes

Use `\"` to include literal quotes within strings:

```drun
task "with quotes":
  run "echo \"Hello, World!\"
echo \"Status: \\\"Running\\\"\"
echo Done"
```

#### Multi-line Strings with Interpolation

Interpolation works seamlessly within multi-line strings:

```drun
task "deploy":
  let $environment = "production"
  let $version = "1.2.3"

  run "echo Deploying to: {$environment}
echo Version: {$version}
echo Status: Ready
./deploy.sh {$environment} {$version}"
```

#### Complex Real-World Example

Combining all features for readable, maintainable scripts:

```drun
task "test coverage" means "Run tests and generate coverage report":
  let $app_env = "test"
  let $coverage_file = "coverage.xml"

  step "Running test suite with coverage"
  run "rm -f {$coverage_file}
docker compose exec \
    -e APP_ENV={$app_env} \
    -e XDEBUG_MODE=coverage \
    -u=www-data \
    php vendor/bin/phpunit --coverage-clover ./{$coverage_file}
docker compose exec \
    -e APP_ENV={$app_env} \
    -u=www-data \
    php bin/console tests:probe-coverage {$coverage_file}"

  success "Coverage report generated: {$coverage_file}"
```

#### Key Features

- **Preserve Line Breaks**: Newlines within strings are preserved in the output
- **Line Continuation**: Use `\` before newline to join lines without a break
- **Escape Sequences**: Support for `\"`, `\\`, `\n`, `\t`, `\r`
- **Interpolation**: Full support for variable and expression interpolation
- **Readability**: Write complex shell scripts in a clean, maintainable format

#### Best Practices

1. **Use line continuation** for long command lines to improve readability
2. **Preserve natural line breaks** for multi-command scripts
3. **Combine with interpolation** for dynamic, reusable command templates
4. **Escape quotes** when passing quoted strings to shell commands

---

### EBNF Grammar

```ebnf
(* Top-level constructs *)
program = { version_statement | project_declaration | snippet_definition | template_task_definition | task_definition | service_definition | orchestration_definition } ;

version_statement = "version" ":" number_literal ;

annotation = "@" identifier "(" [ string_literal { "," string_literal } ] ")" ;

(* Project declaration *)
project_declaration = "project" string_literal [ "version" string_literal ] ":"
                     { project_setting } ;

project_setting = "set" identifier "to" expression
                | "set" identifier "as" "list" "to" array_literal
                | "include" string_literal
                | "before" "any" "task" ":" statement_block
                | "after" "any" "task" ":" statement_block
                | "requires" "tools" ":" { tool_requirement }
                | shell_config ;

(* Reusable declarations *)
snippet_definition = { annotation } "snippet" string_literal ":" statement_block ;

template_task_definition = { annotation } "template" "task" string_literal ":"
                         { task_property }
                         statement_block ;

(* Task definition *)
task_definition = { annotation } "task" task_name [ "mode" string_literal ] [ "means" string_literal ] ":"
                 { task_property }
                 statement_block ;

task_name = string_literal | identifier_like ;

task_property = parameter_declaration
              | dependency_declaration
              | lifecycle_hook
              | variable_declaration ;

(* Parameters *)
parameter_declaration = "requires" "tools" ":" { tool_requirement }
                      | "requires" parameter_spec
                      | "given" parameter_spec
                      | "accepts" parameter_spec ;

parameter_spec = identifier [ parameter_constraint ] [ parameter_default ] ;

tool_requirement = tool_name { comparison_operator ( string_literal | number ) } ;

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
                  | conditional_when_statement
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
              { "else" "if" condition ":" statement_block }
              [ "else" ":" statement_block ] ;

conditional_when_statement = "when" condition ":" statement_block
                           [ "otherwise" ":" statement_block ] ;

for_statement = "for" "each" variable "in" ( expression | array_literal ) [ "in" "parallel" ] ":"
               statement_block ;

try_statement = "try" ":" statement_block
               { "catch" identifier ":" statement_block }
               [ "finally" ":" statement_block ] ;

(* Task calls *)
task_call_statement = "call" "task" task_name [ "with" parameter_list ] ;

parameter_list = parameter_assignment { parameter_assignment } ;

parameter_assignment = identifier "=" ( string_literal | number ) ;

(* Conditions *)
condition = logical_expression ;

logical_expression = comparison_expression
                   { ( "and" | "or" ) comparison_expression } ;

comparison_expression = additive_expression
                      [ comparison_operator additive_expression
                      | "is" version_order "than" "version" additive_expression ] ;

version_order = "older" | "newer" ;

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
                     | capture_expression_statement
                     | capture_shell_statement ;

capture_expression_statement = "capture" identifier "from" expression ;

capture_shell_statement = "capture" "from" "shell" string_literal "as" variable
                        | "capture" "from" "shell" "as" variable ":" statement_block ;

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
                    | "if" tool_list availability_verb "available" [ "and" "version" comparison_operator version_value ] ":" statement_block [ "else" ":" statement_block ]
                    | "if" tool_list availability_verb "not" "available" [ "and" "version" comparison_operator version_value ] ":" statement_block [ "else" ":" statement_block ]
                    | "if" ( tool_name | string_literal ) "version" comparison_operator version_value ":" statement_block [ "else" ":" statement_block ]
                    | "when" "in" environment_name "environment" ":" statement_block [ "else" ":" statement_block ] ;

tool_list = ( tool_name | string_literal ) { "," ( tool_name | string_literal ) } ;

availability_verb = "is" | "are" ;

detection_target = "project" "type"
                 | tool_name [ "version" ] ;

tool_alternatives = ( tool_name | string_literal ) { "or" ( tool_name | string_literal ) } ;

tool_name = identifier ;

environment_name = "ci" | "local" | "production" | "staging" | "development" | identifier ;

variable_name = "$" identifier ;
version_value = string_literal | number_literal ;

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
identifier_like = identifier { "-" identifier } ;

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

(* Service and orchestration declarations *)
service_definition = "service" string_literal "in" string_literal ":" statement_block ;

orchestration_definition = "orchestrate" string_literal ":" statement_block ;

orchestration_action_statement = "orchestrate" string_literal orchestration_action
                               [ service_filter ]
                               [ orchestration_option_block ]
                               [ "starting" "from" ( string_literal | variable | interpolation ) ] ;

orchestration_action = "start"
                     | "up"
                     | "stop"
                     | "restart"
                     | "recreate"
                     | "status"
                     | "show" "endpoints"
                     | "endpoints"
                     | "health"
                     | "health_check"
                     | "build"
                     | "pull"
                     | "down"
                     | "logs"
                     | "clone" "repositories"
                     | "update" "repositories"
                     | "list" "branches" [ string_literal ]
                     | "switch" "branch" "to" "default"
                     | "set" "all" "branches" "to" "default" ;

service_filter = "services" array_literal
               | "service" ( string_literal | variable | interpolation ) ;

orchestration_option_block = "with" orchestration_option { [ "," ] orchestration_option } ;

orchestration_option = "cache" string_literal
                     | "branch" string_literal
                     | "timeout" string_literal ;

(* Comments *)
single_line_comment = "#" { any_character_except_newline } ;
multiline_comment = "/*" { any_character } "*/" ;
```

---
