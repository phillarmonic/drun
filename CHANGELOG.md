# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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
