// Package filevalue implements safe scalar reads and updates for common text formats.
package filevalue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type Kind string

const (
	String  Kind = "string"
	Number  Kind = "number"
	Boolean Kind = "boolean"
)

type Scalar struct {
	Text string
	Kind Kind
}

var decimalNumberPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// Adapter is the internal contract implemented by every supported file format.
type Adapter interface {
	Read(selector string, data []byte) (Scalar, error)
	Update(selector string, data []byte, value, missingPolicy, valueType string) ([]byte, Scalar, error)
}

type adapterFuncs struct {
	read   func(string, []byte) (Scalar, error)
	update func(string, []byte, string, string, string) ([]byte, Scalar, error)
}

func (a adapterFuncs) Read(selector string, data []byte) (Scalar, error) {
	return a.read(selector, data)
}
func (a adapterFuncs) Update(selector string, data []byte, value, policy, valueType string) ([]byte, Scalar, error) {
	return a.update(selector, data, value, policy, valueType)
}

var adapters = map[string]Adapter{
	"property": adapterFuncs{read: readProperty, update: func(s string, d []byte, v, p, _ string) ([]byte, Scalar, error) { return updateProperty(s, d, v, p) }},
	"drun":     adapterFuncs{read: readDrun, update: updateDrun},
	"match": adapterFuncs{read: readMatch, update: func(s string, d []byte, v, p, _ string) ([]byte, Scalar, error) {
		if p == "add" {
			return nil, Scalar{}, fmt.Errorf("regex match updates do not support additions")
		}
		return updateMatch(s, d, v)
	}},
	"json": adapterFuncs{read: func(s string, d []byte) (Scalar, error) { _, v, e := findJSONScalar(d, s); return v, e }, update: updateJSON},
	"yaml": adapterFuncs{read: readYAML, update: updateYAML},
	"toml": adapterFuncs{read: readTOML, update: updateTOML},
}

var drunProjectDeclarationPattern = regexp.MustCompile(`(?m)^[\t ]*project[\t ]+"(?:\\.|[^"\\\r\n])*"[\t ]+version[\t ]+"((?:\\.|[^"\\\r\n])*)"[\t ]*:[^\r\n]*\r?$`)

func drunProjectVersionSpan(selector string, data []byte) (int, int, Scalar, error) {
	if selector != "project.version" {
		return 0, 0, Scalar{}, fmt.Errorf("unsupported drun selector %q; expected %q", selector, "project.version")
	}
	matches := drunProjectDeclarationPattern.FindAllSubmatchIndex(data, -1)
	if len(matches) == 0 {
		return 0, 0, Scalar{}, fmt.Errorf("drun project declaration with a version was not found")
	}
	if len(matches) != 1 {
		return 0, 0, Scalar{}, fmt.Errorf("drun project version is ambiguous: found %d project declarations", len(matches))
	}
	match := matches[0]
	start, end := match[2], match[3]
	return start, end, Scalar{Text: string(data[start:end]), Kind: String}, nil
}

func readDrun(selector string, data []byte) (Scalar, error) {
	_, _, value, err := drunProjectVersionSpan(selector, data)
	return value, err
}

func updateDrun(selector string, data []byte, value, missingPolicy, valueType string) ([]byte, Scalar, error) {
	if missingPolicy != "" && missingPolicy != "fail" {
		return nil, Scalar{}, fmt.Errorf("drun project version updates do not support %q", missingPolicy)
	}
	if valueType != "" {
		return nil, Scalar{}, fmt.Errorf("drun project versions do not accept an explicit scalar type")
	}
	if strings.ContainsAny(value, "\"\r\n") {
		return nil, Scalar{}, fmt.Errorf("drun project version cannot contain quotes or newlines")
	}
	start, end, _, err := drunProjectVersionSpan(selector, data)
	if err != nil {
		return nil, Scalar{}, err
	}
	updated := make([]byte, 0, len(data)-end+start+len(value))
	updated = append(updated, data[:start]...)
	updated = append(updated, value...)
	updated = append(updated, data[end:]...)
	return updated, Scalar{Text: value, Kind: String}, nil
}

// ReadFile resolves one scalar from path.
func ReadFile(format, selector, path string) (Scalar, error) {
	// #nosec G304 -- paths are explicitly supplied by the Drun program.
	data, err := os.ReadFile(path)
	if err != nil {
		return Scalar{}, err
	}
	return Read(format, selector, data)
}

