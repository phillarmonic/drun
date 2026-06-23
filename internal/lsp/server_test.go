package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
