package lsp

import "strings"

// functionHoverEntries documents the built-in functions from
// internal/builtins.Registry. They are only ever used inside interpolation
// braces ({...}), so they are matched separately from statement keywords.
var functionHoverEntries = []hoverEntry{
	{"current git commit", `{current git commit}`, "Current commit hash", "Returns the current Git commit hash. Pass `'short'` for the abbreviated 7-character form."},
	{"current git branch", `{current git branch}`, "Current branch name", "Returns the name of the currently checked-out Git branch."},
	{"now.format", `{now.format('2006-01-02 15:04:05')}`, "Formatted current time", "Formats the current time using a Go time layout (the reference time is `Mon Jan 2 15:04:05 MST 2006`)."},
	{"file exists", `{file exists('path')}`, "File existence check", "Returns the string `\"true\"` when the given path is an existing file, otherwise `\"false\"`."},
	{"dir exists", `{dir exists('path')}`, "Directory existence check", "Returns the string `\"true\"` when the given path is an existing directory, otherwise `\"false\"`."},
	{"env", `{env('VAR', 'default')}`, "Environment variable", "Reads an environment variable. The optional second argument supplies a fallback when the variable is unset."},
	{"pwd", `{pwd}`, "Working directory", "Returns the current working directory. Pass `'basename'` to return only the final path segment."},
	{"hostname", `{hostname}`, "System hostname", "Returns the machine hostname."},
	{"os", `{os}`, "Operating system", "Returns the operating system drun is running on: `windows`, `linux`, or `darwin`."},
	{"shell", `{shell}`, "Active shell", "Returns the shell drun executes commands with, such as `bash`, `zsh`, `pwsh`, `powershell`, or `cmd`."},
	{"start progress", `{start progress('message')}`, "Begin a progress report", "Starts a progress indicator with an initial message. Pass an optional second argument to name the tracker when using more than one."},
	{"update progress", `{update progress(50, 'message')}`, "Update a progress report", "Updates the progress percentage (0-100) and message. Pass an optional third argument to target a named tracker."},
	{"finish progress", `{finish progress('message')}`, "Complete a progress report", "Marks a progress indicator as finished with a final message. Pass an optional second argument to target a named tracker."},
	{"start timer", `{start timer('name')}`, "Start a timer", "Records the start time for the named timer."},
	{"stop timer", `{stop timer('name')}`, "Stop a timer", "Records the stop time for the named timer."},
	{"show elapsed time", `{show elapsed time('name')}`, "Report elapsed time", "Reports the elapsed duration for the named timer between its start and stop."},
	{"docker compose command", `{docker compose command}`, "Docker Compose command", "Returns the Docker Compose command to use (for example `docker compose` or `docker-compose`). An optional argument selects the project path."},
	{"compose_cmd", `{compose_cmd}`, "Docker Compose command", "Alias of `docker compose command`. Returns the Docker Compose command to use for the current project."},
	{"docker compose status", `{docker compose status}`, "Docker Compose status", "Returns the Compose project status: `up`, `down`, or `mixed`. An optional argument selects the project path."},
	{"secret", `{secret('key', 'default', 'namespace')}`, "Read a secret", "Retrieves a secret by key from the secrets manager. The optional second argument is a default value and the optional third selects a namespace."},
	{"available tasks", `{available tasks(', ', 'omit')}`, "List available tasks", "Returns the user-defined task names joined by an optional separator (default `, `). Additional arguments omit tasks with matching exact names."},
	{"dns_resolve", `{dns_resolve('example.com')}`, "Resolve a domain", "Resolves a domain name to its IP address."},
	{"dns_check", `{dns_check('example.com', 'A')}`, "Check a DNS record", "Checks a DNS record of the given type (for example `A`, `AAAA`, `MX`) for a domain."},
	{"dns_validate", `{dns_validate('example.com', '93.184.216.34')}`, "Validate a DNS record", "Verifies that a domain resolves to the expected IP address."},
}

// functionHover returns hover documentation when the cursor rests on a
// built-in function name inside an interpolation region ({...}).
func functionHover(code string, cursor, lineNo int) *hover {
	if cursor < 0 || cursor > len(code) {
		return nil
	}
	open := strings.LastIndex(code[:cursor], "{")
	if open < 0 || strings.Contains(code[open:cursor], "}") {
		return nil
	}
	closeRel := strings.Index(code[cursor:], "}")
	if closeRel < 0 {
		return nil
	}
	base := open + 1
	region := code[base : cursor+closeRel]

	var best *hoverEntry
	bestStart, bestEnd := -1, -1
	for i := range functionHoverEntries {
		entry := &functionHoverEntries[i]
		for from := 0; ; {
			offset := strings.Index(region[from:], entry.Phrase)
			if offset < 0 {
				break
			}
			start := base + from + offset
			end := start + len(entry.Phrase)
			if phraseBoundary(code, start, end) && cursor >= start && cursor <= end &&
				(best == nil || len(entry.Phrase) > len(best.Phrase)) {
				best, bestStart, bestEnd = entry, start, end
			}
			from = from + offset + 1
		}
	}
	if best == nil {
		return nil
	}

	return &hover{
		Contents: markupContent{Kind: "markdown", Value: hoverMarkdown(*best)},
		Range: lspRange{
			Start: position{Line: lineNo, Character: utf16Column(code[:bestStart])},
			End:   position{Line: lineNo, Character: utf16Column(code[:bestEnd])},
		},
	}
}
