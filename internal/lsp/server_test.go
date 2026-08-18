package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerPublishesDiagnosticsForInvalidDocument(t *testing.T) {
	input := joinFrames(
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///workspace/spec.drun","languageId":"drun","version":1,"text":"version: 2.0\n\ntask \"broken\"\n  info \"missing colon\"\n"}}}`),
		frame(`{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"exit","params":{}}`),
	)

	var output bytes.Buffer
	server := NewServer(bytes.NewReader(input), &output)
	if err := server.Run(); err != nil {
		t.Fatalf("server run failed: %v", err)
	}

	messages := decodeFrames(t, output.Bytes())
	if len(messages) < 3 {
		t.Fatalf("expected at least 3 output messages, got %d", len(messages))
	}

	var diagnosticsMsg message
	foundDiagnostics := false
	for _, msg := range messages {
		if msg.Method == "textDocument/publishDiagnostics" {
			diagnosticsMsg = msg
			foundDiagnostics = true
			break
		}
	}
	if !foundDiagnostics {
		t.Fatalf("expected publishDiagnostics notification, got %#v", messages)
	}

	var params publishDiagnosticsParams
	if err := json.Unmarshal(diagnosticsMsg.Params, &params); err != nil {
		t.Fatalf("unmarshal diagnostics params: %v", err)
	}
	if len(params.Diagnostics) == 0 {
		t.Fatalf("expected at least one diagnostic")
	}
	if params.Diagnostics[0].Source != "xdrun" {
		t.Fatalf("expected xdrun diagnostic source, got %q", params.Diagnostics[0].Source)
	}
}

