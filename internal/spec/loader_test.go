package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/drun/internal/model"
)

func TestLoader_Load_ValidSpec(t *testing.T) {
	// Create a temporary directory and file
	tempDir := t.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	specContent := `
version: 0.1

env:
  TEST_VAR: "test_value"

vars:
  app_name: "test_app"

recipes:
  test:
    help: "Test recipe"
    run: echo "Hello {{ .app_name }}"
`

	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tempDir)
	spec, err := loader.Load(specFile)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if spec.Version != "0.1" {
		t.Errorf("Expected version '0.1', got %q", spec.Version)
	}

	if spec.Env["TEST_VAR"] != "test_value" {
		t.Errorf("Expected TEST_VAR='test_value', got %q", spec.Env["TEST_VAR"])
	}

	if spec.Vars["app_name"] != "test_app" {
		t.Errorf("Expected app_name='test_app', got %v", spec.Vars["app_name"])
	}

	if len(spec.Recipes) != 1 {
		t.Fatalf("Expected 1 recipe, got %d", len(spec.Recipes))
	}

	testRecipe := spec.Recipes["test"]
	if testRecipe.Help != "Test recipe" {
		t.Errorf("Expected help='Test recipe', got %q", testRecipe.Help)
	}
}

func TestLoader_Load_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	invalidYAML := `
version: 0.1
recipes:
  test:
    help: "Test recipe"
    run: echo "Hello
      # Missing closing quote and bad indentation
      invalid: [unclosed
`

	err := os.WriteFile(specFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tempDir)
	_, err = loader.Load(specFile)

	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

func TestLoader_Load_MissingRecipes(t *testing.T) {
	tempDir := t.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	specContent := `
version: 0.1
env:
  TEST_VAR: "test_value"
# Missing recipes section
`

	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tempDir)
	_, err = loader.Load(specFile)

	if err == nil {
		t.Fatal("Expected error for missing recipes, got nil")
	}
}

func TestLoader_Load_RecipeWithoutRunOrDeps(t *testing.T) {
	tempDir := t.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	specContent := `
version: 0.1
recipes:
  invalid:
    help: "Recipe without run or deps"
    # Missing both run and deps
`

	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tempDir)
	_, err = loader.Load(specFile)

	if err == nil {
		t.Fatal("Expected error for recipe without run or deps, got nil")
	}
}

func TestLoader_Load_RecipeWithDepsOnly(t *testing.T) {
	tempDir := t.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	specContent := `
version: 0.1
recipes:
  base:
    help: "Base recipe"
    run: echo "base"
  
  deps_only:
    help: "Recipe with only dependencies"
    deps: [base]
`

	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tempDir)
	spec, err := loader.Load(specFile)

	if err != nil {
		t.Fatalf("Expected no error for recipe with deps only, got %v", err)
	}

	if len(spec.Recipes) != 2 {
		t.Fatalf("Expected 2 recipes, got %d", len(spec.Recipes))
	}
}

func TestLoader_Load_WithIncludes(t *testing.T) {
	tempDir := t.TempDir()

	// Create included file
	includedFile := filepath.Join(tempDir, "included.yml")
	includedContent := `
version: 0.1
env:
  INCLUDED_VAR: "included_value"

recipes:
  included_recipe:
    help: "Included recipe"
    run: echo "from included file"
`

	err := os.WriteFile(includedFile, []byte(includedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write included file: %v", err)
	}

	// Create main file with include
	mainFile := filepath.Join(tempDir, "drun.yml")
	mainContent := `
version: 0.1
include:
  - "included.yml"

env:
  MAIN_VAR: "main_value"

recipes:
  main_recipe:
    help: "Main recipe"
    run: echo "from main file"
`

	err = os.WriteFile(mainFile, []byte(mainContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	loader := NewLoader(tempDir)
	spec, err := loader.Load(mainFile)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that both env vars are present
	if spec.Env["INCLUDED_VAR"] != "included_value" {
		t.Errorf("Expected INCLUDED_VAR='included_value', got %q", spec.Env["INCLUDED_VAR"])
	}

	if spec.Env["MAIN_VAR"] != "main_value" {
		t.Errorf("Expected MAIN_VAR='main_value', got %q", spec.Env["MAIN_VAR"])
	}

	// Check that both recipes are present
	if len(spec.Recipes) != 2 {
		t.Fatalf("Expected 2 recipes, got %d", len(spec.Recipes))
	}

	if _, exists := spec.Recipes["included_recipe"]; !exists {
		t.Error("Expected included_recipe to be present")
	}

	if _, exists := spec.Recipes["main_recipe"]; !exists {
		t.Error("Expected main_recipe to be present")
	}
}

func TestLoader_Load_DefaultFilenames(t *testing.T) {
	tempDir := t.TempDir()

	// Test each default filename
	for _, filename := range DefaultFilenames {
		t.Run(filename, func(t *testing.T) {
			// Create subdirectory if needed
			dir := filepath.Dir(filepath.Join(tempDir, filename))
			if dir != tempDir {
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
			}

			specFile := filepath.Join(tempDir, filename)
			specContent := `
version: 0.1
recipes:
  test:
    help: "Test recipe"
    run: echo "test"
`

			err := os.WriteFile(specFile, []byte(specContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			loader := NewLoader(tempDir)
			spec, err := loader.Load("") // Empty string should trigger default file search

			if err != nil {
				t.Fatalf("Expected no error for %s, got %v", filename, err)
			}

			if spec.Version != "0.1" {
				t.Errorf("Expected version '0.1', got %q", spec.Version)
			}

			// Clean up for next test
			_ = os.Remove(specFile)
			if dir != tempDir {
				_ = os.RemoveAll(dir)
			}
		})
	}
}

func TestLoader_Load_NonexistentFile(t *testing.T) {
	tempDir := t.TempDir()

	loader := NewLoader(tempDir)
	_, err := loader.Load("nonexistent.yml")

	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}

func TestLoader_setDefaults(t *testing.T) {
	loader := NewLoader(".")
	spec := &model.Spec{}

	loader.setDefaults(spec)

	// Check that defaults are set
	if spec.Defaults.WorkingDir != "." {
		t.Errorf("Expected working_dir='.', got %q", spec.Defaults.WorkingDir)
	}

	if spec.Defaults.Shell != "auto" {
		t.Errorf("Expected shell='auto', got %q", spec.Defaults.Shell)
	}

	if !spec.Defaults.ExportEnv {
		t.Error("Expected export_env=true")
	}

	if !spec.Defaults.InheritEnv {
		t.Error("Expected inherit_env=true")
	}
}
