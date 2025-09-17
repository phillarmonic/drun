package tmpl

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/phillarmonic/drun/internal/http"
	"github.com/phillarmonic/drun/internal/model"
	"github.com/phillarmonic/drun/internal/pool"
)

// Engine handles template rendering with custom functions
type Engine struct {
	snippets      map[string]string
	recipePrerun  []string                 // Recipe-prerun snippets that execute before every recipe
	recipePostrun []string                 // Recipe-postrun snippets that execute after every recipe
	templateCache sync.Map                 // Cache compiled templates by hash
	funcMap       template.FuncMap         // Pre-computed function map
	currentCtx    *model.ExecutionContext  // Current execution context for functions
	httpClient    *http.TemplateHTTPClient // HTTP client for template functions
}

// NewEngine creates a new template engine
func NewEngine(snippets map[string]string, recipePrerun []string, recipePostrun []string) *Engine {
	e := &Engine{
		snippets:      snippets,
		recipePrerun:  recipePrerun,
		recipePostrun: recipePostrun,
	}
	// Pre-compute function map for better performance
	e.funcMap = e.getFuncMap()
	return e
}

// NewEngineWithHTTP creates a new template engine with HTTP support
func NewEngineWithHTTP(snippets map[string]string, recipePrerun []string, recipePostrun []string, httpEndpoints map[string]model.HTTPEndpoint, secrets map[string]string) *Engine {
	return NewEngineWithHTTPAndVersion(snippets, recipePrerun, recipePostrun, httpEndpoints, secrets, "dev")
}

// NewEngineWithHTTPAndVersion creates a new template engine with HTTP support and version
func NewEngineWithHTTPAndVersion(snippets map[string]string, recipePrerun []string, recipePostrun []string, httpEndpoints map[string]model.HTTPEndpoint, secrets map[string]string, version string) *Engine {
	e := &Engine{
		snippets:      snippets,
		recipePrerun:  recipePrerun,
		recipePostrun: recipePostrun,
		httpClient:    http.NewTemplateHTTPClientWithVersion(httpEndpoints, secrets, version),
	}
	// Pre-compute function map for better performance
	e.funcMap = e.getFuncMap()
	return e
}

// Render renders a template string with the given context
func (e *Engine) Render(templateStr string, ctx *model.ExecutionContext) (string, error) {
	// Set current context for template functions
	e.currentCtx = ctx
	defer func() { e.currentCtx = nil }()

	// Use template caching for better performance
	tmpl, err := e.getOrCompileTemplate(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to get template: %w", err)
	}

	buf := pool.GetStringBuilder()
	defer pool.PutStringBuilder(buf)

	if err := tmpl.Execute(buf, e.buildTemplateData(ctx)); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// getOrCompileTemplate gets a cached template or compiles and caches a new one
func (e *Engine) getOrCompileTemplate(templateStr string) (*template.Template, error) {
	// Create a hash of the template string for caching
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(templateStr)))

	// Check cache first
	if cached, ok := e.templateCache.Load(hash); ok {
		return cached.(*template.Template), nil
	}

	// Compile template
	tmpl, err := template.New("drun").
		Funcs(e.funcMap).
		Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Cache the compiled template
	e.templateCache.Store(hash, tmpl)

	return tmpl, nil
}

