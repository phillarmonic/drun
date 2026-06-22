package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/engine"
	drunErrors "github.com/phillarmonic/drun/v2/internal/errors"
)

const (
	textDocumentSyncFull = 1

	completionItemKindText     = 1
	completionItemKindFunction = 3
	completionItemKindKeyword  = 14
)

var taskNamePattern = regexp.MustCompile(`(?m)^\s*task\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_-]*))`)

var keywordCompletions = []completionItem{
	{Label: "task", Kind: completionItemKindKeyword, Detail: "Declare a task"},
	{Label: "project", Kind: completionItemKindKeyword, Detail: "Declare a project"},
	{Label: "given", Kind: completionItemKindKeyword, Detail: "Optional parameter"},
	{Label: "requires", Kind: completionItemKindKeyword, Detail: "Required parameter"},
	{Label: "depends on", Kind: completionItemKindKeyword, Detail: "Task dependency declaration"},
	{Label: "if", Kind: completionItemKindKeyword, Detail: "Conditional statement"},
	{Label: "else if", Kind: completionItemKindKeyword, Detail: "Conditional branch"},
	{Label: "when", Kind: completionItemKindKeyword, Detail: "Conditional statement"},
	{Label: "otherwise", Kind: completionItemKindKeyword, Detail: "Fallback branch"},
	{Label: "for each", Kind: completionItemKindKeyword, Detail: "Loop statement"},
	{Label: "run", Kind: completionItemKindKeyword, Detail: "Run a shell command"},
	{Label: "exec", Kind: completionItemKindKeyword, Detail: "Execute a shell command"},
	{Label: "shell", Kind: completionItemKindKeyword, Detail: "Shell command statement"},
	{Label: "capture", Kind: completionItemKindKeyword, Detail: "Capture expression or shell output"},
	{Label: "use workdir", Kind: completionItemKindKeyword, Detail: "Change working directory"},
	{Label: "call task", Kind: completionItemKindKeyword, Detail: "Call another task"},
	{Label: "orchestrate", Kind: completionItemKindKeyword, Detail: "Orchestration definition or action"},
	{Label: "service", Kind: completionItemKindKeyword, Detail: "Service definition"},
	{Label: "attached", Kind: completionItemKindKeyword, Detail: "Interactive run modifier"},
}

type Server struct {
	in    *bufio.Reader
	out   io.Writer
	docs  map[string]string
	state serverState
}

type serverState struct {
	shutdownRequested bool
}

type message struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
	ServerInfo   serverInfo         `json:"serverInfo"`
}

type serverCapabilities struct {
	TextDocumentSync   int                `json:"textDocumentSync"`
	CompletionProvider *completionOptions `json:"completionProvider,omitempty"`
}

type completionOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type textDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId,omitempty"`
	Version    int    `json:"version,omitempty"`
	Text       string `json:"text"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type versionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version,omitempty"`
}

type textDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type didChangeParams struct {
	TextDocument   versionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []textDocumentContentChangeEvent `json:"contentChanges"`
}

type didCloseParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type completionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type publishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

type diagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity,omitempty"`
	Source   string   `json:"source,omitempty"`
	Message  string   `json:"message"`
}

type lspRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type completionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		in:   bufio.NewReader(in),
		out:  out,
		docs: make(map[string]string),
	}
}

func (s *Server) Run() error {
	for {
		payload, err := s.readPayload()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var msg message
		if err := json.Unmarshal(payload, &msg); err != nil {
			if writeErr := s.writeResponse(message{
				JSONRPC: "2.0",
				ID:      msg.ID,
				Error: &responseError{
					Code:    -32700,
					Message: "invalid JSON-RPC payload",
				},
			}); writeErr != nil {
				return writeErr
			}
			continue
		}

		shouldExit, err := s.handleMessage(msg)
		if err != nil {
			return err
		}
		if shouldExit {
			return nil
		}
	}
}