func TestServerCompletionIncludesKeywordsAndTasks(t *testing.T) {
	input := joinFrames(
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///workspace/spec.drun","languageId":"drun","version":1,"text":"version: 2.0\n\ntask \"deploy\":\n  info \"ok\"\n"}}}`),
		frame(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///workspace/spec.drun"}}}`),
		frame(`{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"exit","params":{}}`),
	)

	var output bytes.Buffer
	server := NewServer(bytes.NewReader(input), &output)
	if err := server.Run(); err != nil {
		t.Fatalf("server run failed: %v", err)
	}

	messages := decodeFrames(t, output.Bytes())
	var completionMsg message
	foundCompletion := false
	for _, msg := range messages {
		if string(msg.ID) == "2" {
			completionMsg = msg
			foundCompletion = true
			break
		}
	}
	if !foundCompletion {
		t.Fatalf("expected completion response, got %#v", messages)
	}

	var items []completionItem
	if err := json.Unmarshal(mustMarshal(completionMsg.Result), &items); err != nil {
		t.Fatalf("unmarshal completion items: %v", err)
	}

	assertCompletionLabel(t, items, "task")
	assertCompletionLabel(t, items, "deploy")
	assertCompletionLabel(t, items, "attached")
	assertCompletionLabel(t, items, "requires tools")
	assertCompletionLabel(t, items, "from tasks")
	assertCompletionLabel(t, items, "branch")
	assertCompletionLabel(t, items, "protected branches")
	assertCompletionLabel(t, items, "conventional commits")
	assertFileValueCompletions(t, items)
}

func TestServerAdvertisesAndReturnsHover(t *testing.T) {
	source := "version: 2.0\n\ntask \"deploy\":\n  run \"go build\"\n"
	input := joinFrames(
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		frame(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///workspace/spec.drun","languageId":"drun","version":1,"text":%q}}}`, source)),
		frame(`{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///workspace/spec.drun"},"position":{"line":3,"character":3}}}`),
		frame(`{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"exit","params":{}}`),
	)

	var output bytes.Buffer
	if err := NewServer(bytes.NewReader(input), &output).Run(); err != nil {
		t.Fatalf("server run failed: %v", err)
	}

	messages := decodeFrames(t, output.Bytes())
	var initializeMsg, hoverMsg message
	for _, msg := range messages {
		switch string(msg.ID) {
		case "1":
			initializeMsg = msg
		case "2":
			hoverMsg = msg
		}
	}

	var initialized initializeResult
	if err := json.Unmarshal(mustMarshal(initializeMsg.Result), &initialized); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if !initialized.Capabilities.HoverProvider {
		t.Fatal("expected hoverProvider capability")
	}

	var got hover
	if err := json.Unmarshal(mustMarshal(hoverMsg.Result), &got); err != nil {
		t.Fatalf("unmarshal hover: %v", err)
	}
	if got.Contents.Kind != "markdown" || !strings.Contains(got.Contents.Value, "Run a shell command") {
		t.Fatalf("unexpected hover contents: %#v", got.Contents)
	}
	if count := strings.Count(got.Contents.Value, "```drun"); count != 3 {
		t.Fatalf("expected syntax plus two highlighted examples, got %d fences:\n%s", count, got.Contents.Value)
	}
	if !strings.Contains(got.Contents.Value, `run "npm run dev" attached`) {
		t.Fatalf("expected attached run example:\n%s", got.Contents.Value)
	}
	if got.Range.Start.Character != 2 || got.Range.End.Character != 5 {
		t.Fatalf("unexpected hover range: %#v", got.Range)
	}
}

func TestHoverCoversCommonStatementsAndIgnoresStringsAndComments(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		column int
		want   string
	}{
		{"task", `task "build":`, 1, "Task declaration"},
		{"longest phrase", `  call task "build"`, 8, "Call another task"},
		{"file value", `  update json "/version" in "package.json" to "2"`, 5, "Update a JSON value"},
		{"changelog promotion", `  promote changelog "CHANGELOG.md" to version "1.5.0"`, 5, "Promote unreleased changelog entries"},
		{"control flow", `  for each $item in $items:`, 7, "Collection loop"},
		{"tool requirements", `  requires tools:`, 12, "Tool requirements"},
		{"unicode column", `é task "build":`, 3, ""},
		{"keyword outside statement position", `  set run to true`, 7, ""},
		{"quoted keyword", `  info "run this later"`, 9, ""},
		{"comment keyword", `  # run something`, 5, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := hoverForSource(test.line, position{Line: 0, Character: test.column})
			if test.want == "" {
				if got != nil {
					t.Fatalf("hoverForSource() = %#v, want nil", got)
				}
				return
			}
			if got == nil || !strings.Contains(got.Contents.Value, test.want) {
				t.Fatalf("hoverForSource() = %#v, want contents containing %q", got, test.want)
			}
		})
	}
}

