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

## [2.29.0] - 2026-09-03

### Added

- Added the `now` built-in function: `{now}` (and `capture <name> from now`) returns the current Unix time in seconds, letting tasks record timestamps for timing purposes.

### Changed

- `for each` over a string value that contains commas now iterates per comma-separated item (each item trimmed); values without commas keep splitting on whitespace. Array literals and typed `as list` parameters iterate their elements directly.
- Pattern-match loops now require an `of $subject` clause naming the variable or parameter to match against: `for each match m in pattern "<regex>" of $subject`.
- `if`/`when` availability and running checks accept any bare tool name (for example hyphenated `my-fake-tool`), not just keyword tools and quoted strings.

### Deprecated

### Removed

### Fixed

- Fixed `call task ... with key="value"` swallowing the next statement when that statement starts with a keyword (e.g. `set $x to ...` on the line after a task call). The lexer does not emit newline tokens between statements inside a block, so the `with` parameter loop kept consuming parameter names from the following line and failed with `expected '=' after parameter name`. Parameter names must now stay on the same source line as the `with` keyword.
- Fixed range, file-line, and pattern-match loops executing against hardcoded sample items. `for $i in range a to b [step s]` iterates the real bounds (including negative steps) and rejects zero/invalid steps; `for each line text in file "f"` reads the file line by line with a clear error for missing files; `for each match m in pattern "<regex>" of $subject` returns the real regex matches of the subject and reports invalid patterns or missing subjects clearly. Dry runs report what would be processed.
- Fixed errors raised inside `catch` bodies being swallowed: they now propagate and fail the task. `rethrow` inside a catch now re-raises the original caught error with its message instead of a generic "rethrown error". A bare `ignore` is valid only inside a catch body (marks the caught error handled) and errors clearly elsewhere.
- Fixed `capture <name> from <expression>` storing the raw expression text: it now interpolates/evaluates the expression like `let`/`set`, and bare `from now` stores the `now` builtin's epoch-seconds timestamp.
- Fixed `accepts $x as list of <elem>` failing validation with "unknown data type: list of ..." at runtime. List parameters now require a list value and validate every element against the declared element type (`list of strings`, `list of numbers`, `list of booleans`); unknown element types are reported clearly.
- Fixed unquoted non-keyword tool names in availability conditions silently evaluating false: `if my-fake-tool is available` now checks the tool (any bare identifier followed by `is`/`are` + `available`/`running`/`not available`/`not running` or a comma-separated tool list), while generic comparisons such as `if tags is not empty:` remain ordinary conditionals.

### Security

## [2.28.0] - 2026-08-11

### Added

- Added the `open url "<target>"` statement for opening URLs and local file paths in the OS default handler (`open` on macOS, `xdg-open` on Linux, `cmd /c start` on Windows). On headless machines, SSH sessions, and CI environments the statement prints a non-fatal warning with the URL and continues. Local paths without a scheme are resolved to absolute paths. Variables in the target are interpolated. Dry runs report the target without opening it.
- Added folder trust for `open url`: because the statement can launch programs, the folder must be trusted before it runs. On first use drun prompts interactively; `xdrun cmd:trust` and `xdrun cmd:untrust` manage trust from the CLI. The trusted-folder list is stored in `~/.drun/trusted.yml` and parent directories cover their children.
- Added LSP support for `open url`: keyword completion and hover documentation with examples.
- Added the `docker network "<name>" [not] exists` condition for `if`/`when` statements, e.g. `if docker network "proxy" exists:`. The condition queries the Docker daemon (via `docker network ls`), supports interpolation in the network name, and evaluates as if the network were missing in `--dry-run` mode (no daemon query). The `docker <resource> "<name>" [not] exists` shape leaves room for future resource variants (containers, images, volumes).
- Added LSP hover documentation with examples for the `if docker network` and `when docker network` conditions.

### Fixed

- Fixed Ctrl+C not stopping long-running tasks on Windows. `(*os.Process).Signal` cannot deliver `os.Interrupt`/`SIGTERM` to a child process on Windows, so the forwarded interrupt was silently discarded and child process trees (e.g. a docs server) kept running. On Windows drun now terminates the whole child process tree via `taskkill /F /T` when the interrupt is received; Unix signal forwarding is unchanged.

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
