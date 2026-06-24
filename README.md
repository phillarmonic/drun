# drun

`drun` is a semantic task automation language for readable project workflows.

`xdrun` is the CLI that executes `.drun` task files. It is designed for automation that reads closer to intent than shell glue, while still running the commands your project needs.

## Install

Use the install script:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
```

Install a specific version:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s v2.10.0
```

Install with Go:

```bash
go install github.com/phillarmonic/drun/v2/cmd/xdrun@latest
```

Install a specific tagged version with Go:

```bash
go install github.com/phillarmonic/drun/v2/cmd/xdrun@v2.17.0
```

The installer detects platform and architecture, installs `xdrun` to `$HOME/.local/bin` by default on Unix systems, and attempts to make it available on your `PATH`.

## Quick Start

Create `.drun/spec.drun`:

```drun
version: 2.0

task "hello" means "Say hello":
  info "Hello from drun"
```

Or run this command to initialize a Drun spec file in the current directory (a folder named .drun will be created). The drun demo file also includes a task named hello.

```bash
xdrun --init
```

Then run it:

```bash
xdrun hello
```

List tasks:

```bash
xdrun --list
```

Install AI guidance files for a repository that uses drun:

```bash
xdrun cmd:skill install drun-basics
```

Dry run a task:

```bash
xdrun hello --dry-run
```

Task parameters use `key=value` syntax:

```bash
xdrun deploy environment=production version=v1.2.3
```

## Basic Use Cases

- Define build, test, and release workflows in a readable DSL.
- Replace ad hoc shell scripts or large Makefiles with named tasks.
- Add validation and defaults to task parameters.
- Share deployment and environment operations in a form non-authors can still review.
- Manage multi-service local stacks and orchestration flows.

## Highlights

- English-like task syntax.
- `key=value` task parameters with CLI flags kept separate.
- Built-in validation, defaults, and control flow.
- Dry-run support for inspecting execution.
- Task modes, including CI buffering and one-run overrides via `--task-mode`.
- Optional `run ... attached` mode for REPL-style commands that need stdin and a terminal.
- Reusable task files and examples for common workflows.
- Orchestration support for multi-service projects.

## Interactive Commands

Use `attached` when a `run` command must stay connected to your terminal, such as REPLs or tools that prompt for input:

```drun
task "repl":
  run "go run command" attached
```

Plain `run "command"` remains non-interactive and is still the default for ordinary automation steps.

## Learn More

- Usage and troubleshooting: [docs/USAGE_AND_TROUBLESHOOTING.md](./docs/USAGE_AND_TROUBLESHOOTING.md)
- Language reference: [DRUN_V2_SPECIFICATION.md](./DRUN_V2_SPECIFICATION.md)
- Examples: [examples/README.md](./examples/README.md)
- Orchestration: [docs/ORCHESTRATION.md](./docs/ORCHESTRATION.md)
- Stateless and partial names: [docs/STATELESS_AND_PARTIAL_NAMES.md](./docs/STATELESS_AND_PARTIAL_NAMES.md)
- Developer documentation: [DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md)
- Architecture: [ARCHITECTURE.md](./ARCHITECTURE.md)
