# drun documentation

drun is a readable automation language executed by the `xdrun` CLI. This site is the canonical home for its user guides, technical language specification, examples, and contributor documentation.

Install the official drun language support extension from your editor's marketplace:

[![VS Code Marketplace](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fraw.githubusercontent.com%2Fphillarmonic%2Fdrun-vscode%2Fmaster%2Fpackage.json&query=%24.version&prefix=v&label=VS%20Code%20Marketplace&logo=visualstudiocode&color=007ACC)](https://marketplace.visualstudio.com/items?itemName=phillarmonic.drun-language-support)
[![Open VSX](https://img.shields.io/open-vsx/v/phillarmonic/drun-language-support?label=Open%20VSX&logo=eclipseide)](https://open-vsx.org/extension/phillarmonic/drun-language-support)

We support **Visual Studio Code**, and any fork that has access to the OpenVSX store, like **Cursor**, **Antigravity**, etc.

**Using a JetBrains IDE**? Install the official drun language support extension directly from the JetBrains Marketplace:

<iframe
  title="Install drun language support"
  width="245"
  height="48"
  frameborder="0"
  src="https://plugins.jetbrains.com/embeddable/install/32865">
</iframe>

## Choose your path

- **New to drun?** Follow the [getting started guide](getting-started/index.md), then browse the [examples](examples/index.md).
- **Writing Drunfiles?** Use the [language specification](reference/language/overview.md) and [built-in actions](reference/language/built-in-actions.md).
- **AI Integration?** Check out how to [use the drun skills on your AI agent](https://phillarmonic.github.io/drun/getting-started/ai-integration/).
- **Orchestrating services?** Follow the [orchestration guide](guides/orchestration.md), with the normative details in the [orchestration specification](reference/orchestration.md).
- **Contributing?** Read the [developer guide](development/index.md), [architecture guide](development/architecture.md), and [contribution workflow](development/contributing.md).

## Documentation structure

| Area                   | Purpose                                                      |
| ---------------------- | ------------------------------------------------------------ |
| Getting started        | Installation, CLI usage, configuration, and troubleshooting  |
| Language specification | Normative syntax, semantics, runtime behavior, and built-ins |
| Guides                 | Task-oriented workflows and feature walkthroughs             |
| Examples               | Complete Drunfiles and usage patterns                        |
| Development            | Architecture, packages, testing, and contribution guidance   |

The language reference deliberately keeps its technical specification style. It is split by concern so each page is searchable, linkable, and small enough to navigate comfortably.
