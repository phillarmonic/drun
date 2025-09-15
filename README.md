# drun (do run)

A YAML-based task runner with first-class positional arguments, with a powerful templating and dependency management.

## Requirements

- **Go 1.25+** - drun requires Go 1.25 or later

## Features

- **YAML Configuration**: Define tasks in a simple, readable YAML format
- **Positional Arguments**: First-class support for positional arguments with validation
- **Templating**: Powerful Go template engine with custom functions
- **Dependencies**: Automatic dependency resolution and execution
- **Cross-Platform**: Works on Linux, macOS, and Windows with appropriate shell selection
- **Dry Run & Explain**: See what would be executed without running it

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

6. **Dry run to see what would execute**:
   ```bash
   ./bin/drun build --dry-run
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
    run: |
      echo "Hello, {{ .name }}!"
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

This is an MVP implementation with core functionality working. Future enhancements may include:
- Recipe-specific command-line flags
- File watching and auto-execution
- Remote includes and caching
- Matrix builds
- Plugin system
