package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/model"
	"github.com/spf13/cobra"
)

func TestIsPositionalArgName(t *testing.T) {
	positionals := []model.PositionalArg{
		{Name: "name", Required: true},
		{Name: "title", Default: "friend"},
		{Name: "message", Default: "Hello"},
	}

	tests := []struct {
		name     string
		argName  string
		expected bool
	}{
		{"existing arg", "name", true},
		{"existing optional arg", "title", true},
		{"existing default arg", "message", true},
		{"non-existing arg", "unknown", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPositionalArgName(positionals, tt.argName)
			if result != tt.expected {
				t.Errorf("isPositionalArgName(%q) = %v, want %v", tt.argName, result, tt.expected)
			}
		})
	}
}

func TestParsePositionalsWithNamed_BasicUsage(t *testing.T) {
	positionals := []model.PositionalArg{
		{Name: "name", Required: true},
		{Name: "title", Default: "friend"},
		{Name: "message", Default: "Hello"},
	}

	tests := []struct {
		name      string
		args      []string
		namedArgs map[string]string
		expected  map[string]any
		wantError bool
	}{
		{
			name:      "all positional",
			args:      []string{"Alice", "Ms.", "Hi"},
			namedArgs: map[string]string{},
			expected: map[string]any{
				"name":    "Alice",
				"title":   "Ms.",
				"message": "Hi",
			},
			wantError: false,
		},
		{
			name:      "all named",
			args:      []string{},
			namedArgs: map[string]string{"name": "Bob", "title": "Mr.", "message": "Greetings"},
			expected: map[string]any{
				"name":    "Bob",
				"title":   "Mr.",
				"message": "Greetings",
			},
			wantError: false,
		},
		{
			name:      "mixed positional and named",
			args:      []string{"Charlie"},
			namedArgs: map[string]string{"title": "Dr.", "message": "Welcome"},
			expected: map[string]any{
				"name":    "Charlie",
				"title":   "Dr.",
				"message": "Welcome",
			},
			wantError: false,
		},
		{
			name:      "required arg missing",
			args:      []string{},
			namedArgs: map[string]string{"title": "Dr."},
			expected:  nil,
			wantError: true,
		},
		{
			name:      "unknown named arg",
			args:      []string{"Alice"},
			namedArgs: map[string]string{"unknown": "value"},
			expected:  nil,
			wantError: true,
		},
		{
			name:      "defaults applied",
			args:      []string{"David"},
			namedArgs: map[string]string{},
			expected: map[string]any{
				"name":    "David",
				"title":   "friend",
				"message": "Hello",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePositionalsWithNamed(positionals, tt.args, tt.namedArgs)

			if tt.wantError {
				if err == nil {
					t.Errorf("parsePositionalsWithNamed() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parsePositionalsWithNamed() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parsePositionalsWithNamed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParsePositionalsWithNamed_OneOfConstraint(t *testing.T) {
	positionals := []model.PositionalArg{
		{Name: "environment", Required: true, OneOf: []string{"dev", "staging", "prod"}},
		{Name: "version", Default: "latest"},
	}

	tests := []struct {
		name      string
		args      []string
		namedArgs map[string]string
		expected  map[string]any
		wantError bool
	}{
		{
			name:      "valid one_of value positional",
			args:      []string{"prod", "v1.2.3"},
			namedArgs: map[string]string{},
			expected: map[string]any{
				"environment": "prod",
				"version":     "v1.2.3",
			},
			wantError: false,
		},
		{
			name:      "valid one_of value named",
			args:      []string{},
			namedArgs: map[string]string{"environment": "staging", "version": "v2.0.0"},
			expected: map[string]any{
				"environment": "staging",
				"version":     "v2.0.0",
			},
			wantError: false,
		},
		{
			name:      "invalid one_of value positional",
			args:      []string{"invalid", "v1.0.0"},
			namedArgs: map[string]string{},
			expected:  nil,
			wantError: true,
		},
		{
			name:      "invalid one_of value named",
			args:      []string{},
			namedArgs: map[string]string{"environment": "invalid"},
			expected:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePositionalsWithNamed(positionals, tt.args, tt.namedArgs)

			if tt.wantError {
				if err == nil {
					t.Errorf("parsePositionalsWithNamed() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parsePositionalsWithNamed() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parsePositionalsWithNamed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParsePositionalsWithNamed_VariadicArgs(t *testing.T) {
	positionals := []model.PositionalArg{
		{Name: "pattern", Default: "*"},
		{Name: "files", Variadic: true},
	}

	tests := []struct {
		name      string
		args      []string
		namedArgs map[string]string
		expected  map[string]any
		wantError bool
	}{
		{
			name:      "variadic with positional args",
			args:      []string{"*.go", "file1.go", "file2.go", "file3.go"},
			namedArgs: map[string]string{},
			expected: map[string]any{
				"pattern": "*.go",
				"files":   []string{"file1.go", "file2.go", "file3.go"},
			},
			wantError: false,
		},
		{
			name:      "variadic with named pattern",
			args:      []string{"file1.go", "file2.go"},
			namedArgs: map[string]string{"pattern": "*.js"},
			expected: map[string]any{
				"pattern": "*.js",
				"files":   []string{"file1.go", "file2.go"},
			},
			wantError: false,
		},
		{
			name:      "variadic named arg",
			args:      []string{},
			namedArgs: map[string]string{"pattern": "*.ts", "files": "main.ts"},
			expected: map[string]any{
				"pattern": "*.ts",
				"files":   []string{"main.ts"},
			},
			wantError: false,
		},
		{
			name:      "no variadic args provided",
			args:      []string{"*.py"},
			namedArgs: map[string]string{},
			expected: map[string]any{
				"pattern": "*.py",
			},
			wantError: false,
		},
		{
			name:      "default pattern with variadic",
			args:      []string{"main.c", "utils.c"},
			namedArgs: map[string]string{},
			expected: map[string]any{
				"pattern": "main.c",            // First arg becomes pattern
				"files":   []string{"utils.c"}, // Remaining args become files
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePositionalsWithNamed(positionals, tt.args, tt.namedArgs)

			if tt.wantError {
				if err == nil {
					t.Errorf("parsePositionalsWithNamed() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parsePositionalsWithNamed() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parsePositionalsWithNamed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseRecipeArgs_NamedArguments(t *testing.T) {
	recipe := model.Recipe{
		Positionals: []model.PositionalArg{
			{Name: "target", Required: true},
			{Name: "arch", Default: "amd64"},
		},
		Flags: map[string]model.Flag{
			"verbose": {Type: "bool", Default: false},
			"output":  {Type: "string", Default: "dist"},
		},
	}

	tests := []struct {
		name              string
		args              []string
		expectedPositions map[string]any
		expectedFlags     map[string]any
		wantError         bool
	}{
		{
			name: "flag-style named arguments",
			args: []string{"--target=myapp", "--arch=arm64", "--verbose", "--output=build"},
			expectedPositions: map[string]any{
				"target": "myapp",
				"arch":   "arm64",
			},
			expectedFlags: map[string]any{
				"verbose": true,
				"output":  "build",
			},
			wantError: false,
		},
		{
			name: "assignment-style named arguments",
			args: []string{"target=webapp", "arch=amd64", "--verbose"},
			expectedPositions: map[string]any{
				"target": "webapp",
				"arch":   "amd64",
			},
			expectedFlags: map[string]any{
				"verbose": true,
				"output":  "dist", // default value
			},
			wantError: false,
		},
		{
			name: "mixed positional and named",
			args: []string{"service", "--arch=arm64", "--output=bin"},
			expectedPositions: map[string]any{
				"target": "service",
				"arch":   "arm64",
			},
			expectedFlags: map[string]any{
				"verbose": false, // default value
				"output":  "bin",
			},
			wantError: false,
		},
		{
			name: "traditional positional only",
			args: []string{"api", "x86", "--verbose"},
			expectedPositions: map[string]any{
				"target": "api",
				"arch":   "x86",
			},
			expectedFlags: map[string]any{
				"verbose": true,
				"output":  "dist", // default value
			},
			wantError: false,
		},
		{
			name:              "unknown named argument",
			args:              []string{"--unknown=value"},
			expectedPositions: nil,
			expectedFlags:     nil,
			wantError:         true,
		},
		{
			name:              "missing required positional",
			args:              []string{"--arch=arm64"},
			expectedPositions: nil,
			expectedFlags:     nil,
			wantError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positionals, flags, err := parseRecipeArgs(recipe, tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseRecipeArgs() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseRecipeArgs() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(positionals, tt.expectedPositions) {
				t.Errorf("parseRecipeArgs() positionals = %v, want %v", positionals, tt.expectedPositions)
			}

			if !reflect.DeepEqual(flags, tt.expectedFlags) {
				t.Errorf("parseRecipeArgs() flags = %v, want %v", flags, tt.expectedFlags)
			}
		})
	}
}

func TestParseRecipeArgs_EdgeCases(t *testing.T) {
	recipe := model.Recipe{
		Positionals: []model.PositionalArg{
			{Name: "input", Required: true},
		},
		Flags: map[string]model.Flag{
			"count": {Type: "int", Default: 1},
		},
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "assignment with equals in value",
			args:      []string{"input=key=value"},
			wantError: false,
		},
		{
			name:      "flag-style without value",
			args:      []string{"--input"},
			wantError: true,
			errorMsg:  "requires a value",
		},
		{
			name:      "regular arg with equals (not named)",
			args:      []string{"file=with=equals"},
			wantError: false,
		},
		{
			name:      "empty named argument value",
			args:      []string{"input="},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseRecipeArgs(recipe, tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseRecipeArgs() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("parseRecipeArgs() error = %v, expected to contain %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("parseRecipeArgs() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseRecipeArgs_ComplexScenario(t *testing.T) {
	// Test a complex recipe similar to the deploy example in the README
	recipe := model.Recipe{
		Positionals: []model.PositionalArg{
			{Name: "environment", Required: true, OneOf: []string{"dev", "staging", "prod"}},
			{Name: "version", Default: "latest"},
			{Name: "features", Variadic: true},
		},
		Flags: map[string]model.Flag{
			"force":   {Type: "bool", Default: false},
			"timeout": {Type: "int", Default: 300},
			"config":  {Type: "string", Default: ""},
		},
	}

	tests := []struct {
		name              string
		args              []string
		expectedPositions map[string]any
		expectedFlags     map[string]any
		wantError         bool
	}{
		{
			name: "complex mixed usage",
			args: []string{"prod", "--version=v1.2.3", "auth", "ui", "--force", "--timeout=600"},
			expectedPositions: map[string]any{
				"environment": "prod",
				"version":     "v1.2.3",
				"features":    []string{"auth", "ui"},
			},
			expectedFlags: map[string]any{
				"force":   true,
				"timeout": 600,
				"config":  "",
			},
			wantError: false,
		},
		{
			name: "all named with variadic",
			args: []string{"--environment=staging", "version=v2.0.0", "--force", "--config=prod.yml"},
			expectedPositions: map[string]any{
				"environment": "staging",
				"version":     "v2.0.0",
			},
			expectedFlags: map[string]any{
				"force":   true,
				"timeout": 300, // default
				"config":  "prod.yml",
			},
			wantError: false,
		},
		{
			name:              "invalid environment with named args",
			args:              []string{"--environment=invalid", "--version=v1.0.0"},
			expectedPositions: nil,
			expectedFlags:     nil,
			wantError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positionals, flags, err := parseRecipeArgs(recipe, tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseRecipeArgs() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseRecipeArgs() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(positionals, tt.expectedPositions) {
				t.Errorf("parseRecipeArgs() positionals = %v, want %v", positionals, tt.expectedPositions)
			}

			if !reflect.DeepEqual(flags, tt.expectedFlags) {
				t.Errorf("parseRecipeArgs() flags = %v, want %v", flags, tt.expectedFlags)
			}
		})
	}
}

// Test completion functionality
func TestCompleteRecipes(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "drun_completion_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory and change to temp dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create test drun.yml
	testConfig := `version: 0.1
recipes:
  build:
    help: "Build the project"
    run: echo "building"
  test:
    help: "Run tests"
    run: echo "testing"
  deploy:
    help: "Deploy the application"
    run: echo "deploying"
`
	configPath := filepath.Join(tempDir, "drun.yml")
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create mock root command with subcommands
	rootCmd := &cobra.Command{
		Use:   "drun",
		Short: "A YAML-based task runner",
	}

	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate completion script",
	}

	cleanupBackupsCmd := &cobra.Command{
		Use:   "cleanup-backups",
		Short: "Clean up old drun backup files",
	}

	helpCmd := &cobra.Command{
		Use:   "help",
		Short: "Help about any command",
	}

	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(cleanupBackupsCmd)
	rootCmd.AddCommand(helpCmd)

	// Test completion with no args (should return recipes first, then drun commands)
	t.Run("completion with no args", func(t *testing.T) {
		// Set configFile to our test config
		originalConfigFile := configFile
		configFile = "drun.yml"
		defer func() { configFile = originalConfigFile }()

		completions, directive := completeRecipes(rootCmd, []string{}, "")

		// Verify we got completions
		if len(completions) == 0 {
			t.Fatal("Expected completions, got none")
		}

		// Verify directive
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("Expected ShellCompDirectiveNoFileComp, got %v", directive)
		}

		// Check that we have recipes first
		foundRecipes := 0
		foundSeparator := false
		foundDrunCommands := 0
		separatorIndex := -1

		for i, comp := range completions {
			if comp == "---\t" {
				foundSeparator = true
				separatorIndex = i
			} else if strings.Contains(comp, "(drun CLI command)") {
				foundDrunCommands++
			} else if comp != "---\t" {
				foundRecipes++
			}
		}

		// Verify we found expected elements
		if foundRecipes != 3 {
			t.Errorf("Expected 3 recipes, found %d", foundRecipes)
		}

		if !foundSeparator {
			t.Error("Expected separator, not found")
		}

		if foundDrunCommands != 3 {
			t.Errorf("Expected 3 drun commands, found %d", foundDrunCommands)
		}

		// Verify separator is in the middle
		if separatorIndex <= 0 || separatorIndex >= len(completions)-1 {
			t.Errorf("Separator should be in the middle, found at index %d", separatorIndex)
		}

		// Verify recipes come before separator
		for i := 0; i < separatorIndex; i++ {
			if strings.Contains(completions[i], "(drun CLI command)") {
				t.Errorf("Found drun command before separator at index %d: %s", i, completions[i])
			}
		}

		// Verify drun commands come after separator
		for i := separatorIndex + 1; i < len(completions); i++ {
			if !strings.Contains(completions[i], "(drun CLI command)") {
				t.Errorf("Expected drun command after separator at index %d: %s", i, completions[i])
			}
		}
	})

	// Test completion with recipe specified (should delegate to recipe argument completion)
	t.Run("completion with recipe specified", func(t *testing.T) {
		originalConfigFile := configFile
		configFile = "drun.yml"
		defer func() { configFile = originalConfigFile }()

		_, directive := completeRecipes(rootCmd, []string{"build"}, "")

		// Should return empty for now since build recipe has no positionals/flags in test
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("Expected ShellCompDirectiveNoFileComp, got %v", directive)
		}

		// The function should delegate to completeRecipeArguments, which returns empty for our simple test recipe
		// This is expected behavior
	})

	// Test completion with non-existent recipe
	t.Run("completion with non-existent recipe", func(t *testing.T) {
		originalConfigFile := configFile
		configFile = "drun.yml"
		defer func() { configFile = originalConfigFile }()

		completions, directive := completeRecipes(rootCmd, []string{"nonexistent"}, "")

		// Should return nil for non-existent recipe
		if completions != nil {
			t.Errorf("Expected nil completions for non-existent recipe, got %v", completions)
		}

		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("Expected ShellCompDirectiveNoFileComp, got %v", directive)
		}
	})

	// Test completion with invalid config file
	t.Run("completion with invalid config", func(t *testing.T) {
		originalConfigFile := configFile
		configFile = "nonexistent.yml"
		defer func() { configFile = originalConfigFile }()

		completions, directive := completeRecipes(rootCmd, []string{}, "")

		// Should return nil when config can't be loaded
		if completions != nil {
			t.Errorf("Expected nil completions for invalid config, got %v", completions)
		}

		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("Expected ShellCompDirectiveNoFileComp, got %v", directive)
		}
	})
}

func TestCompleteRecipes_OnlyRecipes(t *testing.T) {
	// Test case where we have recipes but no subcommands
	tempDir, err := os.MkdirTemp("", "drun_completion_test_recipes_only")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create test drun.yml
	testConfig := `version: 0.1
recipes:
  start:
    help: "Start the service"
    run: echo "starting"
`
	configPath := filepath.Join(tempDir, "drun.yml")
	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create root command with no subcommands
	rootCmd := &cobra.Command{
		Use:   "drun",
		Short: "A YAML-based task runner",
	}

	originalConfigFile := configFile
	configFile = "drun.yml"
	defer func() { configFile = originalConfigFile }()

	completions, _ := completeRecipes(rootCmd, []string{}, "")

	// Should only have the recipe, no separator
	if len(completions) != 1 {
		t.Errorf("Expected 1 completion, got %d: %v", len(completions), completions)
	}

	if completions[0] != "start\tStart the service" {
		t.Errorf("Expected 'start\\tStart the service', got %s", completions[0])
	}
}

// Note: TestCompleteRecipes_OnlySubcommands was removed because drun requires
// at least one recipe to be defined - a config with no recipes fails validation
