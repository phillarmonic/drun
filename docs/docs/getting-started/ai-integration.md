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
