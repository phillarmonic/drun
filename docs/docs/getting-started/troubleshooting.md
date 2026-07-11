# Troubleshooting

## macOS reports `signal: killed`

macOS Gatekeeper may have quarantined a manually downloaded binary. Remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine ~/.local/bin/xdrun
```

If you installed elsewhere, replace the path with the location of your `xdrun` binary.

## `xdrun` is installed but not found

Check whether your shell can locate the binary:

```bash
which xdrun
echo $PATH
```

On Linux and macOS, the installer uses `~/.local/bin` by default:

```bash
export PATH="$PATH:$HOME/.local/bin"
```

On Windows, the installer uses `~/bin`. Add that directory to your Windows user `PATH`, then open a new terminal.

## No drun spec is found

Create the default spec:

```bash
xdrun --init
```

Or select an existing file explicitly:

```bash
xdrun --file path/to/spec.drun --list
```

`xdrun` checks workspace configuration and conventional paths including `.drun/spec.drun`, `spec.drun`, `infra/drun/spec.drun`, and `ops/drun/spec.drun`.

## A template cannot be loaded

First confirm that the official catalog is reachable and list its current entries:

```bash
xdrun --list-templates
```

Template initialization requires a name:

```bash
xdrun --init --template go-cli
```

When using `--from-template`, make sure the reference points to a manifest or to a local directory containing `templates.yaml`.

## Get more diagnostic output

Use verbose output for task execution:

```bash
xdrun --verbose task-name
```

For parser diagnostics, inspect the available debug modes:

```bash
xdrun --help
```
