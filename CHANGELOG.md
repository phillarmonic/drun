# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added the `docker network "<name>" [not] exists` condition for `if`/`when` statements, e.g. `if docker network "proxy" exists:`. The condition queries the Docker daemon (via `docker network ls`), supports interpolation in the network name, and evaluates as if the network were missing in `--dry-run` mode (no daemon query). The `docker <resource> "<name>" [not] exists` shape leaves room for future resource variants (containers, images, volumes).
- Added LSP hover documentation with examples for the `if docker network` and `when docker network` conditions.

### Changed

### Deprecated

### Removed

### Fixed

- Fixed Ctrl+C not stopping long-running tasks on Windows. `(*os.Process).Signal` cannot deliver `os.Interrupt`/`SIGTERM` to a child process on Windows, so the forwarded interrupt was silently discarded and child process trees (e.g. a docs server) kept running. On Windows drun now terminates the whole child process tree via `taskkill /F /T` when the interrupt is received; Unix signal forwarding is unchanged.

### Security

## [2.28.0] - 2026-08-11

### Added

- Added the `open url "<target>"` statement for opening URLs and local file paths in the OS default handler (`open` on macOS, `xdg-open` on Linux, `cmd /c start` on Windows). On headless machines, SSH sessions, and CI environments the statement prints a non-fatal warning with the URL and continues. Local paths without a scheme are resolved to absolute paths. Variables in the target are interpolated. Dry runs report the target without opening it.
- Added folder trust for `open url`: because the statement can launch programs, the folder must be trusted before it runs. On first use drun prompts interactively; `xdrun cmd:trust` and `xdrun cmd:untrust` manage trust from the CLI. The trusted-folder list is stored in `~/.drun/trusted.yml` and parent directories cover their children.
- Added LSP support for `open url`: keyword completion and hover documentation with examples.

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [2.27.0] - 2026-08-10

### Added

- Added the `promote changelog` statement for [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) release management: `promote changelog "CHANGELOG.md" to version "X.Y.Z" [on "YYYY-MM-DD"]` moves the `## [Unreleased]` entries into a dated release section, leaves an emptied `Unreleased` skeleton behind, and rewrites `[Unreleased]: .../compare/<prev>...HEAD` comparison links when present. Re-running it for a version whose release section already exists merges new `Unreleased` entries into that section (a no-op when there is nothing new), so release preparation tasks stay idempotent. Honors dry runs and writes atomically with preserved permissions.
- Added LSP support for `promote changelog`: keyword completion and hover documentation with examples.
- Added TextMate grammar coverage for `promote changelog` in the vendored language artifacts.

### Changed

### Deprecated

### Removed

### Fixed

### Security