// UpdateFile validates and atomically applies one scalar update.
func UpdateFile(format, selector, path, value, missingPolicy, valueType string) (bool, Scalar, error) {
	// #nosec G304 -- paths are explicitly supplied by the Drun program.
	data, err := os.ReadFile(path)
	if err != nil {
		return false, Scalar{}, err
	}
	updated, scalar, err := Update(format, selector, data, value, missingPolicy, valueType)
	if err != nil {
		return false, Scalar{}, err
	}
	if bytes.Equal(data, updated) {
		return false, scalar, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return false, Scalar{}, err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".drun-file-value-*")
	if err != nil {
		return false, Scalar{}, err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		_ = tmp.Close()
		return false, Scalar{}, err
	}
	if _, err := tmp.Write(updated); err != nil {
		_ = tmp.Close()
		return false, Scalar{}, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return false, Scalar{}, err
	}
	if err := tmp.Close(); err != nil {
		return false, Scalar{}, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return false, Scalar{}, err
	}
	return true, scalar, nil
}

func Read(format, selector string, data []byte) (Scalar, error) {
	adapter, ok := adapters[format]
	if !ok {
		return Scalar{}, fmt.Errorf("unsupported file value format %q", format)
	}
	return adapter.Read(selector, data)
}

func Update(format, selector string, data []byte, value, missingPolicy, valueType string) ([]byte, Scalar, error) {
	adapter, ok := adapters[format]
	if !ok {
		return nil, Scalar{}, fmt.Errorf("unsupported file value format %q", format)
	}
	return adapter.Update(selector, data, value, missingPolicy, valueType)
}

func scalarFromText(value, kind string) (Scalar, error) {
	switch Kind(kind) {
	case String, "":
		return Scalar{Text: value, Kind: String}, nil
	case Number:
		if !decimalNumberPattern.MatchString(value) {
			return Scalar{}, fmt.Errorf("%q is not a number", value)
		}
		return Scalar{Text: value, Kind: Number}, nil
	case Boolean:
		if value != "true" && value != "false" {
			return Scalar{}, fmt.Errorf("%q is not a boolean", value)
		}
		return Scalar{Text: value, Kind: Boolean}, nil
	default:
		return Scalar{}, fmt.Errorf("unsupported scalar type %q", kind)
	}
}

func encodedScalar(s Scalar, format string) string {
	if s.Kind == String {
		switch format {
		case "json":
			b, _ := json.Marshal(s.Text)
			return string(b)
		case "yaml":
			b, _ := yaml.Marshal(s.Text)
			return strings.TrimSpace(string(b))
		case "toml":
			return strconv.Quote(s.Text)
		}
	}
	return s.Text
}

type lineSpan struct{ start, end, valueStart, valueEnd int }

func isPropertySpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\f'
}

func decodePropertyKey(raw string) (string, bool) {
	var decoded strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			decoded.WriteByte(raw[i])
			continue
		}
		i++
		if i == len(raw) {
			return "", false
		}
		switch raw[i] {
		case 't':
			decoded.WriteByte('\t')
		case 'n':
			decoded.WriteByte('\n')
		case 'r':
			decoded.WriteByte('\r')
		case 'f':
			decoded.WriteByte('\f')
		case 'u':
			if i+4 >= len(raw) {
				return "", false
			}
			value, err := strconv.ParseUint(raw[i+1:i+5], 16, 16)
			if err != nil {
				return "", false
			}
			decoded.WriteRune(rune(value))
			i += 4
		default:
			decoded.WriteByte(raw[i])
		}
	}
	return decoded.String(), true
}

func propertyValueSpan(key, line string) (int, int, bool) {
	lineEnd := len(strings.TrimSuffix(line, "\r"))
	i := 0
	for i < lineEnd && isPropertySpace(line[i]) {
		i++
	}
	if i == lineEnd || line[i] == '#' || line[i] == '!' {
		return 0, 0, false
	}

	keyStart := i
	escaped := false
	for i < lineEnd {
		b := line[i]
		if escaped {
			escaped = false
			i++
			continue
		}
		if b == '\\' {
			escaped = true
			i++
			continue
		}
		if b == '=' || b == ':' || isPropertySpace(b) {
			break
		}
		i++
	}
	decodedKey, valid := decodePropertyKey(line[keyStart:i])
	if !valid || decodedKey != key {
		return 0, 0, false
	}

	for i < lineEnd && isPropertySpace(line[i]) {
		i++
	}
	if i < lineEnd && (line[i] == '=' || line[i] == ':') {
		i++
	}
	for i < lineEnd && isPropertySpace(line[i]) {
		i++
	}
	return i, lineEnd, true
}