// RenderStep renders a step with the given context
func (e *Engine) RenderStep(step model.Step, ctx *model.ExecutionContext) (model.Step, error) {
	// Calculate total capacity for all lines
	totalCapacity := len(e.recipePrerun) + len(step.Lines) + len(e.recipePostrun)
	allLines := make([]string, 0, totalCapacity)

	// Add recipe-prerun snippets first
	for _, prerunSnippet := range e.recipePrerun {
		// Render each recipe-prerun snippet as a template to support variables
		rendered, err := e.Render(prerunSnippet, ctx)
		if err != nil {
			return model.Step{}, fmt.Errorf("failed to render recipe-prerun snippet: %w", err)
		}
		// Split rendered snippet into lines and add them
		snippetLines := strings.Split(rendered, "\n")
		for _, line := range snippetLines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				allLines = append(allLines, trimmed)
			}
		}
	}

	// Add the original step lines
	allLines = append(allLines, step.Lines...)

	// Add recipe-postrun snippets last
	for _, postrunSnippet := range e.recipePostrun {
		// Render each recipe-postrun snippet as a template to support variables
		rendered, err := e.Render(postrunSnippet, ctx)
		if err != nil {
			return model.Step{}, fmt.Errorf("failed to render recipe-postrun snippet: %w", err)
		}
		// Split rendered snippet into lines and add them
		snippetLines := strings.Split(rendered, "\n")
		for _, line := range snippetLines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				allLines = append(allLines, trimmed)
			}
		}
	}

	// Join all lines into a single script for template rendering
	script := strings.Join(allLines, "\n")

	// Render the entire script as one template
	rendered, err := e.Render(script, ctx)
	if err != nil {
		return model.Step{}, fmt.Errorf("failed to render step: %w", err)
	}

	// Split back into lines
	lines := strings.Split(rendered, "\n")

	// Remove empty lines at the beginning and end
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return model.Step{Lines: lines}, nil
}

// buildTemplateData builds the data map for template execution
func (e *Engine) buildTemplateData(ctx *model.ExecutionContext) map[string]any {
	// Use pooled map for better performance
	data := pool.GetStringMap()

	// Note: We can't use defer here because we're returning the map
	// The caller is responsible for returning it to the pool if needed
	// For now, we'll create a new map to avoid complexity

	// Pre-allocate map with estimated size to reduce allocations
	estimatedSize := 3 + len(ctx.Vars) + len(ctx.Env) + len(ctx.Flags) + len(ctx.Positionals)
	result := make(map[string]any, estimatedSize)

	// Add system info first (can be overridden by user data)
	result["os"] = ctx.OS
	result["arch"] = ctx.Arch
	result["hostname"] = ctx.Hostname

	// Add all context data (these can override system info)
	for k, v := range ctx.Vars {
		result[k] = v
	}
	for k, v := range ctx.Env {
		result[k] = v
	}
	for k, v := range ctx.Secrets {
		result[k] = v
	}
	// Add flags under both direct access and .flags namespace for flexibility
	for k, v := range ctx.Flags {
		result[k] = v
	}
	if len(ctx.Flags) > 0 {
		result["flags"] = ctx.Flags
	}

	for k, v := range ctx.Positionals {
		result[k] = v
	}

	// Return the pooled map for reuse
	pool.PutStringMap(data)

	return result
}

// getFuncMap returns the template function map
func (e *Engine) getFuncMap() template.FuncMap {
	// Start with sprig functions
	funcMap := sprig.TxtFuncMap()

	// Add custom functions
	funcMap["snippet"] = e.snippetFunc
	funcMap["env"] = envFunc
	funcMap["now"] = nowFunc
	funcMap["sha256"] = sha256Func
	funcMap["shellquote"] = shellquoteFunc
	funcMap["os"] = osFunc
	funcMap["arch"] = archFunc
	funcMap["hostname"] = hostnameFunc

	// Command detection functions
	funcMap["dockerCompose"] = dockerComposeFunc
	funcMap["dockerBuildx"] = dockerBuildxFunc
	funcMap["hasCommand"] = hasCommandFunc

	// Status and messaging functions
	funcMap["info"] = infoFunc
	funcMap["warn"] = warnFunc
	funcMap["error"] = errorFunc
	funcMap["success"] = successFunc
	funcMap["step"] = stepFunc

	// Git functions
	funcMap["gitBranch"] = gitBranchFunc
	funcMap["gitCommit"] = gitCommitFunc
	funcMap["gitShortCommit"] = gitShortCommitFunc
	funcMap["isDirty"] = isDirtyFunc

	// Package manager detection
	funcMap["packageManager"] = packageManagerFunc
	funcMap["hasFile"] = hasFileFunc

	// Environment detection
	funcMap["isCI"] = isCIFunc

	// Secret functions
	funcMap["secret"] = e.secretFunc
	funcMap["hasSecret"] = e.hasSecretFunc

	// String manipulation functions
	funcMap["truncate"] = truncateFunc
	funcMap["stringContains"] = stringContainsFunc

	// HTTP functions (if HTTP client is available)
	if e.httpClient != nil {
		httpFuncs := e.httpClient.GetTemplateFunctions()
		for name, fn := range httpFuncs {
			funcMap[name] = fn
		}
	}

	return funcMap
}

