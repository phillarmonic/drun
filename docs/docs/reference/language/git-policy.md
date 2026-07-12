# Git Policy and Hooks

Drun allows you to define project-wide Git conventions (branch naming and commit messages) directly in the project block, and automatically validate them at runtime or through Git hooks.

## Project Git Policy

The `git policy:` block is used to configure your conventions:

```drun
project "myapp":
  git policy:
    branch:
      default branches: "master", "main"
      naming: "{type}/{identifier}-{description}"
      types: "feat", "fix", "chore"
    commit:
      messages: "conventional commits"
      ban: "WIP", "wip", "fixup"
      min length: 10
      extract identifier from branch
      enforce signed commits
```

### Settings

- `branch`: Block for branch-specific rules.
    - `default branches`: Branches that are exempt from the naming rules (for example, `main` and `develop`).
    - `naming`: The required pattern for feature branches. Supports `{type}`, `{identifier}`, and `{description}` placeholders.
    - `types`: Allowed values for the `{type}` placeholder.
- `commit`: Block for commit-specific rules.
    - `messages`: The required pattern for commit messages. Use `"conventional commits"` to enforce the Conventional Commits header format (`type(scope): description` with optional `!`).
    - `ban`: A list of exact commit messages that are rejected, such as `WIP`.
    - `min length`: The minimum number of characters for a commit message.
    - `extract identifier from branch`: Pulls the `{identifier}` from the current branch name and enforces its presence in the commit message. When used with `"conventional commits"`, the identifier can appear anywhere in the commit message.
    - `enforce signed commits`: Validates that commits are signed with GPG or SSH.

Example conventional commit messages that pass:

```text
feat(parser): PHIL-01 support conventional commit validation
fix(engine)!: CORE-77 reject invalid commit headers
chore(release): 1.2.3
```

## Validation (`git validate`)

You can manually trigger Git policy validation inside any task using the `git validate` statement:

```drun
task "pre-flight" means "Run checks before push":
  git validate branch_name
  git validate commit_message
  git validate signed_commits

  # Or validate everything at once:
  git validate all
```

## Git Hooks Lifecycle (`cmd:hook`)

Instead of manually running checks in tasks, you can use drun to manage and enforce Git hooks on developer machines:

```bash
# Install drun Git hooks (commit-msg and pre-push) to enforce the policy
xdrun cmd:hook install

# List installed hooks and their status
xdrun cmd:hook list

# Uninstall drun Git hooks
xdrun cmd:hook uninstall
```

When installed, drun automatically checks commit messages against your policy and blocks pushes if commits are unsigned when `enforce signed commits` is enabled.