func propertySpans(key string, data []byte) []lineSpan {
	var spans []lineSpan
	for start := 0; start <= len(data); {
		end := bytes.IndexByte(data[start:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += start
		}
		line := string(data[start:end])
		if valueStart, valueEnd, ok := propertyValueSpan(key, line); ok {
			spans = append(spans, lineSpan{start: start, end: end, valueStart: start + valueStart, valueEnd: start + valueEnd})
		}
		if end == len(data) {
			break
		}
		start = end + 1
	}
	return spans
}

func readProperty(key string, data []byte) (Scalar, error) {
	spans := propertySpans(key, data)
	if len(spans) != 1 {
		return Scalar{}, fmt.Errorf("property %q matched %d times", key, len(spans))
	}
	return Scalar{Text: string(data[spans[0].valueStart:spans[0].valueEnd]), Kind: String}, nil
}

func updateProperty(key string, data []byte, value, policy string) ([]byte, Scalar, error) {
	spans := propertySpans(key, data)
	s := Scalar{Text: value, Kind: String}
	if len(spans) > 1 {
		return nil, Scalar{}, fmt.Errorf("property %q matched %d times", key, len(spans))
	}
	if len(spans) == 0 {
		if policy != "add" {
			return nil, Scalar{}, fmt.Errorf("property %q does not exist", key)
		}
		newline := "\n"
		if bytes.Contains(data, []byte("\r\n")) {
			newline = "\r\n"
		}
		out := append([]byte(nil), data...)
		if len(out) > 0 && !bytes.HasSuffix(out, []byte("\n")) {
			out = append(out, []byte(newline)...)
		}
		out = append(out, []byte(key+"="+value+newline)...)
		return out, s, nil
	}
	sp := spans[0]
	out := append([]byte(nil), data[:sp.valueStart]...)
	out = append(out, value...)
	out = append(out, data[sp.valueEnd:]...)
	return out, s, nil
}

func matchCapture(pattern string, data []byte) (*regexp.Regexp, []int, int, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, nil, 0, err
	}
	group := -1
	for i, n := range re.SubexpNames() {
		if n == "value" {
			if group >= 0 {
				return nil, nil, 0, fmt.Errorf("pattern has multiple value captures")
			}
			group = i
		}
	}
	if group < 1 {
		return nil, nil, 0, fmt.Errorf("pattern must contain a named value capture")
	}
	matches := re.FindAllSubmatchIndex(data, -1)
	if len(matches) != 1 {
		return nil, nil, 0, fmt.Errorf("pattern matched %d times", len(matches))
	}
	if matches[0][2*group] < 0 {
		return nil, nil, 0, fmt.Errorf("value capture did not participate in match")
	}
	return re, matches[0], group, nil
}

func readMatch(pattern string, data []byte) (Scalar, error) {
	_, m, g, err := matchCapture(pattern, data)
	if err != nil {
		return Scalar{}, err
	}
	return Scalar{Text: string(data[m[2*g]:m[2*g+1]]), Kind: String}, nil
}

func updateMatch(pattern string, data []byte, value string) ([]byte, Scalar, error) {
	_, m, g, err := matchCapture(pattern, data)
	if err != nil {
		return nil, Scalar{}, err
	}
	out := append([]byte(nil), data[:m[2*g]]...)
	out = append(out, value...)
	out = append(out, data[m[2*g+1]:]...)
	return out, Scalar{Text: value, Kind: String}, nil
}

type jsonNodeKind int

const (
	jsonObject jsonNodeKind = iota
	jsonArray
	jsonString
	jsonNumber
	jsonBoolean
	jsonNull
)

type jsonNode struct {
	start, end int
	kind       jsonNodeKind
	close      int
	firstKey   int
	members    map[string][]*jsonNode
}

type jsonScanner struct {
	data []byte
	pos  int
}

