# Implementation notes

## Implementation Notes

### Architecture Overview

drun parses its semantic language directly into an abstract syntax tree and executes it through the engine. The active implementation is organized under `internal/lexer`, `internal/parser`, `internal/ast`, and `internal/engine`.

### Engine Components

1. **Lexer** (`internal/lexer/`): Tokenizes the semantic language.
2. **Parser** (`internal/parser/`): Builds the abstract syntax tree.
3. **AST** (`internal/ast/`): Defines language node structures.
4. **Engine** (`internal/engine/`): Plans and executes parsed tasks directly.
5. **Runtime services**: Provide built-in actions, detection, interpolation, and shell integration.

### Domain Separation

Each component is organized into its own domain package:

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
    parser *parser.Parser
    engine *engine.Engine
}

func (e *ExecutionEngine) Execute(source string, args []string) error {
    project, err := e.parser.Parse(source)
    if err != nil {
        return err
    }

    return e.engine.Execute(project, args)
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

Drun exposes a simple stdio LSP entrypoint through the CLI:

```bash
xdrun cmd:lsp
```

The current implementation is intentionally small and focused on editor essentials:

- `initialize`, `shutdown`, and `exit`
- Full text-document sync
- Parser-backed diagnostics
- Simple keyword and task-name completions

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

### Implementation And Validation Contract

When drun adds or changes language behavior, contributors should treat the following as part of the feature:

- Update this specification in the same change whenever syntax, semantics, or normative examples change.
- Add focused parser, domain, and engine tests that cover the new behavior, not just manual verification.
- Update `.drun/spec.drun` when the feature affects repository-local workflows so the project continues to exercise its own language.
- Finish validation with `xdrun ci`, which is the project-level end-to-end check for the current local workflow.
