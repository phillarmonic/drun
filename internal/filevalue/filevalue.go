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
	"unicode"

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
	"match": adapterFuncs{read: readMatch, update: func(s string, d []byte, v, p, _ string) ([]byte, Scalar, error) {
		if p == "add" {
			return nil, Scalar{}, fmt.Errorf("regex match updates do not support or add")
		}
		return updateMatch(s, d, v)
	}},
	"json": adapterFuncs{read: func(s string, d []byte) (Scalar, error) { _, v, e := findJSONScalar(d, s); return v, e }, update: updateJSON},
	"yaml": adapterFuncs{read: readYAML, update: updateYAML},
	"toml": adapterFuncs{read: readTOML, update: updateTOML},
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
		if _, err := strconv.ParseFloat(value, 64); err != nil {
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
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "!") {
			idx := strings.IndexAny(line, "=:")
			if idx >= 0 && strings.TrimSpace(line[:idx]) == key {
				vs := idx + 1
				for vs < len(line) && (line[vs] == ' ' || line[vs] == '\t') {
					vs++
				}
				ve := len(strings.TrimSuffix(line, "\r"))
				spans = append(spans, lineSpan{start: start, end: end, valueStart: start + vs, valueEnd: start + ve})
			}
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

type jsonSpan struct {
	start, end int
	kind       Kind
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
func (s *jsonScanner) value(path []string) (jsonSpan, error) {
	s.ws()
	start := s.pos
	if start >= len(s.data) {
		return jsonSpan{}, fmt.Errorf("missing JSON value")
	}
	switch s.data[s.pos] {
	case '{':
		return s.object(path)
	case '[':
		return jsonSpan{}, fmt.Errorf("JSON arrays are not supported")
	case '"':
		_, a, b, e := s.stringToken()
		return jsonSpan{a, b, String}, e
	default:
		s.pos++
		for s.pos < len(s.data) && !strings.ContainsRune(",}] \t\r\n", rune(s.data[s.pos])) {
			s.pos++
		}
		raw := string(s.data[start:s.pos])
		if raw == "true" || raw == "false" {
			return jsonSpan{start, s.pos, Boolean}, nil
		}
		if _, e := strconv.ParseFloat(raw, 64); e == nil {
			return jsonSpan{start, s.pos, Number}, nil
		}
		return jsonSpan{}, fmt.Errorf("unsupported JSON scalar %q", raw)
	}
}
func (s *jsonScanner) skipValue() error {
	s.ws()
	if s.pos >= len(s.data) {
		return fmt.Errorf("missing value")
	}
	switch s.data[s.pos] {
	case '"':
		_, _, _, e := s.stringToken()
		return e
	case '{':
		s.pos++
		s.ws()
		if s.pos < len(s.data) && s.data[s.pos] == '}' {
			s.pos++
			return nil
		}
		for {
			if _, _, _, e := s.stringToken(); e != nil {
				return e
			}
			s.ws()
			if s.pos >= len(s.data) || s.data[s.pos] != ':' {
				return fmt.Errorf("expected colon")
			}
			s.pos++
			if e := s.skipValue(); e != nil {
				return e
			}
			s.ws()
			if s.data[s.pos] == '}' {
				s.pos++
				return nil
			}
			if s.data[s.pos] != ',' {
				return fmt.Errorf("expected comma")
			}
			s.pos++
		}
	case '[':
		s.pos++
		s.ws()
		if s.pos < len(s.data) && s.data[s.pos] == ']' {
			s.pos++
			return nil
		}
		for {
			if e := s.skipValue(); e != nil {
				return e
			}
			s.ws()
			if s.data[s.pos] == ']' {
				s.pos++
				return nil
			}
			if s.data[s.pos] != ',' {
				return fmt.Errorf("expected comma")
			}
			s.pos++
		}
	default:
		s.pos++
		for s.pos < len(s.data) && !strings.ContainsRune(",}] \t\r\n", rune(s.data[s.pos])) {
			s.pos++
		}
		return nil
	}
}
func (s *jsonScanner) object(path []string) (jsonSpan, error) {
	s.pos++
	s.ws()
	if len(path) == 0 {
		return jsonSpan{}, fmt.Errorf("JSON objects are not scalar")
	}
	for {
		s.ws()
		if s.pos >= len(s.data) || s.data[s.pos] == '}' {
			return jsonSpan{}, fmt.Errorf("JSON pointer segment %q not found", path[0])
		}
		key, _, _, err := s.stringToken()
		if err != nil {
			return jsonSpan{}, err
		}
		s.ws()
		if s.pos >= len(s.data) || s.data[s.pos] != ':' {
			return jsonSpan{}, fmt.Errorf("expected colon")
		}
		s.pos++
		if key == path[0] {
			if len(path) == 1 {
				return s.value(nil)
			}
			s.ws()
			if s.pos >= len(s.data) || s.data[s.pos] != '{' {
				return jsonSpan{}, fmt.Errorf("JSON pointer parent %q is not an object", key)
			}
			return s.object(path[1:])
		}
		if err := s.skipValue(); err != nil {
			return jsonSpan{}, err
		}
		s.ws()
		if s.pos < len(s.data) && s.data[s.pos] == ',' {
			s.pos++
			continue
		}
		if s.pos < len(s.data) && s.data[s.pos] == '}' {
			return jsonSpan{}, fmt.Errorf("JSON pointer segment %q not found", path[0])
		}
		return jsonSpan{}, fmt.Errorf("invalid JSON object")
	}
}
func decodePointer(pointer string) ([]string, error) {
	if pointer == "" || pointer[0] != '/' {
		return nil, fmt.Errorf("JSON selector must be an RFC 6901 pointer")
	}
	parts := strings.Split(pointer[1:], "/")
	for i, p := range parts {
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		parts[i] = p
	}
	return parts, nil
}
func findJSONScalar(data []byte, pointer string) (jsonSpan, Scalar, error) {
	parts, err := decodePointer(pointer)
	if err != nil {
		return jsonSpan{}, Scalar{}, err
	}
	s := jsonScanner{data: data}
	sp, err := s.value(parts)
	if err != nil {
		return jsonSpan{}, Scalar{}, err
	}
	var text string
	switch sp.kind {
	case String:
		if err := json.Unmarshal(data[sp.start:sp.end], &text); err != nil {
			return jsonSpan{}, Scalar{}, err
		}
	default:
		text = string(data[sp.start:sp.end])
	}
	return sp, Scalar{Text: text, Kind: sp.kind}, nil
}
func updateJSON(selector string, data []byte, value, policy, valueType string) ([]byte, Scalar, error) {
	sp, current, err := findJSONScalar(data, selector)
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
	out := append([]byte(nil), data[:sp.start]...)
	out = append(out, enc...)
	out = append(out, data[sp.end:]...)
	if !json.Valid(out) {
		return nil, Scalar{}, fmt.Errorf("updated JSON is invalid")
	}
	return out, next, nil
}
func addJSON(selector string, data []byte, value, valueType string) ([]byte, Scalar, error) {
	parts, err := decodePointer(selector)
	if err != nil || len(parts) == 0 {
		return nil, Scalar{}, err
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, Scalar{}, err
	}
	parent := root
	for _, p := range parts[:len(parts)-1] {
		v, ok := parent[p]
		if !ok {
			return nil, Scalar{}, fmt.Errorf("JSON parent %q does not exist", p)
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, Scalar{}, fmt.Errorf("JSON parent %q is not an object", p)
		}
		parent = m
	}
	leaf := parts[len(parts)-1]
	if _, ok := parent[leaf]; ok {
		return nil, Scalar{}, fmt.Errorf("JSON value already exists")
	}
	s, err := scalarFromText(value, valueType)
	if err != nil {
		return nil, Scalar{}, err
	}
	switch s.Kind {
	case String:
		parent[leaf] = s.Text
	case Boolean:
		parent[leaf] = s.Text == "true"
	case Number:
		n, _ := strconv.ParseFloat(s.Text, 64)
		parent[leaf] = n
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, Scalar{}, err
	}
	if bytes.HasSuffix(data, []byte("\n")) {
		out = append(out, '\n')
	}
	return out, s, nil
}

func yamlPath(selector string) []string {
	if selector == "" {
		return nil
	}
	return strings.Split(selector, ".")
}
func yamlRoot(data []byte) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}
	return &doc, nil
}
func yamlNodeAt(doc *yaml.Node, parts []string) (*yaml.Node, *yaml.Node, error) {
	node := doc.Content[0]
	var parent *yaml.Node
	for _, part := range parts {
		if node.Kind != yaml.MappingNode {
			return nil, nil, fmt.Errorf("YAML parent %q is not a mapping", part)
		}
		parent = node
		var next *yaml.Node
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == part {
				next = node.Content[i+1]
				break
			}
		}
		if next == nil {
			return nil, parent, fmt.Errorf("YAML key %q does not exist", part)
		}
		node = next
	}
	return node, parent, nil
}
func scalarFromYAML(n *yaml.Node) (Scalar, error) {
	if n.Kind != yaml.ScalarNode {
		return Scalar{}, fmt.Errorf("YAML selection is not scalar")
	}
	switch n.Tag {
	case "!!bool":
		return Scalar{n.Value, Boolean}, nil
	case "!!int", "!!float":
		return Scalar{n.Value, Number}, nil
	default:
		return Scalar{n.Value, String}, nil
	}
}
func readYAML(selector string, data []byte) (Scalar, error) {
	doc, err := yamlRoot(data)
	if err != nil {
		return Scalar{}, err
	}
	n, _, err := yamlNodeAt(doc, yamlPath(selector))
	if err != nil {
		return Scalar{}, err
	}
	return scalarFromYAML(n)
}
func updateYAML(selector string, data []byte, value, policy, valueType string) ([]byte, Scalar, error) {
	doc, err := yamlRoot(data)
	if err != nil {
		return nil, Scalar{}, err
	}
	parts := yamlPath(selector)
	n, _, findErr := yamlNodeAt(doc, parts)
	var s Scalar
	if findErr == nil {
		cur, e := scalarFromYAML(n)
		if e != nil {
			return nil, Scalar{}, e
		}
		s, e = scalarFromText(value, string(cur.Kind))
		if e != nil {
			return nil, Scalar{}, e
		}
		n.Value = s.Text
		n.Tag = map[Kind]string{String: "!!str", Number: "!!float", Boolean: "!!bool"}[s.Kind]
	} else {
		if policy != "add" {
			return nil, Scalar{}, findErr
		}
		if len(parts) < 1 {
			return nil, Scalar{}, findErr
		}
		parent, _, e := yamlNodeAt(doc, parts[:len(parts)-1])
		if e != nil {
			return nil, Scalar{}, e
		}
		if parent.Kind != yaml.MappingNode {
			return nil, Scalar{}, fmt.Errorf("YAML parent is not a mapping")
		}
		s, e = scalarFromText(value, valueType)
		if e != nil {
			return nil, Scalar{}, e
		}
		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: parts[len(parts)-1]}
		val := &yaml.Node{Kind: yaml.ScalarNode, Tag: map[Kind]string{String: "!!str", Number: "!!float", Boolean: "!!bool"}[s.Kind], Value: s.Text}
		parent.Content = append(parent.Content, key, val)
	}
	out, err := yaml.Marshal(doc)
	return out, s, err
}