func (s *jsonScanner) ws() {
	for s.pos < len(s.data) && unicode.IsSpace(rune(s.data[s.pos])) {
		s.pos++
	}
}
func (s *jsonScanner) stringToken() (string, int, int, error) {
	s.ws()
	start := s.pos
	if start >= len(s.data) || s.data[start] != '"' {
		return "", 0, 0, fmt.Errorf("expected JSON string")
	}
	s.pos++
	esc := false
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		s.pos++
		if esc {
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if c == '"' {
			var v string
			if err := json.Unmarshal(s.data[start:s.pos], &v); err != nil {
				return "", 0, 0, err
			}
			return v, start, s.pos, nil
		}
	}
	return "", 0, 0, fmt.Errorf("unterminated JSON string")
}
func (s *jsonScanner) value() (*jsonNode, error) {
	s.ws()
	start := s.pos
	if start >= len(s.data) {
		return nil, fmt.Errorf("missing JSON value")
	}
	switch s.data[s.pos] {
	case '{':
		return s.object()
	case '[':
		return s.array()
	case '"':
		_, a, b, err := s.stringToken()
		return &jsonNode{start: a, end: b, kind: jsonString, firstKey: -1}, err
	default:
		s.pos++
		for s.pos < len(s.data) && !strings.ContainsRune(",}] \t\r\n", rune(s.data[s.pos])) {
			s.pos++
		}
		raw := string(s.data[start:s.pos])
		if raw == "true" || raw == "false" {
			return &jsonNode{start: start, end: s.pos, kind: jsonBoolean, firstKey: -1}, nil
		}
		if raw == "null" {
			return &jsonNode{start: start, end: s.pos, kind: jsonNull, firstKey: -1}, nil
		}
		return &jsonNode{start: start, end: s.pos, kind: jsonNumber, firstKey: -1}, nil
	}
}

func (s *jsonScanner) object() (*jsonNode, error) {
	start := s.pos
	s.pos++
	s.ws()
	node := &jsonNode{start: start, kind: jsonObject, firstKey: -1, members: map[string][]*jsonNode{}}
	if s.pos < len(s.data) && s.data[s.pos] == '}' {
		node.close = s.pos
		s.pos++
		node.end = s.pos
		return node, nil
	}
	for {
		s.ws()
		key, keyStart, _, err := s.stringToken()
		if err != nil {
			return nil, err
		}
		if node.firstKey < 0 {
			node.firstKey = keyStart
		}
		s.ws()
		if s.pos >= len(s.data) || s.data[s.pos] != ':' {
			return nil, fmt.Errorf("expected colon")
		}
		s.pos++
		value, err := s.value()
		if err != nil {
			return nil, err
		}
		node.members[key] = append(node.members[key], value)
		s.ws()
		if s.pos < len(s.data) && s.data[s.pos] == ',' {
			s.pos++
			continue
		}
		if s.pos < len(s.data) && s.data[s.pos] == '}' {
			node.close = s.pos
			s.pos++
			node.end = s.pos
			return node, nil
		}
		return nil, fmt.Errorf("invalid JSON object")
	}
}

func (s *jsonScanner) array() (*jsonNode, error) {
	start := s.pos
	s.pos++
	s.ws()
	if s.pos < len(s.data) && s.data[s.pos] == ']' {
		s.pos++
		return &jsonNode{start: start, end: s.pos, kind: jsonArray, firstKey: -1}, nil
	}
	for {
		if _, err := s.value(); err != nil {
			return nil, err
		}
		s.ws()
		if s.pos < len(s.data) && s.data[s.pos] == ',' {
			s.pos++
			continue
		}
		if s.pos < len(s.data) && s.data[s.pos] == ']' {
			s.pos++
			return &jsonNode{start: start, end: s.pos, kind: jsonArray, firstKey: -1}, nil
		}
		return nil, fmt.Errorf("invalid JSON array")
	}
}

func decodePointer(pointer string) ([]string, error) {
	if pointer == "" || pointer[0] != '/' {
		return nil, fmt.Errorf("JSON selector must be an RFC 6901 pointer")
	}
	parts := strings.Split(pointer[1:], "/")
	for i, p := range parts {
		var decoded strings.Builder
		for j := 0; j < len(p); j++ {
			if p[j] != '~' {
				decoded.WriteByte(p[j])
				continue
			}
			if j+1 >= len(p) || (p[j+1] != '0' && p[j+1] != '1') {
				return nil, fmt.Errorf("invalid RFC 6901 escape in JSON selector")
			}
			j++
			if p[j] == '0' {
				decoded.WriteByte('~')
			} else {
				decoded.WriteByte('/')
			}
		}
		parts[i] = decoded.String()
	}
	return parts, nil
}

