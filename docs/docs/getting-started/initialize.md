# Initialize a spec

A drun spec describes the tasks available in a project. By default, `xdrun` creates it at `.drun/spec.drun`.

## Normal mode

Use normal mode when you want an annotated starter with example `hello`, `build`, `test`, and `deploy` tasks:

```bash
xdrun --init
```

List the generated tasks:

```bash
xdrun --list
```

## Minimal mode

Use minimal mode when you want the smallest starting point: a project declaration and a single `default` task.

```bash
xdrun --init-minimal
```

Open `.drun/spec.drun` and replace the default task or add your own tasks.

## Choose another location

Pass `--file` with either initialization mode:

```bash
xdrun --init --file ops/drun/spec.drun
xdrun --init-minimal --file infra/drun/spec.drun
```

Built-in spec locations are discovered automatically. When you choose another location, `xdrun` also saves it as the workspace default. You can explicitly set an existing file as the default with:

```bash
xdrun --set-workspace path/to/spec.drun
```

For framework-specific boilerplate, [initialize from a project template](templates.md). Otherwise, continue to [running tasks](run.md).
