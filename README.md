# drun (do run)

A **high-performance** YAML-based task runner with first-class positional arguments, powerful templating, and intelligent dependency management. Optimized for speed with microsecond-level operations and minimal memory usage.


## Requirements

- **Go 1.25+** - drun requires Go 1.25 or later

## Features

- **YAML Configuration**: Define tasks in a simple, readable YAML format
- **Positional Arguments**: First-class support for positional arguments with validation
- **Named Arguments**: Pass positional arguments by name for clarity (`--name=value` or `name=value`)
- **Templating**: Powerful Go template engine with custom functions and caching
- **Dependencies**: Automatic dependency resolution and parallel execution
- **High Performance**: Microsecond-level operations with intelligent caching
- **Cross-Platform**: Works on Linux, macOS, and Windows with appropriate shell selection
- **Dry Run & Explain**: See what would be executed without running it
- **Recipe Flags**: Command-line flags specific to individual recipes

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/phillarmonic/drun/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| **Linux** | x86_64 | `drun-linux-amd64` (UPX compressed) |
| **Linux** | ARM64 | `drun-linux-arm64` (UPX compressed) |
| **macOS** | Intel | `drun-darwin-amd64` |
| **macOS** | Apple Silicon | `drun-darwin-arm64` |
| **Windows** | x86_64 | `drun-windows-amd64.exe` (UPX compressed) |
| **Windows** | ARM64 | `drun-windows-arm64.exe` |

All binaries are **statically linked** and have **no dependencies**.

### Install Script

```bash
# Install latest version (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash

# Install specific version
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s v1.0.0
```

### Build from Source

## Quick Start

1. **Build drun**:
   ```bash
   go build -o bin/drun ./cmd/drun
   ```

## Testing

Run the comprehensive test suite (includes mandatory golangci-lint):

```bash
# Basic tests (includes linting, unit tests, build verification)
./test.sh

# With coverage report
./test.sh -c

# Verbose with race detection
./test.sh -v -r

# All options
./test.sh -v -c -r -b
```

Or run components manually:

```bash
# Linting (required - auto-installs golangci-lint if needed)
golangci-lint run ./...

# Unit tests only
go test ./internal/...

# With coverage
go test -cover ./internal/...

# CI-optimized test suite
./test-ci.sh
```

2. **Initialize a new project**:
   ```bash
   ./bin/drun --init
   ```

3. **List available recipes**:
   ```bash
   ./bin/drun --list
   ```

4. **Run a recipe**:
   ```bash
   ./bin/drun build
   ```

5. **Use positional arguments**:
   ```bash
   ./bin/drun release v1.0.0 amd64
   ```

6. **Use named arguments for clarity**:
   ```bash
   # Flag-style named arguments
   ./bin/drun release --version=v1.0.0 --arch=amd64
   
   # Assignment-style named arguments
   ./bin/drun release version=v1.0.0 arch=amd64
   
   # Mix positional and named
   ./bin/drun release v1.0.0 --arch=amd64
   ```

7. **Dry run to see what would execute**:
   ```bash
   ./bin/drun build --dry-run
   ```

8. **Run performance benchmarks**:
   ```bash
   ./test.sh -b
   ```

## Performance

drun is engineered for **high performance** and **low resource usage**. Extensive optimizations ensure fast execution even for large projects with complex dependency graphs.

### Benchmarks

Performance benchmarks on Apple M4 (your results may vary):

| Component | Operation | Time | Memory | Allocations |
|-----------|-----------|------|--------|-------------|
| **YAML Loading** | Simple spec | 2.5μs | 704 B | 5 allocs |
| **YAML Loading** | Large spec (100 recipes) | 8.6μs | 756 B | 5 allocs |
| **Template Rendering** | Basic template | 29μs | 3.9 KB | 113 allocs |
| **Template Rendering** | Complex template | 51μs | 7.0 KB | 93 allocs |
| **DAG Building** | Simple dependency graph | 3.1μs | 10.7 KB | 109 allocs |
| **DAG Building** | Complex dependencies | 3.9μs | 12.4 KB | 123 allocs |
| **Topological Sort** | 100 nodes | 2.5μs | 8.0 KB | 137 allocs |

### Optimization Impact

Our performance optimizations deliver significant improvements:

| Component | Before | After | **Improvement** |
|-----------|--------|-------|-----------------|
| **Template Rendering** | 40μs, 60KB | **29μs, 4KB** | **1.4x faster, 15x less memory** |
| **YAML Loading** | 361μs, 42KB | **2.5μs, 704B** | **144x faster, 59x less memory** |
| **Large Spec Loading** | 3.4ms, 657KB | **8.6μs, 756B** | **396x faster, 869x less memory** |
| **DAG Building** | 4.4μs, 14KB | **3.1μs, 11KB** | **1.4x faster, 22% less memory** |
| **Topological Sort** | 4.7μs, 10KB | **2.5μs, 8KB** | **1.9x faster, 20% less memory** |

