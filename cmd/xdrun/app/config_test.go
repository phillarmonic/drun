package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
	"gopkg.in/yaml.v3"
)

func TestInferProjectNameFromWorkingDirUsesFolderName(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "sample-service")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if got := inferProjectNameFromWorkingDir(); got != "sample-service" {
		t.Fatalf("inferProjectNameFromWorkingDir() = %q, want %q", got, "sample-service")
	}
}

func TestGenerateStarterConfigUsesWorkingDirectoryName(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "starter-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	config := generateStarterConfig(false)
	if !strings.Contains(config, `project "starter-app" version "1.0":`) {
		t.Fatalf("generateStarterConfig() did not embed working directory name:\n%s", config)
	}

	assertGeneratedConfigParses(t, config)
}

func TestGenerateStarterConfigMinimalContainsOnlyWelcomeTask(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "minimal-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	config := generateStarterConfig(true)
	if !strings.Contains(config, `project "minimal-app" version "1.0":`) {
		t.Fatalf("generateStarterConfig(true) did not embed working directory name:\n%s", config)
	}
	if !strings.Contains(config, `task "default" means "Welcome":`) {
		t.Fatalf("generateStarterConfig(true) did not include default welcome task:\n%s", config)
	}
	if strings.Contains(config, `task "hello" means "Say hello":`) {
		t.Fatalf("generateStarterConfig(true) should not include extra tasks:\n%s", config)
	}

	assertGeneratedConfigParses(t, config)
}

