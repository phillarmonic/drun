package lsp

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/phillarmonic/drun/v2/internal/patterns"
)

type hoverEntry struct {
	Phrase      string
	Syntax      string
	Summary     string
	Description string
}

var hoverEntries = []hoverEntry{
	{"template task", `template task "name" means "description":`, "Reusable task template", "Defines a task body that can be included by other tasks with `use snippet` or template-oriented composition."},
	{"requires tools", `requires tools:`, "Tool requirements", "Declares tools that must be available before a task runs. Requirements can include version constraints, defaults, and provisioning sources."},
	{"depends on", `depends on task-a, task-b`, "Task dependencies", "Runs the named tasks before the current task. Comma-separated dependencies may run in parallel; `and then` expresses sequential execution."},
	{"call task", `call task "name" with key="value"`, "Call another task", "Executes another task explicitly and optionally supplies its parameters."},
	{"for each", `for each $item in $items:`, "Collection loop", "Runs the nested statements once for every value in a collection."},
	{"else if", `else if <condition>:`, "Conditional branch", "Adds another condition to the preceding `if` statement."},
	{"use workdir", `use workdir "path":`, "Scoped working directory", "Runs the nested statements with a different working directory, then restores the previous directory."},
	{"wait", `wait <number|{$variable}> second(s)|minute(s)|hour(s)`, "Pause execution", "Pauses task execution for a fixed duration, then continues with the next statement. The duration can be a number literal or an interpolated variable. To wait until a service responds instead, use `wait for service at \"url\" to be ready`."},
	{"get property", `get property "key" from "file" as $value`, "Read a properties value", "Reads a key from a Java properties file and assigns it to a variable."},
	{"get json", `get json "/pointer" from "file" as $value`, "Read a JSON value", "Reads a value selected by JSON Pointer and assigns it to a variable."},
	{"get yaml", `get yaml "path" from "file" as $value`, "Read a YAML value", "Reads a value selected by its YAML path and assigns it to a variable."},
	{"get toml", `get toml "path" from "file" as $value`, "Read a TOML value", "Reads a value selected by its dotted path and assigns it to a variable."},
	{"get match", `get match "pattern" from "file" as $value`, "Read a regular-expression match", "Matches file content and assigns the selected capture to a variable."},
	{"check property", `check property "key" in "file" equals <value>`, "Check a properties value", "Fails the task unless the selected value satisfies the comparison."},
	{"check json", `check json "/pointer" in "file" equals <value>`, "Check a JSON value", "Fails the task unless the JSON Pointer value satisfies the comparison."},
	{"check yaml", `check yaml "path" in "file" equals <value>`, "Check a YAML value", "Fails the task unless the selected YAML value satisfies the comparison."},
	{"check toml", `check toml "path" in "file" equals <value>`, "Check a TOML value", "Fails the task unless the selected TOML value satisfies the comparison."},
	{"check match", `check match "pattern" in "file" equals <value>`, "Check a regular-expression match", "Fails the task unless the selected capture satisfies the comparison."},
	{"update property", `update property "key" in "file" to <value>`, "Update a properties value", "Rewrites the selected property while preserving the rest of the file."},
	{"update json", `update json "/pointer" in "file" to <value>`, "Update a JSON value", "Rewrites the value selected by JSON Pointer."},
	{"update yaml", `update yaml "path" in "file" to <value>`, "Update a YAML value", "Rewrites the value selected by its YAML path."},
	{"update toml", `update toml "path" in "file" to <value>`, "Update a TOML value", "Rewrites the value selected by its dotted path."},
	{"update match", `update match "pattern" in "file" to <value>`, "Update a regular-expression match", "Replaces the selected regular-expression capture."},
	{"promote changelog", `promote changelog "file" to version "X.Y.Z" [on "YYYY-MM-DD"]`, "Promote unreleased changelog entries", "Moves the `## [Unreleased]` section of a Keep a Changelog file into a new dated release section, leaving an emptied Unreleased section behind. When the file has an `[Unreleased]: .../compare/<prev>...HEAD` link, the comparison links are updated too. The date defaults to today; use `on \"YYYY-MM-DD\"` to override it. Re-running for a version whose section already exists merges new Unreleased entries into it instead of failing, so release preparation is idempotent."},
	{"open url", `open url "<target>"`, "Open in default application", "Opens a URL or local file path in the OS default handler (browser, file viewer, etc.). The folder must be trusted before execution (prompted interactively or via `xdrun cmd:trust`). In headless, SSH, or CI sessions it prints the target instead of failing. Local paths without a scheme are resolved to absolute paths. Variables in the target are interpolated at execution time."},
	{"test connection", `test connection to "<host>" on port <port> [timeout "<duration>"]`, "TCP port check", "Probes a TCP port natively (a dial, no external tools such as `nc`). The check succeeds when something is listening on the port - i.e. the port is in use - and fails the task otherwise. `timeout` accepts Go durations (`\"500ms\"`, `\"10s\"`) or bare seconds and defaults to 5 seconds. To branch on the result instead of failing the task, use the `if port <port> is open on \"host\"` condition."},
	{"check if port", `check if port <port> is open on "<host>" [timeout "<duration>"]`, "TCP port check", "Probes a TCP port natively (a dial, no external tools such as `nc`). This is an alternate spelling of `test connection to \"host\" on port N`: it succeeds when the port is in use and fails the task otherwise. To branch on the result instead of failing the task, use the `if port <port> is open on \"host\"` condition."},
	{"if port", `if port <port> is [not] open on "<host>" [with timeout "<duration>"]:`, "TCP port condition", "Probes a TCP port natively and runs the nested statements when the probe result matches. `is open` is true when a dial succeeds, i.e. something is listening on the port (the port is in use). Host, port, and the optional timeout all support interpolation; the timeout accepts Go durations or bare seconds and defaults to 5 seconds. In `--dry-run` mode no connection is opened and the condition evaluates as if the port were closed."},
	{"when port", `when port <port> is [not] open on "<host>" [with timeout "<duration>"]:`, "TCP port condition", "Probes a TCP port natively and runs the nested statements when the probe result matches; use `otherwise` for the fallback branch. `is open` is true when a dial succeeds, i.e. something is listening on the port (the port is in use). Host, port, and the optional timeout all support interpolation. In `--dry-run` mode no connection is opened and the condition evaluates as if the port were closed."},
	{"if docker network", `if docker network "<name>" [not] exists:`, "Docker network condition", "Asks the Docker daemon whether a network exists and runs the nested statements when the answer matches. `exists` is true when the daemon lists the network; use `not exists` for the inverse. The network name supports interpolation. In `--dry-run` mode the daemon is not queried and the condition evaluates as if the network were missing."},
	{"when docker network", `when docker network "<name>" [not] exists:`, "Docker network condition", "Asks the Docker daemon whether a network exists and runs the nested statements when the answer matches; use `otherwise` for the fallback branch. `exists` is true when the daemon lists the network; use `not exists` for the inverse. The network name supports interpolation. In `--dry-run` mode the daemon is not queried and the condition evaluates as if the network were missing."},
	{"git policy", `git policy:`, "Git policy", "Defines repository conventions such as branch naming, protected branches, and commit-message rules."},
	{"git validate", `git validate`, "Validate Git policy", "Checks the current repository against the configured Git policy."},
	{"version", `version: 2.0`, "Language version", "Selects the Drun language version used to parse this file."},
	{"project", `project "name" version "value":`, "Project declaration", "Declares project metadata and project-level settings."},
	{"task", `task "name" means "description":`, "Task declaration", "Defines a named automation command. The optional `means` text is shown in task listings."},
	{"given", `given $name as <type> defaults to <value>`, "Optional task parameter", "Declares an optional typed parameter, usually with a default value."},
	{"requires", `requires $name as <type>`, "Required task parameter", "Declares a typed parameter that callers must provide."},
	{"run", `run "command"`, "Run a shell command", "Executes a command through the configured platform shell. Add `attached` for interactive terminal use."},
	{"exec", `exec "program"`, "Execute a program", "Executes a command directly rather than through shell command parsing."},
	{"capture", `capture "command" as $value`, "Capture command output", "Runs a command and assigns its standard output to a variable."},
	{"shell", `shell "command"`, "Run through the configured shell", "Executes a command with the shell configured for the current platform."},
	{"if", `if <condition>:`, "Conditional execution", "Runs the nested statements only when the expression is true."},
	{"when", `when <condition>:`, "Readable conditional execution", "Runs the nested statements when the condition matches; use `otherwise` for the fallback branch."},
	{"otherwise", `otherwise:`, "Fallback branch", "Runs when the preceding `when` branches did not match."},
	{"set", `set $name to <value>`, "Set a value", "Assigns a value for later interpolation and expression evaluation."},
	{"include", `include tasks from "source"`, "Include reusable definitions", "Imports tasks, templates, or snippets from another Drun source."},
	{"info", `info "message"`, "Informational output", "Prints a normal informational message, with variable interpolation."},
	{"step", `step "message"`, "Progress output", "Prints a step or progress message for the current task."},
	{"success", `success "message"`, "Success output", "Prints a successful-result message."},
	{"warn", `warn "message"`, "Warning output", "Prints a warning without failing the task."},
	{"fail", `fail "message"`, "Fail the task", "Stops execution and reports the supplied failure message."},
	{"create file", `create file "path"`, "Create a file", "Creates a file at the requested path."},
	{"create directory", `create (dir|directory|folder) "path"`, "Create a directory", "Creates a directory. The nouns `dir`, `directory`, and `folder` are interchangeable."},
	{"create folder", `create (dir|directory|folder) "path"`, "Create a directory", "Creates a directory. The nouns `dir`, `directory`, and `folder` are interchangeable."},
	{"create dir", `create (dir|directory|folder) "path"`, "Create a directory", "Creates a directory. The nouns `dir`, `directory`, and `folder` are interchangeable."},
	{"copy", `copy "source" to "destination"`, "Copy a file or directory", "Copies a filesystem entry to a new path."},
	{"move", `move "source" to "destination"`, "Move a file or directory", "Moves or renames a filesystem entry."},
	{"delete file", `delete file "path"`, "Delete a file", "Removes the selected file."},
	{"delete directory", `delete (dir|directory|folder) "path"`, "Delete a directory", "Removes a directory. The nouns `dir`, `directory`, and `folder` are interchangeable."},
	{"delete folder", `delete (dir|directory|folder) "path"`, "Delete a directory", "Removes a directory. The nouns `dir`, `directory`, and `folder` are interchangeable."},
	{"delete dir", `delete (dir|directory|folder) "path"`, "Delete a directory", "Removes a directory. The nouns `dir`, `directory`, and `folder` are interchangeable."},
	{"read file", `read file "path" as $content`, "Read a file", "Reads file content and optionally captures it in a variable."},
	{"write", `write "content" to file "path"`, "Write a file", "Writes content to a file, replacing its existing content."},
	{"append", `append "content" to file "path"`, "Append to a file", "Adds content to the end of a file."},
	{"download", `download "url" to "path"`, "Download a file", "Downloads a URL, with optional extraction, overwrite, permission, header, and authentication settings."},
	{"get request", `get request to "url"`, "HTTP GET request", "Sends an HTTP GET request with optional headers, authentication, timeout, and retry settings."},
	{"post request", `post request to "url" with body "content"`, "HTTP POST request", "Sends an HTTP POST request with optional body, headers, authentication, timeout, and retry settings."},
	{"put request", `put request to "url" with body "content"`, "HTTP PUT request", "Sends an HTTP PUT request with optional body and request settings."},
	{"patch request", `patch request to "url" with body "content"`, "HTTP PATCH request", "Sends an HTTP PATCH request with optional body and request settings."},
	{"git", `git <operation>`, "Git operation", "Runs a Git operation such as clone, status, add, commit, branch, switch, fetch, pull, push, or merge."},
	{"docker", `docker <operation> <resource>`, "Docker operation", "Runs a Docker image, container, compose, or registry operation."},
	{"service", `service "name":`, "Service declaration", "Defines a service used by orchestration, including repository, build, health-check, and runtime configuration."},
	{"orchestrate", `orchestrate "name":`, "Orchestration", "Defines or runs a coordinated set of services."},
}