func (s *Server) handleMessage(msg message) (bool, error) {
	switch msg.Method {
	case "initialize":
		return false, s.writeResponse(message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result: initializeResult{
				Capabilities: serverCapabilities{
					TextDocumentSync: textDocumentSyncFull,
					CompletionProvider: &completionOptions{
						ResolveProvider: false,
					},
				},
				ServerInfo: serverInfo{
					Name:    "xdrun-lsp",
					Version: "0.1.0",
				},
			},
		})
	case "initialized":
		return false, nil
	case "shutdown":
		s.state.shutdownRequested = true
		return false, s.writeResponse(message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result:  nil,
		})
	case "exit":
		if s.state.shutdownRequested {
			return true, nil
		}
		return false, fmt.Errorf("received exit before shutdown")
	case "textDocument/didOpen":
		var params didOpenParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return false, err
		}
		s.docs[params.TextDocument.URI] = params.TextDocument.Text
		return false, s.publishDiagnostics(params.TextDocument.URI, params.TextDocument.Text)
	case "textDocument/didChange":
		var params didChangeParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return false, err
		}
		if len(params.ContentChanges) > 0 {
			s.docs[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
		}
		return false, s.publishDiagnostics(params.TextDocument.URI, s.docs[params.TextDocument.URI])
	case "textDocument/didClose":
		var params didCloseParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return false, err
		}
		delete(s.docs, params.TextDocument.URI)
		return false, s.writeNotification("textDocument/publishDiagnostics", publishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: []diagnostic{},
		})
	case "textDocument/completion":
		var params completionParams
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return false, err
		}
		text := s.docs[params.TextDocument.URI]
		items := completionsForSource(text)
		return false, s.writeResponse(message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Result:  items,
		})
	default:
		if len(msg.ID) == 0 {
			return false, nil
		}
		return false, s.writeResponse(message{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &responseError{
				Code:    -32601,
				Message: "method not found",
			},
		})
	}
}

func (s *Server) publishDiagnostics(uri, text string) error {
	diagnostics := diagnosticsForSource(uri, text)
	return s.writeNotification("textDocument/publishDiagnostics", publishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

func diagnosticsForSource(uri, text string) []diagnostic {
	filename := filenameFromURI(uri)
	if filename == "" {
		filename = uri
	}

	_, err := engine.ParseStringWithFilename(text, filename)
	if err == nil {
		return []diagnostic{}
	}

	if errorList, ok := err.(*drunErrors.ParseErrorList); ok {
		diagnostics := make([]diagnostic, 0, len(errorList.Errors))
		for _, parseErr := range errorList.Errors {
			startLine := max(parseErr.Token.Line-1, 0)
			startChar := max(parseErr.Token.Column-1, 0)
			endChar := startChar + max(len(parseErr.Token.Literal), 1)
			diagnostics = append(diagnostics, diagnostic{
				Range: lspRange{
					Start: position{Line: startLine, Character: startChar},
					End:   position{Line: startLine, Character: endChar},
				},
				Severity: 1,
				Source:   "xdrun",
				Message:  parseErr.Message,
			})
		}
		return diagnostics
	}

	return []diagnostic{{
		Range: lspRange{
			Start: position{Line: 0, Character: 0},
			End:   position{Line: 0, Character: 1},
		},
		Severity: 1,
		Source:   "xdrun",
		Message:  err.Error(),
	}}
}

func completionsForSource(text string) []completionItem {
	items := make([]completionItem, 0, len(keywordCompletions)+8)
	items = append(items, keywordCompletions...)

	seen := map[string]struct{}{}
	for _, item := range items {
		seen[item.Label] = struct{}{}
	}

	if program, err := engine.ParseStringWithFilename(text, "<completion>"); err == nil {
		items = appendTaskCompletions(items, seen, program)
		return items
	}

	for _, match := range taskNamePattern.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if name == "" {
			name = match[2]
		}
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		items = append(items, completionItem{
			Label:  name,
			Kind:   completionItemKindFunction,
			Detail: "Task",
		})
		seen[name] = struct{}{}
	}

	return items
}

func appendTaskCompletions(items []completionItem, seen map[string]struct{}, program *ast.Program) []completionItem {
	for _, task := range program.Tasks {
		if _, exists := seen[task.Name]; exists {
			continue
		}
		items = append(items, completionItem{
			Label:  task.Name,
			Kind:   completionItemKindFunction,
			Detail: "Task",
		})
		seen[task.Name] = struct{}{}
	}
	return items
}

func (s *Server) readPayload() ([]byte, error) {
	headers := make(map[string]string)
	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}

	contentLength, err := strconv.Atoi(headers["content-length"])
	if err != nil {
		return nil, fmt.Errorf("invalid content length: %w", err)
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(s.in, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Server) writeResponse(msg message) error {
	return s.writeMessage(msg)
}

func (s *Server) writeNotification(method string, params any) error {
	return s.writeMessage(message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  mustMarshal(params),
	})
}

func (s *Server) writeMessage(msg message) error {
	if msg.JSONRPC == "" {
		msg.JSONRPC = "2.0"
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	var frame bytes.Buffer
	fmt.Fprintf(&frame, "Content-Length: %d\r\n\r\n", len(payload))
	frame.Write(payload)
	_, err = s.out.Write(frame.Bytes())
	return err
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func filenameFromURI(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if parsed.Scheme != "file" {
		return raw
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return parsed.Path
	}
	if path == "" {
		return raw
	}
	return path
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