func parseJSON(data []byte) (*jsonNode, error) {
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON document")
	}
	s := jsonScanner{data: data}
	root, err := s.value()
	if err != nil {
		return nil, err
	}
	s.ws()
	if s.pos != len(data) {
		return nil, fmt.Errorf("invalid trailing JSON content")
	}
	return root, nil
}

func jsonMember(object *jsonNode, key string) (*jsonNode, bool, error) {
	if object.kind != jsonObject {
		return nil, false, fmt.Errorf("JSON pointer parent is not an object")
	}
	values := object.members[key]
	if len(values) > 1 {
		return nil, false, fmt.Errorf("JSON pointer member %q is duplicated", key)
	}
	if len(values) == 0 {
		return nil, false, nil
	}
	return values[0], true, nil
}

func findJSONNode(data []byte, pointer string) (*jsonNode, error) {
	parts, err := decodePointer(pointer)
	if err != nil {
		return nil, err
	}
	node, err := parseJSON(data)
	if err != nil {
		return nil, err
	}
	if node.kind != jsonObject {
		return nil, fmt.Errorf("JSON pointer root is not an object")
	}
	for _, part := range parts {
		next, found, err := jsonMember(node, part)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("JSON pointer segment %q not found", part)
		}
		node = next
	}
	return node, nil
}

func scalarFromJSONNode(data []byte, node *jsonNode) (Scalar, error) {
	var kind Kind
	switch node.kind {
	case jsonString:
		kind = String
	case jsonNumber:
		kind = Number
	case jsonBoolean:
		kind = Boolean
	case jsonObject:
		return Scalar{}, fmt.Errorf("JSON objects are not scalar")
	case jsonArray:
		return Scalar{}, fmt.Errorf("JSON arrays are not supported")
	default:
		return Scalar{}, fmt.Errorf("JSON null is not a supported scalar")
	}
	var text string
	switch kind {
	case String:
		if err := json.Unmarshal(data[node.start:node.end], &text); err != nil {
			return Scalar{}, err
		}
	default:
		text = string(data[node.start:node.end])
	}
	return Scalar{Text: text, Kind: kind}, nil
}

func findJSONScalar(data []byte, pointer string) (*jsonNode, Scalar, error) {
	node, err := findJSONNode(data, pointer)
	if err != nil {
		return nil, Scalar{}, err
	}
	scalar, err := scalarFromJSONNode(data, node)
	return node, scalar, err
}

func updateJSON(selector string, data []byte, value, policy, valueType string) ([]byte, Scalar, error) {
	node, current, err := findJSONScalar(data, selector)
	if err != nil {
		if policy != "add" {
			return nil, Scalar{}, err
		}
		return addJSON(selector, data, value, valueType)
	}
	next, err := scalarFromText(value, string(current.Kind))
	if err != nil {
		return nil, Scalar{}, err
	}
	enc := encodedScalar(next, "json")
	out := append([]byte(nil), data[:node.start]...)
	out = append(out, enc...)
	out = append(out, data[node.end:]...)
	if !json.Valid(out) {
		return nil, Scalar{}, fmt.Errorf("updated JSON is invalid")
	}
	return out, next, nil
}

