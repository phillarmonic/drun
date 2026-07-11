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

To initialize the project spec under `infra/.drun/spec.drun` instead:

```bash
xdrun --init --file infra/.drun/spec.drun
```

Or under `infra/drun/spec.drun`:

```bash
xdrun --init --file ops/drun/spec.drun
```

Initialize from a local template repository:

```bash
xdrun --list-templates --templates-repo ../drun-templates
xdrun --init --template go-cli --templates-repo ../drun-templates
xdrun --init --from-template ../drun-templates --template go-cli
```

When `--from-template` points at a local directory, `xdrun` looks for `templates.yaml` at that directory root. This is useful when developing templates locally before publishing a remote manifest.

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

## Tool Provisioning

When a `requires tools:` entry opts into `provision`, `drun` resolves installers from provisioning catalogs in this order:

1. Project `provisioning sources:`
2. User `provisioningSources` from `~/.drun/config.yml`
3. The official first-party catalog at `github:phillarmonic/drun-provisionings/provisionings.yaml@master`
4. The embedded fallback catalog shipped with `drun`

The official catalog is implicit. You only need to declare `provisioning sources:` when you want to override or extend it.

Use `--allow-tool-version-changes` when a provisionable requirement is already installed but needs an upgrade or downgrade to satisfy the declared version:

```bash
xdrun --allow-tool-version-changes lint
```

Project example:

```drun
project "api":
  provisioning sources:
    "./.drun/provisionings.yaml"

  requires tools:
    golangci-lint >= "1.64" provision
    gosec >= "2.22" <= "2.22" provision
    govulncheck provision
```

See [examples/73-tool-provisioning.drun](./examples/73-tool-provisioning.drun) for a fuller example covering project overrides, the implicit first-party catalog, the embedded fallback, and exact-version requests.

## Learn More

- Usage and troubleshooting: [usage and troubleshooting](./docs/docs/getting-started/usage-and-troubleshooting.md)
- Language reference: [language specification](./docs/docs/reference/language/overview.md)
- Examples: [examples](./docs/docs/examples/index.md)
- Orchestration: [orchestration guide](./docs/docs/guides/orchestration.md)
- Stateless and partial names: [stateless mode and partial names](./docs/docs/guides/stateless-and-partial-names.md)
- Developer documentation: [developer guide](./docs/docs/development/index.md)
- Architecture: [architecture](./docs/docs/development/architecture.md)