func TestInitializeConfigUsesOfficialManifestWhenTemplateNameProvided(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "official-template-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	manifest := `version: "1"
templates:
  go-cli:
    kind: go-cli
    description: "Go CLI starter"
    source: "go-cli.drun"
`
	templateSpec := `version: 2.0

project "starter" version "1.0":
task "default" means "Welcome":
	info "{{project_name}} Drun Spec"
`

	originalFetcher := initTemplateContentFetcher
	initTemplateContentFetcher = func(url string) ([]byte, error) {
		switch url {
		case "/tmp/official/templates.yaml":
			return []byte(manifest), nil
		case "/tmp/official/go-cli.drun":
			return []byte(templateSpec), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	t.Cleanup(func() {
		initTemplateContentFetcher = originalFetcher
	})
	err = InitializeConfig("", false, false, "", "go-cli", "/tmp/official")
	if err != nil {
		t.Fatalf("InitializeConfig() error = %v", err)
	}

	content, err := os.ReadFile(".drun/spec.drun")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), `project "official-template-app" version "1.0":`) {
		t.Fatalf("generated config did not use official manifest:\n%s", string(content))
	}
}

func TestInitializeConfigRejectsManifestWithoutTemplateName(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "template-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	err = InitializeConfig("", false, false, "https://example.com/templates.yaml", "", "")
	if err == nil || !strings.Contains(err.Error(), "--from-template requires --template") {
		t.Fatalf("InitializeConfig() error = %v, want manifest/template validation", err)
	}
}

func TestInitializeConfigFromTemplateAppliesGoRewrite(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "my-tool")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	manifest := `version: "1"
templates:
  go-cli:
    kind: go-cli
    description: "Go CLI starter"
    source: "https://templates.example/go-cli.drun"
`
	templateSpec := `# drun (do-run) CLI is a fast, semantic task runner with
# its own powerful automation language. Effortless tasks, serious speed.
# Learn more at https://github.com/phillarmonic/drun

version: 2.0

project "example-app" version "1.0":
task "default" means "Welcome":
	info "{{project_name}} Drun Spec"

task "build" means "Build {{binary_name}}":
	step "Building {{binary_name}}..."
	run "go build -ldflags=\"-X 'main.version=v0.0.1 (dev build)'\" -o ./bin/example-app ./cmd/example-app"
	success "Build completed for {{binary_name}}"

task "install" means "Install {{binary_name}}":
	step "Installing {{binary_name}}..."
	run "go install ./cmd/example-app"
	success "Install completed for {{module_name}}"
`

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	goMod := "module github.com/example/my-tool\n\ngo 1.24.0\n"
	if err := os.WriteFile("go.mod", []byte(goMod), 0600); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	originalFetcher := initTemplateContentFetcher
	initTemplateContentFetcher = func(url string) ([]byte, error) {
		switch url {
		case "https://templates.example/templates.yaml":
			return []byte(manifest), nil
		case "https://templates.example/go-cli.drun":
			return []byte(templateSpec), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	t.Cleanup(func() {
		initTemplateContentFetcher = originalFetcher
	})

	if err := InitializeConfig("", false, false, "https://templates.example/templates.yaml", "go-cli", ""); err != nil {
		t.Fatalf("InitializeConfig() error = %v", err)
	}

	content, err := os.ReadFile(".drun/spec.drun")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	config := string(content)
	if !strings.Contains(config, `project "my-tool" version "1.0":`) {
		t.Fatalf("generated config did not rewrite project name:\n%s", config)
	}
	if !strings.Contains(config, `run "go build -ldflags=\"-X 'main.version=v0.0.1 (dev build)'\" -o ./bin/my-tool ./cmd/my-tool"`) {
		t.Fatalf("generated config did not rewrite go build command:\n%s", config)
	}
	if !strings.Contains(config, `run "go install ./cmd/my-tool"`) {
		t.Fatalf("generated config did not rewrite go install command:\n%s", config)
	}
	if !strings.Contains(config, `success "Install completed for github.com/example/my-tool"`) {
		t.Fatalf("generated config did not rewrite module placeholder:\n%s", config)
	}

	assertGeneratedConfigParses(t, config)
}

func TestInitTemplateManifestSupportsMapAndSequence(t *testing.T) {
	t.Run("map", func(t *testing.T) {
		input := `version: "1"
templates:
  go-cli:
    kind: go-cli
    source: "https://example.com/go-cli.drun"
`
		var manifest initTemplateManifest
		if err := yaml.Unmarshal([]byte(input), &manifest); err != nil {
			t.Fatalf("yaml.Unmarshal() error = %v", err)
		}
		if len(manifest.Templates) != 1 || manifest.Templates[0].Name != "go-cli" {
			t.Fatalf("unexpected templates: %+v", manifest.Templates)
		}
	})

	t.Run("sequence", func(t *testing.T) {
		input := `version: "1"
templates:
  - name: go-cli
    kind: go-cli
    source: "https://example.com/go-cli.drun"
`
		var manifest initTemplateManifest
		if err := yaml.Unmarshal([]byte(input), &manifest); err != nil {
			t.Fatalf("yaml.Unmarshal() error = %v", err)
		}
		if len(manifest.Templates) != 1 || manifest.Templates[0].Name != "go-cli" {
			t.Fatalf("unexpected templates: %+v", manifest.Templates)
		}
	})
}

func TestLoadInitTemplateManifestResolvesRelativeLocalSources(t *testing.T) {
	manifest := `version: "1"
templates:
  go-cli:
    kind: go-cli
    source: "templates/go-cli.drun"
`

	originalFetcher := initTemplateContentFetcher
	initTemplateContentFetcher = func(url string) ([]byte, error) {
		if url == "/catalog/templates.yaml" {
			return []byte(manifest), nil
		}
		return nil, os.ErrNotExist
	}
	t.Cleanup(func() {
		initTemplateContentFetcher = originalFetcher
	})

	loaded, err := loadInitTemplateManifest("/catalog/templates.yaml")
	if err != nil {
		t.Fatalf("loadInitTemplateManifest() error = %v", err)
	}
	if loaded.Templates[0].Source != "/catalog/templates/go-cli.drun" {
		t.Fatalf("resolved source = %q", loaded.Templates[0].Source)
	}
}

func TestLoadInitTemplateManifestResolvesRelativeGitHubSources(t *testing.T) {
	manifest := `version: "1"
templates:
  go-cli:
    kind: go-cli
    source: "templates/go-cli.drun"
`

	originalFetcher := initTemplateContentFetcher
	initTemplateContentFetcher = func(url string) ([]byte, error) {
		if url == "github:phillarmonic/drun-templates/catalog/templates.yaml@main" {
			return []byte(manifest), nil
		}
		return nil, os.ErrNotExist
	}
	t.Cleanup(func() {
		initTemplateContentFetcher = originalFetcher
	})

	loaded, err := loadInitTemplateManifest("github:phillarmonic/drun-templates/catalog/templates.yaml@main")
	if err != nil {
		t.Fatalf("loadInitTemplateManifest() error = %v", err)
	}
	if loaded.Templates[0].Source != "github:phillarmonic/drun-templates/catalog/templates/go-cli.drun@main" {
		t.Fatalf("resolved source = %q", loaded.Templates[0].Source)
	}
}

func TestInitTemplateManifestTemplateByName(t *testing.T) {
	manifest := &initTemplateManifest{
		Templates: []initTemplateEntry{
			{Name: "go-cli", Source: "https://example.com/go-cli.drun", Kind: initTemplateKindGoCLI},
		},
	}

	entry, err := manifest.templateByName("go-cli")
	if err != nil {
		t.Fatalf("templateByName() error = %v", err)
	}
	if entry.Source != "https://example.com/go-cli.drun" {
		t.Fatalf("templateByName() source = %q", entry.Source)
	}

	if _, err := manifest.templateByName("missing"); err == nil {
		t.Fatal("templateByName() expected error for missing template")
	}
}

func TestListInitTemplatesUsesDefaultManifest(t *testing.T) {
	manifest := `version: "1"
templates:
  go-cli:
    kind: go-cli
    description: "Go CLI starter"
    source: "go-cli.drun"
  api:
    source: "api.drun"
`

	originalFetcher := initTemplateContentFetcher
	initTemplateContentFetcher = func(url string) ([]byte, error) {
		if url == "/catalog/templates.yaml" {
			return []byte(manifest), nil
		}
		return nil, os.ErrNotExist
	}
	t.Cleanup(func() {
		initTemplateContentFetcher = originalFetcher
	})
	output := captureStdout(t, func() {
		if err := ListInitTemplates("", "/catalog"); err != nil {
			t.Fatalf("ListInitTemplates() error = %v", err)
		}
	})

	if !strings.Contains(output, "Available init templates (/catalog/templates.yaml):") {
		t.Fatalf("unexpected list output:\n%s", output)
	}
	if !strings.Contains(output, "  - api\n") || !strings.Contains(output, "  - go-cli: Go CLI starter\n") {
		t.Fatalf("unexpected list output:\n%s", output)
	}
}

func TestResolveDefaultTemplateManifestPrefersTemplatesRepo(t *testing.T) {
	manifest, err := resolveDefaultTemplateManifest("", "/workspace/drun-templates")
	if err != nil {
		t.Fatalf("resolveDefaultTemplateManifest() error = %v", err)
	}
	if manifest != "/workspace/drun-templates/templates.yaml" {
		t.Fatalf("manifest = %q", manifest)
	}
}

func TestResolveDefaultTemplateManifestUsesTemplatesYAMLForLocalDirectory(t *testing.T) {
	tempRoot := t.TempDir()

	manifest, err := resolveDefaultTemplateManifest(tempRoot, "")
	if err != nil {
		t.Fatalf("resolveDefaultTemplateManifest() error = %v", err)
	}
	if manifest != filepath.Join(tempRoot, "templates.yaml") {
		t.Fatalf("manifest = %q", manifest)
	}
}

func TestInitializeConfigAcceptsLocalTemplateDirectoryViaFromTemplate(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "directory-template-app")
	if err := os.Mkdir(projectDir, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	templateRepo := filepath.Join(tempRoot, "drun-templates")
	if err := os.Mkdir(templateRepo, 0750); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	manifest := `version: "1"
templates:
  go-cli:
    kind: go-cli
    source: "go-cli.drun"
`
	templateSpec := `version: 2.0

project "starter" version "1.0":
task "default" means "Welcome":
	info "{{project_name}} Drun Spec"
`

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	originalFetcher := initTemplateContentFetcher
	initTemplateContentFetcher = func(url string) ([]byte, error) {
		switch url {
		case filepath.Join(templateRepo, "templates.yaml"):
			return []byte(manifest), nil
		case filepath.Join(templateRepo, "go-cli.drun"):
			return []byte(templateSpec), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	t.Cleanup(func() {
		initTemplateContentFetcher = originalFetcher
	})

	if err := InitializeConfig("", false, false, templateRepo, "go-cli", ""); err != nil {
		t.Fatalf("InitializeConfig() error = %v", err)
	}

	content, err := os.ReadFile(".drun/spec.drun")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), `project "directory-template-app" version "1.0":`) {
		t.Fatalf("generated config did not use local template directory:\n%s", string(content))
	}
}

func TestFindConfigFileFindsInfraDefaultLocation(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "infra-project")
	if err := os.MkdirAll(filepath.Join(projectDir, "infra", ".drun"), 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	specPath := filepath.Join(projectDir, "infra", ".drun", "spec.drun")
	if err := os.WriteFile(specPath, []byte("version: 2.0\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	withWorkingDir(t, projectDir, func() {
		got, err := FindConfigFile("")
		if err != nil {
			t.Fatalf("FindConfigFile() error = %v", err)
		}
		if got != "infra/.drun/spec.drun" {
			t.Fatalf("FindConfigFile() = %q, want %q", got, "infra/.drun/spec.drun")
		}
	})
}

func TestFindConfigFileFindsInfraDrunDefaultLocation(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "infra-drun-project")
	if err := os.MkdirAll(filepath.Join(projectDir, "infra", "drun"), 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	specPath := filepath.Join(projectDir, "infra", "drun", "spec.drun")
	if err := os.WriteFile(specPath, []byte("version: 2.0\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	withWorkingDir(t, projectDir, func() {
		got, err := FindConfigFile("")
		if err != nil {
			t.Fatalf("FindConfigFile() error = %v", err)
		}
		if got != "infra/drun/spec.drun" {
			t.Fatalf("FindConfigFile() = %q, want %q", got, "infra/drun/spec.drun")
		}
	})
}

func TestInitializeMinimalConfigDoesNotCreateWorkspaceDefaultForBuiltInInfraLocation(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "infra-default-init")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	withWorkingDir(t, projectDir, func() {
		if err := InitializeConfig("infra/drun/spec.drun", false, true, "", "", ""); err != nil {
			t.Fatalf("InitializeConfig() error = %v", err)
		}

		if _, err := os.Stat(filepath.Join(projectDir, ".drun", ".drun_workspace.yml")); !os.IsNotExist(err) {
			t.Fatalf("workspace default file should not be created for built-in default path, got err=%v", err)
		}
	})
}

func TestInitializeConfigCreatesWorkspaceDefaultForCustomLocation(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "custom-default-init")
	if err := os.MkdirAll(projectDir, 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	withWorkingDir(t, projectDir, func() {
		if err := InitializeConfig("automation/project.drun", false, false, "", "", ""); err != nil {
			t.Fatalf("InitializeConfig() error = %v", err)
		}

		workspaceConfigPath := filepath.Join(projectDir, ".drun", ".drun_workspace.yml")
		content, err := os.ReadFile(workspaceConfigPath)
		if err != nil {
			t.Fatalf("ReadFile(workspace config) error = %v", err)
		}
		if !strings.Contains(string(content), "defaultTaskFile: automation/project.drun") {
			t.Fatalf("workspace config did not persist custom default:\n%s", string(content))
		}
	})
}

func TestFindConfigFileUsesHomeExtraSearchPaths(t *testing.T) {
	tempRoot := t.TempDir()
	projectDir := filepath.Join(tempRoot, "custom-search-project")
	if err := os.MkdirAll(filepath.Join(projectDir, "automation"), 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	specPath := filepath.Join(projectDir, "automation", "project.drun")
	if err := os.WriteFile(specPath, []byte("version: 2.0\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	homeDir := filepath.Join(tempRoot, "home")
	if err := os.MkdirAll(filepath.Join(homeDir, ".drun"), 0750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	userConfig := "extraTaskFileSearchPaths:\n  - automation/project.drun\n"
	if err := os.WriteFile(filepath.Join(homeDir, ".drun", "config.yml"), []byte(userConfig), 0600); err != nil {
		t.Fatalf("WriteFile(config.yml) error = %v", err)
	}

	withEnv(t, "HOME", homeDir, func() {
		withWorkingDir(t, projectDir, func() {
			got, err := FindConfigFile("")
			if err != nil {
				t.Fatalf("FindConfigFile() error = %v", err)
			}
			if got != "automation/project.drun" {
				t.Fatalf("FindConfigFile() = %q, want %q", got, "automation/project.drun")
			}
		})
	})
}

func assertGeneratedConfigParses(t *testing.T, config string) {
	t.Helper()

	l := lexer.NewLexer(config)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if program == nil {
		t.Fatal("generated config did not parse")
	}
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("generated config parse errors: %v\n%s", errs, config)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	os.Stdout = writer
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		done <- buf.String()
	}()

	fn()

	_ = writer.Close()
	os.Stdout = originalStdout
	return <-done
}

func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("Chdir() restore error = %v", chdirErr)
		}
	})

	fn()
}

func withEnv(t *testing.T, key, value string, fn func()) {
	t.Helper()

	originalValue, hadValue := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Setenv(%q) error = %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if hadValue {
			err = os.Setenv(key, originalValue)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("restore env %q error = %v", key, err)
		}
	})

	fn()
}