func addJSON(selector string, data []byte, value, valueType string) ([]byte, Scalar, error) {
	parts, err := decodePointer(selector)
	if err != nil {
		return nil, Scalar{}, err
	}
	if valueType == "" {
		return nil, Scalar{}, fmt.Errorf("added JSON values require an explicit scalar type")
	}
	root, err := parseJSON(data)
	if err != nil {
		return nil, Scalar{}, err
	}
	if root.kind != jsonObject {
		return nil, Scalar{}, fmt.Errorf("JSON pointer root is not an object")
	}
	parent := root
	for _, p := range parts[:len(parts)-1] {
		child, found, err := jsonMember(parent, p)
		if err != nil {
			return nil, Scalar{}, err
		}
		if !found {
			return nil, Scalar{}, fmt.Errorf("JSON parent %q does not exist", p)
		}
		if child.kind != jsonObject {
			return nil, Scalar{}, fmt.Errorf("JSON parent %q is not an object", p)
		}
		parent = child
	}
	leaf := parts[len(parts)-1]
	if _, found, err := jsonMember(parent, leaf); err != nil {
		return nil, Scalar{}, err
	} else if found {
		return nil, Scalar{}, fmt.Errorf("JSON value already exists")
	}
	scalar, err := scalarFromText(value, valueType)
	if err != nil {
		return nil, Scalar{}, err
	}
	encoded := encodedScalar(scalar, "json")
	if !json.Valid([]byte(encoded)) {
		return nil, Scalar{}, fmt.Errorf("invalid JSON scalar %q", value)
	}
	encodedKey, _ := json.Marshal(leaf)

	triviaStart := parent.close
	for triviaStart > parent.start+1 && unicode.IsSpace(rune(data[triviaStart-1])) {
		triviaStart--
	}
	closingTrivia := data[triviaStart:parent.close]
	pretty := bytes.Contains(closingTrivia, []byte("\n")) || bytes.Contains(closingTrivia, []byte("\r"))
	newline := []byte("\n")
	if bytes.Contains(data, []byte("\r\n")) {
		newline = []byte("\r\n")
	}
	memberIndent := []byte("  ")
	if parent.firstKey >= 0 {
		lineStart := bytes.LastIndexByte(data[:parent.firstKey], '\n') + 1
		if indent := data[lineStart:parent.firstKey]; len(indent) > 0 {
			memberIndent = indent
		}
	} else if pretty {
		lineStart := bytes.LastIndexByte(data[:parent.close], '\n') + 1
		memberIndent = append(append([]byte(nil), data[lineStart:parent.close]...), ' ', ' ')
	}

	insertion := make([]byte, 0, len(encodedKey)+len(encoded)+len(memberIndent)+8)
	if parent.firstKey >= 0 {
		insertion = append(insertion, ',')
	}
	if pretty {
		insertion = append(insertion, newline...)
		insertion = append(insertion, memberIndent...)
	}
	insertion = append(insertion, encodedKey...)
	if pretty {
		insertion = append(insertion, ':', ' ')
	} else {
		insertion = append(insertion, ':')
	}
	insertion = append(insertion, encoded...)

	out := append([]byte(nil), data[:triviaStart]...)
	out = append(out, insertion...)
	out = append(out, data[triviaStart:]...)
	if _, err := parseJSON(out); err != nil {
		return nil, Scalar{}, fmt.Errorf("updated JSON is invalid: %w", err)
	}
	return out, scalar, nil
}

func unsupportedSelector(format, selector, reason string) error {
	return fmt.Errorf("unsupported %s selector %q (%s); use match for this source shape", format, selector, reason)
}

func yamlPath(selector string) ([]string, error) {
	parts := strings.Split(selector, ".")
	for _, part := range parts {
		if part == "" {
			return nil, unsupportedSelector("YAML", selector, "selectors must be non-empty dot-separated mapping keys")
		}
	}
	return parts, nil
}
func yamlRoot(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) != 1 {
		return nil, fmt.Errorf("empty YAML document")
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, unsupportedSelector("YAML", "", "the document root is not a mapping")
	}
	return &doc, nil
}
func yamlNodeAt(doc *yaml.Node, parts []string) (*yaml.Node, bool, error) {
	node := doc.Content[0]
	for _, part := range parts {
		if node.Kind != yaml.MappingNode {
			return nil, false, unsupportedSelector("YAML", strings.Join(parts, "."), fmt.Sprintf("parent of %q is not a mapping", part))
		}
		var matches []*yaml.Node
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return nil, false, unsupportedSelector("YAML", strings.Join(parts, "."), "complex or non-string mapping keys are not supported")
			}
			if key.Value == part {
				matches = append(matches, node.Content[i+1])
			}
		}
		if len(matches) > 1 {
			return nil, false, fmt.Errorf("YAML key %q is duplicated", part)
		}
		if len(matches) == 0 {
			return node, false, nil
		}
		node = matches[0]
	}
	return node, true, nil
}
func scalarFromYAML(n *yaml.Node) (Scalar, error) {
	if n.Kind != yaml.ScalarNode {
		return Scalar{}, unsupportedSelector("YAML", "", "selection is not a scalar string, number, or boolean")
	}
	switch n.Tag {
	case "!!bool":
		if n.Value != "true" && n.Value != "false" {
			return Scalar{}, fmt.Errorf("unsupported YAML boolean %q", n.Value)
		}
		return Scalar{Text: n.Value, Kind: Boolean}, nil
	case "!!int", "!!float":
		if _, err := scalarFromText(n.Value, string(Number)); err != nil {
			return Scalar{}, fmt.Errorf("unsupported YAML number %q", n.Value)
		}
		return Scalar{Text: n.Value, Kind: Number}, nil
	case "!!str":
		return Scalar{Text: n.Value, Kind: String}, nil
	default:
		return Scalar{}, unsupportedSelector("YAML", "", fmt.Sprintf("tag %s is not a scalar string, number, or boolean", n.Tag))
	}
}