func tomlParts(selector string) []string { return strings.Split(selector, ".") }
func tomlFind(selector string, data []byte) (lineSpan, Scalar, error) {
	parts := tomlParts(selector)
	key := parts[len(parts)-1]
	wantSection := strings.Join(parts[:len(parts)-1], ".")
	section := ""
	var found []lineSpan
	var scalar Scalar
	offset := 0
	for _, lineBytes := range bytes.SplitAfter(data, []byte("\n")) {
		line := strings.TrimSuffix(strings.TrimSuffix(string(lineBytes), "\n"), "\r")
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			section = strings.TrimSpace(trim[1 : len(trim)-1])
			offset += len(lineBytes)
			continue
		}
		if trim == "" || strings.HasPrefix(trim, "#") || section != wantSection {
			offset += len(lineBytes)
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 || strings.TrimSpace(line[:idx]) != key {
			offset += len(lineBytes)
			continue
		}
		vs := idx + 1
		for vs < len(line) && (line[vs] == ' ' || line[vs] == '\t') {
			vs++
		}
		ve := len(line)
		if c := strings.Index(line[vs:], " #"); c >= 0 {
			ve = vs + c
		}
		raw := strings.TrimSpace(line[vs:ve])
		start := vs
		end := vs + len(strings.TrimRight(line[vs:ve], " \t"))
		kind := String
		text := raw
		if strings.HasPrefix(raw, "\"") {
			if decoded, e := strconv.Unquote(raw); e == nil {
				text = decoded
			} else {
				return lineSpan{}, Scalar{}, e
			}
		} else if raw == "true" || raw == "false" {
			kind = Boolean
		} else if _, e := strconv.ParseFloat(raw, 64); e == nil {
			kind = Number
		} else {
			return lineSpan{}, Scalar{}, fmt.Errorf("unsupported TOML scalar %q", raw)
		}
		found = append(found, lineSpan{valueStart: offset + start, valueEnd: offset + end})
		scalar = Scalar{text, kind}
		offset += len(lineBytes)
	}
	if len(found) != 1 {
		return lineSpan{}, Scalar{}, fmt.Errorf("TOML key %q matched %d times", selector, len(found))
	}
	return found[0], scalar, nil
}
func readTOML(selector string, data []byte) (Scalar, error) {
	_, s, e := tomlFind(selector, data)
	return s, e
}
func updateTOML(selector string, data []byte, value, policy, valueType string) ([]byte, Scalar, error) {
	sp, cur, err := tomlFind(selector, data)
	if err == nil {
		s, e := scalarFromText(value, string(cur.Kind))
		if e != nil {
			return nil, Scalar{}, e
		}
		enc := encodedScalar(s, "toml")
		out := append([]byte(nil), data[:sp.valueStart]...)
		out = append(out, enc...)
		out = append(out, data[sp.valueEnd:]...)
		return out, s, nil
	}
	if policy != "add" {
		return nil, Scalar{}, err
	}
	parts := tomlParts(selector)
	section := strings.Join(parts[:len(parts)-1], ".")
	key := parts[len(parts)-1]
	if section != "" && !bytes.Contains(data, []byte("["+section+"]")) {
		return nil, Scalar{}, fmt.Errorf("TOML parent %q does not exist", section)
	}
	s, e := scalarFromText(value, valueType)
	if e != nil {
		return nil, Scalar{}, e
	}
	newline := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		newline = "\r\n"
	}
	out := append([]byte(nil), data...)
	if len(out) > 0 && !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, []byte(newline)...)
	}
	out = append(out, []byte(key+" = "+encodedScalar(s, "toml")+newline)...)
	return out, s, nil
}
