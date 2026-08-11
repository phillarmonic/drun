# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added the `open url "<target>"` statement for opening URLs and local file paths in the OS default handler (`open` on macOS, `xdg-open` on Linux, `cmd /c start` on Windows). On headless machines, SSH sessions, and CI environments the statement prints a non-fatal warning with the URL and continues. Local paths without a scheme are resolved to absolute paths. Variables in the target are interpolated. Dry runs report the target without opening it.
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
