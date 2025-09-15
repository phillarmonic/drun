package tmpl

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/phillarmonic/drun/internal/model"
	"github.com/phillarmonic/drun/internal/pool"
)

// Engine handles template rendering with custom functions
type Engine struct {
	snippets      map[string]string
	templateCache sync.Map         // Cache compiled templates by hash
	funcMap       template.FuncMap // Pre-computed function map
}

// NewEngine creates a new template engine
func NewEngine(snippets map[string]string) *Engine {
	e := &Engine{
		snippets: snippets,
	}
	// Pre-compute function map for better performance
	e.funcMap = e.getFuncMap()
	return e
}

// Render renders a template string with the given context
func (e *Engine) Render(templateStr string, ctx *model.ExecutionContext) (string, error) {
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
	for k, v := range ctx.Flags {
		result[k] = v
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
