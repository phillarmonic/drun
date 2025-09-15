package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkLoader_Load benchmarks the spec loading performance
func BenchmarkLoader_Load(b *testing.B) {
	tempDir := b.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	// Create a realistic spec file
	specContent := `
version: 0.1

env:
  NODE_ENV: development
  API_URL: http://localhost:3000

vars:
  app_name: myapp
  version: 1.0.0

defaults:
  shell: bash
  timeout: 300s

snippets:
  setup_node: |
    node --version
    npm --version
  docker_login: |
    echo "Logging into Docker..."
    docker login

recipes:
  deps:
    help: "Install dependencies"
    run: |
      {{ snippet "setup_node" }}
      npm install

  test:
    help: "Run tests"
    deps: [deps]
    flags:
      unit:
        type: string
        default: ""
        help: "Run specific unit tests"
      coverage:
        type: bool
        default: false
        help: "Generate coverage report"
    run: |
      {{ if .unit }}
      npm test -- --testNamePattern="{{ .unit }}"
      {{ else }}
      npm test
      {{ end }}
      {{ if .coverage }}
      npm run coverage
      {{ end }}

  build:
    help: "Build the application"
    deps: [deps, test]
    cache_key: "build-{{ sha256 \"package.json package-lock.json\" }}"
    run: |
      npm run build
      echo "Build complete"

  deploy:
    help: "Deploy to production"
    deps: [build]
    env:
      NODE_ENV: production
    run: |
      {{ snippet "docker_login" }}
      docker build -t {{ .app_name }}:{{ .version }} .
      docker push {{ .app_name }}:{{ .version }}

  all:
    help: "Run all tasks"
    deps: [test, build]
    parallel_deps: true
    run: echo "All tasks completed"
`

	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load("")
		if err != nil {
			b.Fatalf("Failed to load spec: %v", err)
		}
	}
}

// BenchmarkLoader_Load_WithIncludes benchmarks loading with include directives
func BenchmarkLoader_Load_WithIncludes(b *testing.B) {
	tempDir := b.TempDir()

	// Create shared files
	sharedDir := filepath.Join(tempDir, "shared")
	err := os.MkdirAll(sharedDir, 0755)
	if err != nil {
		b.Fatalf("Failed to create shared dir: %v", err)
	}

	// Create multiple include files
	for i := 0; i < 5; i++ {
		includeFile := filepath.Join(sharedDir, fmt.Sprintf("common%d.yml", i))
		includeContent := fmt.Sprintf(`
recipes:
  task%d:
    help: "Task %d"
    run: echo "Running task %d"
`, i, i, i)
		err := os.WriteFile(includeFile, []byte(includeContent), 0644)
		if err != nil {
			b.Fatalf("Failed to write include file: %v", err)
		}
	}

	// Main spec file
	specFile := filepath.Join(tempDir, "drun.yml")
	specContent := `
version: 0.1

include:
  - "shared/*.yml"

recipes:
  main:
    help: "Main task"
    deps: [task0, task1, task2, task3, task4]
    run: echo "Main task complete"
`

	err = os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write main spec file: %v", err)
	}

	loader := NewLoader(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load("")
		if err != nil {
			b.Fatalf("Failed to load spec: %v", err)
		}
	}
}

// BenchmarkLoader_Load_LargeSpec benchmarks loading a large specification
func BenchmarkLoader_Load_LargeSpec(b *testing.B) {
	tempDir := b.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	// Generate a large spec with many recipes
	specContent := `version: 0.1

env:
  NODE_ENV: development

vars:
  app_name: myapp

recipes:
`

	// Add 100 recipes to test scalability
	for i := 0; i < 100; i++ {
		recipeContent := fmt.Sprintf(`  recipe%d:
    help: "Recipe %d"
    deps: %s
    flags:
      flag%d:
        type: string
        default: "value%d"
        help: "Flag %d"
    run: |
      echo "Running recipe %d"
      echo "Flag value: {{ .flag%d }}"
`, i, i, getDeps(i), i, i, i, i, i)
		specContent += recipeContent
	}

	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Load("")
		if err != nil {
			b.Fatalf("Failed to load spec: %v", err)
		}
	}
}

// getDeps generates dependencies for a recipe based on its index
func getDeps(i int) string {
	if i == 0 {
		return "[]"
	}
	if i < 5 {
		return fmt.Sprintf("[recipe%d]", i-1)
	}
	// Create some complex dependency patterns
	return fmt.Sprintf("[recipe%d, recipe%d]", i-1, i-2)
}
