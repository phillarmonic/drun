package compatibility_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/engine"
	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

// TestV2CompatibilityContract protects representative behavior accepted by
// stable v2 releases. Existing fixtures are append-only: changing an assertion
// requires the compatibility-exception process documented for maintainers.
func TestV2CompatibilityContract(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		task     string
		params   map[string]string
		tasks    []string
		contains []string
		excludes []string
		wantErr  string
	}{
		{
			name:     "task discovery selection and interpolation",
			fixture:  "tasks-and-interpolation.drun",
			task:     "hello",
			params:   map[string]string{"name": "Ada"},
			tasks:    []string{"hello", "second task"},
			contains: []string{"Hello, Ada!"},
		},
		{
			name:     "required parameter and default",
			fixture:  "parameters-and-defaults.drun",
			task:     "deploy",
			params:   map[string]string{"environment": "staging"},
			tasks:    []string{"deploy"},
			contains: []string{"Deploying latest to staging"},
		},
		{
			name:    "parameter validation",
			fixture: "parameters-and-defaults.drun",
			task:    "deploy",
			params:  map[string]string{"environment": "invalid"},
			tasks:   []string{"deploy"},
			wantErr: "must be one of",
		},
		{
			name:     "conditional production branch",
			fixture:  "control-flow.drun",
			task:     "select path",
			params:   map[string]string{"environment": "production"},
			tasks:    []string{"select path", "iterate"},
			contains: []string{"Production path"},
			excludes: []string{"Non-production path"},
		},
		{
			name:     "conditional fallback branch",
			fixture:  "control-flow.drun",
			task:     "select path",
			params:   map[string]string{"environment": "dev"},
			tasks:    []string{"select path", "iterate"},
			contains: []string{"Non-production path"},
			excludes: []string{"Production path"},
		},
		{
			name:     "loop behavior",
			fixture:  "control-flow.drun",
			task:     "iterate",
			tasks:    []string{"select path", "iterate"},
			contains: []string{"Item one", "Item two"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("testdata", "v2.0", test.fixture))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			p := parser.NewParser(lexer.NewLexer(string(source)))
			program := p.ParseProgram()
			if errors := p.Errors(); len(errors) > 0 {
				t.Fatalf("parse fixture: %v", errors)
			}

			var taskNames []string
			for _, task := range program.Tasks {
				taskNames = append(taskNames, task.Name)
			}
			if !slices.Equal(taskNames, test.tasks) {
				t.Fatalf("tasks = %v, want %v", taskNames, test.tasks)
			}

			var output bytes.Buffer
			runner := engine.NewEngine(&output)
			runner.SetDryRun(true)
			err = runner.ExecuteWithParams(program, test.task, test.params)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want error containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("execute fixture: %v", err)
			}

			for _, expected := range test.contains {
				if !strings.Contains(output.String(), expected) {
					t.Errorf("output does not contain %q:\n%s", expected, output.String())
				}
			}
			for _, unexpected := range test.excludes {
				if strings.Contains(output.String(), unexpected) {
					t.Errorf("output unexpectedly contains %q:\n%s", unexpected, output.String())
				}
			}
		})
	}
}
