package filevalue

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPropertiesExactKeysAndSeparators(t *testing.T) {
	tests := []struct {
		name string
		line string
		key  string
		want string
	}{
		{name: "equals", line: "pluginVersion = 1.0.1", key: "pluginVersion", want: "1.0.1"},
		{name: "colon", line: "pluginVersion:\t1.0.1", key: "pluginVersion", want: "1.0.1"},
		{name: "whitespace", line: "  pluginVersion\t  1.0.1", key: "pluginVersion", want: "1.0.1"},
		{name: "escaped key", line: `plugin\:version = 1.0.1`, key: "plugin:version", want: "1.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := Read("property", tt.key, []byte(tt.line+"\n"))
			if err != nil || value.Text != tt.want {
				t.Fatalf("read = %#v, %v", value, err)
			}
		})
	}

	data := []byte("pluginVersionSuffix=wrong\npluginVersion=right\n")
	value, err := Read("property", "pluginVersion", data)
	if err != nil || value.Text != "right" {
		t.Fatalf("exact read = %#v, %v", value, err)
	}
}

func TestPropertiesUpdatePreservesLayoutCRLFAndIsIdempotent(t *testing.T) {
	data := []byte("# plugin\r\npluginVersion :\t1.0.1\r\nother=x\r\n")
	want := []byte("# plugin\r\npluginVersion :\t1.0.2\r\nother=x\r\n")
	updated, _, err := Update("property", "pluginVersion", data, "1.0.2", "fail", "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(updated, want) {
		t.Fatalf("layout changed: %q", updated)
	}
	idempotent, _, err := Update("property", "pluginVersion", updated, "1.0.2", "fail", "")
	if err != nil || !bytes.Equal(idempotent, updated) {
		t.Fatalf("idempotent update = %q, %v", idempotent, err)
	}
}

func TestPropertiesAddUsesExistingNewlineStyle(t *testing.T) {
	data := []byte("pluginVersion=1.0.1\r\n")
	added, _, err := Update("property", "newKey", data, "yes", "add", "")
	if err != nil || !bytes.Equal(added, []byte("pluginVersion=1.0.1\r\nnewKey=yes\r\n")) {
		t.Fatalf("add = %q, %v", added, err)
	}

	withoutFinalNewline, _, err := Update("property", "newKey", []byte("pluginVersion=1.0.1"), "yes", "add", "")
	if err != nil || !bytes.Equal(withoutFinalNewline, []byte("pluginVersion=1.0.1\nnewKey=yes\n")) {
		t.Fatalf("add without final newline = %q, %v", withoutFinalNewline, err)
	}
}

func TestPropertiesRejectDuplicates(t *testing.T) {
	data := []byte("pluginVersion=1.0.1\npluginVersion:duplicate\n")
	if _, err := Read("property", "pluginVersion", data); err == nil {
		t.Fatal("expected duplicate read error")
	}
	if updated, _, err := Update("property", "pluginVersion", data, "2.0.0", "fail", ""); err == nil || updated != nil {
		t.Fatalf("duplicate update = %q, %v", updated, err)
	}
}

func TestRegexReadAndUpdateOnlyNamedCapture(t *testing.T) {
	data := []byte("header\r\nrelease {\r\n  VERSION=1.0.1 # keep\r\n}\r\nfooter\r\n")
	pattern := `(?m)^  VERSION=(?P<value>[^\r\n ]+) # keep\r?$`

	value, err := Read("match", pattern, data)
	if err != nil || value != (Scalar{Text: "1.0.1", Kind: String}) {
		t.Fatalf("read = %#v, %v", value, err)
	}

	updated, scalar, err := Update("match", pattern, data, "2.0.0", "fail", "")
	if err != nil || scalar != (Scalar{Text: "2.0.0", Kind: String}) {
		t.Fatalf("update scalar = %#v, %v", scalar, err)
	}
	want := []byte("header\r\nrelease {\r\n  VERSION=2.0.0 # keep\r\n}\r\nfooter\r\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("update changed surrounding bytes:\n got: %q\nwant: %q", updated, want)
	}
}

func TestRegexRequiresOneParticipatingNamedValueCapture(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		data    string
	}{
		{name: "missing capture", pattern: `VERSION=([^ ]+)`, data: "VERSION=1"},
		{name: "duplicate capture name", pattern: `(?P<value>A)|(?P<value>B)`, data: "A"},
		{name: "capture did not participate", pattern: `(?P<value>A)?B`, data: "B"},
		{name: "no match", pattern: `VERSION=(?P<value>[^\r\n ]+)`, data: "name=demo"},
		{name: "multiple matches", pattern: `VERSION=(?P<value>[^\r\n ]+)`, data: "VERSION=1\nVERSION=2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if value, err := Read("match", tt.pattern, []byte(tt.data)); err == nil {
				t.Fatalf("read = %#v, expected error", value)
			}
			if updated, _, err := Update("match", tt.pattern, []byte(tt.data), "3", "fail", ""); err == nil || updated != nil {
				t.Fatalf("update = %q, %v; expected safe failure", updated, err)
			}
		})
	}
}

