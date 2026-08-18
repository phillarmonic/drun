package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestAnnotationCompletionIncludesPlatformSnippetAndDocumentation(t *testing.T) {
	items := completionsForSource("file:///workspace/spec.drun", "version: 2.0\n\n@")

	var platformItem *completionItem
	for i := range items {
		if items[i].Label == "@platform" {
			platformItem = &items[i]
			break
		}
	}
	if platformItem == nil {
		t.Fatal("expected @platform completion")
	}
	if platformItem.InsertText != `@platform("${1|linux,mac,windows|}")` {
		t.Fatalf("insertText = %q", platformItem.InsertText)
	}
	if platformItem.InsertTextFormat != completionTextFormatSnippet {
		t.Fatalf("insertTextFormat = %d, want snippet", platformItem.InsertTextFormat)
	}
	if platformItem.Documentation == nil || !strings.Contains(platformItem.Documentation.Value, "Supported values") {
		t.Fatalf("unexpected documentation: %#v", platformItem.Documentation)
	}
}

func TestServerAnnotationCompletionReplacesTypedPrefix(t *testing.T) {
	source := "version: 2.0\n\n@pla"
	input := joinFrames(
		frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`),
		frame(fmt.Sprintf(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///workspace/spec.drun","languageId":"drun","version":1,"text":%q}}}`, source)),
		frame(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///workspace/spec.drun"},"position":{"line":2,"character":4}}}`),
		frame(`{"jsonrpc":"2.0","id":3,"method":"shutdown","params":{}}`),
		frame(`{"jsonrpc":"2.0","method":"exit","params":{}}`),
	)

	var output bytes.Buffer
	if err := NewServer(bytes.NewReader(input), &output).Run(); err != nil {
		t.Fatalf("server run failed: %v", err)
	}

	var initializeMsg, completionMsg message
	for _, msg := range decodeFrames(t, output.Bytes()) {
		switch string(msg.ID) {
		case "1":
			initializeMsg = msg
		case "2":
			completionMsg = msg
		}
	}
	var initialized initializeResult
	if err := json.Unmarshal(mustMarshal(initializeMsg.Result), &initialized); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if initialized.Capabilities.CompletionProvider == nil ||
		len(initialized.Capabilities.CompletionProvider.TriggerCharacters) != 1 ||
		initialized.Capabilities.CompletionProvider.TriggerCharacters[0] != "@" {
		t.Fatalf("completion triggers = %#v", initialized.Capabilities.CompletionProvider)
	}

	var items []completionItem
	if err := json.Unmarshal(mustMarshal(completionMsg.Result), &items); err != nil {
		t.Fatalf("unmarshal completion items: %v", err)
	}
	for _, item := range items {
		if item.Label != "@platform" {
			continue
		}
		if item.TextEdit == nil || item.TextEdit.Range.Start.Character != 0 || item.TextEdit.Range.End.Character != 4 {
			t.Fatalf("unexpected annotation text edit: %#v", item.TextEdit)
		}
		if item.TextEdit.NewText != `@platform("${1|linux,mac,windows|}")` {
			t.Fatalf("newText = %q", item.TextEdit.NewText)
		}
		return
	}
	t.Fatal("expected @platform completion in protocol response")
}

func TestPlatformAnnotationHoverIncludesValuesScopeAndVariantBehavior(t *testing.T) {
	got := hoverForSource(`@platform("linux", "mac")`, position{Line: 0, Character: 5})
	if got == nil {
		t.Fatal("expected @platform hover")
	}
	for _, expected := range []string{
		"Platform-specific declaration",
		"`linux`, `mac`, `windows`",
		"task, template task, or snippet",
		"matching variant",
	} {
		if !strings.Contains(got.Contents.Value, expected) {
			t.Fatalf("hover does not contain %q:\n%s", expected, got.Contents.Value)
		}
	}
	if got.Range.Start.Character != 0 || got.Range.End.Character != len("@platform") {
		t.Fatalf("unexpected hover range: %#v", got.Range)
	}
}
