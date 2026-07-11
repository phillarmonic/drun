# Pattern macros

drun v2 includes a comprehensive set of built-in pattern macros that provide common validation patterns without requiring complex regular expressions.

## Available pattern macros

- **`semver`**: Basic semantic versioning (e.g., `v1.2.3`)
- **`semver_extended`**: Extended semantic versioning with pre-release and build metadata (e.g., `v2.0.1-RC2`, `v1.0.0-alpha.1+build.123`)
- **`uuid`**: UUID format (e.g., `550e8400-e29b-41d4-a716-446655440000`)
- **`url`**: HTTP/HTTPS URL format
- **`ipv4`**: IPv4 address format (e.g., `192.168.1.1`)
- **`slug`**: URL slug format (lowercase, hyphens only, e.g., `my-project-name`)
- **`docker_tag`**: Docker image tag format
- **`git_branch`**: Git branch name format

## Usage examples

```drun
task "deploy" means "Deploy with validation":
  # Basic semantic versioning
  requires $version as string matching semver

  # Extended semantic versioning
  requires $release as string matching semver_extended

  # UUID validation
  requires $deployment_id as string matching uuid

  # URL validation
  requires $api_endpoint as string matching url

  # IPv4 address validation
  requires $server_ip as string matching ipv4

  # Slug validation for project names
  requires $project_slug as string matching slug

  # Docker tag validation
  requires $image_tag as string matching docker_tag

  # Git branch validation
  requires $branch as string matching git_branch

  info "Deploying {version} to {server_ip}"
```

## Pattern macros vs raw patterns

Pattern macros can be used alongside raw regex patterns:

```drun
task "validation_examples":
  # Using pattern macros (recommended)
  requires $version as string matching semver
  requires $id as string matching uuid

  # Using raw patterns (for custom validation)
  requires $custom as string matching pattern "^custom-[0-9]+$"

  # Email validation (built-in)
  requires $email as string matching email format
```

## Error messages

Pattern macros provide descriptive error messages:

```text
# Semver validation error
Error: parameter 'version': value '1.2.3' does not match semver pattern (Basic semantic versioning (e.g., v1.2.3))

# UUID validation error
Error: parameter 'id': value 'not-a-uuid' does not match uuid pattern (UUID format (e.g., 550e8400-e29b-41d4-a716-446655440000))
```