// snippetFunc returns the content of a snippet and renders it with current context
func (e *Engine) snippetFunc(name string) string {
	if content, exists := e.snippets[name]; exists {
		// Return the raw content - it will be rendered as part of the template
		return content
	}
	return fmt.Sprintf("{{ERROR: snippet '%s' not found}}", name)
}

// envFunc gets an environment variable
func envFunc(name string) string {
	return os.Getenv(name)
}

// nowFunc formats the current time
func nowFunc(format string) string {
	return time.Now().Format(format)
}

// sha256Func computes SHA256 hash of a string
func sha256Func(input string) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// shellquoteFunc quotes a string for shell usage
func shellquoteFunc(input string) string {
	// Simple shell quoting - wrap in single quotes and escape single quotes
	return "'" + strings.ReplaceAll(input, "'", "'\"'\"'") + "'"
}

// osFunc returns the current OS
func osFunc() string {
	return runtime.GOOS
}

// archFunc returns the current architecture
func archFunc() string {
	return runtime.GOARCH
}

// hostnameFunc returns the hostname
func hostnameFunc() string {
	hostname, _ := os.Hostname()
	return hostname
}

// DockerComposeHelper provides Docker Compose functionality for templates
type DockerComposeHelper struct {
	command string
}

// String returns the Docker Compose command (for backward compatibility)
func (d DockerComposeHelper) String() string {
	return d.command
}

// IsRunning checks if any Docker Compose services are currently running
// Returns true if any services are in "running" state, false otherwise
func (d DockerComposeHelper) IsRunning() bool {
	if d.command == "" {
		return false
	}

	// Use docker compose ps to check running services
	// The command will be either "docker compose" or "docker-compose"
	var cmd *exec.Cmd
	if strings.Contains(d.command, " ") {
		// "docker compose" - split into parts
		parts := strings.Fields(d.command)
		args := append(parts[1:], "ps", "--services", "--filter", "status=running")
		cmd = exec.Command(parts[0], args...)
	} else {
		// "docker-compose" - single command
		cmd = exec.Command(d.command, "ps", "--services", "--filter", "status=running")
	}

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// If there's any output, it means there are running services
	return strings.TrimSpace(string(output)) != ""
}

// dockerComposeFunc detects the available Docker Compose command and returns a helper
func dockerComposeFunc() DockerComposeHelper {
	var command string

	// Check for docker compose (CLI plugin) first
	if hasDockerAndSubcommand("compose") {
		command = "docker compose"
	} else if hasCommandFunc("docker-compose") {
		// Fall back to docker-compose (standalone)
		command = "docker-compose"
	}

	return DockerComposeHelper{command: command}
}

// dockerBuildxFunc detects the available Docker Buildx command
func dockerBuildxFunc() string {
	// Check for docker buildx (CLI plugin) first
	if hasDockerAndSubcommand("buildx") {
		return "docker buildx"
	}

	// Fall back to docker-buildx (standalone)
	if hasCommandFunc("docker-buildx") {
		return "docker-buildx"
	}

	return ""
}

