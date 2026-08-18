package lsp

import (
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/patterns"
)

func TestMacroHover(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		column int
		want   []string
	}{
		{
			name:   "semver macro",
			line:   `  requires $version as string matching semver`,
			column: 42,
			want:   []string{"Pattern macro", "Basic semantic versioning", `^v\d+\.\d+\.\d+$`},
		},
		{
			name:   "underscore macro name",
			line:   `  requires $release as string matching semver_optional_v`,
			column: 45,
			want:   []string{"Pattern macro", "optional", "semver_optional_v"},
		},
		{
			name:   "docker_tag macro",
			line:   `  requires $tag as string matching docker_tag`,
			column: 40,
			want:   []string{"Pattern macro", "Docker image tag"},
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

func TestMacroHoverCoversAllMacros(t *testing.T) {
	for name := range patterns.GetAllMacros() {
		line := "  requires $value as string matching " + name
		column := strings.Index(line, name) + 1
		got := hoverForSource(line, position{Line: 0, Character: column})
		if got == nil {
			t.Errorf("macro %q has no hover", name)
			continue
		}
		if !strings.Contains(got.Contents.Value, name) {
			t.Errorf("macro %q hover missing its name:\n%s", name, got.Contents.Value)
		}
	}
}

func TestMacroHoverIgnoresNonMatchingContext(t *testing.T) {
	// A variable that merely shares a macro name is not documented.
	line := `  set $semver to "1.0.0"`
	if got := hoverForSource(line, position{Line: 0, Character: 8}); got != nil {
		if strings.Contains(got.Contents.Value, "Pattern macro") {
			t.Errorf("unexpected macro hover for non-matching context:\n%s", got.Contents.Value)
		}
	}
}
