package lsp

import (
	"regexp"
	"strings"
)

type definitionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type location struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

var callTaskRefPattern = regexp.MustCompile(`call\s+task\s+(?:"([^"]*)"|([A-Za-z_][A-Za-z0-9_.-]*))`)
var useSnippetRefPattern = regexp.MustCompile(`use\s+snippet\s+(?:"([^"]*)"|([A-Za-z_][A-Za-z0-9_.-]*))`)
var dependsOnRefPattern = regexp.MustCompile(`depends\s+on\s+(.+)$`)
var refNameTokenPattern = regexp.MustCompile(`"([^"]*)"|([A-Za-z_][A-Za-z0-9_.-]*)`)

var taskDeclPattern = regexp.MustCompile(`^\s*(?:template\s+)?task\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))`)
var snippetDeclPattern = regexp.MustCompile(`^\s*snippet\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))`)

func definitionsForSource(uri, source string, pos position) []location {
	lines := strings.Split(source, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) || pos.Character < 0 {
		return nil
	}

	code := codeBeforeComment(lines[pos.Line])
	cursor := byteOffsetForUTF16(code, pos.Character)
	if cursor < 0 {
		return nil
	}

	kind, name, ok := referenceAtPosition(code, cursor)
	if !ok {
		return nil
	}

	target := definitionLocation(uri, source, kind, name)
	if target == nil {
		return nil
	}
	return []location{*target}
}

// referenceAtPosition reports the kind ("task" or "snippet") and name of the
// reference whose name token contains the cursor byte offset.
func referenceAtPosition(code string, cursor int) (string, string, bool) {
	statementStart := len(code) - len(strings.TrimLeft(code, " \t"))

	if match := callTaskRefPattern.FindStringSubmatchIndex(code); match != nil && match[0] == statementStart {
		if name, found := refNameAtCursor(code, match, cursor); found {
			return "task", name, true
		}
	}

	if match := useSnippetRefPattern.FindStringSubmatchIndex(code); match != nil && match[0] == statementStart {
		if name, found := refNameAtCursor(code, match, cursor); found {
			return "snippet", name, true
		}
	}

	if match := dependsOnRefPattern.FindStringSubmatchIndex(code); match != nil && match[0] == statementStart {
		rest := code[match[2]:match[3]]
		for _, token := range refNameTokenPattern.FindAllStringSubmatchIndex(rest, -1) {
			name, found := refNameAtCursor(rest, token, cursor-match[2])
			if !found {
				continue
			}
			if name == "and" || name == "then" {
				continue
			}
			return "task", name, true
		}
	}

	return "", "", false
}

// refNameAtCursor extracts the referenced name from an alternation match whose
// group 1 is a quoted name and group 2 a bare name, and reports whether the
// cursor byte offset falls within the whole token (quotes included).
func refNameAtCursor(src string, match []int, cursor int) (string, bool) {
	if start, end := match[2], match[3]; start >= 0 && end > start {
		if cursor >= start-1 && cursor <= end+1 {
			return src[start:end], true
		}
		return "", false
	}
	if start, end := match[4], match[5]; start >= 0 && end > start {
		if cursor >= start && cursor <= end {
			return src[start:end], true
		}
	}
	return "", false
}

func definitionLocation(uri, source, kind, name string) *location {
	pattern := taskDeclPattern
	if kind == "snippet" {
		pattern = snippetDeclPattern
	}

	for lineNumber, rawLine := range strings.Split(source, "\n") {
		code := codeBeforeComment(rawLine)
		match := pattern.FindStringSubmatchIndex(code)
		if match == nil {
			continue
		}

		nameStart, nameEnd := match[2], match[3]
		if nameStart < 0 {
			nameStart, nameEnd = match[4], match[5]
		}
		if code[nameStart:nameEnd] != name {
			continue
		}

		return &location{
			URI: uri,
			Range: lspRange{
				Start: position{Line: lineNumber, Character: utf16Column(code[:nameStart])},
				End:   position{Line: lineNumber, Character: utf16Column(code[:nameEnd])},
			},
		}
	}
	return nil
}
