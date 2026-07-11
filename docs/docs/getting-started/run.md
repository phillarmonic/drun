# Run tasks

Run `xdrun` from the project directory followed by a task name:

```bash
xdrun hello
```

If you used minimal initialization, the generated task is named `default`:

```bash
xdrun default
```

## List available tasks

```bash
xdrun --list
```

## Pass parameters

Task parameters use `key=value` syntax:

```bash
xdrun deploy environment=production version=v1.2.3
```

CLI behavior uses flags. For example, preview a task without executing it:

```bash
xdrun deploy environment=production --dry-run
```

## Run a specific spec

`xdrun` discovers `.drun/spec.drun` and other conventional locations automatically. Use `--file` when you need to select a particular spec:

```bash
xdrun --file examples/01-hello-world.drun hello
```

Next, learn how [project templates](templates.md) can generate useful boilerplate for your stack.