// Extra examples keep the compact signature above useful while showing the
// common variants that are easiest to understand as real Drun.
var hoverExtraExamples = map[string][]string{
	"task": {
		"task \"test\" means \"Run the test suite\":\n  run \"go test ./...\"",
		"task \"release\":\n  requires $version as string\n  run \"goreleaser release --clean\"",
	},
	"requires tools": {
		"requires tools:\n  go >= \"1.24\"",
		"requires tools:\n  node >= \"22\" provision\n  pnpm provision",
	},
	"depends on": {
		"depends on lint, test, security_scan",
		"depends on build then publish",
	},
	"run": {
		"run \"go test ./...\"",
		"run \"npm run dev\" attached",
	},
	"capture": {
		"capture \"git rev-parse --short HEAD\" as $revision",
	},
	"if": {
		"if $environment is \"production\":\n  call task \"deploy-production\"",
	},
	"when": {
		"when os is \"darwin\":\n  run \"brew install jq\"\notherwise:\n  run \"apt-get install jq\"",
	},
	"for each": {
		"for each $package in $packages:\n  info \"Building ${package}\"",
	},
	"get json": {
		"get json \"/version\" from \"package.json\" as $version",
	},
	"check json": {
		"check json \"/name\" in \"package.json\" equals \"drun-language-support\"",
	},
	"update json": {
		"update json \"/version\" in \"package.json\" to $version",
	},
	"promote changelog": {
		"promote changelog \"CHANGELOG.md\" to version \"{$release_version}\"",
		"promote changelog \"CHANGELOG.md\" to version \"1.5.0\" on \"2026-09-01\"",
		"task \"prepare-release\" means \"Prepare a release\":\n  requires $version as string matching semver_optional_v\n  set $release_version to \"{$version without prefix 'v'}\"\n  update json \"/version\" in \"package.json\" to \"{$release_version}\" or fail\n  promote changelog \"CHANGELOG.md\" to version \"{$release_version}\"",
	},
	"use workdir": {
		"use workdir \"frontend\":\n  run \"pnpm test\"",
	},
	"wait": {
		"wait 5 seconds",
		"wait {$backoff} minutes",
		"wait 1 hour",
		"for each $attempt in [\"1\", \"2\", \"3\"]:\n  try:\n    run \"./flaky-deploy.sh\"\n    break\n  catch:\n    warn \"Attempt {$attempt} failed, backing off\"\n    wait {$attempt} minutes",
		"wait for service at \"https://api.local/health\" to be ready timeout \"60s\"",
	},
	"call task": {
		"call task \"build\"",
		"call task \"deploy\" with environment=\"staging\"",
	},
	"download": {
		"download \"https://example.com/tool.tar.gz\" extract to \".tools/tool\" remove archive",
	},
	"git": {
		"git clone repository \"https://github.com/acme/project.git\"",
		"git commit changes with message \"Release ${version}\"",
	},
	"docker": {
		"docker build image \"acme/app\" with tag $version",
		"docker compose up",
	},
	"service": {
		"service \"api\" in \"./services/api\" means \"REST API\":\n  health check:\n    type \"tcp\"\n    endpoint \"localhost:8080\"",
	},
	"orchestrate": {
		"orchestrate \"development\" means \"Local services\":\n  services [\"api\", \"database\"]\n  strategy \"dependency-based\"",
	},
	"open url": {
		"open url \"https://example.com/docs\"",
		"open url \"{$base_url}/releases/tag/v{$version}\"",
		"open url \"./coverage/index.html\"",
		"task \"docs\" means \"Open the project documentation\":\n  let $docs_url = \"https://docs.example.com\"\n  open url \"{$docs_url}/getting-started\"",
	},
	"test connection": {
		"test connection to \"database.example.com\" on port 5432",
		"test connection to \"localhost\" on port 8080 timeout \"10s\"",
	},
	"check if port": {
		"check if port 6379 is open on \"redis.local\"",
		"check if port 8080 is open on \"localhost\" timeout \"2s\"",
	},
	"if port": {
		"if port 5432 is open on \"localhost\":\n  info \"PostgreSQL is already running\"\nelse:\n  call task \"start-database\"",
		"if port {$redis_port} is not open on \"{$redis_host}\" with timeout \"2s\":\n  warn \"Redis is down - skipping cache warmup\"",
	},
	"when port": {
		"when port 8080 is open on \"localhost\":\n  info \"Dev server is up\"\notherwise:\n  info \"Dev server is not running\"",
	},
	"if docker network": {
		"if docker network \"proxy\" exists:\n  info \"Reusing the existing proxy network\"\nelse:\n  call task \"create-proxy-network\"",
		"if docker network \"{$app_network}\" not exists:\n  warn \"Network {$app_network} is missing - orchestration will create it\"",
	},
	"when docker network": {
		"when docker network \"legacy-bridge\" exists:\n  info \"Attaching to the legacy network\"\notherwise:\n  info \"Using the default bridge\"",
	},
}