func setYAMLScalar(node *yaml.Node, scalar Scalar) {
	node.Kind = yaml.ScalarNode
	node.Style = 0
	node.Value = scalar.Text
	switch scalar.Kind {
	case String:
		node.Tag = "!!str"
	case Boolean:
		node.Tag = "!!bool"
	case Number:
		node.Tag = "!!float"
		if !strings.ContainsAny(scalar.Text, ".eE") {
			node.Tag = "!!int"
		}
	}
}

func readYAML(selector string, data []byte) (Scalar, error) {
	parts, err := yamlPath(selector)
	if err != nil {
		return Scalar{}, err
	}
	doc, err := yamlRoot(data)
	if err != nil {
		return Scalar{}, err
	}
	n, found, err := yamlNodeAt(doc, parts)
	if err != nil {
		return Scalar{}, err
	}
	if !found {
		return Scalar{}, fmt.Errorf("YAML key %q does not exist", selector)
	}
	return scalarFromYAML(n)
}
func updateYAML(selector string, data []byte, value, policy, valueType string) ([]byte, Scalar, error) {
	parts, err := yamlPath(selector)
	if err != nil {
		return nil, Scalar{}, err
	}
	doc, err := yamlRoot(data)
	if err != nil {
		return nil, Scalar{}, err
	}
	n, found, findErr := yamlNodeAt(doc, parts)
	if findErr != nil {
		return nil, Scalar{}, findErr
	}
	var s Scalar
	if found {
		cur, e := scalarFromYAML(n)
		if e != nil {
			return nil, Scalar{}, e
		}
		s, e = scalarFromText(value, string(cur.Kind))
		if e != nil {
			return nil, Scalar{}, e
		}
		setYAMLScalar(n, s)
	} else {
		if policy != "add" {
			return nil, Scalar{}, fmt.Errorf("YAML key %q does not exist", selector)
		}
		parent, parentFound, e := yamlNodeAt(doc, parts[:len(parts)-1])
		if e != nil {
			return nil, Scalar{}, e
		}
		if !parentFound || parent.Kind != yaml.MappingNode {
			return nil, Scalar{}, fmt.Errorf("YAML parent for %q does not exist", selector)
		}
		if valueType == "" {
			return nil, Scalar{}, fmt.Errorf("added YAML values require an explicit scalar type")
		}
		s, e = scalarFromText(value, valueType)
		if e != nil {
			return nil, Scalar{}, e
		}
		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: parts[len(parts)-1]}
		val := &yaml.Node{}
		setYAMLScalar(val, s)
		parent.Content = append(parent.Content, key, val)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, Scalar{}, err
	}
	if _, err := yamlRoot(out); err != nil {
		return nil, Scalar{}, fmt.Errorf("updated YAML is invalid: %w", err)
	}
	return out, s, nil
}

func tomlParts(selector string) ([]string, error) {
	var parsed map[string]any
	if strings.TrimSpace(selector) == "" || strings.ContainsAny(selector, "\r\n") {
		return nil, unsupportedSelector("TOML", selector, "selector is not a dotted key")
	}
	if err := toml.Unmarshal([]byte(selector+" = true\n"), &parsed); err != nil {
		return nil, unsupportedSelector("TOML", selector, "selector is not valid TOML dotted-key syntax")
	}
	var parts []string
	var current any = parsed
	for {
		mapping, ok := current.(map[string]any)
		if !ok || len(mapping) != 1 {
			break
		}
		for key, value := range mapping {
			parts = append(parts, key)
			current = value
		}
	}
	if value, ok := current.(bool); !ok || !value || len(parts) == 0 {
		return nil, unsupportedSelector("TOML", selector, "selector is not a dotted key")
	}
	return parts, nil
}

