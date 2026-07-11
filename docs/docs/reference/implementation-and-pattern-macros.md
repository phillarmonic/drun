# Implementation notes and pattern macros

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
└── cmd/xdrun/
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


---

