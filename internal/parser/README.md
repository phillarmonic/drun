# Parser Module

This directory contains the drun v2 parser, refactored into domain-specific files for better organization and maintainability.

## Architecture

The parser follows a **domain-driven design** where each file handles a specific aspect of the drun language:

### Core Parser (`parser.go`) - 115 lines
The main orchestration layer containing:
- `Parser` struct definition
- Constructor functions (`NewParser`, `NewParserWithSource`)
- Token advancement (`nextToken`)
- Main parsing loop (`ParseProgram`)

This file delegates all actual parsing work to specialized domain files.

---

## Domain-Specific Parser Files

### Project-Level Parsing

#### `parser_project.go` - 807 lines
Handles project-level declarations and configuration:
- **Version statements** (`version: 2`)
- **Project statements** (`project "name"`)
- **Set statements** (global variable declarations)
- **Include statements** (importing other files)
- **Project parameters** (`given $param as type`)
- **Snippet definitions** (`snippet "name"`)
- **Shell configuration** (`shell:`, platform-specific configs)
- **Lifecycle hooks** (`before:`, `after:`)

---

### Task & Execution

#### `parser_task.go` - 432 lines
Parses task definitions and execution flow:
- **Task statements** (`task "name"`)
- **Task templates** (`template task "name"`)
- **Task bodies** (parameters, dependencies, statements)
- **Dependency declarations** (`depends on X, Y`)
- **Task-from-template** instantiation

#### `parser_control.go` - 603 lines
Control flow constructs:
- **Conditional statements** (`if`, `when`, `otherwise`)
- **Loop statements** (`for`, `for each`)
- **Break/continue** statements
- **Control flow body parsing** (statements within control blocks)

---

### Data & Variables

#### `parser_parameter.go` - 311 lines
Parameter declarations and validation:
- **Parameter statements** (`requires $param as type`)
- **Advanced constraints** (range, pattern, validation)
- **Type checking** (string, number, boolean)
- **Pattern macros** (url, email, semver, etc.)

#### `parser_variable.go` - 281 lines
Variable operations:
- **Variable declarations** (`let $var = value`)
- **Variable assignment** (`set $var = value`)
- **Variable transformation** (`transform $var`)
- **Variable capture** from shell commands

#### `parser_expression.go` - 183 lines
Expression parsing:
- **Binary expressions** (arithmetic, logical)
- **Literal expressions** (strings, numbers, booleans)
- **Identifier expressions** (variables, references)
- **Function calls** (`function(args)`)
- **Array literals** (`[item1, item2]`)

---

### Actions & Operations

#### `parser_action.go` - 105 lines
Action statements and task calls:
- **Action statements** (`info`, `step`, `success`, `warning`, `error`)
- **Task call statements** (`call task-name`)

#### `parser_shell.go` - 175 lines
Shell command execution:
- **Shell statements** (`run "command"`)
- **Multiline shell commands**
- **Command capture** (`capture $var from command`)

---

### External Systems

#### `parser_docker.go` - 101 lines
Docker operations:
- **Docker commands** (`docker build`, `docker run`, etc.)
- Container management

#### `parser_git.go` - 256 lines
Git operations:
- **Git commands** (`git clone`, `git commit`, etc.)
- Branch and tag management

#### `parser_http.go` - 383 lines
HTTP operations:
- **HTTP requests** (`get`, `post`, `put`, `delete`, etc.)
- **Download operations** (`download from URL to path`)

#### `parser_network.go` - 174 lines
Network operations:
- **Health checks** (`check health`)
- **Port testing** (`check if port X is open`)
- **Ping operations**

---

### File Operations

#### `parser_file.go` - 295 lines
File system operations:
- **File creation** (`create file`, `create directory`)
- **File manipulation** (`copy`, `move`, `delete`)
- **File I/O** (`read`, `write`, `append`)
- **File checks** (`check if file exists`)

---

### Advanced Features

#### `parser_detection.go` - 212 lines
Smart detection operations:
- **Tool detection** (`detect docker`, `detect kubernetes`)
- **Environment detection** (`detect os`, `detect platform`)
- **Context-aware parsing**

#### `parser_error.go` - 111 lines
Error handling constructs:
- **Try-catch statements** (`try:`, `catch:`)
- **Throw statements** (`throw "error"`)
- **Error recovery**

---

### Utilities

#### `parser_helpers.go` - 461 lines
Helper functions and utilities:
- **Token type checking** (`isDockerToken`, `isGitToken`, etc.)
- **Token expectation** (`expectPeek`, `expectPeekSkipNewlines`)
- **Error management** (`addError`, `peekError`)
- **Parsing utilities** (`parseStringList`, `parseConditionExpression`)
- **Pattern matching** (`isPortCheckPattern`, `isDetectionContext`)

---

## Statistics

- **Total lines:** 5,466
- **Domain files:** 16
- **Main parser:** 115 lines (97.6% reduction from original 4,874 lines)
- **Test coverage:** 58/58 examples passing

## Design Principles

1. **Single Responsibility:** Each file handles one domain
2. **Same Package:** All files in `parser` package to avoid circular dependencies
3. **Method Organization:** Methods grouped by parsing domain, not by type
4. **Clear Naming:** File names indicate their domain (`parser_<domain>.go`)
5. **Minimal Core:** `parser.go` contains only orchestration logic

## Usage

All parser files are part of the same `parser` package, so they can call each other's methods freely:

```go
// In parser_task.go
func (p *Parser) parseTaskStatement() *ast.TaskStatement {
    // Can call methods from parser_control.go
    controlFlow := p.parseControlFlowStatement()
    
    // Can call methods from parser_helpers.go
    if p.isActionToken(p.curToken.Type) {
        // ...
    }
}
```

## Future Enhancements

- Consider extracting common patterns into shared helpers
- Add inline documentation for complex parsing logic
- Create parser benchmarks for performance testing