func TestServerAdvertisesAndReturnsDefinition(t *testing.T) {
	source := "version: 2.0\n\ntask \"lint\":\n  run \"golangci-lint run\"\n\ntask \"build\":\n  call task lint\n"
	input := joinFrames(
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		frame(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///workspace/spec.drun","languageId":"drun","version":1,"text":%q}}}`, source)),
		frame(`{"jsonrpc":"2.0","id":2,"method":"textDocument/definition","params":{"textDocument":{"uri":"file:///workspace/spec.drun"},"position":{"line":6,"character":13}}}`),
		frame(`{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"exit","params":{}}`),
	)

	var output bytes.Buffer
	if err := NewServer(bytes.NewReader(input), &output).Run(); err != nil {
		t.Fatalf("server run failed: %v", err)
	}

	messages := decodeFrames(t, output.Bytes())
	var initializeMsg, definitionMsg message
	for _, msg := range messages {
		switch string(msg.ID) {
		case "1":
			initializeMsg = msg
		case "2":
			definitionMsg = msg
		}
	}

	var initialized initializeResult
	if err := json.Unmarshal(mustMarshal(initializeMsg.Result), &initialized); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if !initialized.Capabilities.DefinitionProvider {
		t.Fatal("expected definitionProvider capability")
	}

	var locations []location
	if err := json.Unmarshal(mustMarshal(definitionMsg.Result), &locations); err != nil {
		t.Fatalf("unmarshal definition result: %v", err)
	}
	if len(locations) != 1 {
		t.Fatalf("expected one definition location, got %#v", locations)
	}
	got := locations[0]
	if got.URI != "file:///workspace/spec.drun" {
		t.Fatalf("unexpected definition URI: %q", got.URI)
	}
	if got.Range.Start.Line != 2 || got.Range.Start.Character != 6 || got.Range.End.Character != 10 {
		t.Fatalf("unexpected definition range: %#v", got.Range)
	}
}

func TestDefinitionsForSource(t *testing.T) {
	source := `version: 2.0

snippet "greet":
  info "hello"

task "lint":
  run "golangci-lint run"

template task "scaffold":
  run "cookiecutter ."

task "release":
  run "goreleaser release"

task "build":
  depends on lint, scaffold and then "release"
  call task "lint"
  call task scaffold
  use snippet "greet"
`
	const uri = "file:///workspace/spec.drun"

	tests := []struct {
		name      string
		line      int
		character int
		wantLine  int
		wantStart int
		wantEnd   int
	}{
		{"quoted call task", 16, 14, 5, 6, 10},
		{"bare call task to template", 17, 14, 8, 15, 23},
		{"depends on first", 15, 15, 5, 6, 10},
		{"depends on template", 15, 22, 8, 15, 23},
		{"depends on quoted", 15, 40, 11, 6, 13},
		{"use snippet", 18, 16, 2, 9, 14},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			locations := definitionsForSource(uri, source, position{Line: test.line, Character: test.character})
			if len(locations) != 1 {
				t.Fatalf("definitionsForSource() = %#v, want one location", locations)
			}
			got := locations[0]
			if got.URI != uri {
				t.Fatalf("definition URI = %q, want %q", got.URI, uri)
			}
			want := lspRange{
				Start: position{Line: test.wantLine, Character: test.wantStart},
				End:   position{Line: test.wantLine, Character: test.wantEnd},
			}
			if got.Range != want {
				t.Fatalf("definition range = %#v, want %#v", got.Range, want)
			}
		})
	}
}

func TestDefinitionsForSourceReturnsNilWithoutReference(t *testing.T) {
	source := "version: 2.0\n\ntask \"lint\":\n  run \"golangci-lint run\"\n  call task missing\n  # call task lint\n"
	const uri = "file:///workspace/spec.drun"

	tests := []struct {
		name      string
		line      int
		character int
	}{
		{"cursor on keyword", 4, 3},
		{"cursor elsewhere on call line", 4, 1},
		{"unrelated statement", 3, 5},
		{"unknown task", 4, 14},
		{"commented call", 5, 13},
		{"line out of range", 42, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := definitionsForSource(uri, source, position{Line: test.line, Character: test.character}); got != nil {
				t.Fatalf("definitionsForSource() = %#v, want nil", got)
			}
		})
	}
}