func scalarFromTOML(value any) (Scalar, error) {
	switch value := value.(type) {
	case string:
		return Scalar{Text: value, Kind: String}, nil
	case bool:
		return Scalar{Text: strconv.FormatBool(value), Kind: Boolean}, nil
	case int64:
		return Scalar{Text: strconv.FormatInt(value, 10), Kind: Number}, nil
	case uint64:
		return Scalar{Text: strconv.FormatUint(value, 10), Kind: Number}, nil
	case float64:
		text := strconv.FormatFloat(value, 'g', -1, 64)
		if _, err := scalarFromText(text, string(Number)); err != nil {
			return Scalar{}, unsupportedSelector("TOML", "", "non-finite numbers are not supported")
		}
		return Scalar{Text: text, Kind: Number}, nil
	case time.Time, time.Duration:
		return Scalar{}, unsupportedSelector("TOML", "", "date and time values are not supported")
	default:
		return Scalar{}, unsupportedSelector("TOML", "", "selection is not a scalar string, number, or boolean")
	}
}

func tomlDocument(data []byte) (map[string]any, error) {
	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	return document, nil
}

func tomlLookup(document map[string]any, parts []string) (any, bool, error) {
	current := document
	for i, part := range parts {
		value, found := current[part]
		if !found {
			return current, false, nil
		}
		if i == len(parts)-1 {
			return value, true, nil
		}
		next, ok := value.(map[string]any)
		if !ok {
			return nil, false, unsupportedSelector("TOML", strings.Join(parts, "."), fmt.Sprintf("parent %q is not a table", part))
		}
		current = next
	}
	return nil, false, nil
}

func readTOML(selector string, data []byte) (Scalar, error) {
	parts, err := tomlParts(selector)
	if err != nil {
		return Scalar{}, err
	}
	document, err := tomlDocument(data)
	if err != nil {
		return Scalar{}, err
	}
	value, found, err := tomlLookup(document, parts)
	if err != nil {
		return Scalar{}, err
	}
	if !found {
		return Scalar{}, fmt.Errorf("TOML key %q does not exist", selector)
	}
	return scalarFromTOML(value)
}
func updateTOML(selector string, data []byte, value, policy, valueType string) ([]byte, Scalar, error) {
	parts, err := tomlParts(selector)
	if err != nil {
		return nil, Scalar{}, err
	}
	document, err := tomlDocument(data)
	if err != nil {
		return nil, Scalar{}, err
	}
	current, found, err := tomlLookup(document, parts)
	if err != nil {
		return nil, Scalar{}, err
	}
	var scalar Scalar
	if found {
		existing, err := scalarFromTOML(current)
		if err != nil {
			return nil, Scalar{}, err
		}
		scalar, err = scalarFromText(value, string(existing.Kind))
		if err != nil {
			return nil, Scalar{}, err
		}
	} else {
		if policy != "add" {
			return nil, Scalar{}, fmt.Errorf("TOML key %q does not exist", selector)
		}
		if valueType == "" {
			return nil, Scalar{}, fmt.Errorf("added TOML values require an explicit scalar type")
		}
		scalar, err = scalarFromText(value, valueType)
		if err != nil {
			return nil, Scalar{}, err
		}
	}
	parentValue, parentFound, err := tomlLookup(document, parts[:len(parts)-1])
	if err != nil {
		return nil, Scalar{}, err
	}
	var parent map[string]any
	if len(parts) == 1 {
		parent = document
	} else if !parentFound {
		return nil, Scalar{}, fmt.Errorf("TOML parent for %q does not exist", selector)
	} else {
		var ok bool
		parent, ok = parentValue.(map[string]any)
		if !ok {
			return nil, Scalar{}, unsupportedSelector("TOML", selector, "parent is not a table")
		}
	}
	switch scalar.Kind {
	case String:
		parent[parts[len(parts)-1]] = scalar.Text
	case Boolean:
		parent[parts[len(parts)-1]] = scalar.Text == "true"
	case Number:
		if strings.ContainsAny(scalar.Text, ".eE") {
			number, _ := strconv.ParseFloat(scalar.Text, 64)
			parent[parts[len(parts)-1]] = number
		} else {
			number, parseErr := strconv.ParseInt(scalar.Text, 10, 64)
			if parseErr != nil {
				return nil, Scalar{}, fmt.Errorf("TOML integer %q is out of range", scalar.Text)
			}
			parent[parts[len(parts)-1]] = number
		}
	}
	out, err := toml.Marshal(document)
	if err != nil {
		return nil, Scalar{}, err
	}
	if _, err := tomlDocument(out); err != nil {
		return nil, Scalar{}, fmt.Errorf("updated TOML is invalid: %w", err)
	}
	return out, scalar, nil
}