func hoverForSource(source string, pos position) *hover {
	lines := strings.Split(source, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) || pos.Character < 0 {
		return nil
	}

	line := lines[pos.Line]
	code := codeBeforeComment(line)
	cursor := byteOffsetForUTF16(code, pos.Character)
	if cursor < 0 {
		return nil
	}
	if macro := macroHover(code, cursor, pos.Line); macro != nil {
		return macro
	}
	statementStart := len(code) - len(strings.TrimLeft(code, " \t"))

	var best *hoverEntry
	bestStart := -1
	entries := append(annotationHoverEntries(), hoverEntries...)
	for i := range entries {
		entry := &entries[i]
		for from := 0; from < len(code); {
			offset := strings.Index(code[from:], entry.Phrase)
			if offset < 0 {
				break
			}
			start := from + offset
			end := start + len(entry.Phrase)
			if start == statementStart && phraseBoundary(code, start, end) && cursor >= start && cursor <= end &&
				(best == nil || len(entry.Phrase) > len(best.Phrase)) {
				best, bestStart = entry, start
			}
			from = start + 1
		}
	}
	if best == nil || insideQuotedString(code, bestStart) {
		return nil
	}

	end := bestStart + len(best.Phrase)
	value := hoverMarkdown(*best)
	return &hover{
		Contents: markupContent{Kind: "markdown", Value: value},
		Range: lspRange{
			Start: position{Line: pos.Line, Character: utf16Column(code[:bestStart])},
			End:   position{Line: pos.Line, Character: utf16Column(code[:end])},
		},
	}
}