func TestServerAdvertisesAndReturnsDocumentSymbols(t *testing.T) {
	source := "version: 2.0\n\nproject \"demo\" version \"1.0\":\n  requires tools:\n    go >= \"1.24\"\n\ntask \"build\" means \"Build it\":\n  requires $target as string\n  depends on lint, test\n  call task \"compile\" with target=$target\n"
	input := joinFrames(
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		frame(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///workspace/spec.drun","languageId":"drun","version":1,"text":%q}}}`, source)),
		frame(`{"jsonrpc":"2.0","id":2,"method":"textDocument/documentSymbol","params":{"textDocument":{"uri":"file:///workspace/spec.drun"}}}`),
		frame(`{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"exit","params":{}}`),
	)

	var output bytes.Buffer
	if err := NewServer(bytes.NewReader(input), &output).Run(); err != nil {
		t.Fatalf("server run failed: %v", err)
	}

	messages := decodeFrames(t, output.Bytes())
	var initializeMsg, symbolsMsg message
	for _, msg := range messages {
		switch string(msg.ID) {
		case "1":
			initializeMsg = msg
		case "2":
			symbolsMsg = msg
		}
	}

	var initialized initializeResult
	if err := json.Unmarshal(mustMarshal(initializeMsg.Result), &initialized); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if !initialized.Capabilities.DocumentSymbolProvider {
		t.Fatal("expected documentSymbolProvider capability")
	}

	var symbols []documentSymbol
	if err := json.Unmarshal(mustMarshal(symbolsMsg.Result), &symbols); err != nil {
		t.Fatalf("unmarshal document symbols: %v", err)
	}
	assertDocumentSymbol(t, symbols, "demo", "Project")
	build := assertDocumentSymbol(t, symbols, "build", "Task")
	assertDocumentSymbol(t, build.Children, "$target", "Parameter")
	assertDocumentSymbol(t, build.Children, "lint, test", "Depends on")
	assertDocumentSymbol(t, build.Children, "compile", "Calls task")
}

func TestDocumentSymbolsBuildUsefulNestedStructure(t *testing.T) {
	source := `project "demo":
	service "api" in "./api":
		health check:
			type "tcp"
	orchestrate "dev":
		services ["api"]

task "deploy":
	requires tools:
		go >= "1.24"
	when $environment is "production":
		if $approved is true:
			call task "release"
	otherwise:
		call task "preview"
`

	symbols := documentSymbolsForSource(source)
	project := assertDocumentSymbol(t, symbols, "demo", "Project")
	assertDocumentSymbol(t, project.Children, "api", "Service")
	assertDocumentSymbol(t, project.Children, "dev", "Orchestration")

	deploy := assertDocumentSymbol(t, symbols, "deploy", "Task")
	assertDocumentSymbol(t, deploy.Children, "Required tools", "Required tools")
	when := assertDocumentSymbol(t, deploy.Children, `$environment is "production"`, "When")
	ifSymbol := assertDocumentSymbol(t, when.Children, "$approved is true", "If")
	assertDocumentSymbol(t, ifSymbol.Children, "release", "Calls task")
	otherwise := assertDocumentSymbol(t, deploy.Children, "Otherwise", "Otherwise")
	assertDocumentSymbol(t, otherwise.Children, "preview", "Calls task")

	if deploy.Range.End.Line != 14 {
		t.Fatalf("deploy range ended on line %d, want 14", deploy.Range.End.Line)
	}
}

func assertDocumentSymbol(t *testing.T, symbols []documentSymbol, name, detail string) documentSymbol {
	t.Helper()
	for _, symbol := range symbols {
		if symbol.Name == name && symbol.Detail == detail {
			return symbol
		}
	}
	t.Fatalf("expected symbol %q (%s) in %#v", name, detail, symbols)
	return documentSymbol{}
}

func TestFileValueDiagnosticsAreLocalized(t *testing.T) {
	tests := []struct {
		name        string
		statement   string
		wantMessage string
	}{
		{
			name:        "check comparison",
			statement:   `check json "/version" in "package.json" matches "2"`,
			wantMessage: "expected 'equals' or 'differs from' in file value check",
		},
		{
			name:        "structured addition type",
			statement:   `update yaml "chart.version" in "Chart.yaml" to "2" or add`,
			wantMessage: "structured additions require 'as string', 'as number', or 'as boolean'",
		},
		{
			name:        "regex addition",
			statement:   `update match "(?P<value>.+)" in "VERSION" to "2" or add as string`,
			wantMessage: "regex match updates do not support 'or add'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := "version: 2.0\n\ntask \"invalid\":\n  " + test.statement + "\n"
			diagnostics := diagnosticsForSource("file:///workspace/spec.drun", source)
			if len(diagnostics) == 0 {
				t.Fatal("expected a diagnostic")
			}
			if diagnostics[0].Message != test.wantMessage {
				t.Fatalf("diagnostic message = %q, want %q", diagnostics[0].Message, test.wantMessage)
			}
			if diagnostics[0].Range.Start.Line != 3 {
				t.Fatalf("diagnostic line = %d, want 3", diagnostics[0].Range.Start.Line)
			}
			if diagnostics[0].Range.Start.Character < 2 {
				t.Fatalf("diagnostic character = %d, want a location within the statement", diagnostics[0].Range.Start.Character)
			}
			if diagnostics[0].Range.End.Character <= diagnostics[0].Range.Start.Character {
				t.Fatalf("diagnostic range is empty: %#v", diagnostics[0].Range)
			}
		})
	}
}