// hasCommandFunc checks if a command is available in PATH
func hasCommandFunc(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// hasDockerAndSubcommand checks if docker command exists and supports a subcommand
func hasDockerAndSubcommand(subcommand string) bool {
	// First check if docker command exists
	if !hasCommandFunc("docker") {
		return false
	}

	// Then check if the subcommand is available by running docker <subcommand> --help
	cmd := exec.Command("docker", subcommand, "--help")
	err := cmd.Run()
	return err == nil
}

// Status and messaging functions
func infoFunc(message string) string {
	return fmt.Sprintf("echo \"ℹ️  %s\"", message)
}

func warnFunc(message string) string {
	return fmt.Sprintf("echo \"⚠️  %s\"", message)
}

func errorFunc(message string) string {
	return fmt.Sprintf("echo \"❌ %s\"", message)
}

func successFunc(message string) string {
	return fmt.Sprintf("echo \"✅ %s\"", message)
}

func stepFunc(message string) string {
	return fmt.Sprintf("echo \"🚀 %s\"", message)
}

// String manipulation functions
func truncateFunc(length int, text string) string {
	if length <= 0 {
		return ""
	}

	// Convert to runes to handle Unicode properly
	runes := []rune(text)
	if len(runes) <= length {
		return text
	}

	return string(runes[:length])
}

// stringContainsFunc checks if a string contains a substring
// Parameters: string, substring (intuitive order, unlike Sprig's contains)
func stringContainsFunc(str, substr string) bool {
	return strings.Contains(str, substr)
}

// Git functions
func gitBranchFunc() string {
	// Try modern git command first (Git 2.22+)
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		if branch != "" {
			return branch
		}
	}

	// Fallback for older git versions or detached HEAD
	cmd = exec.Command("git", "symbolic-ref", "--short", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		if branch != "" {
			return branch
		}
	}

	// If we're in detached HEAD, try to get a descriptive name
	cmd = exec.Command("git", "describe", "--tags", "--exact-match")
	output, err = cmd.Output()
	if err == nil {
		tag := strings.TrimSpace(string(output))
		if tag != "" {
			return tag
		}
	}

	// Last resort: return short commit hash with "detached" prefix
	cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		commit := strings.TrimSpace(string(output))
		if commit != "" {
			return "detached@" + commit
		}
	}

	return "unknown"
}

func gitCommitFunc() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	commit := strings.TrimSpace(string(output))
	if commit == "" {
		return "unknown"
	}
	return commit
}

func gitShortCommitFunc() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	commit := strings.TrimSpace(string(output))
	if commit == "" {
		return "unknown"
	}
	return commit
}

func isDirtyFunc() bool {
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD", "--")
	err := cmd.Run()
	return err != nil // If command fails, working directory is dirty
}

// Package manager detection
func packageManagerFunc() string {
	// Check for lock files to determine package manager
	if hasFileFunc("pnpm-lock.yaml") {
		return "pnpm"
	}
	if hasFileFunc("yarn.lock") {
		return "yarn"
	}
	if hasFileFunc("bun.lockb") {
		return "bun"
	}
	if hasFileFunc("package.json") {
		return "npm"
	}

	// Check for Python
	if hasFileFunc("pyproject.toml") || hasFileFunc("requirements.txt") {
		return "pip"
	}

	// Check for Go
	if hasFileFunc("go.mod") {
		return "go"
	}

	return ""
}

func hasFileFunc(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// Environment detection
func isCIFunc() bool {
	// Check common CI environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TRAVIS", "CIRCLECI"}
	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	return false
}

// Secret functions - these need to be methods to access engine context
func (e *Engine) secretFunc(name string) string {
	if e.currentCtx != nil && e.currentCtx.Secrets != nil {
		if value, exists := e.currentCtx.Secrets[name]; exists {
			return value
		}
	}
	return ""
}

func (e *Engine) hasSecretFunc(name string) bool {
	if e.currentCtx != nil && e.currentCtx.Secrets != nil {
		_, exists := e.currentCtx.Secrets[name]
		return exists
	}
	return false
}
