package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	templateText := "version: 2.0\n\nproject \"{{project_name}}\" version \"1.0\":\ntemplate task \"build-template\":\n  run \"go build -o ./bin/{{binary_name}} {{cmd_path}}\"\n"

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
