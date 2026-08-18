package lsp

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/platform"
)

type annotationEntry struct {
	Name        string
	Syntax      string
	Summary     string
	Description string
	InsertText  string
	Examples    []string
}

var annotationEntries = []annotationEntry{
	{
		Name:    "platform",
		Syntax:  `@platform("linux", "mac", "windows")`,
		Summary: "Platform-specific declaration",
		Description: fmt.Sprintf(
			"Restricts the following task, template task, or snippet to one or more platforms. Supported values: `%s`. The legacy value `darwin` is accepted and normalized to `mac`. Platform-tagged tasks may share a name; Drun selects the matching variant and then an unannotated fallback.",
			strings.Join(platform.CanonicalNames(), "`, `"),
		),
		InsertText: `@platform("${1|linux,mac,windows|}")`,
		Examples: []string{
			"@platform(\"linux\", \"mac\")\ntask \"shell\":\n  run \"bash\" attached",
			"@platform(\"windows\")\ntask \"shell\":\n  run \"pwsh.exe\" attached",
		},
	},
}

func annotationCompletionItems(replacementRange *lspRange) []completionItem {
	items := make([]completionItem, 0, len(annotationEntries))
	for _, entry := range annotationEntries {
		hoverEntry := entry.hoverEntry()
		item := completionItem{
			Label:            "@" + entry.Name,
			Kind:             completionItemKindKeyword,
			Detail:           entry.Summary,
			Documentation:    &markupContent{Kind: "markdown", Value: hoverMarkdown(hoverEntry)},
			InsertText:       entry.InsertText,
			InsertTextFormat: completionTextFormatSnippet,
		}
		if replacementRange != nil {
			item.TextEdit = &textEdit{Range: *replacementRange, NewText: entry.InsertText}
		}
		items = append(items, item)
	}
	return items
}

func annotationCompletionRange(source string, positions ...position) *lspRange {
	if len(positions) == 0 {
		return nil
	}
	pos := positions[0]
	lines := strings.Split(source, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	cursor := byteOffsetForUTF16(line, pos.Character)
	if cursor < 0 {
		return nil
	}

	start := cursor
	for start > 0 && isWordByte(line[start-1]) {
		start--
	}
	if start == 0 || line[start-1] != '@' {
		return nil
	}
	start--

	return &lspRange{
		Start: position{Line: pos.Line, Character: utf16Column(line[:start])},
		End:   pos,
	}
}

func annotationHoverEntries() []hoverEntry {
	entries := make([]hoverEntry, 0, len(annotationEntries))
	for _, entry := range annotationEntries {
		entries = append(entries, entry.hoverEntry())
	}
	return entries
}

func annotationExamples(phrase string) []string {
	for _, entry := range annotationEntries {
		if phrase == "@"+entry.Name {
			return entry.Examples
		}
	}
	return nil
}

func (entry annotationEntry) hoverEntry() hoverEntry {
	return hoverEntry{
		Phrase:      "@" + entry.Name,
		Syntax:      entry.Syntax,
		Summary:     entry.Summary,
		Description: entry.Description,
	}
}
