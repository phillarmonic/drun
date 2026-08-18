package lsp

import (
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/builtins"
)

func TestFunctionHover(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		column int
		want   []string
	}{
		{
			name:   "now.format inside quoted interpolation",
			line:   `  given $ts defaults to "{now.format('2006-01-02')}"`,
			column: 30,
			want:   []string{"Formatted current time", "Go time layout"},
		},
		{
			name:   "current git branch in unquoted interpolation",
			line:   `  set $b to {current git branch}`,
			column: 20,
			want:   []string{"Current branch name"},
		},
		{
			name:   "multi-word longest match wins",
			line:   `  info "status: {docker compose status}"`,
			column: 30,
			want:   []string{"Docker Compose status", "mixed"},
		},
		{
			name:   "env function",
			line:   `  info "env: {env('HOME', '/root')}"`,
			column: 14,
			want:   []string{"Environment variable"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := hoverForSource(test.line, position{Line: 0, Character: test.column})
			if got == nil {
				t.Fatalf("hoverForSource() = nil, want contents containing %q", test.want)
			}
			for _, want := range test.want {
				if !strings.Contains(got.Contents.Value, want) {
					t.Errorf("hoverForSource() missing %q in:\n%s", want, got.Contents.Value)
				}
			}
		})
	}
}

func TestFunctionHoverCoversRegistry(t *testing.T) {
	documented := make(map[string]bool, len(functionHoverEntries))
	for _, entry := range functionHoverEntries {
		documented[entry.Phrase] = true
	}
	for name := range builtins.Registry {
		if !documented[name] {
			t.Errorf("built-in function %q has no hover entry", name)
		}
	}
}

func TestFunctionHoverIgnoresOutsideInterpolation(t *testing.T) {
	// The word "os" outside an interpolation must not be documented.
	line := `  info "the os is here"`
	if got := hoverForSource(line, position{Line: 0, Character: 12}); got != nil {
		if strings.Contains(got.Contents.Value, "Operating system") {
			t.Errorf("unexpected function hover outside interpolation:\n%s", got.Contents.Value)
		}
	}
}
