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

- Fixed `call task ... with` arguments being stored raw and interpolated only in the callee. The callee does not inherit the caller's parameters, so forwarding a parameter by the same name (`with app-version="{$app-version}"`) resolved to the placeholder itself, and a shell command then expanded the `$app` portion. Those arguments are now interpolated in the calling task before the call, including hyphenated names, and the callee's constraint checks see the resolved value.

### Security

## [2.30.0] - 2026-09-09

### Added

Interactive mode
- `xdrun debug` now captures the standard Go 1.27 `goroutineleak` profile (text and binary) on the critical-memory path, and drun's goroutines carry pprof labels (`drun.component`, `drun.worker`, `drun.variable`, `drun.orchestration`, `drun.service`) so panic and SIGQUIT tracebacks identify which pool, worker, or orchestration loop a goroutine belongs to.

### Changed

- Modernized the codebase onto Go 1.27 and drove `golangci-lint` from 801 findings on the go 1.27.1 tree (844 at planning time) to zero across 209 files. Every fix is mechanical or semantics-preserving: formatting (`gofumpt`/`goimports`), `perfsprint`, `modernize`, `usestdlibvars`, `errorlint` (`%v` → `%w` wrapping, and `==`/type-assertion error matching → `errors.Is`/`errors.As`), `govet` `shadow` renames, `fieldalignment` struct reordering, `wastedassign`, `nilness`, and `gocritic`. The 34 findings that remain are deliberate and each is suppressed in place with a written reason.
- JSON handling uses `encoding/json/v2` where it earns its place: the LSP JSON-RPC envelope (v2 matches member names case-sensitively and rejects duplicate members, as JSON-RPC 2.0 requires), GitHub/GitLab CLI response decoding, and the self-updater's API responses. JSON emitted into user files and small human-facing debug serializations stay on v1 so their bytes are unchanged.
- Module dependencies refreshed to their current same-major releases: cobra 1.10.2, `x/crypto` 0.57.0 together with `x/term` 0.46.0, `go-keyring` 0.2.8, `figlet` 1.3.1, and the archive-compression stack (`klauspost/compress`, `minlz`, `sevenzip`, `brotli`, `rardecode`, `lz4`, `xz`).

### Fixed

- Engine cleanup now runs when a task fails. `os.Exit` moved out of task execution into the command handler, so `defer eng.Cleanup()` releases remote-include temporary files and closes the include cache manager on the task-execution-failure and parameter-validation-failure paths, where it was previously skipped. The stderr text and exit code are unchanged.
- `xdrun` workspace config loading only falls back to defaults when the config file is genuinely absent (`fs.ErrNotExist`); permission and I/O failures on that file now surface as errors instead of being silently masked as a missing file.

### Security

- A provisioning manifest fetched from a remote source is now confined to its temporary clone directory: absolute, drive-qualified, and `..`-escaping manifest paths are rejected, so a remote source cannot address files outside its own clone. The remaining `gosec` findings are deliberate authorizations — spec-, CLI-, and hook-declared paths and commands — and each is annotated in place with the provenance of the path or command name.

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
- Fixed data races when `for each ... in parallel`/range bodies run concurrently: builtin-error collection is now per interpolation call and engine output writes are synchronized, so the race detector (`go test -race`) no longer trips on parallel loops. Loop tests that embed temp-file paths now use forward slashes so they are not corrupted by drun string-literal escape handling on Windows CI.

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
