# Language overview

## Overview

drun v2 introduces a semantic, English-like domain-specific language (DSL) for defining automation tasks. It features a **completely new execution engine** that directly interprets and executes the semantic language without compilation to intermediate formats.

### Key Features

- **Natural Language Syntax**: Write automation in English-like sentences
- **Native Execution Engine**: Direct interpretation and execution of v2 syntax
- **Shell Backend**: All constructs execute as shell commands when needed
- **Smart Inference**: Automatic detection of tools, environments, and patterns
- **Type Safety**: Static analysis with runtime validation
- **Simple CLI**: Parameters use `key=value` syntax (no `--` dashes needed for task params)

### Architecture

drun v2 uses a **new execution engine** with the following components:

1. **Lexer**: Tokenizes the semantic language source code
2. **Parser**: Builds an Abstract Syntax Tree (AST) from tokens
3. **Engine**: Directly executes the AST without intermediate compilation
4. **Runtime**: Provides built-in actions, smart detection, and shell integration
5. **CLI (xdrun)**: Command-line interface that accepts tasks and parameters using `key=value` syntax

### CLI Usage Pattern

```bash
xdrun [task_name] [param1=value1] [param2=value2] [--cli-flags]
      └─ task     └─ task parameters (no dashes) ─┘ └─ xdrun flags ─┘
```

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