func TestServerTemplateFilesSupportTemplatePlaceholders(t *testing.T) {
	tempRoot := t.TempDir()
	templateDir := filepath.Join(tempRoot, "drun-templates", "templates")
	if err := os.MkdirAll(templateDir, 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRoot, "drun-templates", "templates.yaml"), []byte("version: \"1\"\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	templatePath := filepath.Join(templateDir, "go-cli.drun")
	templateURI := "file://" + filepath.ToSlash(templatePath)
	templateText := "version: 2.0\n\nproject \"{{project_name}}\" version \"1.0\":\ntemplate task \"build-template\":\n  get property \"{{project_name}}.version\" from \"gradle.properties\" as $version\n  run \"go build -o ./bin/{{binary_name}} {{cmd_path}}\"\n"

	input := joinFrames(
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		frame(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"%s","languageId":"drun","version":1,"text":%q}}}`, templateURI, templateText)),
		frame(fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"%s"}}}`, templateURI)),
		frame(`{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"exit","params":{}}`),
	)

	var output bytes.Buffer
	server := NewServer(bytes.NewReader(input), &output)
	if err := server.Run(); err != nil {
		t.Fatalf("server run failed: %v", err)
	}

	messages := decodeFrames(t, output.Bytes())

	var diagnosticsMsg message
	var completionMsg message
	for _, msg := range messages {
		switch {
		case msg.Method == "textDocument/publishDiagnostics":
			diagnosticsMsg = msg
		case string(msg.ID) == "2":
			completionMsg = msg
		}
	}

	var params publishDiagnosticsParams
	if err := json.Unmarshal(diagnosticsMsg.Params, &params); err != nil {
		t.Fatalf("unmarshal diagnostics params: %v", err)
	}
	if len(params.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics for template placeholders, got %#v", params.Diagnostics)
	}

	var items []completionItem
	if err := json.Unmarshal(mustMarshal(completionMsg.Result), &items); err != nil {
		t.Fatalf("unmarshal completion items: %v", err)
	}

	assertCompletionLabel(t, items, "template task")
	assertCompletionLabel(t, items, "build-template")
	assertFileValueCompletions(t, items)
}

func assertFileValueCompletions(t *testing.T, items []completionItem) {
	t.Helper()
	for _, operation := range []string{"get", "check", "update"} {
		for _, format := range []string{"property", "json", "yaml", "toml", "match"} {
			assertCompletionLabel(t, items, operation+" "+format)
		}
	}
}

func assertCompletionLabel(t *testing.T, items []completionItem, label string) {
	t.Helper()
	for _, item := range items {
		if item.Label == label {
			return
		}
	}
	t.Fatalf("expected completion label %q in %#v", label, items)
}

func frame(payload string) []byte {
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload))
}

func joinFrames(frames ...[]byte) []byte {
	return bytes.Join(frames, nil)
}

func decodeFrames(t *testing.T, data []byte) []message {
	t.Helper()
	reader := bytes.NewReader(data)
	server := NewServer(reader, io.Discard)

	var messages []message
	for {
		payload, err := server.readPayload()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("read payload: %v", err)
		}
		var msg message
		if err := json.Unmarshal(payload, &msg); err != nil {
			t.Fatalf("unmarshal output message: %v", err)
		}
		messages = append(messages, msg)
	}
	return messages
}
