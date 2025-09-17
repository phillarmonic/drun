package tmpl

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v1/model"
)

func TestNewEngine(t *testing.T) {
	snippets := map[string]string{
		"test": "echo test",
	}

	engine := NewEngine(snippets, nil, nil)

	if engine == nil {
		t.Fatal("Expected engine to be created, got nil")
	}
}

func TestEngine_Render_SimpleTemplate(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"name": "world",
		},
	}

	result, err := engine.Render("Hello {{ .name }}!", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "Hello world!" {
		t.Errorf("Expected 'Hello world!', got %q", result)
	}
}

func TestEngine_Render_WithEnvironment(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	ctx := &model.ExecutionContext{
		Env: map[string]string{
			"USER": "testuser",
		},
	}

	result, err := engine.Render("User: {{ .USER }}", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "User: testuser" {
		t.Errorf("Expected 'User: testuser', got %q", result)
	}
}

func TestEngine_Render_WithFlags(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	ctx := &model.ExecutionContext{
		Flags: map[string]any{
			"verbose": true,
			"count":   42,
		},
	}

	result, err := engine.Render("Verbose: {{ .verbose }}, Count: {{ .count }}", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "Verbose: true, Count: 42" {
		t.Errorf("Expected 'Verbose: true, Count: 42', got %q", result)
	}
}

func TestEngine_Render_WithPositionals(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	ctx := &model.ExecutionContext{
		Positionals: map[string]any{
			"version": "1.0.0",
			"arch":    "amd64",
		},
	}

	result, err := engine.Render("Release {{ .version }} for {{ .arch }}", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "Release 1.0.0 for amd64" {
		t.Errorf("Expected 'Release 1.0.0 for amd64', got %q", result)
	}
}

func TestEngine_Render_WithSprigFunctions(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"text": "hello world",
		},
	}

	result, err := engine.Render("{{ .text | upper }}", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "HELLO WORLD" {
		t.Errorf("Expected 'HELLO WORLD', got %q", result)
	}
}

func TestEngine_Render_WithSnippet(t *testing.T) {
	snippets := map[string]string{
		"greeting": "Hello World!",
	}

	engine := NewEngine(snippets, nil, nil)

	ctx := &model.ExecutionContext{}

	result, err := engine.Render(`{{ snippet "greeting" }}`, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "Hello World!" {
		t.Errorf("Expected 'Hello World!', got %q", result)
	}
}

func TestEngine_Render_InvalidTemplate(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	ctx := &model.ExecutionContext{}

	_, err := engine.Render("{{ .invalid", ctx)

	if err == nil {
		t.Fatal("Expected error for invalid template, got nil")
	}
}

func TestEngine_RenderStep(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	step := model.Step{
		Lines: []string{
			"echo Starting {{ .name }}",
			"echo Processing...",
			"echo Done with {{ .name }}",
		},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"name": "test",
		},
	}

	result, err := engine.RenderStep(step, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{
		"echo Starting test",
		"echo Processing...",
		"echo Done with test",
	}

	if len(result.Lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(result.Lines))
	}

	for i, line := range result.Lines {
		if line != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}

func TestEngine_RenderStep_WithConditionals(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	step := model.Step{
		Lines: []string{
			"{{ if .debug }}",
			"echo Debug mode enabled",
			"{{ end }}",
			"echo Always runs",
		},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"debug": true,
		},
	}

	result, err := engine.RenderStep(step, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Debug: log actual output
	t.Logf("Actual result lines: %+v", result.Lines)

	expected := []string{
		"echo Debug mode enabled",
		"",
		"echo Always runs",
	}

	if len(result.Lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(result.Lines))
	}

	for i, line := range result.Lines {
		if line != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i, expected[i], line)
		}
	}
}

func TestEngine_VariablePrecedence(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	// Test that positionals override other variables
	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"name": "from_vars",
		},
		Env: map[string]string{
			"name": "from_env",
		},
		Flags: map[string]any{
			"name": "from_flags",
		},
		Positionals: map[string]any{
			"name": "from_positionals",
		},
		OS:       "test_os",
		Arch:     "test_arch",
		Hostname: "test_host",
	}

	result, err := engine.Render("{{ .name }}", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result != "from_positionals" {
		t.Errorf("Expected 'from_positionals', got %q", result)
	}
}
