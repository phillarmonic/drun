package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/engine"
	drunErrors "github.com/phillarmonic/drun/v2/internal/errors"
	"github.com/phillarmonic/drun/v2/internal/platform"
)

const (
	textDocumentSyncFull = 1

	completionItemKindText     = 1
	completionItemKindFunction = 3
	completionItemKindKeyword  = 14
)

var taskNamePattern = regexp.MustCompile(`(?m)^\s*(?:template\s+)?task\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_-]*))`)
var templatePlaceholderPattern = regexp.MustCompile(`\{\{[A-Za-z_][A-Za-z0-9_-]*\}\}`)

var keywordCompletions = []completionItem{
	{Label: "task", Kind: completionItemKindKeyword, Detail: "Declare a task"},
	{Label: "template task", Kind: completionItemKindKeyword, Detail: "Declare a task template"},
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
	{Label: "get property", Kind: completionItemKindKeyword, Detail: "Read a properties value into a variable"},
	{Label: "get json", Kind: completionItemKindKeyword, Detail: "Read a JSON value into a variable"},
	{Label: "get yaml", Kind: completionItemKindKeyword, Detail: "Read a YAML value into a variable"},
	{Label: "get toml", Kind: completionItemKindKeyword, Detail: "Read a TOML value into a variable"},
	{Label: "get match", Kind: completionItemKindKeyword, Detail: "Read a regular-expression capture into a variable"},
	{Label: "check property", Kind: completionItemKindKeyword, Detail: "Compare a properties value"},
	{Label: "check json", Kind: completionItemKindKeyword, Detail: "Compare a JSON value"},
	{Label: "check yaml", Kind: completionItemKindKeyword, Detail: "Compare a YAML value"},
	{Label: "check toml", Kind: completionItemKindKeyword, Detail: "Compare a TOML value"},
	{Label: "check match", Kind: completionItemKindKeyword, Detail: "Compare a regular-expression capture"},
	{Label: "update property", Kind: completionItemKindKeyword, Detail: "Update a properties value"},
	{Label: "update json", Kind: completionItemKindKeyword, Detail: "Update a JSON value"},
	{Label: "update yaml", Kind: completionItemKindKeyword, Detail: "Update a YAML value"},
	{Label: "update toml", Kind: completionItemKindKeyword, Detail: "Update a TOML value"},
	{Label: "update match", Kind: completionItemKindKeyword, Detail: "Update a regular-expression capture"},
	{Label: "use workdir", Kind: completionItemKindKeyword, Detail: "Change working directory"},
	{Label: "call task", Kind: completionItemKindKeyword, Detail: "Call another task"},
	{Label: "orchestrate", Kind: completionItemKindKeyword, Detail: "Orchestration definition or action"},
	{Label: "service", Kind: completionItemKindKeyword, Detail: "Service definition"},
	{Label: "attached", Kind: completionItemKindKeyword, Detail: "Interactive run modifier"},
	{Label: "git policy", Kind: completionItemKindKeyword, Detail: "Git conventions policy block"},
	{Label: "git validate", Kind: completionItemKindKeyword, Detail: "Validate git conventions"},
	{Label: "default branches", Kind: completionItemKindKeyword, Detail: "Default branch list (inside git policy)"},
	{Label: "protected branches", Kind: completionItemKindKeyword, Detail: "Branches protected from local commits by git hooks"},
	{Label: "branch naming", Kind: completionItemKindKeyword, Detail: "Branch naming convention (inside git policy)"},
	{Label: "commit messages", Kind: completionItemKindKeyword, Detail: "Commit message convention (inside git policy)"},
	{Label: "conventional commits", Kind: completionItemKindKeyword, Detail: "Built-in commit message policy mode"},
	{Label: "enforce signed commits", Kind: completionItemKindKeyword, Detail: "Require GPG/SSH signed commits"},
	{Label: "ban", Kind: completionItemKindKeyword, Detail: "Ban a commit message pattern"},
	{Label: "min length", Kind: completionItemKindKeyword, Detail: "Minimum commit message length"},
	{Label: "extract identifier from branch", Kind: completionItemKindKeyword, Detail: "Auto-extract ticket ID from branch name"},
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
		items := completionsForSource(params.TextDocument.URI, text)
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

	lspSource := sourceForLSP(filename, text)

	_, err := engine.ParseStringWithFilename(lspSource, filename)
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

func completionsForSource(uri, text string) []completionItem {
	items := make([]completionItem, 0, len(keywordCompletions)+8)
	items = append(items, keywordCompletions...)

	seen := map[string]struct{}{}
	for _, item := range items {
		seen[item.Label] = struct{}{}
	}

	filename := filenameFromURI(uri)
	lspSource := sourceForLSP(filename, text)

	if program, err := engine.ParseStringWithFilename(lspSource, "<completion>"); err == nil {
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
	taskVariants := make(map[string][]*ast.TaskStatement)
	for _, task := range program.Tasks {
		taskVariants[task.Name] = append(taskVariants[task.Name], task)
	}
	for _, task := range program.Tasks {
		variants := taskVariants[task.Name]
		if len(variants) == 1 {
			if _, exists := seen[task.Name]; exists {
				continue
			}
			items = append(items, completionItem{
				Label:  task.Name,
				Kind:   completionItemKindFunction,
				Detail: completionDetailForTask(task),
			})
			seen[task.Name] = struct{}{}
			continue
		}

		items = append(items, completionItem{
			Label:  task.Name,
			Kind:   completionItemKindFunction,
			Detail: completionDetailForTask(task),
		})
	}
	for _, template := range program.Templates {
		if _, exists := seen[template.Name]; exists {
			continue
		}
		items = append(items, completionItem{
			Label:  template.Name,
			Kind:   completionItemKindFunction,
			Detail: "Template task",
		})
		seen[template.Name] = struct{}{}
	}
	return items
}

func completionDetailForTask(task *ast.TaskStatement) string {
	meta, err := platform.ValidateAnnotations("task", task.Name, task.Annotations)
	if err == nil && len(meta.Platforms) > 0 {
		return "Task [" + platform.FormatList(meta.Platforms) + "]"
	}
	return "Task"
}

func sourceForLSP(filename, text string) string {
	if !isTemplateEditingContext(filename, text) {
		return text
	}

	return templatePlaceholderPattern.ReplaceAllStringFunc(text, func(match string) string {
		return strings.Repeat("x", len(match))
	})
}

func isTemplateEditingContext(filename, text string) bool {
	if !templatePlaceholderPattern.MatchString(text) {
		return false
	}

	if filename == "" {
		return false
	}

	for dir := filepath.Dir(filename); dir != "." && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "templates.yaml")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return strings.Contains(text, "{{project_name}}") ||
		strings.Contains(text, "{{binary_name}}") ||
		strings.Contains(text, "{{cmd_path}}") ||
		strings.Contains(text, "{{module_name}}")
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