### Performance Features

- **⚡ Template Caching**: Compiled templates cached by hash for instant reuse
- **🧠 Smart Pre-allocation**: Memory pools and capacity-aware data structures
- **📊 Spec Caching**: YAML specs cached with file modification tracking
- **🔄 Optimized DAG**: Highly efficient dependency graph construction
- **💾 Memory Pools**: Reusable objects reduce GC pressure
- **🎯 Lazy Evaluation**: Only compute what's needed when needed

### Real-World Performance

- **Startup time**: Sub-millisecond for cached specs
- **Large projects**: 100+ recipes process in microseconds
- **Memory usage**: Minimal footprint with intelligent caching
- **Parallel execution**: Efficient DAG-based task scheduling
- **Template rendering**: Up to 20x faster than naive implementations

Run benchmarks yourself:
```bash
./test.sh -b  # Includes comprehensive performance benchmarks
```

## Configuration

drun automatically looks for configuration files in this order:
- `drun.yml`
- `drun.yaml` 
- `.drun.yml`
- `.drun.yaml`
- `.drun/drun.yml`
- `.drun/drun.yaml`
- `ops.drun.yml`
- `ops.drun.yaml`

Use `drun --init` to create a starter configuration, or see the included examples for comprehensive configurations.

### Basic Recipe

```yaml
version: 0.1

recipes:
  hello:
    help: "Say hello"
    run: |
      echo "Hello, World!"
```

### Recipe with Positional Arguments

```yaml
recipes:
  greet:
    help: "Greet someone"
    positionals:
      - name: name
        required: true
      - name: title
        default: "friend"
    run: |
      echo "Hello, {{ .title }} {{ .name }}!"
```

**Usage examples:**
```bash
# Traditional positional arguments
drun greet Alice
drun greet Bob Mr.

# Named arguments (flag-style)
drun greet --name=Alice --title=Ms.

# Named arguments (assignment-style)  
drun greet name=Bob title=Dr.

# Mixed usage
drun greet Alice --title=Ms.
```

### Advanced Named Arguments

```yaml
recipes:
  deploy:
    help: "Deploy to environment with version"
    positionals:
      - name: environment
        required: true
        one_of: ["dev", "staging", "prod"]
      - name: version
        default: "latest"
      - name: features
        variadic: true
    flags:
      force:
        type: bool
        default: false
    run: |
      echo "Deploying {{ .version }} to {{ .environment }}"
      {{ if .features }}echo "Features: {{ range .features }}{{ . }} {{ end }}"{{ end }}
      {{ if .force }}echo "Force deployment enabled"{{ end }}
```

**Usage examples:**
```bash
# All positional
drun deploy prod v1.2.3 feature1 feature2 --force

# All named arguments
drun deploy --environment=prod --version=v1.2.3 --force

# Mixed style
drun deploy prod --version=v1.2.3 --force

# Assignment style with variadic
drun deploy environment=staging version=v1.1.0 features=auth,ui --force
```

### Recipe with Dependencies

```yaml
recipes:
  test:
    help: "Run tests"
    deps: [build]
    run: |
      go test ./...
      
  build:
    help: "Build the project"
    run: |
      go build ./...
```

## Command Line Options

- `--init`: Initialize a new drun.yml configuration file
- `--list, -l`: List available recipes
- `--dry-run`: Show what would be executed without running
- `--explain`: Show rendered scripts and environment variables
- `--update`: Update drun to the latest version from GitHub releases
- `--file, -f`: Specify configuration file (default: auto-discover)
- `--jobs, -j`: Number of parallel jobs for dependencies
- `--set`: Set variables (KEY=VALUE format)
- `--shell`: Override shell type (linux/darwin/windows)

## Template Functions

drun includes many built-in template functions:

- `{{ now "2006-01-02" }}`: Current time formatting
- `{{ .version }}`: Access positional arguments and variables
- `{{ env "HOME" }}`: Environment variables
- `{{ snippet "name" }}`: Include reusable snippets
- `{{ shellquote .arg }}`: Shell-safe quoting
- Plus all [Sprig](https://masterminds.github.io/sprig/) functions

## Examples

See the included `drun.yml` for a comprehensive example showing:
- Positional arguments with validation
- Conditional templating
- Environment variable templating
- Dependency management
- Cross-platform shell commands

## Status

This is a feature-complete implementation with robust functionality. Recent additions include:
- ✅ **Named Arguments**: Pass positional arguments by name using `--name=value` or `name=value` syntax
- ✅ **Recipe-specific command-line flags**: Define custom flags per recipe
- ✅ **High-performance optimizations**: Microsecond-level operations with intelligent caching

Future enhancements may include:
- File watching and auto-execution
- Remote includes and caching
- Matrix builds
- Plugin system
