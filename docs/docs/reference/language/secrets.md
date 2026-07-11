# Secrets

## Secrets Management

drun v2 provides secure, built-in secrets management for storing and retrieving sensitive data like API keys, passwords, and tokens.

### Features

- **Automatic Project Isolation**: Secrets are automatically namespaced by project name from drun config
- **Platform Integration**: Uses native keychains (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- **Automatic Fallback**: Seamlessly falls back to AES-256-GCM encrypted file storage when platform keychain is unavailable or has permission issues
- **Interpolation Support**: Access secrets directly in strings using `{secret('key')}`
- **Namespace Override**: Optionally specify custom namespaces for shared secrets

### Secret Operations

#### Setting Secrets

```drun
# Basic usage (automatic project namespace)
secret set "api_key" to "secret123"
secret set "db_password" to "super_secure_pass"

# With custom namespace
secret set "shared_token" to "token456" in namespace "team-shared"

# From environment variable
secret set "github_token" to ${GITHUB_TOKEN}
```

#### Retrieving Secrets

```drun
# In interpolation (recommended)
info "Connecting with key: {secret('api_key')}"
run "curl -H 'Authorization: Bearer {secret('api_key')}' https://api.example.com"

# With default value
info "Webhook: {secret('webhook_url', 'https://default.webhook.com')}"

# From custom namespace
info "Token: {secret('shared_token', '', 'team-shared')}"
```

#### Checking Secret Existence

```drun
# Check if secret exists
secret exists "api_key"

# With namespace
secret exists "shared_token" in namespace "team-shared"
```

#### Listing Secrets

```drun
# List all secrets in current project namespace
secret list

# List from custom namespace
secret list from namespace "team-shared"
```

#### Deleting Secrets

```drun
# Delete secret
secret delete "api_key"

# Delete from custom namespace
secret delete "shared_token" from namespace "team-shared"
```

### Complete Example

```drun
version: 2.0

project "api-deployment" version "1.0":

task "configure":
  info "Setting up secrets..."
  secret set "api_key" to prompt "Enter API key:" masked
  secret set "db_password" to prompt "Enter database password:" masked
  success "Secrets configured"

task "deploy":
  depends on "configure"

  # Check if secrets exist
  secret exists "api_key"
  secret exists "db_password"

  # Use secrets in deployment
  run """
    kubectl create secret generic app-secrets \
      --from-literal=api-key={secret('api_key')} \
      --from-literal=db-password={secret('db_password')}
  """

  success "Deployed with secrets"

task "cleanup":
  info "Removing secrets..."
  secret delete "api_key"
  secret delete "db_password"
  success "Secrets removed"
```

### Security Features

- **Per-Project Isolation**: Secrets are automatically scoped to project names, preventing accidental access
- **Platform Keychains**: Integrates with OS-native secure storage when available
- **Encrypted Storage**: AES-256-GCM encryption with PBKDF2 key derivation for fallback storage
- **Memory Safety**: Secure clearing of sensitive data from memory
- **Input Validation**: Keys, namespaces, and values are validated before storage

### secret() Function

The `secret()` builtin function allows seamless secret access in interpolations:

**Syntax:**

```drun
{secret('key')}                           # Get from current project namespace
{secret('key', 'default')}                # Get with default value
{secret('key', '', 'namespace')}          # Get from specific namespace
```

**Examples:**

```drun
# Simple usage
info "API Key: {secret('api_key')}"

# With default for optional secrets
run "webhook_url={secret('webhook', 'https://default.com')}"

# From shared namespace
run "deploy --token {secret('deploy_token', '', 'ci-secrets')}"

# In complex strings
run "curl -u admin:{secret('password')} https://api.example.com"
```

### CLI Secret Management

In addition to managing secrets within tasks, drun provides a standalone `cmd:secret` command for managing secrets directly from the command line. This is useful for:

- Setting up secrets before running tasks
- Managing secrets across multiple projects
- Inspecting and listing stored secrets
- Team collaboration via shared namespaces

#### Command Syntax

```bash
# Add secrets
xdrun cmd:secret add <key> [value] [flags]

# List secrets
xdrun cmd:secret list [flags]
xdrun cmd:secret list-all [flags]

# Remove secrets
xdrun cmd:secret remove <key> [flags]
```

#### Namespace Flags

- `--project`, `-p`: Use project scope (auto-detects project name from drun config)
- `--global`, `-g`: Use global scope (shared across all projects)
- `--namespace <name>`, `-n <name>`: Use custom namespace

**Automatic Workspace Detection:**
When you run `cmd:secret` commands without specifying a namespace, drun automatically detects your project name from the current workspace's `.drun/spec.drun` or configured drun file. This provides automatic project isolation without manual namespace management.

#### Examples

**Add Secrets:**

```bash
# Add to default namespace (auto-detects project if in workspace, prompts for value)
xdrun cmd:secret add api_key

# Add with masked input (secure)
xdrun cmd:secret add api_key --masked

# Add to global scope
xdrun cmd:secret add --global shared_token "team-token-123"

# Add to project scope (auto-detects project name from drun config)
xdrun cmd:secret add --project db_password "secret-pass"

# Add to custom namespace
xdrun cmd:secret add --namespace team-alpha team_key "alpha-secret"
```

**List Secrets:**

```bash
# List in default namespace (auto-detects project if in workspace)
xdrun cmd:secret list

# List in global scope
xdrun cmd:secret list --global

# List in project scope (auto-detects project name)
xdrun cmd:secret list --project

# List all secrets across all namespaces
xdrun cmd:secret list-all

# Show secret values (use with caution)
xdrun cmd:secret list --show-values
```

**Remove Secrets:**

```bash
# Remove from default namespace (auto-detects project if in workspace)
xdrun cmd:secret remove api_key

# Remove from global scope
xdrun cmd:secret rm --global shared_token  # 'rm' is an alias

# Remove from project scope (auto-detects project name)
xdrun cmd:secret delete --project db_password  # 'delete' is an alias
```

#### Using CLI-Managed Secrets in Tasks

Secrets managed via CLI are accessible in tasks using the `secret()` function:

```drun
version: 2.0
project "my-app" version "1.0":

task "deploy":
  # Access project-scoped secret
  info "Deploying with key: {secret('api_key')}"

  # Access global secret
  info "Team token: {secret('shared_token', '', 'global')}"

  # Access custom namespace secret
  info "Alpha key: {secret('team_key', '', 'team-alpha')}"
```

#### Security Notes

1. **Masked Input**: Use `--masked` flag for secure password entry
2. **Command History**: Avoid passing secrets directly on command line
3. **Show Values**: Only use `--show-values` in secure environments
4. **Platform Storage**: Secrets stored in native keychains (macOS Keychain, Windows Credential Manager, Linux Secret Service)

### Best Practices

1. **Use Project Namespaces**: Let drun automatically scope secrets to projects
2. **Set Defaults**: Provide sensible defaults for optional secrets
3. **Clean Up**: Delete secrets after use in temporary workflows
4. **Audit Access**: Use `secret list` to review stored secrets
5. **Avoid Hardcoding**: Never commit secrets to version control

### Project AI Skill Installation

In addition to executing tasks, `xdrun` can install project-level AI guidance files for repositories that use drun. This helps assistants such as Codex, Claude Code, Cursor, and Copilot understand the expected drun/xdrun workflow.

#### Command Syntax

```bash
xdrun cmd:skill install drun-basics [--target <directory>] [--force]
```

#### Installed Files

The `drun-basics` bundle installs a shared guide plus agent-specific entrypoints:

- `.drun/ai/drun-basics.md`
- `.codex/skills/drun-basics/SKILL.md`
- `.cursor/rules/drun-basics.mdc`
- `AGENTS.md`
- `CLAUDE.md`
- `.github/copilot-instructions.md`

#### Behavior

- **Managed blocks**: For mergeable markdown entrypoints such as `AGENTS.md`, `CLAUDE.md`, and `.github/copilot-instructions.md`, drun inserts or updates a named managed block instead of replacing the whole file.
- **Default mode**: Creates missing standalone guidance files and skips existing standalone generated files.
- **Force mode**: `--force` overwrites existing standalone generated files.
- **Target directory**: `--target` installs the bundle into another repository root.

#### Example

```bash
# Install guidance into the current repository
xdrun cmd:skill install drun-basics

# Install into a sibling repository and overwrite existing generated files
xdrun cmd:skill install basics --target ../my-service --force
```
