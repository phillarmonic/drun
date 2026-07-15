package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/filevalue"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

func TestExecuteFileValueGetCheckUpdateAndDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gradle.properties")
	if err := os.WriteFile(path, []byte("pluginVersion=1.0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &ExecutionContext{Variables: map[string]string{"next": "1.0.2"}}
	out := &bytes.Buffer{}
	e := NewEngine(out)
	if err := e.executeFileValue(&statement.FileValue{Operation: "get", Format: "property", Selector: "pluginVersion", Target: path, CaptureVar: "current"}, ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Variables["current"] != "1.0.1" {
		t.Fatalf("capture = %q", ctx.Variables["current"])
	}
	err := e.executeFileValue(&statement.FileValue{Operation: "check", Format: "property", Selector: "pluginVersion", Target: path, Comparison: "equals", Expected: "9.9.9"}, ctx)
	if err == nil || !strings.Contains(err.Error(), "actual \"1.0.1\"") || !strings.Contains(err.Error(), path) {
		t.Fatalf("check error = %v", err)
	}
	e.SetDryRun(true)
	if err := e.executeFileValue(&statement.FileValue{Operation: "update", Format: "property", Selector: "pluginVersion", Target: path, Value: "{$next}", MissingPolicy: "fail"}, ctx); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "pluginVersion=1.0.1\n" {
		t.Fatal("dry run mutated file")
	}
	e.SetDryRun(false)
	if err := e.executeFileValue(&statement.FileValue{Operation: "update", Format: "property", Selector: "pluginVersion", Target: path, Value: "{$next}", MissingPolicy: "fail"}, ctx); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != "pluginVersion=1.0.2\n" {
		t.Fatalf("update = %q", data)
	}
}

func TestProjectVersionCheckAndUpdateUseCurrentDrunFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "release.drun")
	source := `version: 2.0
project "demo" version "1.0.1":
task "release":
  requires $next
  check project version equals "1.0.1"
  update project version to "{$next}"
`
	if err := os.WriteFile(path, []byte(source), 0o640); err != nil {
		t.Fatal(err)
	}
	p := parser.NewParser(lexer.NewLexer(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	engine := NewEngine(&bytes.Buffer{})
	engine.SetDryRun(true)
	if err := engine.ExecuteWithParamsAndFile(program, "release", map[string]string{"next": "1.0.2"}, path); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if actual, _ := os.ReadFile(path); string(actual) != source {
		t.Fatalf("dry run changed spec:\n%s", actual)
	}
	engine.SetDryRun(false)
	if err := engine.ExecuteWithParamsAndFile(program, "release", map[string]string{"next": "1.0.2"}, path); err != nil {
		t.Fatalf("update: %v", err)
	}
	actual, _ := os.ReadFile(path)
	if !strings.Contains(string(actual), `project "demo" version "1.0.2":`) {
		t.Fatalf("project version was not updated:\n%s", actual)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}

func TestFileValueDocumentedFormatsThroughParserAndEngine(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"gradle.properties": "pluginVersion=1.0.1\n",
		"package.json":      "{\n  \"version\": \"1.0.1\"\n}\n",
		"Chart.yaml":        "chart:\n  appVersion: 1.0.1\n",
		"Cargo.toml":        "[package]\nversion = \"1.0.1\"\n",
		"VERSION.txt":       "VERSION=1.0.1\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	source := `version: 2.0
task "versions":
  requires $dir
  requires $next
  requires $property_selector
  get property "{$property_selector}" from "{$dir}/gradle.properties" as $plugin_version
  check property "pluginVersion" in "{$dir}/gradle.properties" equals "1.0.1"
  check property "pluginVersion" in "{$dir}/gradle.properties" differs from "9.9.9"
  update property "pluginVersion" in "{$dir}/gradle.properties" to "{$next}" or fail
  get json "/version" from "{$dir}/package.json" as $package_version
  update json "/version" in "{$dir}/package.json" to "{$next}" or fail
  get yaml "chart.appVersion" from "{$dir}/Chart.yaml" as $chart_version
  update yaml "chart.appVersion" in "{$dir}/Chart.yaml" to "{$next}" or fail
  get toml "package.version" from "{$dir}/Cargo.toml" as $crate_version
  update toml "package.version" in "{$dir}/Cargo.toml" to "{$next}" or fail
  get match "(?m)^VERSION=(?P<value>[^\\r\\n]+)$" from "{$dir}/VERSION.txt" as $version
  update match "(?m)^VERSION=(?P<value>[^\\r\\n]+)$" in "{$dir}/VERSION.txt" to "{$next}" or fail
`
	l := lexer.NewLexer(source)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	params := map[string]string{"dir": dir, "next": "1.0.2", "property_selector": "pluginVersion"}
	before := make(map[string][]byte, len(files))
	for name := range files {
		before[name], _ = os.ReadFile(filepath.Join(dir, name))
	}
	engine := NewEngine(&bytes.Buffer{})
	engine.SetDryRun(true)
	if err := engine.ExecuteWithParams(program, "versions", params); err != nil {
		t.Fatalf("dry-run execution: %v", err)
	}
	for name, want := range before {
		actual, _ := os.ReadFile(filepath.Join(dir, name))
		if !bytes.Equal(actual, want) {
			t.Fatalf("dry run changed %s", name)
		}
	}

	engine.SetDryRun(false)
	if err := engine.ExecuteWithParams(program, "versions", params); err != nil {
		t.Fatalf("execution: %v", err)
	}
	selectors := map[string]struct {
		format   string
		selector string
	}{
		"gradle.properties": {format: "property", selector: "pluginVersion"},
		"package.json":      {format: "json", selector: "/version"},
		"Chart.yaml":        {format: "yaml", selector: "chart.appVersion"},
		"Cargo.toml":        {format: "toml", selector: "package.version"},
		"VERSION.txt":       {format: "match", selector: `(?m)^VERSION=(?P<value>[^\r\n]+)$`},
	}
	after := make(map[string][]byte, len(files))
	for name, selected := range selectors {
		path := filepath.Join(dir, name)
		value, err := filevalue.ReadFile(selected.format, selected.selector, path)
		if err != nil || value.Text != "1.0.2" {
			t.Fatalf("%s value = %#v, %v", name, value, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %v", name, info.Mode().Perm())
		}
		after[name], _ = os.ReadFile(path)
	}

	idempotentSource := strings.Replace(source, `equals "1.0.1"`, `equals "1.0.2"`, 1)
	idempotentParser := parser.NewParser(lexer.NewLexer(idempotentSource))
	idempotentProgram := idempotentParser.ParseProgram()
	if len(idempotentParser.Errors()) > 0 {
		t.Fatalf("idempotent parser errors: %v", idempotentParser.Errors())
	}
	if err := engine.ExecuteWithParams(idempotentProgram, "versions", params); err != nil {
		t.Fatalf("idempotent execution: %v", err)
	}
	for name, want := range after {
		actual, _ := os.ReadFile(filepath.Join(dir, name))
		if !bytes.Equal(actual, want) {
			t.Fatalf("second execution changed %s", name)
		}
	}
}

func TestFileValueFailedPreconditionsNeverModifyFilesThroughParserAndEngine(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		selector string
		content  string
	}{
		{name: "property missing", format: "property", selector: "missing", content: "version=1\n"},
		{name: "json collection", format: "json", selector: "/versions", content: "{\"versions\":[1,2]}\n"},
		{name: "yaml collection", format: "yaml", selector: "release.versions", content: "release:\n  versions: [1, 2]\n"},
		{name: "toml collection", format: "toml", selector: "release.versions", content: "[release]\nversions = [1, 2]\n"},
		{name: "regex ambiguous", format: "match", selector: `(?m)^VERSION=(?P<value>[^\r\n]+)$`, content: "VERSION=1\nVERSION=2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "value.txt")
			original := []byte(tt.content)
			if err := os.WriteFile(path, original, 0o640); err != nil {
				t.Fatal(err)
			}
			source := fmt.Sprintf("version: 2.0\ntask \"bad\":\n  update %s %q in %q to \"next\" or fail\n", tt.format, tt.selector, path)
			l := lexer.NewLexer(source)
			p := parser.NewParser(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			if err := NewEngine(&bytes.Buffer{}).Execute(program, "bad"); err == nil {
				t.Fatal("expected precondition failure")
			}
			actual, _ := os.ReadFile(path)
			if !bytes.Equal(actual, original) {
				t.Fatalf("file changed: %q", actual)
			}
			info, _ := os.Stat(path)
			if info.Mode().Perm() != 0o640 {
				t.Fatalf("mode changed: %v", info.Mode().Perm())
			}
		})
	}
}