// macroHover returns hover documentation when the cursor rests on a built-in
// pattern macro name that follows the `matching` keyword.
func macroHover(code string, cursor, lineNo int) *hover {
	if cursor < 0 || cursor > len(code) {
		return nil
	}
	start := cursor
	for start > 0 && isWordByte(code[start-1]) {
		start--
	}
	end := cursor
	for end < len(code) && isWordByte(code[end]) {
		end++
	}
	if start == end {
		return nil
	}
	macro, ok := patterns.GetMacro(code[start:end])
	if !ok {
		return nil
	}
	// Only treat the word as a macro when it directly follows `matching`, so
	// identifiers that happen to share a macro name are not documented.
	prefix := strings.TrimRight(code[:start], " \t")
	if !strings.HasSuffix(prefix, "matching") {
		return nil
	}
	if before := len(prefix) - len("matching"); before > 0 && isWordByte(prefix[before-1]) {
		return nil
	}
	if insideQuotedString(code, start) {
		return nil
	}

	return &hover{
		Contents: markupContent{Kind: "markdown", Value: macroHoverMarkdown(macro)},
		Range: lspRange{
			Start: position{Line: lineNo, Character: utf16Column(code[:start])},
			End:   position{Line: lineNo, Character: utf16Column(code[:end])},
		},
	}
}

