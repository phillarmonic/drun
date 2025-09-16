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
version: 1.0

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

	if spec.Version != 1.0 {
		t.Errorf("Expected version 1.0, got %v", spec.Version)
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
version: 1.0
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
version: 1.0
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
version: 1.0
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
version: 1.0
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
version: 1.0
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
version: 1.0
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
version: 1.0
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

			if spec.Version != 1.0 {
				t.Errorf("Expected version 1.0, got %v", spec.Version)
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

// Tests for new lifecycle and namespacing features

func TestLoader_Load_WithAllLifecycleBlocks(t *testing.T) {
	tempDir := t.TempDir()
	specFile := filepath.Join(tempDir, "drun.yml")

	specContent := `
version: 1.0

recipe-prerun:
  - echo "recipe-prerun block 1"
  - echo "recipe-prerun block 2"

recipe-postrun:
  - echo "recipe-postrun block 1"

before:
  - echo "before block 1"
  - echo "before block 2"

after:
  - echo "after block 1"
  - echo "after block 2"

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
	spec, err := loader.Load(specFile)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check recipe-prerun blocks
	if len(spec.RecipePrerun) != 2 {
		t.Errorf("Expected 2 recipe-prerun blocks, got %d", len(spec.RecipePrerun))
	}
	if spec.RecipePrerun[0] != "echo \"recipe-prerun block 1\"" {
		t.Errorf("Expected first recipe-prerun block to be 'echo \"recipe-prerun block 1\"', got %q", spec.RecipePrerun[0])
	}

	// Check recipe-postrun blocks
	if len(spec.RecipePostrun) != 1 {
		t.Errorf("Expected 1 recipe-postrun block, got %d", len(spec.RecipePostrun))
	}
	if spec.RecipePostrun[0] != "echo \"recipe-postrun block 1\"" {
		t.Errorf("Expected first recipe-postrun block to be 'echo \"recipe-postrun block 1\"', got %q", spec.RecipePostrun[0])
	}

	// Check before blocks
	if len(spec.Before) != 2 {
		t.Errorf("Expected 2 before blocks, got %d", len(spec.Before))
	}
	if spec.Before[0] != "echo \"before block 1\"" {
		t.Errorf("Expected first before block to be 'echo \"before block 1\"', got %q", spec.Before[0])
	}

	// Check after blocks
	if len(spec.After) != 2 {
		t.Errorf("Expected 2 after blocks, got %d", len(spec.After))
	}
	if spec.After[0] != "echo \"after block 1\"" {
		t.Errorf("Expected first after block to be 'echo \"after block 1\"', got %q", spec.After[0])
	}
}

func TestLoader_MergeSpecsWithNamespace(t *testing.T) {
	loader := NewLoader(".")

	// Base spec
	base := &model.Spec{
		Recipes: map[string]model.Recipe{
			"build": {
				Help: "Local build",
				Run:  model.Step{Lines: []string{"echo local build"}},
			},
		},
	}

	// Included spec that will be namespaced
	included := &model.Spec{
		Recipes: map[string]model.Recipe{
			"build": {
				Help: "Docker build",
				Run:  model.Step{Lines: []string{"docker build ."}},
			},
			"push": {
				Help: "Docker push",
				Run:  model.Step{Lines: []string{"docker push"}},
			},
		},
	}

	// Merge with namespace
	loader.mergeSpecsWithNamespace(base, included, "docker")

	// Check that recipes are properly namespaced
	if len(base.Recipes) != 3 {
		t.Errorf("Expected 3 recipes after merge, got %d", len(base.Recipes))
	}

	// Original recipe should still exist
	if _, exists := base.Recipes["build"]; !exists {
		t.Error("Original 'build' recipe should still exist")
	}

	// Namespaced recipes should exist
	if _, exists := base.Recipes["docker:build"]; !exists {
		t.Error("Namespaced 'docker:build' recipe should exist")
	}
	if _, exists := base.Recipes["docker:push"]; !exists {
		t.Error("Namespaced 'docker:push' recipe should exist")
	}

	// Check that the namespaced recipe has the correct content
	dockerBuild := base.Recipes["docker:build"]
	if dockerBuild.Help != "Docker build" {
		t.Errorf("Expected docker:build help to be 'Docker build', got %q", dockerBuild.Help)
	}
}

func TestLoader_ProcessIncludePattern_WithNamespace(t *testing.T) {
	tempDir := t.TempDir()

	// Create main spec file
	mainFile := filepath.Join(tempDir, "drun.yml")
	mainContent := `
version: 1.0
include:
  - "docker::shared/docker.yml"
recipes:
  build:
    help: "Local build"
    run: echo "local build"
`

	// Create shared directory and docker file
	sharedDir := filepath.Join(tempDir, "shared")
	err := os.MkdirAll(sharedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	dockerFile := filepath.Join(sharedDir, "docker.yml")
	dockerContent := `
version: 1.0
recipes:
  build:
    help: "Docker build"
    run: docker build .
  push:
    help: "Docker push"
    run: docker push
`

	err = os.WriteFile(mainFile, []byte(mainContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	err = os.WriteFile(dockerFile, []byte(dockerContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write docker file: %v", err)
	}

	loader := NewLoader(tempDir)
	spec, err := loader.Load(mainFile)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have 3 recipes: build, docker:build, docker:push
	if len(spec.Recipes) != 3 {
		t.Errorf("Expected 3 recipes, got %d", len(spec.Recipes))
	}

	// Check that both local and namespaced recipes exist
	if _, exists := spec.Recipes["build"]; !exists {
		t.Error("Local 'build' recipe should exist")
	}
	if _, exists := spec.Recipes["docker:build"]; !exists {
		t.Error("Namespaced 'docker:build' recipe should exist")
	}
	if _, exists := spec.Recipes["docker:push"]; !exists {
		t.Error("Namespaced 'docker:push' recipe should exist")
	}

	// Check that the recipes have the correct content
	localBuild := spec.Recipes["build"]
	if localBuild.Help != "Local build" {
		t.Errorf("Expected local build help to be 'Local build', got %q", localBuild.Help)
	}

	dockerBuild := spec.Recipes["docker:build"]
	if dockerBuild.Help != "Docker build" {
		t.Errorf("Expected docker:build help to be 'Docker build', got %q", dockerBuild.Help)
	}
}

func TestLoader_MergeAllLifecycleBlocks(t *testing.T) {
	loader := NewLoader(".")

	// Base spec with some lifecycle blocks
	base := &model.Spec{
		RecipePrerun:  []string{"echo base recipe-prerun"},
		RecipePostrun: []string{"echo base recipe-postrun"},
		Before:        []string{"echo base before"},
		After:         []string{"echo base after"},
	}

	// Included spec with additional lifecycle blocks
	included := &model.Spec{
		RecipePrerun:  []string{"echo included recipe-prerun 1", "echo included recipe-prerun 2"},
		RecipePostrun: []string{"echo included recipe-postrun"},
		Before:        []string{"echo included before 1", "echo included before 2"},
		After:         []string{"echo included after"},
	}

	// Merge specs
	loader.mergeSpecsWithNamespace(base, included, "")

	// Check that recipe-prerun blocks are merged (appended)
	expectedRecipePrerun := []string{
		"echo base recipe-prerun",
		"echo included recipe-prerun 1",
		"echo included recipe-prerun 2",
	}
	if len(base.RecipePrerun) != len(expectedRecipePrerun) {
		t.Errorf("Expected %d recipe-prerun blocks, got %d", len(expectedRecipePrerun), len(base.RecipePrerun))
	}
	for i, expected := range expectedRecipePrerun {
		if i < len(base.RecipePrerun) && base.RecipePrerun[i] != expected {
			t.Errorf("Expected recipe-prerun block %d to be %q, got %q", i, expected, base.RecipePrerun[i])
		}
	}

	// Check that recipe-postrun blocks are merged (appended)
	expectedRecipePostrun := []string{
		"echo base recipe-postrun",
		"echo included recipe-postrun",
	}
	if len(base.RecipePostrun) != len(expectedRecipePostrun) {
		t.Errorf("Expected %d recipe-postrun blocks, got %d", len(expectedRecipePostrun), len(base.RecipePostrun))
	}
	for i, expected := range expectedRecipePostrun {
		if i < len(base.RecipePostrun) && base.RecipePostrun[i] != expected {
			t.Errorf("Expected recipe-postrun block %d to be %q, got %q", i, expected, base.RecipePostrun[i])
		}
	}

	// Check that before blocks are merged (appended)
	expectedBefore := []string{
		"echo base before",
		"echo included before 1",
		"echo included before 2",
	}
	if len(base.Before) != len(expectedBefore) {
		t.Errorf("Expected %d before blocks, got %d", len(expectedBefore), len(base.Before))
	}
	for i, expected := range expectedBefore {
		if i < len(base.Before) && base.Before[i] != expected {
			t.Errorf("Expected before block %d to be %q, got %q", i, expected, base.Before[i])
		}
	}

	// Check that after blocks are merged (appended)
	expectedAfter := []string{
		"echo base after",
		"echo included after",
	}
	if len(base.After) != len(expectedAfter) {
		t.Errorf("Expected %d after blocks, got %d", len(expectedAfter), len(base.After))
	}
	for i, expected := range expectedAfter {
		if i < len(base.After) && base.After[i] != expected {
			t.Errorf("Expected after block %d to be %q, got %q", i, expected, base.After[i])
		}
	}
}