func TestRegexRejectsAddPolicy(t *testing.T) {
	data := []byte("name=demo\n")
	updated, _, err := Update("match", `VERSION=(?P<value>[^ ]+)`, data, "1", "add", "")
	if err == nil || updated != nil {
		t.Fatalf("update = %q, %v; expected additions to be rejected", updated, err)
	}
}

func TestJSONPointerPreservesLayoutAndTypes(t *testing.T) {
	data := []byte("{\n  \"name\": \"demo\",\n  \"version\": \"1.0.1\",\n  \"enabled\": true\n}\n")
	updated, scalar, err := Update("json", "/version", data, "1.0.2", "fail", "")
	if err != nil || scalar.Kind != String {
		t.Fatalf("update = %#v, %v", scalar, err)
	}
	want := strings.Replace(string(data), `"1.0.1"`, `"1.0.2"`, 1)
	if string(updated) != want {
		t.Fatalf("layout changed:\n%s", updated)
	}
	if _, _, err := Update("json", "/enabled", data, "not-bool", "fail", ""); err == nil {
		t.Fatal("expected bool type error")
	}
	added, _, err := Update("json", "/build", data, "7", "add", "number")
	if err != nil {
		t.Fatal(err)
	}
	value, err := Read("json", "/build", added)
	if err != nil || value.Kind != Number || value.Text != "7" {
		t.Fatalf("added = %#v, %v", value, err)
	}
}

func TestJSONRFC6901PointerResolution(t *testing.T) {
	data := []byte(`{"a/b":{"m~n":"value"},"":"empty-key"}`)

	value, err := Read("json", "/a~1b/m~0n", data)
	if err != nil || value != (Scalar{Text: "value", Kind: String}) {
		t.Fatalf("escaped pointer = %#v, %v", value, err)
	}
	value, err = Read("json", "/", data)
	if err != nil || value.Text != "empty-key" {
		t.Fatalf("empty-key pointer = %#v, %v", value, err)
	}
	for _, selector := range []string{"", "a/b", "/bad~", "/bad~2escape"} {
		if _, err := Read("json", selector, data); err == nil {
			t.Errorf("selector %q should be rejected", selector)
		}
	}
}

