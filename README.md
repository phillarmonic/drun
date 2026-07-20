# drun automation language (with AI support)

<p align="center">
  <img src="images/drun_500_transp.png" width="500" alt="Drun" />
</p>

`drun` is a semantic task automation language for readable project workflows, with AI skills support.

`xdrun` is the CLI that executes `.drun` task files. It is designed for automation that reads closer to intent than shell glue, while still running the commands your project needs.

Full documentation lives at [phillarmonic.github.io/drun](https://phillarmonic.github.io/drun/), including the [AI integration guide](https://phillarmonic.github.io/drun/getting-started/ai-integration/).

Official editor extensions are available for [VS Code](https://marketplace.visualstudio.com/items?itemName=phillarmonic.drun-language-support), [Open VSX](https://open-vsx.org/extension/phillarmonic/drun-language-support), and [JetBrains IDEs](https://plugins.jetbrains.com/plugin/32865-drun-language-support).

## Why drun?

- Define build, test, release, and operations workflows in a readable DSL.
- Replace ad hoc shell scripts or large Makefiles with named tasks.
- Add validation, defaults, and tool checks to task parameters and project workflows.
- Share automation in a form non-authors can still review.
- Keep AI guidance close to the automation with project skills.

## Install

Use the install script:

```bash
curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
```

Or install with Go:

```bash
go install github.com/phillarmonic/drun/v2/cmd/xdrun@latest
```

See the [install guide](./docs/docs/getting-started/install.md) for platform notes, pinned versions, and verification.

## Quick Start

Create `.drun/spec.drun`:

```drun
version: 2.0

task "hello" means "Say hello":
  info "Hello from drun"
```

Then run it:

```bash
xdrun hello
```

You can also initialize a starter spec:

```bash
xdrun --init
```

Install AI guidance files for a repository that uses drun:

```bash
xdrun cmd:skill install drun-basics
```

See [getting started](./docs/docs/getting-started/index.md) for initialization options, templates, task parameters, dry runs, and shell autocomplete.

## Editor Support

**Using VS Code?** Install the official extension with LSP support from the [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=phillarmonic.drun-language-support).

[![VS Code Marketplace](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fraw.githubusercontent.com%2Fphillarmonic%2Fdrun-vscode%2Fmaster%2Fpackage.json&query=%24.version&prefix=v&label=VS%20Code%20Marketplace&logo=visualstudiocode&color=007ACC)](https://marketplace.visualstudio.com/items?itemName=phillarmonic.drun-language-support)

**Using Cursor, Antigravity, or another VS Code fork?** Install the official extension from the [Open VSX Registry](https://open-vsx.org/extension/phillarmonic/drun-language-support).

[![Open VSX](https://img.shields.io/open-vsx/v/phillarmonic/drun-language-support?label=Open%20VSX&logo=eclipseide)](https://open-vsx.org/extension/phillarmonic/drun-language-support)

**Using a JetBrains IDE?** Install the official plugin from the [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/32865-drun-language-support).

[![JetBrains Marketplace](https://img.shields.io/jetbrains/plugin/v/32865?label=JetBrains%20Marketplace&logo=jetbrains)](https://plugins.jetbrains.com/plugin/32865-drun-language-support)

## Highlights

- English-like task syntax.
- Context-aware shell completion for tasks and declared `key=value` parameter names.
- `key=value` task parameters with CLI flags kept separate.
- Built-in validation, defaults, and control flow.
- Dry-run support for inspecting execution.
- Task modes, including CI buffering and one-run overrides via `--task-mode`.
- Optional `run ... attached` mode for REPL-style commands that need stdin and a terminal.
- Reusable task files and examples for common workflows.
- Orchestration support for multi-service projects.

## Learn More

- Getting started: [installation, initialization, running tasks, and templates](./docs/docs/getting-started/index.md)
- AI integration: [install guidance files for AI agents](./docs/docs/getting-started/ai-integration.md)
- Troubleshooting: [common installation and usage problems](./docs/docs/getting-started/troubleshooting.md)
- Language reference: [language specification](./docs/docs/reference/language/overview.md)
- Runtime and tool requirements: [detection, execution, and errors](./docs/docs/reference/runtime/detection-execution-and-errors.md)
- Git policy hooks: [branch and commit policy enforcement](./docs/docs/reference/language/git-policy.md)
- Examples: [examples](./docs/docs/examples/index.md)
- Orchestration: [orchestration guide](./docs/docs/guides/orchestration.md)
- Stateless and partial names: [stateless mode and partial names](./docs/docs/guides/stateless-and-partial-names.md)
- Developer documentation: [developer guide](./docs/docs/development/index.md)
- Architecture: [architecture](./docs/docs/development/architecture.md)
