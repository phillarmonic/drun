package lsp

import (
	"regexp"
	"strings"
)

const (
	symbolKindFile      = 1
	symbolKindModule    = 2
	symbolKindNamespace = 3
	symbolKindClass     = 5
	symbolKindMethod    = 6
	symbolKindFunction  = 12
	symbolKindVariable  = 13
	symbolKindConstant  = 14
	symbolKindObject    = 19
	symbolKindOperator  = 25
)

type documentSymbolParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type documentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Children       []documentSymbol `json:"children,omitempty"`
	Range          lspRange         `json:"range"`
	SelectionRange lspRange         `json:"selectionRange"`
	Kind           int              `json:"kind"`
}

type symbolPattern struct {
	re        *regexp.Regexp
	label     string
	kind      int
	container bool
}

var documentSymbolPatterns = []symbolPattern{
	{re: regexp.MustCompile(`^version\s*:\s*(\S+)`), kind: symbolKindConstant, label: "Language version", container: false},
	{re: regexp.MustCompile(`^project\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_-]*))`), kind: symbolKindNamespace, label: "Project", container: true},
	{re: regexp.MustCompile(`^template\s+task\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))`), kind: symbolKindFunction, label: "Task template", container: true},
	{re: regexp.MustCompile(`^task\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))`), kind: symbolKindFunction, label: "Task", container: true},
	{re: regexp.MustCompile(`^service\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))`), kind: symbolKindClass, label: "Service", container: true},
	{re: regexp.MustCompile(`^orchestrate\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))(?::|\s+means)`), kind: symbolKindClass, label: "Orchestration", container: true},
	{re: regexp.MustCompile(`^snippet\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))`), kind: symbolKindFunction, label: "Snippet", container: true},
	{re: regexp.MustCompile(`^requires\s+tools\s*:`), kind: symbolKindObject, label: "Required tools", container: true},
	{re: regexp.MustCompile(`^provisioning\s+sources\s*:`), kind: symbolKindObject, label: "Provisioning sources", container: true},
	{re: regexp.MustCompile(`^git\s+policy\s*:`), kind: symbolKindObject, label: "Git policy", container: true},
	{re: regexp.MustCompile(`^shell\s+config\s*:`), kind: symbolKindObject, label: "Shell configuration", container: true},
	{re: regexp.MustCompile(`^scm\s*:`), kind: symbolKindModule, label: "SCM", container: true},
	{re: regexp.MustCompile(`^(?:given|requires)\s+(\$[A-Za-z_][A-Za-z0-9_-]*)`), kind: symbolKindVariable, label: "Parameter", container: false},
	{re: regexp.MustCompile(`^depends\s+on\s+(.+)$`), kind: symbolKindOperator, label: "Depends on", container: false},
	{re: regexp.MustCompile(`^call\s+task\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_.-]*))`), kind: symbolKindMethod, label: "Calls task", container: false},
	{re: regexp.MustCompile(`^else\s+if\s+(.+):$`), kind: symbolKindOperator, label: "Else if", container: true},
	{re: regexp.MustCompile(`^if\s+(.+):$`), kind: symbolKindOperator, label: "If", container: true},
	{re: regexp.MustCompile(`^when\s+(.+):$`), kind: symbolKindOperator, label: "When", container: true},
	{re: regexp.MustCompile(`^otherwise\s*:$`), kind: symbolKindOperator, label: "Otherwise", container: true},
	{re: regexp.MustCompile(`^for\s+each\s+(.+):$`), kind: symbolKindOperator, label: "For each", container: true},
	{re: regexp.MustCompile(`^try\s*:$`), kind: symbolKindOperator, label: "Try", container: true},
	{re: regexp.MustCompile(`^catch(?:\s+(.+))?\s*:$`), kind: symbolKindOperator, label: "Catch", container: true},
	{re: regexp.MustCompile(`^use\s+workdir\s+(?:"([^"]+)"|(\S+))\s*:$`), kind: symbolKindFile, label: "Working directory", container: true},
}

type openDocumentSymbol struct {
	symbol *documentSymbol
	indent int
}

func documentSymbolsForSource(source string) []documentSymbol {
	lines := strings.Split(source, "\n")
	symbols := make([]documentSymbol, 0)
	stack := make([]openDocumentSymbol, 0)

	for lineNumber, rawLine := range lines {
		code := codeBeforeComment(rawLine)
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}

		indent := indentationWidth(code)
		for len(stack) > 0 && indent <= stack[len(stack)-1].indent {
			closeDocumentSymbol(stack[len(stack)-1].symbol, lines, lineNumber-1)
			stack = stack[:len(stack)-1]
		}

		symbol, isContainer := documentSymbolForLine(code, trimmed, lineNumber)
		if symbol == nil {
			continue
		}

		if len(stack) == 0 {
			symbols = append(symbols, *symbol)
			symbol = &symbols[len(symbols)-1]
		} else {
			parent := stack[len(stack)-1].symbol
			parent.Children = append(parent.Children, *symbol)
			symbol = &parent.Children[len(parent.Children)-1]
		}
		if isContainer {
			stack = append(stack, openDocumentSymbol{symbol: symbol, indent: indent})
		}
	}

	lastLine := len(lines) - 1
	for len(stack) > 0 {
		closeDocumentSymbol(stack[len(stack)-1].symbol, lines, lastLine)
		stack = stack[:len(stack)-1]
	}
	return symbols
}

func documentSymbolForLine(code, trimmed string, lineNumber int) (*documentSymbol, bool) {
	leadingBytes := len(code) - len(strings.TrimLeft(code, " \t"))
	for _, pattern := range documentSymbolPatterns {
		match := pattern.re.FindStringSubmatchIndex(trimmed)
		if match == nil {
			continue
		}

		name, nameStart, nameEnd := symbolName(pattern, trimmed, match)
		lineEnd := utf16Column(code)
		selectionStart := utf16Column(code[:leadingBytes+nameStart])
		selectionEnd := utf16Column(code[:leadingBytes+nameEnd])
		symbol := &documentSymbol{
			Name:   name,
			Detail: pattern.label,
			Kind:   pattern.kind,
			Range: lspRange{
				Start: position{Line: lineNumber, Character: 0},
				End:   position{Line: lineNumber, Character: lineEnd},
			},
			SelectionRange: lspRange{
				Start: position{Line: lineNumber, Character: selectionStart},
				End:   position{Line: lineNumber, Character: selectionEnd},
			},
		}
		return symbol, pattern.container
	}
	return nil, false
}

func symbolName(pattern symbolPattern, line string, match []int) (string, int, int) {
	for group := 1; group < len(match)/2; group++ {
		start, end := match[group*2], match[group*2+1]
		if start >= 0 && end >= start {
			value := strings.TrimSpace(line[start:end])
			if value != "" {
				return value, start, end
			}
		}
	}
	start, end := match[0], match[1]
	name := pattern.label
	if name == "" {
		name = strings.TrimSpace(line[start:end])
	}
	return name, start, end
}

func closeDocumentSymbol(symbol *documentSymbol, lines []string, lastLine int) {
	if lastLine < symbol.Range.Start.Line {
		lastLine = symbol.Range.Start.Line
	}
	for lastLine > symbol.Range.Start.Line && strings.TrimSpace(lines[lastLine]) == "" {
		lastLine--
	}
	symbol.Range.End = position{Line: lastLine, Character: utf16Column(lines[lastLine])}
}

func indentationWidth(line string) int {
	width := 0
	for _, r := range line {
		switch r {
		case ' ':
			width++
		case '\t':
			width += 4
		default:
			return width
		}
	}
	return width
}