func TestJSONRejectsArraysContainersDuplicatesAndMissingParents(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		data     string
	}{
		{name: "root array", selector: "/0", data: `["value"]`},
		{name: "array parent", selector: "/items/0", data: `{"items":["value"]}`},
		{name: "array leaf", selector: "/items", data: `{"items":[]}`},
		{name: "object leaf", selector: "/metadata", data: `{"metadata":{"version":"1"}}`},
		{name: "null leaf", selector: "/version", data: `{"version":null}`},
		{name: "duplicate leaf", selector: "/version", data: `{"version":"1","version":"2"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if value, err := Read("json", tt.selector, []byte(tt.data)); err == nil {
				t.Fatalf("read = %#v, expected rejection", value)
			}
		})
	}

	data := []byte("{\n  \"name\": \"demo\"\n}\n")
	if updated, _, err := Update("json", "/metadata/version", data, "1", "add", "string"); err == nil || updated != nil {
		t.Fatalf("missing parent update = %q, %v", updated, err)
	}
	if updated, _, err := Update("json", "/name/version", data, "1", "add", "string"); err == nil || updated != nil {
		t.Fatalf("scalar parent update = %q, %v", updated, err)
	}
}

func TestJSONUpdatesPreserveExistingScalarTypes(t *testing.T) {
	data := []byte(`{"version":"1.0.0","count":1,"enabled":true}`)
	tests := []struct {
		selector string
		value    string
		want     Scalar
	}{
		{selector: "/version", value: "2.0.0", want: Scalar{Text: "2.0.0", Kind: String}},
		{selector: "/count", value: "2.5e2", want: Scalar{Text: "2.5e2", Kind: Number}},
		{selector: "/enabled", value: "false", want: Scalar{Text: "false", Kind: Boolean}},
	}
	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			updated, scalar, err := Update("json", tt.selector, data, tt.value, "fail", "")
			if err != nil || scalar != tt.want {
				t.Fatalf("update scalar = %#v, %v", scalar, err)
			}
			actual, err := Read("json", tt.selector, updated)
			if err != nil || actual != tt.want {
				t.Fatalf("read updated scalar = %#v, %v", actual, err)
			}
		})
	}

	for _, tt := range []struct {
		selector string
		value    string
	}{
		{selector: "/count", value: "two"},
		{selector: "/count", value: "01"},
		{selector: "/enabled", value: "1"},
	} {
		if updated, _, err := Update("json", tt.selector, data, tt.value, "fail", ""); err == nil || updated != nil {
			t.Errorf("type-invalid update %s=%q returned %q, %v", tt.selector, tt.value, updated, err)
		}
	}
}

func TestJSONTypedLeafAddition(t *testing.T) {
	data := []byte("{\n  \"metadata\": {}\n}\n")
	tests := []struct {
		selector  string
		value     string
		valueType string
		want      Scalar
	}{
		{selector: "/name", value: "demo", valueType: "string", want: Scalar{Text: "demo", Kind: String}},
		{selector: "/build", value: "7", valueType: "number", want: Scalar{Text: "7", Kind: Number}},
		{selector: "/enabled", value: "true", valueType: "boolean", want: Scalar{Text: "true", Kind: Boolean}},
		{selector: "/metadata/version", value: "1.2.3", valueType: "string", want: Scalar{Text: "1.2.3", Kind: String}},
	}
	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			updated, scalar, err := Update("json", tt.selector, data, tt.value, "add", tt.valueType)
			if err != nil || scalar != tt.want {
				t.Fatalf("add scalar = %#v, %v", scalar, err)
			}
			actual, err := Read("json", tt.selector, updated)
			if err != nil || actual != tt.want {
				t.Fatalf("read added scalar = %#v, %v", actual, err)
			}
		})
	}

	for _, tt := range []struct {
		value     string
		valueType string
	}{
		{value: "missing-type"},
		{value: "NaN", valueType: "number"},
		{value: "yes", valueType: "boolean"},
		{value: "value", valueType: "object"},
	} {
		if updated, _, err := Update("json", "/new", data, tt.value, "add", tt.valueType); err == nil || updated != nil {
			t.Errorf("invalid typed addition %q as %q returned %q, %v", tt.value, tt.valueType, updated, err)
		}
	}
}

func TestJSONNPMAndComposerVersionEditsPreserveUnrelatedBytes(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "npm LF",
			data: "{\n  \"name\" : \"demo\",\n  \"version\": \"1.0.0\",\n  \"scripts\": { \"test\": \"go test ./...\" }\n}\n",
			want: "{\n  \"name\" : \"demo\",\n  \"version\": \"2.0.0\",\n  \"scripts\": { \"test\": \"go test ./...\" }\n}\n",
		},
		{
			name: "Composer CRLF",
			data: "{\r\n\t\"name\": \"vendor/demo\",\r\n\t\"version\": \"1.0.0\",\r\n\t\"require\": {\"php\": \"^8.3\"}\r\n}\r\n",
			want: "{\r\n\t\"name\": \"vendor/demo\",\r\n\t\"version\": \"2.0.0\",\r\n\t\"require\": {\"php\": \"^8.3\"}\r\n}\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, _, err := Update("json", "/version", []byte(tt.data), "2.0.0", "fail", "")
			if err != nil {
				t.Fatal(err)
			}
			if string(updated) != tt.want {
				t.Fatalf("unrelated JSON bytes changed:\n got: %q\nwant: %q", updated, tt.want)
			}
		})
	}
}

func TestJSONAdditionPreservesExistingLayoutAndNewlineStyle(t *testing.T) {
	data := []byte("{\r\n  \"name\" : \"demo\",\r\n  \"metadata\": {\r\n  }\r\n}\r\n")
	updated, _, err := Update("json", "/metadata/version", data, "1.2.3", "add", "string")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("{\r\n  \"name\" : \"demo\",\r\n  \"metadata\": {\r\n    \"version\": \"1.2.3\"\r\n  }\r\n}\r\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("addition changed existing layout:\n got: %q\nwant: %q", updated, want)
	}

	compact := []byte(`{"name" : "demo"}`)
	updated, _, err = Update("json", "/version", compact, "1.0.0", "add", "string")
	if err != nil || string(updated) != `{"name" : "demo","version":"1.0.0"}` {
		t.Fatalf("compact addition = %q, %v", updated, err)
	}
}

func TestJSONValidationFailureNeverReturnsRewrittenContent(t *testing.T) {
	data := []byte("{\n  \"version\": \"1.0.0\",\n}\n")
	if updated, _, err := Update("json", "/version", data, "2.0.0", "fail", ""); err == nil || updated != nil {
		t.Fatalf("invalid source update = %q, %v", updated, err)
	}
	if updated, _, err := Update("json", "/build", data, "7", "add", "number"); err == nil || updated != nil {
		t.Fatalf("invalid source addition = %q, %v", updated, err)
	}
}

func TestYAMLAndTOMLScalars(t *testing.T) {
	yamlData := []byte("chart:\n  appVersion: 1.0.1\n  enabled: true\n")
	value, err := Read("yaml", "chart.appVersion", yamlData)
	if err != nil || value.Text != "1.0.1" {
		t.Fatalf("yaml read = %#v, %v", value, err)
	}
	updated, _, err := Update("yaml", "chart.appVersion", yamlData, "1.0.2", "fail", "")
	if err != nil {
		t.Fatal(err)
	}
	if v, e := Read("yaml", "chart.appVersion", updated); e != nil || v.Text != "1.0.2" {
		t.Fatalf("yaml update = %#v, %v", v, e)
	}

	tomlData := []byte("[package]\nname = \"demo\"\nversion = \"1.0.1\" # release\n")
	value, err = Read("toml", "package.version", tomlData)
	if err != nil || value.Text != "1.0.1" {
		t.Fatalf("toml read = %#v, %v", value, err)
	}
	updated, _, err = Update("toml", "package.version", tomlData, "1.0.2", "fail", "")
	if err != nil {
		t.Fatal(err)
	}
	if v, e := Read("toml", "package.version", updated); e != nil || v != (Scalar{Text: "1.0.2", Kind: String}) {
		t.Fatalf("toml update = %#v, %v", v, e)
	}
	idempotent, _, err := Update("toml", "package.version", updated, "1.0.2", "fail", "")
	if err != nil || !bytes.Equal(idempotent, updated) {
		t.Fatalf("toml deterministic serialization = %q, %v", idempotent, err)
	}
}

func TestYAMLRetainsScalarTypesAndAddsTypedLeaves(t *testing.T) {
	data := []byte("release:\n  name: demo\n  build: 7\n  enabled: true\n")
	tests := []struct {
		selector string
		value    string
		want     Scalar
	}{
		{selector: "release.name", value: "stable", want: Scalar{Text: "stable", Kind: String}},
		{selector: "release.build", value: "8.5", want: Scalar{Text: "8.5", Kind: Number}},
		{selector: "release.enabled", value: "false", want: Scalar{Text: "false", Kind: Boolean}},
	}
	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			updated, scalar, err := Update("yaml", tt.selector, data, tt.value, "fail", "")
			if err != nil || scalar != tt.want {
				t.Fatalf("update = %#v, %v", scalar, err)
			}
			if actual, err := Read("yaml", tt.selector, updated); err != nil || actual != tt.want {
				t.Fatalf("read updated = %#v, %v", actual, err)
			}
		})
	}

	updated, scalar, err := Update("yaml", "release.channel", data, "beta", "add", "string")
	if err != nil || scalar != (Scalar{Text: "beta", Kind: String}) {
		t.Fatalf("add = %#v, %v", scalar, err)
	}
	if actual, err := Read("yaml", "release.channel", updated); err != nil || actual != scalar {
		t.Fatalf("read added = %#v, %v", actual, err)
	}
	idempotent, _, err := Update("yaml", "release.channel", updated, "beta", "fail", "")
	if err != nil || !bytes.Equal(idempotent, updated) {
		t.Fatalf("yaml deterministic serialization = %q, %v", idempotent, err)
	}
}

func TestYAMLRejectsUnsupportedShapesAndMissingParents(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		data     string
	}{
		{name: "collection", selector: "release.tags", data: "release:\n  tags: [one, two]\n"},
		{name: "complex key", selector: "release.name", data: "release:\n  ? [complex, key]\n  : value\n"},
		{name: "empty path segment", selector: "release..name", data: "release:\n  name: demo\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Read("yaml", tt.selector, []byte(tt.data)); err == nil || !strings.Contains(err.Error(), "use match") {
				t.Fatalf("error = %v", err)
			}
		})
	}

	data := []byte("release:\n  name: demo\n")
	if updated, _, err := Update("yaml", "missing.name", data, "x", "add", "string"); err == nil || updated != nil {
		t.Fatalf("missing parent update = %q, %v", updated, err)
	}
	if updated, _, err := Update("yaml", "release.channel", data, "beta", "add", ""); err == nil || updated != nil {
		t.Fatalf("untyped add = %q, %v", updated, err)
	}
}

func TestTOMLNativeDottedSelectorsAndScalarTypes(t *testing.T) {
	data := []byte("[package.metadata]\n\"release.channel\" = \"stable\"\nbuild = 7\nenabled = true\n")
	tests := []struct {
		selector string
		value    string
		want     Scalar
	}{
		{selector: `package.metadata."release.channel"`, value: "beta", want: Scalar{Text: "beta", Kind: String}},
		{selector: "package.metadata.build", value: "8.5", want: Scalar{Text: "8.5", Kind: Number}},
		{selector: "package.metadata.enabled", value: "false", want: Scalar{Text: "false", Kind: Boolean}},
	}
	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			updated, scalar, err := Update("toml", tt.selector, data, tt.value, "fail", "")
			if err != nil || scalar != tt.want {
				t.Fatalf("update = %#v, %v", scalar, err)
			}
			if actual, err := Read("toml", tt.selector, updated); err != nil || actual != tt.want {
				t.Fatalf("read updated = %#v, %v", actual, err)
			}
		})
	}

	updated, scalar, err := Update("toml", "package.metadata.channel", data, "beta", "add", "string")
	if err != nil || scalar != (Scalar{Text: "beta", Kind: String}) {
		t.Fatalf("add = %#v, %v", scalar, err)
	}
	if actual, err := Read("toml", "package.metadata.channel", updated); err != nil || actual != scalar {
		t.Fatalf("read added = %#v, %v", actual, err)
	}
}

func TestTOMLRejectsUnsupportedShapesAndMissingParents(t *testing.T) {
	data := []byte("[package]\nname = \"demo\"\ntags = [\"one\", \"two\"]\nreleased = 1979-05-27T07:32:00Z\n")
	for _, selector := range []string{"package.tags", "package.released", "package..name"} {
		t.Run(selector, func(t *testing.T) {
			if _, err := Read("toml", selector, data); err == nil || !strings.Contains(err.Error(), "use match") {
				t.Fatalf("error = %v", err)
			}
		})
	}
	if updated, _, err := Update("toml", "missing.name", data, "x", "add", "string"); err == nil || updated != nil {
		t.Fatalf("missing parent update = %q, %v", updated, err)
	}
	if updated, _, err := Update("toml", "package.channel", data, "beta", "add", ""); err == nil || updated != nil {
		t.Fatalf("untyped add = %q, %v", updated, err)
	}
}

func TestUpdateFileIsAtomicAndPreservesModeOnValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gradle.properties")
	original := []byte("pluginVersion=1.0.1\npluginVersion=duplicate\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	initialInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := UpdateFile("property", "pluginVersion", path, "2.0.0", "fail", ""); err == nil {
		t.Fatal("expected duplicate error")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, original) {
		t.Fatal("file changed after validation failure")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != initialInfo.Mode().Perm() {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}

func TestDrunProjectVersionReadAndUpdatePreserveSpecLayout(t *testing.T) {
	data := []byte("# release metadata\r\nproject \"demo\"  version \"1.0.1\": # keep this comment\r\n  info \"ready\"\r\n")
	value, err := Read("drun", "project.version", data)
	if err != nil || value != (Scalar{Text: "1.0.1", Kind: String}) {
		t.Fatalf("read = %#v, %v", value, err)
	}
	updated, scalar, err := Update("drun", "project.version", data, "1.0.2", "fail", "")
	if err != nil || scalar != (Scalar{Text: "1.0.2", Kind: String}) {
		t.Fatalf("update = %#v, %v", scalar, err)
	}
	want := []byte("# release metadata\r\nproject \"demo\"  version \"1.0.2\": # keep this comment\r\n  info \"ready\"\r\n")
	if !bytes.Equal(updated, want) {
		t.Fatalf("layout changed:\n%s", updated)
	}
}

func TestDrunProjectVersionRejectsMissingAmbiguousAndUnsafeUpdates(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "missing", data: "version: 2.0\n"},
		{name: "ambiguous", data: "project \"one\" version \"1\":\nproject \"two\" version \"2\":\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Read("drun", "project.version", []byte(tt.data)); err == nil {
				t.Fatal("expected read error")
			}
		})
	}
	data := []byte("project \"demo\" version \"1\":\n")
	if updated, _, err := Update("drun", "project.version", data, "2", "add", ""); err == nil || updated != nil {
		t.Fatalf("add update = %q, %v", updated, err)
	}
	if updated, _, err := Update("drun", "project.version", data, "bad\"version", "fail", ""); err == nil || updated != nil {
		t.Fatalf("unsafe update = %q, %v", updated, err)
	}
}
