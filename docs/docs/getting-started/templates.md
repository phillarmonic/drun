# Project templates

Project templates create a ready-to-edit drun spec for common frameworks and toolchains. The official catalog lives at [`phillarmonic/drun-templates`](https://github.com/phillarmonic/drun-templates) and is configured in `xdrun` automatically, so normal usage does not require cloning the repository or passing a catalog URL.

## List available templates

```bash
xdrun --list-templates
```

The official catalog includes starters for Go CLI projects, Node.js, pnpm, Bun, Deno, Python with `venv` or `uv`, Rust with Cargo, Java with Gradle or Maven, and Docker-backed PHP applications.

## Initialize from a template

Choose a name from the list and pass it to `--template`:

```bash
xdrun --init --template go-cli
```

For example:

```bash
xdrun --init --template node-app
xdrun --init --template python-uv
xdrun --init --template rust-cargo
```

The template is fetched from the official catalog, rendered with project-specific values, validated as drun syntax, and written to `.drun/spec.drun`.

## Use another catalog

Use `--from-template` for a remote manifest or local repository:

```bash
xdrun --list-templates --from-template /path/to/drun-templates
xdrun --init --from-template /path/to/drun-templates --template go-cli
```

When the path is a directory, `xdrun` reads `templates.yaml` from its root. Remote manifests can use `github:`, `drunhub:`, or `https:` references:

```bash
xdrun --init \
  --from-template github:owner/repo/templates.yaml@main \
  --template project-starter
```

See the [official template repository](https://github.com/phillarmonic/drun-templates) for the current catalog and template source.
