package tmpl

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/phillarmonic/drun/internal/model"
)

// Engine handles template rendering with custom functions
type Engine struct {
	snippets map[string]string
}

// NewEngine creates a new template engine
func NewEngine(snippets map[string]string) *Engine {
	return &Engine{
		snippets: snippets,
	}
}

// Render renders a template string with the given context
func (e *Engine) Render(templateStr string, ctx *model.ExecutionContext) (string, error) {
	tmpl, err := template.New("drun").
		Funcs(e.getFuncMap()).
		Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, e.buildTemplateData(ctx)); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderStep renders a step with the given context
func (e *Engine) RenderStep(step model.Step, ctx *model.ExecutionContext) (model.Step, error) {
	// Join all lines into a single script for template rendering
	script := step.String()

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
	data := make(map[string]any)

	// Add system info first (can be overridden by user data)
	data["os"] = ctx.OS
	data["arch"] = ctx.Arch
	data["hostname"] = ctx.Hostname

	// Add all context data (these can override system info)
	for k, v := range ctx.Vars {
		data[k] = v
	}
	for k, v := range ctx.Env {
		data[k] = v
	}
	for k, v := range ctx.Flags {
		data[k] = v
	}
	for k, v := range ctx.Positionals {
		data[k] = v
	}

	return data
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
