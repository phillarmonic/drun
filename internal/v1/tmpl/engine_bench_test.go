package tmpl

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v1/model"
)

// BenchmarkEngine_Render benchmarks template rendering performance
func BenchmarkEngine_Render(b *testing.B) {
	engine := NewEngine(nil, nil, nil)

	template := `
Hello {{ .name }}!
Your environment is: {{ .env.NODE_ENV }}
App version: {{ .version }}
{{ if .debug }}
Debug mode is enabled
{{ end }}
{{ range $i := until .count }}
Item {{ add $i 1 }}: {{ $.name }}
{{ end }}
`

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"name":    "TestUser",
			"version": "1.0.0",
			"debug":   true,
			"count":   10,
		},
		Env: map[string]string{
			"NODE_ENV": "production",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Render(template, ctx)
		if err != nil {
			b.Fatalf("Failed to render template: %v", err)
		}
	}
}

// BenchmarkEngine_RenderStep benchmarks step rendering performance
func BenchmarkEngine_RenderStep(b *testing.B) {
	engine := NewEngine(map[string]string{
		"setup":   "echo 'Setting up environment'",
		"cleanup": "echo 'Cleaning up'",
	}, nil, nil)

	step := model.Step{
		Lines: []string{
			"{{ snippet \"setup\" }}",
			"echo 'Processing {{ .name }}'",
			"{{ if .verbose }}",
			"echo 'Verbose mode enabled'",
			"{{ range $item := .items }}",
			"echo 'Processing item: {{ $item }}'",
			"{{ end }}",
			"{{ end }}",
			"echo 'Build version: {{ .version }}'",
			"{{ snippet \"cleanup\" }}",
		},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"name":    "MyApp",
			"version": "2.1.0",
			"verbose": true,
			"items":   []string{"file1.go", "file2.go", "file3.go"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.RenderStep(step, ctx)
		if err != nil {
			b.Fatalf("Failed to render step: %v", err)
		}
	}
}

// BenchmarkEngine_Render_Complex benchmarks complex template rendering
func BenchmarkEngine_Render_Complex(b *testing.B) {
	engine := NewEngine(map[string]string{
		"docker_build": `
docker build \
  --build-arg VERSION={{ .version }} \
  --build-arg NODE_ENV={{ .env.NODE_ENV }} \
  --tag {{ .registry }}/{{ .org }}/{{ .app_name }}:{{ .version }} \
  .
`,
		"docker_push": `
docker push {{ .registry }}/{{ .org }}/{{ .app_name }}:{{ .version }}
`,
	}, nil, nil)

	template := `
#!/bin/bash
set -euo pipefail

echo "Starting build process for {{ .app_name }} v{{ .version }}"

{{ if .clean }}
echo "Cleaning previous builds..."
rm -rf dist/
{{ end }}

{{ range $env := .environments }}
echo "Building for environment: {{ $env }}"
export NODE_ENV={{ $env }}

{{ if eq $env "production" }}
npm run build:prod
{{ else }}
npm run build:dev
{{ end }}

{{ if $.docker }}
{{ snippet "docker_build" }}
{{ if $.push }}
{{ snippet "docker_push" }}
{{ end }}
{{ end }}

{{ end }}

echo "Build completed successfully!"
`

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"app_name":     "myapp",
			"version":      "1.2.3",
			"clean":        true,
			"docker":       true,
			"push":         true,
			"registry":     "ghcr.io",
			"org":          "myorg",
			"environments": []string{"development", "staging", "production"},
		},
		Env: map[string]string{
			"NODE_ENV": "production",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Render(template, ctx)
		if err != nil {
			b.Fatalf("Failed to render complex template: %v", err)
		}
	}
}

// BenchmarkEngine_Render_WithSprig benchmarks Sprig function usage
func BenchmarkEngine_Render_WithSprig(b *testing.B) {
	engine := NewEngine(nil, nil, nil)

	template := `
{{ $data := dict "users" (list "alice" "bob" "charlie") "config" (dict "debug" true "port" 8080) }}
{{ range $user := $data.users }}
User: {{ $user | title }}
Config: {{ toJson $data.config }}
Hash: {{ sha256sum $user }}
UUID: {{ uuidv4 }}
{{ end }}

{{ $now := now "2006-01-02T15:04:05Z" }}
Timestamp: {{ $now }}
{{ range $i := until 5 }}
Item {{ $i }}: {{ randAlphaNum 10 }}
{{ end }}
`

	ctx := &model.ExecutionContext{
		Vars: map[string]any{},
		Env:  map[string]string{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Render(template, ctx)
		if err != nil {
			b.Fatalf("Failed to render Sprig template: %v", err)
		}
	}
}
