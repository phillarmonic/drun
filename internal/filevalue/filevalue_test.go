package filevalue

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPropertiesReadUpdateAddAndDuplicate(t *testing.T) {
	data := []byte("# plugin\r\npluginVersion = 1.0.1\r\nother=x\r\n")
	value, err := Read("property", "pluginVersion", data)
	if err != nil || value.Text != "1.0.1" {
		t.Fatalf("read = %#v, %v", value, err)
	}
	updated, _, err := Update("property", "pluginVersion", data, "1.0.2", "fail", "")
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != "# plugin\r\npluginVersion = 1.0.2\r\nother=x\r\n" {
		t.Fatalf("layout changed: %q", updated)
	}
	added, _, err := Update("property", "newKey", data, "yes", "add", "")
	if err != nil || !bytes.HasSuffix(added, []byte("newKey=yes\r\n")) {
		t.Fatalf("add = %q, %v", added, err)
	}
	_, err = Read("property", "a", []byte("a=1\na=2\n"))
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestRegexNamedCaptureSafety(t *testing.T) {
	data := []byte("prefix VERSION=1.0.1 suffix\n")
	pattern := `VERSION=(?P<value>[^ ]+)`
	updated, _, err := Update("match", pattern, data, "1.0.2", "fail", "")
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != "prefix VERSION=1.0.2 suffix\n" {
		t.Fatalf("update = %q", updated)
	}
	if _, _, err := Update("match", `VERSION=([^ ]+)`, data, "2", "fail", ""); err == nil {
		t.Fatal("expected named capture error")
	}
	if _, _, err := Update("match", `(?P<value>VERSION)`, []byte("VERSION VERSION"), "x", "fail", ""); err == nil {
		t.Fatal("expected multiple match error")
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
	if string(updated) != "[package]\nname = \"demo\"\nversion = \"1.0.2\" # release\n" {
		t.Fatalf("toml layout = %q", updated)
	}
}

func TestUpdateFileIsAtomicAndPreservesModeOnValidationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gradle.properties")
	original := []byte("pluginVersion=1.0.1\npluginVersion=duplicate\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
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
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}
