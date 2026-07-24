# AI Agents Integration

In a hurry? Need to work with drun faster? Teach your AI agents how to work with drun!

## Install the skill in your repository

Teaching your AI agents how to use drun is pretty straightforward. Run the following command in the working directory of your project:

```bash
xdrun cmd:skill install drun-basics
```

This will install a skill file at `.drun/ai/drun-basics.md`, and refer to it in your `AGENTS.md` file.

If the `AGENTS.md` file doesn't yet exist, it will be created. If it does, a new section versioned by drun will be added, similar to this:

```markdown
<!-- drun:skill:drun-basics:start -->
When tasks mention drun...
<!-- drun:skill:drun-basics:end -->
```

As drun evolves, in the future, the contents of this specific block in your `AGENTS.md` file will be updated. Don't worry, we don't mess with the rest of the file.

The instruction set surface on `AGENTS.md` is very light on purpose, considering this affects the amount of input tokens your tools consume. Then, if drun specific actions are required, the skill is red from the actual folder.

## Saving money on AI usage with drun

**How to save money (and context) on AI agent usage (input tokens) with the drun CI execution mode**

Drun is designed for speed and efficiency. When running tasks that have a lot of noisy logs, but which the logs don't matter for Large Language Model usage when an error isn't thrown, you can use the **CI execution mode**.

The CI mode polls the stdin/stderr of every command being executed by a task until something on the pipeline exits with a code other than `0`.

In that occasion, **and only in that occasion**, then, the logs are dumped to the stdout/stderr.

This can save you a lot of noise in **context usage** and **input tokens** when integrating the use of AI agents in your work 

Take, for example, drun's [CI pipeline](https://github.com/phillarmonic/drun/blob/master/.drun/spec.drun):

```drun
task "ci" mode "ci" means "Runs the whole CI pipeline":
    info "Executing the local CI pipeline..."
    step "Dependency vulnerability checks"
    call task vuln
    step "Linter"
    call task lint
    step "Unit tests"
    call task test
    step "Checking for regressions"
    run "./scripts/test-regressions.sh"
    step "Security checks"
    call task sec
    info "Semantic fuzzing"
    call task fuzz with iterations=50
    success "CI executed successfully end-to-end"
```

The the statement `mode "ci"` hints at the drun interpreter this supposed to be executed in the CI silent mode by default.

The normal mode of our pipeline generates around 700 lines worth of logs, 46 thousand characters of text, and around **15 thousand input tokens**, just for these logs. And they are noise, **they add nothing valuable to the context of the LLM**.

The CI mode gives us:

```log
# Run the command
xdrun ci                                                                                                                                                             feature/lsp-goodies!+
ℹ️  Executing the local CI pipeline...
┌─────────────────────────────────┐
│ Dependency vulnerability checks │
└─────────────────────────────────┘
┌────────┐
│ Linter │
└────────┘
ℹ️  Running PHPCS
┌────────────┐
│ Unit tests │
└────────────┘
ℹ️  Running PHPUnit
┌──────────────────────────┐
│ Checking for regressions │
└──────────────────────────┘
┌─────────────────┐
│ Security checks │
└─────────────────┘
ℹ️  Running gosec
ℹ️  Semantic fuzzing
┌────────────────────────────────────────────────────────────┐
│ Executing semantic fuzz tests against example-based inputs │
│ Iterations: 50                                             │
└────────────────────────────────────────────────────────────┘
✅  CI executed successfully end-to-end
```

**That's around 204 tokens**. You can make it completely silent if you want no information on ci mode by having no step, info statements. **0 token usage**.

In the event a command fails in CI mode, only that command's stdout/stderr is dumped, so, overall, you always have less noise.

You can still inspect everything that's happening on CI mode backed tasks by overriding the default behavior with:

```bash
# Our task is called ci as well, it could be 'potato' if you wanted to
xdrun ci --task-mode=normal
```
