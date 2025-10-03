package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

func TestTernaryOperatorInterpolation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		params         map[string]string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "ternary with boolean true",
			input: `version: 2.0

task "test":
  given $enabled as boolean defaults to "true"
  info "Result: {$enabled ? '--flag' : ''}"`,
			taskName:       "test",
			params:         map[string]string{"enabled": "true"},
			expectedOutput: []string{"Result: --flag"},
		},
		{
			name: "ternary with boolean false",
			input: `version: 2.0

task "test":
  given $enabled as boolean defaults to "false"
  info "Result: {$enabled ? '--flag' : '--no-flag'}"`,
			taskName:       "test",
			params:         map[string]string{"enabled": "false"},
			expectedOutput: []string{"Result: --no-flag"},
			notExpected:    []string{"Result: --flag"},
		},
		{
			name: "ternary in docker build command",
			input: `version: 2.0

task "build":
  given $no_cache as boolean defaults to "false"
  info "docker build {$no_cache ? '--no-cache' : ''} -t myapp ."`,
			taskName:       "build",
			params:         map[string]string{"no_cache": "true"},
			expectedOutput: []string{"docker build --no-cache -t myapp ."},
		},
		{
			name: "ternary with empty string for false",
			input: `version: 2.0

task "test":
  given $debug as boolean defaults to "false"
  info "Flags: {$debug ? '--verbose' : ''}"`,
			taskName:       "test",
			params:         map[string]string{"debug": "false"},
			expectedOutput: []string{"Flags: "},
		},
		{
			name: "ternary with 'yes' as truthy value",
			input: `version: 2.0

task "test":
  given $option defaults to "yes"
  info "Status: {$option ? 'enabled' : 'disabled'}"`,
			taskName:       "test",
			params:         map[string]string{"option": "yes"},
			expectedOutput: []string{"Status: enabled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			var output bytes.Buffer
			engine := NewEngine(&output)

			err := engine.ExecuteWithParams(program, tt.taskName, tt.params)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			outputStr := output.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestIfThenElseInterpolation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		params         map[string]string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "if-then-else with boolean",
			input: `version: 2.0

task "test":
  given $prod as boolean defaults to "true"
  info "Mode: {if $prod then 'production' else 'development'}"`,
			taskName:       "test",
			params:         map[string]string{"prod": "true"},
			expectedOutput: []string{"Mode: production"},
			notExpected:    []string{"Mode: development"},
		},
		{
			name: "if-then-else with string comparison",
			input: `version: 2.0

task "test":
  given $env defaults to "staging"
  info "Config: {if $env is 'production' then 'prod.yml' else 'dev.yml'}"`,
			taskName:       "test",
			params:         map[string]string{"env": "staging"},
			expectedOutput: []string{"Config: dev.yml"},
			notExpected:    []string{"Config: prod.yml"},
		},
		{
			name: "if-then-else with is comparison matching",
			input: `version: 2.0

task "test":
  given $env defaults to "production"
  info "Config: {if $env is 'production' then 'prod.yml' else 'dev.yml'}"`,
			taskName:       "test",
			params:         map[string]string{"env": "production"},
			expectedOutput: []string{"Config: prod.yml"},
			notExpected:    []string{"Config: dev.yml"},
		},
		{
			name: "if-then-else with is not comparison",
			input: `version: 2.0

task "test":
  given $env defaults to "dev"
  info "Replicas: {if $env is not 'production' then '1' else '3'}"`,
			taskName:       "test",
			params:         map[string]string{"env": "dev"},
			expectedOutput: []string{"Replicas: 1"},
			notExpected:    []string{"Replicas: 3"},
		},
		{
			name: "if-then-else with is not comparison (production)",
			input: `version: 2.0

task "test":
  given $env defaults to "production"
  info "Replicas: {if $env is not 'production' then '1' else '3'}"`,
			taskName:       "test",
			params:         map[string]string{"env": "production"},
			expectedOutput: []string{"Replicas: 3"},
			notExpected:    []string{"Replicas: 1"},
		},
		{
			name: "if-then-else in docker command",
			input: `version: 2.0

task "build":
  given $cache defaults to "no"
  info "docker build {if $cache is 'yes' then '' else '--no-cache'} -t myapp ."`,
			taskName:       "build",
			params:         map[string]string{"cache": "no"},
			expectedOutput: []string{"docker build --no-cache -t myapp ."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			var output bytes.Buffer
			engine := NewEngine(&output)

			err := engine.ExecuteWithParams(program, tt.taskName, tt.params)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			outputStr := output.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestConditionalInterpolationRealWorld(t *testing.T) {
	input := `version: 2.0

task "docker-build":
  given $no_cache as boolean defaults to "false"
  given $platform defaults to "linux/amd64"
  given $push as boolean defaults to "false"
  
  info "Building Docker image..."
  info "docker build {$no_cache ? '--no-cache' : ''} --platform {$platform} {$push ? '--push' : ''} -t myapp:latest ."
  success "Build command generated!"`

	tests := []struct {
		name           string
		params         map[string]string
		expectedOutput string
	}{
		{
			name:           "default build",
			params:         map[string]string{},
			expectedOutput: "docker build  --platform linux/amd64  -t myapp:latest .",
		},
		{
			name:           "no cache build",
			params:         map[string]string{"no_cache": "true"},
			expectedOutput: "docker build --no-cache --platform linux/amd64  -t myapp:latest .",
		},
		{
			name:           "push build",
			params:         map[string]string{"push": "true"},
			expectedOutput: "docker build  --platform linux/amd64 --push -t myapp:latest .",
		},
		{
			name:           "no cache and push",
			params:         map[string]string{"no_cache": "true", "push": "true"},
			expectedOutput: "docker build --no-cache --platform linux/amd64 --push -t myapp:latest .",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := lexer.NewLexer(input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			var output bytes.Buffer
			engine := NewEngine(&output)

			err := engine.ExecuteWithParams(program, "docker-build", tt.params)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			outputStr := output.String()
			if !strings.Contains(outputStr, tt.expectedOutput) {
				t.Errorf("Expected output to contain %q, but got:\n%s", tt.expectedOutput, outputStr)
			}
		})
	}
}
