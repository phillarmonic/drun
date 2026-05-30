# Usage and Troubleshooting

This guide collects practical user-facing information that does not belong in the top-level README: installation issues, file discovery, shell completion, self-update, and a few common built-in workflows.

## Installation Troubleshooting

### macOS: `signal: killed`

If `xdrun` fails on macOS with `signal: killed`, macOS Gatekeeper has likely quarantined the binary.

Fix it with:

```bash
sudo xattr -d com.apple.quarantine /usr/local/bin/xdrun
```

If you installed to a different location, use that path instead:

```bash
xattr -d com.apple.quarantine /path/to/xdrun
```

The `install.sh` script already tries to remove quarantine attributes automatically, but manually downloaded binaries or some update paths may still require this step.

### `xdrun` is installed but not found

The installer attempts to put `xdrun` on your `PATH`. If your shell still cannot find it:

1. Verify where it was installed.
2. Confirm that directory is on `PATH`.
3. Restart the shell or reload your shell profile.

Example:

```bash
echo $PATH
which xdrun
```

If needed, add the install directory manually:

```bash
export PATH="$PATH:/usr/local/bin"
```

## File Discovery and Configuration

`xdrun` looks for task files in this order:

1. Workspace default from `.drun/.drun_workspace`
2. Default file at `.drun/spec.drun`
3. Explicit file path passed with `--file`

Create a starter file:

```bash
xdrun --init
```

Create a custom file and save it as the workspace default:

```bash
xdrun --init --file=my-project.drun --save-as-default
```

Point the workspace to an existing file:

```bash
xdrun --set-workspace my-project.drun
```

Run a task from an explicit file:

```bash
xdrun --file examples/01-hello-world.drun hello
```

## Shell Completion

`xdrun` provides completion helpers for common shells through the built-in completion command.

Typical usage:

```bash
xdrun cmd:completion
```

If you want the generated completion script or shell-specific setup details, check the CLI help for the completion command in your environment.

## Self-Update

Update `xdrun` in place:

```bash
xdrun --self-update
```

The updater is designed to:

- Download the new version
- Replace the current binary
- Keep backups under `~/.drun/`
- Restore the previous version if the update fails

## Makefile Conversion

If you are migrating from Make, use the built-in converter:

```bash
xdrun cmd:from makefile --input Makefile --output tasks.drun
```

There is a fuller walkthrough in [examples/makefile-conversion/README.md](../examples/makefile-conversion/README.md).

## Secret Management CLI

`xdrun` includes a secret management command for storing and retrieving secrets outside task execution.

Common operations:

```bash
xdrun cmd:secret --help
```

Secrets are designed to work with project scoping and can be used from task interpolation as part of normal drun workflows.

## Built-in Command Convention

drun reserves the `cmd:` prefix for built-in commands so they do not collide with user-defined task names.

Examples:

```bash
xdrun cmd:completion
xdrun cmd:from makefile
xdrun cmd:secret
```

## Parameter and CLI Syntax

Task parameters use `key=value`.

```bash
xdrun deploy environment=production version=v1.2.3
```

CLI behavior flags still use `--flag` syntax.

```bash
xdrun deploy environment=production --dry-run
xdrun --list
```

## Where to Go Next

- Language reference: [../DRUN_V2_SPECIFICATION.md](../DRUN_V2_SPECIFICATION.md)
- Examples: [../examples/README.md](../examples/README.md)
- Orchestration: [./ORCHESTRATION.md](./ORCHESTRATION.md)
- Developer docs: [../DEVELOPER_GUIDE.md](../DEVELOPER_GUIDE.md)
