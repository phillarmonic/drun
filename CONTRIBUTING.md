# Contributing to drun

Thank you for your interest in contributing to drun! This guide will help you get started with contributing code, documentation, or bug reports.

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Setup](#development-setup)
3. [Code Organization](#code-organization)
4. [Making Changes](#making-changes)
5. [Testing](#testing)
6. [Code Style](#code-style)
7. [Submitting Changes](#submitting-changes)
8. [Adding New Features](#adding-new-features)
9. [Documentation](#documentation)
10. [Getting Help](#getting-help)

---

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Basic understanding of compilers/interpreters (helpful but not required)

### First Steps

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/drun.git`
3. Read the [Architecture Guide](./ARCHITECTURE.md) to understand the system
4. Read the [Developer Guide](./DEVELOPER_GUIDE.md) for detailed codebase documentation
5. Browse the [examples/](./examples/) directory to see drun in action

---

## Development Setup

### Build from Source

```bash
cd drun
go build -o xdrun ./cmd/drun
```

### Run Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/parser/...

# With coverage
go test -cover ./...

# Run example files (regression tests)
./scripts/test.sh
```

### Install Locally

```bash
# Install to $GOPATH/bin
go install ./cmd/drun

# Or use the build script
./scripts/build.sh
```

---

## Code Organization

drun follows a clean, layered architecture:

```
cmd/drun/              # CLI entry point
├── main.go            # Main entry (440 lines)
└── app/               # CLI modules
    ├── update.go      # Self-update logic
    ├── config.go      # Configuration management
    ├── completion.go  # Shell completion
    └── runner.go      # Task runner

internal/
├── ast/               # Abstract Syntax Tree (15 files)
├── parser/            # Syntax parser (26 files)
├── engine/            # Execution engine (36 files)
│   ├── interpolation/ # Variable interpolation
│   ├── hooks/         # Lifecycle hooks
│   └── includes/      # Include resolution
├── lexer/             # Tokenization (6 files)
└── (support packages) # builtins, shell, detection, etc.
```

See [internal/README.md](./internal/README.md) for detailed package documentation.

---

## Making Changes

### Branching Strategy

1. Create a feature branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes with clear, focused commits

3. Keep your branch up to date:
   ```bash
   git fetch origin
   git rebase origin/main
   ```

### Commit Messages

Write clear commit messages:

```
Add HTTP timeout configuration

- Add timeout option to HTTP statements
- Default to 30 seconds
- Add tests for timeout behavior
```

**Format:**
- First line: Short summary (50 chars or less)
- Blank line
- Detailed description if needed
- Reference issues: `Fixes #123` or `Related to #456`

---

## Testing

### Test Requirements

All contributions must include tests:

1. **Unit tests** for new functions/methods
2. **Integration tests** for new features
3. **Example files** for new language constructs

### Writing Tests

#### Unit Test Example

```go
// internal/parser/parser_docker_test.go
func TestParseDockerBuild(t *testing.T) {
    input := `build docker image "myapp:latest"`
    l := lexer.New(input)
    p := New(l)
    
    stmt, err := p.parseDockerStatement()
    if err != nil {
        t.Fatalf("parser error: %v", err)
    }
    
    if stmt.Action != "build" {
        t.Errorf("expected action 'build', got '%s'", stmt.Action)
    }
}
```

#### Integration Test Example

```go
// internal/engine/docker_test.go
func TestDockerBuildExecution(t *testing.T) {
    source := `
    version: 2.0
    task "test":
      build docker image "test:latest"
    `
    
    eng := NewEngine(os.Stdout)
    program, _ := ParseString(source)
    eng.LoadProject(program)
    
    err := eng.RunTask("test", nil)
    // Assert results...
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/parser/

# Run with verbose output
go test -v ./internal/engine/

# Run with coverage
go test -cover ./...

# Run regression tests (all examples)
./scripts/test.sh
```

---

## Code Style

### File Organization

**Keep files small and focused:**
- AST definitions: 100-200 lines
- Parsers: 200-300 lines  
- Executors: 150-250 lines
- Helpers: 100-200 lines
- Maximum: 500 lines per file

**Group by domain, not layer:**

Good:
```
ast_docker.go
parser_docker.go
executor_docker.go
```

Bad:
```
all_ast_types.go (5000 lines)
```

### Naming Conventions

```go
// Exported (public) - PascalCase
type Engine struct { ... }
func NewEngine() *Engine { ... }
func (e *Engine) RunTask(name string) error { ... }

// Unexported (private) - camelCase
func (e *Engine) executeStatement(stmt ast.Statement) error { ... }
func (e *Engine) interpolateVariables(s string) string { ... }

// Constants - PascalCase for exported, camelCase for private
const DefaultTimeout = 30
const maxRetries = 3
```

### Error Handling

Always provide context in errors:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to execute task '%s': %w", taskName, err)
}

// Bad
if err != nil {
    return err
}
```

### Documentation

Document all exported types and functions:

```go
// NewEngine creates a new execution engine.
// The output writer receives all command output and status messages.
func NewEngine(output io.Writer) *Engine {
    // ...
}
```

### Code Formatting

- Use `go fmt` before committing
- Use `go vet` to catch common issues
- Follow standard Go conventions
- Keep functions focused and simple

---

## Submitting Changes

### Pull Request Process

1. **Ensure all tests pass:**
   ```bash
   go test ./...
   ./scripts/test.sh
   ```

2. **Update documentation:**
   - Add/update comments for public APIs
   - Update README.md if adding user-facing features
   - Add example files for new language features

3. **Create pull request:**
   - Write a clear title and description
   - Reference any related issues
   - Explain what changed and why
   - Include examples of new functionality

4. **Pull request checklist:**
   - [ ] Tests pass locally
   - [ ] All examples work
   - [ ] Code follows style guidelines
   - [ ] Documentation updated
   - [ ] Commit messages are clear

### Code Review

- Address review feedback promptly
- Be open to suggestions
- Ask questions if something is unclear
- Update your branch as requested

---

## Adding New Features

### Adding a New Action Type

Example: Adding `notify slack "message"` support

#### 1. Define AST Node

Create `internal/ast/ast_slack.go`:

```go
package ast

// SlackStatement represents a Slack notification action
type SlackStatement struct {
    Action  string // "notify"
    Channel string
    Message string
}

func (s *SlackStatement) statementNode() {}
```

#### 2. Define Domain Statement

Create `internal/domain/statement/slack.go` (or add to `statement.go`):

```go
package statement

// Slack represents a Slack notification action at the domain level
type Slack struct {
    Action  string
    Channel string
    Message string
}

func (s *Slack) Type() StatementType { return "slack" }
```

#### 3. Add Domain Converter

Add to `internal/domain/statement/converter.go`:

```go
// In FromAST function
case *ast.SlackStatement:
    return &Slack{
        Action:  s.Action,
        Channel: s.Channel,
        Message: s.Message,
    }, nil

// In ToAST function (if needed for execution bridge)
case *Slack:
    return &ast.SlackStatement{
        Action:  s.Action,
        Channel: s.Channel,
        Message: s.Message,
    }, nil
```

#### 4. Add Parser

Create `internal/parser/parser_slack.go`:

```go
package parser

import "github.com/phillarmonic/drun/internal/ast"

func (p *Parser) parseSlackStatement() (*ast.SlackStatement, error) {
    stmt := &ast.SlackStatement{}
    
    // Consume "notify"
    stmt.Action = p.curToken.Literal
    
    if !p.expectPeek(IDENT) || p.curToken.Literal != "slack" {
        return nil, p.error("expected 'slack' after 'notify'")
    }
    
    // Parse channel and message...
    
    return stmt, nil
}
```

Wire it up in `parser_action.go`:

```go
case "notify":
    if p.peekTokenIs(IDENT) && p.peekToken.Literal == "slack" {
        return p.parseSlackStatement()
    }
```

#### 5. Add Executor

Create `internal/engine/executor_slack.go`:

```go
package engine

import "github.com/phillarmonic/drun/internal/ast"

func (e *Engine) executeSlack(stmt *ast.SlackStatement, ctx *ExecutionContext) error {
    // Interpolate variables
    message := e.interpolateVariables(stmt.Message, ctx)
    
    // Send to Slack...
    
    return nil
}
```

Wire it up in `executeDomainStatement` in `engine.go`:

```go
case *statement.Slack:
    return e.executeSlack(&ast.SlackStatement{
        Action:  s.Action,
        Channel: s.Channel,
        Message: s.Message,
    }, ctx)
```

#### 6. Add Tests

Create tests in:
- `internal/parser/parser_slack_test.go` - Parser tests
- `internal/domain/statement/slack_test.go` - Domain converter tests
- `internal/engine/executor_slack_test.go` - Executor tests

#### 7. Add Example

Create `examples/XX-slack-notifications.drun`:

```drun
version: 2.0

task "deploy":
  step "Deploying application"
  notify slack "Deployment started"
  run "deploy.sh"
  notify slack "Deployment complete"
```

### Adding New Built-in Functions

Add to `internal/builtins/builtins.go`:

```go
func YourFunction(args ...string) (string, error) {
    // Validate arguments
    if len(args) < 1 {
        return "", fmt.Errorf("yourFunction requires at least 1 argument")
    }
    
    // Implementation
    result := doSomething(args[0])
    
    return result, nil
}
```

Register in the builtins map and add tests.

---

## Documentation

### Code Documentation

- Document all exported types and functions
- Explain complex algorithms or logic
- Include examples in comments when helpful

### User Documentation

When adding user-facing features:

1. Update [README.md](./README.md)
2. Update [DRUN_V2_SPECIFICATION.md](./DRUN_V2_SPECIFICATION.md)
3. Add examples to [examples/](./examples/)
4. Update [ROADMAP.md](./ROADMAP.md) status

### Developer Documentation

When changing architecture:

1. Update [ARCHITECTURE.md](./ARCHITECTURE.md)
2. Update [internal/README.md](./internal/README.md)
3. Update this contributing guide if needed

---

## Getting Help

### Resources

- **Architecture Guide:** [ARCHITECTURE.md](./ARCHITECTURE.md)
- **Developer Guide:** [DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md)
- **Package Guide:** [internal/README.md](./internal/README.md)
- **Language Spec:** [DRUN_V2_SPECIFICATION.md](./DRUN_V2_SPECIFICATION.md)

### Questions?

- Open a discussion on GitHub
- Check existing issues for similar questions
- Read through the examples directory

### Found a Bug?

1. Check if it's already reported
2. Create a new issue with:
   - Description of the bug
   - Steps to reproduce
   - Expected vs actual behavior
   - drun version and OS
   - Minimal example file if applicable

---

## Development Workflow Example

Here's a typical workflow for adding a feature:

```bash
# 1. Create branch
git checkout -b feature/add-email-notifications

# 2. Make changes
# - Add AST node in internal/ast/ast_email.go
# - Add parser in internal/parser/parser_email.go
# - Add executor in internal/engine/executor_email.go
# - Add tests

# 3. Test locally
go test ./...
./scripts/test.sh

# 4. Add example
echo 'version: 2.0
task "test":
  send email to "user@example.com" subject "Test"
' > examples/XX-email-test.drun

xdrun -f examples/XX-email-test.drun test

# 5. Update documentation
# - Edit README.md
# - Edit DRUN_V2_SPECIFICATION.md

# 6. Commit
git add .
git commit -m "Add email notification support

- Add email statement to AST
- Implement email parser
- Implement email executor
- Add tests and example
- Update documentation"

# 7. Push and create PR
git push origin feature/add-email-notifications
```

---

## Architecture Principles

Keep these principles in mind when contributing:

### 1. Single Responsibility

Each file/package should have one clear purpose.

### 2. Separation of Concerns

- Lexer: Tokenization only
- Parser: Syntax analysis only
- Engine: Execution only
- Each executor: One statement type only

### 3. Dependency Direction

```
CLI → Engine → Parser → Lexer → AST
      ↓
  Support Packages
```

Higher-level components depend on lower-level ones, not vice versa.

### 4. Testability

Every component should be testable in isolation.

### 5. Clarity Over Cleverness

Prefer clear, simple code over clever optimizations.

---

## Code of Conduct

- Be respectful and constructive
- Welcome newcomers
- Focus on the code, not the person
- Assume good intentions
- Help others learn and grow

---

## License

By contributing to drun, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to drun!
