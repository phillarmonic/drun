package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/phillarmonic/drun/v2/internal/cache"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
	"github.com/phillarmonic/drun/v2/internal/remote"
	"gopkg.in/yaml.v3"
)

const (
	initTemplateKindGoCLI          = "go-cli"
	initTemplateCacheDuration      = time.Minute
	initTemplateFetchTimeout       = 30 * time.Second
	initTemplateManifestVerion     = "1"
	officialTemplateManifestRemote = "github:phillarmonic/drun-templates/templates.yaml@main"
)

var initTemplateContentFetcher = fetchInitTemplateContent

type initTemplateManifest struct {
	Version   string              `yaml:"version"`
	Templates []initTemplateEntry `yaml:"-"`
}

type initTemplateEntry struct {
	Name        string `yaml:"name"`
	Source      string `yaml:"source"`
	Kind        string `yaml:"kind,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type initTemplateVariables struct {
	ProjectName string
	BinaryName  string
	CmdPath     string
	ModuleName  string
}

func (m *initTemplateManifest) UnmarshalYAML(node *yaml.Node) error {
	var seq struct {
		Version   string              `yaml:"version"`
		Templates []initTemplateEntry `yaml:"templates"`
	}
	if err := node.Decode(&seq); err == nil && len(seq.Templates) > 0 {
		m.Version = seq.Version
		m.Templates = seq.Templates
		return nil
	}

	var mp struct {
		Version   string               `yaml:"version"`
		Templates map[string]yaml.Node `yaml:"templates"`
	}
	if err := node.Decode(&mp); err != nil {
		return err
	}

	m.Version = mp.Version
	m.Templates = make([]initTemplateEntry, 0, len(mp.Templates))
	for name, raw := range mp.Templates {
		var entry initTemplateEntry
		if err := raw.Decode(&entry); err != nil {
			return fmt.Errorf("invalid template %q: %w", name, err)
		}
		if entry.Name == "" {
			entry.Name = name
		}
		m.Templates = append(m.Templates, entry)
	}

	return nil
}

func generateConfigFromTemplate(manifestURL, templateName, templatesRepo string) (string, error) {
	manifestRef, err := resolveDefaultTemplateManifest(manifestURL, templatesRepo)
	if err != nil {
		return "", err
	}

	manifest, err := loadInitTemplateManifest(manifestRef)
	if err != nil {
		return "", err
	}

	entry, err := manifest.templateByName(templateName)
	if err != nil {
		return "", err
	}

	content, err := initTemplateContentFetcher(entry.Source)
	if err != nil {
		return "", fmt.Errorf("failed to fetch template spec %q: %w", entry.Source, err)
	}

	rendered, err := applyInitTemplate(string(content), entry.Kind)
	if err != nil {
		return "", err
	}

	if err := validateGeneratedConfig(rendered); err != nil {
		return "", fmt.Errorf("rendered template is not valid drun: %w", err)
	}

	return rendered, nil
}

func loadInitTemplateManifest(manifestURL string) (*initTemplateManifest, error) {
	content, err := initTemplateContentFetcher(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template manifest %q: %w", manifestURL, err)
	}

	var manifest initTemplateManifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse template manifest: %w", err)
	}
	if len(manifest.Templates) == 0 {
		return nil, fmt.Errorf("template manifest contains no templates")
	}
	if manifest.Version != "" && manifest.Version != initTemplateManifestVerion {
		return nil, fmt.Errorf("unsupported template manifest version %q", manifest.Version)
	}

	for i := range manifest.Templates {
		entry := &manifest.Templates[i]
		if strings.TrimSpace(entry.Name) == "" {
			return nil, fmt.Errorf("template manifest contains an entry without a name")
		}
		if strings.TrimSpace(entry.Source) == "" {
			return nil, fmt.Errorf("template %q is missing a source", entry.Name)
		}
		resolvedSource, err := resolveTemplateSource(manifestURL, entry.Source)
		if err != nil {
			return nil, fmt.Errorf("template %q source resolution failed: %w", entry.Name, err)
		}
		entry.Source = resolvedSource
	}

	return &manifest, nil
}

func (m *initTemplateManifest) templateByName(name string) (*initTemplateEntry, error) {
	for _, entry := range m.Templates {
		if entry.Name == name {
			entryCopy := entry
			return &entryCopy, nil
		}
	}
	return nil, fmt.Errorf("template %q not found in manifest", name)
}

func fetchInitTemplateContent(url string) ([]byte, error) {
	if isLocalTemplatePath(url) {
		content, err := os.ReadFile(url)
		if err != nil {
			return nil, err
		}
		return content, nil
	}

	protocol, path, ref, err := remote.ParseRemoteURL(url)
	if err != nil {
		return nil, err
	}

	cacheManager, err := cache.NewManager(initTemplateCacheDuration, false)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize remote template cache: %w", err)
	}
	defer func() { _ = cacheManager.Close() }()

	cacheKey := cache.GenerateKey(url, ref)
	if content, hit, err := cacheManager.Get(cacheKey); err == nil && hit {
		return content, nil
	}

	githubFetcher := remote.NewGitHubFetcher()
	var fetcher remote.Fetcher
	switch protocol {
	case "github":
		fetcher = githubFetcher
	case "https":
		fetcher = remote.NewHTTPSFetcher()
	case "drunhub":
		fetcher = remote.NewDrunhubFetcher(githubFetcher)
	default:
		return nil, fmt.Errorf("unsupported template protocol %q", protocol)
	}

	ctx, cancel := context.WithTimeout(context.Background(), initTemplateFetchTimeout)
	defer cancel()

	content, err := fetcher.Fetch(ctx, path, ref)
	if err != nil {
		if stale, ok := cacheManager.GetStale(cacheKey); ok {
			return stale, nil
		}
		return nil, err
	}

	if err := cacheManager.Set(cacheKey, content, initTemplateCacheDuration); err != nil {
		return nil, fmt.Errorf("failed to cache remote template content: %w", err)
	}

	return content, nil
}

func resolveDefaultTemplateManifest(manifestRef, templatesRepo string) (string, error) {
	resolveLocalDir := func(ref string) (string, error) {
		if !isLocalTemplatePath(ref) {
			return ref, nil
		}

		// #nosec G304,G703 -- template manifest refs intentionally allow user-selected local paths.
		info, err := os.Stat(ref)
		if err != nil {
			if os.IsNotExist(err) {
				return ref, nil
			}
			return "", fmt.Errorf("failed to inspect template manifest path %q: %w", ref, err)
		}
		if info.IsDir() {
			return filepath.Join(ref, "templates.yaml"), nil
		}
		return ref, nil
	}

	if manifestRef != "" {
		return resolveLocalDir(manifestRef)
	}
	if templatesRepo != "" {
		return filepath.Join(templatesRepo, "templates.yaml"), nil
	}
	if explicitManifest := os.Getenv("DRUN_TEMPLATES_MANIFEST"); explicitManifest != "" {
		return resolveLocalDir(explicitManifest)
	}
	if repoRoot := os.Getenv("DRUN_TEMPLATES_REPO"); repoRoot != "" {
		return filepath.Join(repoRoot, "templates.yaml"), nil
	}
	return officialTemplateManifestRemote, nil
}

func ListInitTemplates(fromTemplate, templatesRepo string) error {
	manifestRef, err := resolveDefaultTemplateManifest(fromTemplate, templatesRepo)
	if err != nil {
		return err
	}

	manifest, err := loadInitTemplateManifest(manifestRef)
	if err != nil {
		return err
	}

	entries := append([]initTemplateEntry(nil), manifest.Templates...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	fmt.Printf("Available init templates (%s):\n", manifestRef)
	for _, entry := range entries {
		if entry.Description != "" {
			fmt.Printf("  - %s: %s\n", entry.Name, entry.Description)
			continue
		}
		fmt.Printf("  - %s\n", entry.Name)
	}

	return nil
}

func resolveTemplateSource(manifestRef, source string) (string, error) {
	if source == "" {
		return "", fmt.Errorf("source is empty")
	}
	if remote.IsRemoteURL(source) || filepath.IsAbs(source) {
		return source, nil
	}

	if remote.IsRemoteURL(manifestRef) {
		protocol, path, ref, err := remote.ParseRemoteURL(manifestRef)
		if err != nil {
			return "", err
		}

		switch protocol {
		case "github", "drunhub":
			dir := filepath.ToSlash(filepath.Dir(path))
			if dir == "." {
				dir = ""
			}
			joined := strings.TrimPrefix(filepath.ToSlash(filepath.Join(dir, source)), "./")
			if ref != "" {
				return fmt.Sprintf("%s:%s@%s", protocol, joined, ref), nil
			}
			return fmt.Sprintf("%s:%s", protocol, joined), nil
		case "https":
			baseURL, err := url.Parse(manifestRef)
			if err != nil {
				return "", err
			}
			relURL, err := url.Parse(source)
			if err != nil {
				return "", err
			}
			return baseURL.ResolveReference(relURL).String(), nil
		default:
			return "", fmt.Errorf("unsupported manifest protocol %q", protocol)
		}
	}

	manifestDir := filepath.Dir(manifestRef)
	return filepath.Join(manifestDir, source), nil
}

func isLocalTemplatePath(path string) bool {
	if path == "" {
		return false
	}
	if strings.Contains(path, "://") || strings.HasPrefix(path, "github:") || strings.HasPrefix(path, "drunhub:") {
		return false
	}
	return true
}

func applyInitTemplate(templateContent, kind string) (string, error) {
	vars := inferInitTemplateVariables()

	rendered := strings.NewReplacer(
		"{{project_name}}", vars.ProjectName,
		"{{binary_name}}", vars.BinaryName,
		"{{cmd_path}}", vars.CmdPath,
		"{{module_name}}", vars.ModuleName,
	).Replace(templateContent)

	rendered = rewriteProjectDeclaration(rendered, vars.ProjectName)

	if kind == initTemplateKindGoCLI {
		rendered = rewriteGoTemplateCommands(rendered, vars)
	}

	return rendered, nil
}

func inferInitTemplateVariables() initTemplateVariables {
	projectName := inferProjectNameFromWorkingDir()
	moduleName := inferGoModuleName(projectName)

	return initTemplateVariables{
		ProjectName: projectName,
		BinaryName:  projectName,
		CmdPath:     "./cmd/" + projectName,
		ModuleName:  moduleName,
	}
}

func inferGoModuleName(projectName string) string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return projectName
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if moduleName != "" {
				return moduleName
			}
		}
	}

	return projectName
}

func rewriteProjectDeclaration(input, projectName string) string {
	projectPattern := regexp.MustCompile(`(?m)^project\s+"[^"]+"`)
	return projectPattern.ReplaceAllString(input, fmt.Sprintf(`project "%s"`, projectName))
}

func rewriteGoTemplateCommands(input string, vars initTemplateVariables) string {
	cmdPathPattern := regexp.MustCompile(`\./cmd/[A-Za-z0-9._-]+`)
	rendered := cmdPathPattern.ReplaceAllString(input, vars.CmdPath)

	binPattern := regexp.MustCompile(`(-o\s+)(\./bin/)([A-Za-z0-9._-]+)`)
	rendered = binPattern.ReplaceAllString(rendered, fmt.Sprintf(`${1}${2}%s`, vars.BinaryName))

	return rendered
}

func validateGeneratedConfig(config string) error {
	l := lexer.NewLexer(config)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if program == nil {
		return fmt.Errorf("generated config did not parse")
	}
	if errs := p.Errors(); len(errs) > 0 {
		return fmt.Errorf("parse errors: %v", errs)
	}
	return nil
}
