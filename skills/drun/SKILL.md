---
name: drun
description: >-
  Use when working with drun specs or xdrun automation: tasks, parameters,
  platforms, interpolation, CI, and hooks.
---

# drun

Use this skill when the task mentions drun, xdrun, `.drun/spec.drun`, task
automation, or repository workflows implemented in drun.

Drun is a fluent automation DSL. The interpreter binary is `xdrun`
("execute drun"). Prefer readable specs that make intent obvious at a glance.

## What to know first

- Main spec location: `.drun/spec.drun`
- Initialize a repo: `xdrun --init`
- List tasks: `xdrun --list`
- Run a task: `xdrun <task>`
- Pass task parameters as `key=value`, for example
  `xdrun deploy environment=production`
- Keep CLI behavior flags separate, for example
  `xdrun deploy environment=production --dry-run`
- Upstream docs and examples: https://github.com/phillarmonic/drun

## Recommended workflow

1. Read the existing drun file before making changes.
2. If there is no spec yet, initialize one with `xdrun --init`.
3. Use `xdrun --list` to inspect task names instead of guessing.
4. For platform-specific workflows, prefer separate declarations with
   `@platform(...)` instead of mixing OS branches into one task when behavior
   differs substantially.
5. Use canonical platform names in new specs: `linux`, `mac`, `windows`.
   Legacy `darwin` still parses, but prefer `mac` in new code and examples.
6. If a task family includes both platform-tagged variants and one unannotated
   task, drun resolves the exact platform variant first and uses the
   unannotated task as the fallback.
7. When adding hard dependencies, declare them with `requires tools:`.
8. Prefer small, readable tasks that explain intent with `means`, `info`, and
   `step`.
9. For AI-driven CI or noisy checks, prefer `mode "ci"` so successful shell
   stdout/stderr stays buffered and only failure output is emitted.
10. After editing a spec, run the narrowest relevant `xdrun` command to verify.

## Project AI guidance

Repositories can install a managed cross-agent guide with:

```bash
xdrun cmd:skill install drun-basics
```

That writes `.drun/ai/drun-basics.md` plus light entrypoints in common agent
files. Prefer that for explicit per-repository onboarding. A Repertoire project
bootstrap can install this complete portable skill globally while managing only
compact activation pointers in the repository.

## Interpolation and variables

- Task and project variables commonly use `{$name}` inside strings.
- Environment variables use shell-style syntax such as `${USER}` or
  `${HOME:-/tmp}`.
- Conditional interpolation supports ternary and `if/then/else`, for example
  `{$debug ? '--debug' : ''}` or
  `{if $environment is 'production' then 'prod.yml' else 'dev.yml'}`.
- Secrets can be read with `secret(...)`, for example `{secret('api_key')}` or
  `{secret('webhook_url', 'https://default.example')}`.
- Undefined drun variables are strict by default. Prefer
  `requires $name` or `given $name defaults to ...`.
- If an expression is hard to scan inline, compute it with
  `set $name to "..."` first.
- Multi-line `run "..."` strings support interpolation and are often cleaner
  than one huge shell line.

Useful examples in the upstream repo:

- `examples/03-interpolation.drun`
- `examples/51-env-var-interpolation.drun`
- `examples/52-conditional-interpolation.drun`
- `examples/62-secrets-interpolation.drun`
- `examples/63-multiline-strings.drun`

## Tool checks

Prefer declarative requirements when a task depends on a binary or minimum
version:

```drun
project "example" version "1.0":
  requires tools:
    go >= "1.21"
    docker
```

Task-level checks are also valid:

```drun
task "test" means "Run the test suite":
  requires tools:
    go
  run "go test ./..."
```

## Writing good specs

- Keep the file readable at a glance.
- Prefer task names that match user intent.
- For the same user-facing workflow across platforms, use duplicate task names
  with disjoint `@platform(...)` annotations so `xdrun <task>` resolves the
  correct variant.
- A task family may include one unannotated task as fallback when no
  platform-specific variant matches.
- Use `given $name defaults to ...` for optional parameters.
- Use `requires $name` for values that must be supplied at runtime.
- Prefer interpolation for task inputs and shell env expansion for true process
  environment values.
- For complex command assembly, compute intermediate values with `set` before
  the `run` step.
- Use `call task ...` instead of duplicating steps.
- Use `mode "ci"` for noisy validation when you want to save output tokens on
  success.
- Keep shell commands explicit inside `run "..."`.

## Lifecycle basics

- Bootstrap with `xdrun --init`.
- Treat `.drun/spec.drun` as the source of truth for project automation.
- Use `xdrun --list` to discover available workflows.
- `mode "ci"` buffers normal shell stdout/stderr and only prints that buffer
  when a command fails.
- Validate with targeted runs such as `xdrun test` or
  `xdrun build --dry-run`.

## Example starter spec

```drun
version: 2.0

project "example" version "1.0":
  requires tools:
    go

task "default" means "Show available automation":
  info "Run xdrun --list to inspect tasks"

task "test" means "Run tests":
  run "go test ./..."

@platform("linux", "mac")
task "shell" means "Open a Unix shell":
  run "bash" attached

@platform("windows")
task "shell" means "Open PowerShell":
  run "pwsh.exe" attached

task "ci" mode "ci" means "Run noisy checks with buffered output":
  run "go test ./..."
```

## Git policy and hooks

Projects can define git conventions with a `git policy:` block. When present,
use `xdrun cmd:hook install` to install drun-managed hooks that enforce them.
The Repertoire catalog does not install these hooks.

```drun
project "example" version "1.0":
  git policy:
    branch:
      default branches: "master", "main"
      protected branches: "master", "main"
      naming: "{type}/{identifier}-{description}"
      types: "feat", "fix", "chore"
    commit:
      messages: "{identifier}: {message}"
      extract identifier from branch
      enforce signed commits
```
