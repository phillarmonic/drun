# GitHub Actions

drun is primarily designed as a local automation tool, but teams can also run it in continuous integration to keep their CI tasks in 1:1 parity with local development. The [`phillarmonic/setup-drun`](https://github.com/phillarmonic/setup-drun) action installs `xdrun` on a GitHub Actions runner.

The action supports Linux, macOS, and Windows runners on both AMD64 and ARM64 architectures.

## Basic usage

Add the setup action before any step that invokes `xdrun`:

```yaml
- name: Setup xdrun
  uses: phillarmonic/setup-drun@v2
```

The action installs the latest stable release by default.

## Specify a version

Set `version` to install an exact release:

```yaml
- name: Setup xdrun
  uses: phillarmonic/setup-drun@v2
  with:
    version: 'v2.0.0'
```

## Pin to a major version

Use a major version when you want compatible updates without changing the workflow for every release:

```yaml
- name: Setup xdrun
  uses: phillarmonic/setup-drun@v2
  with:
    version: 'v2'
```

This resolves to the newest stable `v2.x.y` release.

## Disable caching

Downloaded binaries are cached by default. Set `cache` to `'false'` to disable this behavior:

```yaml
- name: Setup xdrun
  uses: phillarmonic/setup-drun@v2
  with:
    version: 'latest'
    cache: 'false'
```

## Use a custom GitHub token

The action uses GitHub's API to resolve and download releases. Supply a token explicitly when your workflow needs a custom token or encounters API rate limits:

```yaml
- name: Setup xdrun
  uses: phillarmonic/setup-drun@v2
  with:
    token: ${{ secrets.GITHUB_TOKEN }}
```

## Complete workflow

The following workflow checks out a repository, installs `xdrun`, and runs the same tasks a developer can run locally:

```yaml
name: CI

on:
  push:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Check out the repository
        uses: actions/checkout@v4

      - name: Setup xdrun
        uses: phillarmonic/setup-drun@v2
        with:
          version: 'v2'

      - name: Run CI tasks
        run: xdrun ci
```

See the [setup-drun repository](https://github.com/phillarmonic/setup-drun) for all inputs, supported platforms, outputs, and troubleshooting guidance.