func macroHoverMarkdown(macro patterns.PatternMacro) string {
	lines := []string{
		fmt.Sprintf("### `%s`", macro.Name),
		"",
		"**Pattern macro**",
		"",
		macro.Description,
		"",
		"**Regex**",
		"",
		"```regex",
		macro.Pattern,
		"```",
		"",
		"**Syntax**",
		"",
		"```drun",
		fmt.Sprintf("requires $value as string matching %s", macro.Name),
		"```",
	}
	return strings.Join(lines, "\n")
}

func hoverMarkdown(entry hoverEntry) string {
	lines := []string{
		fmt.Sprintf("### `%s`", entry.Phrase),
		"",
		fmt.Sprintf("**%s**", entry.Summary),
		"",
		entry.Description,
		"",
		"**Syntax**",
		"",
		"```drun",
		entry.Syntax,
		"```",
	}
	examples := append([]string(nil), annotationExamples(entry.Phrase)...)
	examples = append(examples, hoverExtraExamples[entry.Phrase]...)
	if len(examples) > 0 {
		lines = append(lines, "", "**Examples**")
		for _, example := range examples {
			lines = append(lines, "", "```drun", example, "```")
		}
	}
	return strings.Join(lines, "\n")
}

func codeBeforeComment(line string) string {
	quoted, escaped := false, false
	for i, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && quoted:
			escaped = true
		case r == '"':
			quoted = !quoted
		case r == '#' && !quoted:
			return line[:i]
		}
	}
	return line
}

func insideQuotedString(line string, offset int) bool {
	quoted, escaped := false, false
	for i, r := range line {
		if i >= offset {
			break
		}
		switch {
		case escaped:
			escaped = false
		case r == '\\' && quoted:
			escaped = true
		case r == '"':
			quoted = !quoted
		}
	}
	return quoted
}

func phraseBoundary(line string, start, end int) bool {
	return (start == 0 || !isWordByte(line[start-1])) && (end == len(line) || !isWordByte(line[end]))
}

func isWordByte(b byte) bool {
	return b == '_' || b == '-' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func byteOffsetForUTF16(s string, column int) int {
	units := 0
	for offset, r := range s {
		if units >= column {
			return offset
		}
		units += len(utf16.Encode([]rune{r}))
		if units > column {
			return offset
		}
	}
	if units == column {
		return len(s)
	}
	return -1
}

func utf16Column(s string) int {
	units := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		units += len(utf16.Encode([]rune{r}))
		s = s[size:]
	}
	return units
}
