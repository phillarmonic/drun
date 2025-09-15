package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/model"
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
