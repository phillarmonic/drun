package app

import (
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
)

func TestResolvePartialTaskNameDeduplicatesPlatformVariants(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "shell"},
			{Name: "shell"},
			{Name: "serve"},
		},
	}

	got, err := ResolvePartialTaskName("she", program)
	if err != nil {
		t.Fatalf("ResolvePartialTaskName() error = %v", err)
	}
	if got != "shell" {
		t.Fatalf("expected shell, got %q", got)
	}
}

// levenshteinDistance uses the Go min builtin for its three-way minimum now that
// the local min helper is gone; these cases pin the exact distances.
func TestLevenshteinDistance(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"identical", "build", "build", 0},
		{"empty source", "", "build", 5},
		{"empty target", "build", "", 5},
		{"both empty", "", "", 0},
		{"single substitution", "build", "buiLd", 1},
		{"single insertion", "build", "builds", 1},
		{"single deletion", "builds", "build", 1},
		{"kitten/sitting", "kitten", "sitting", 3},
		{"completely different", "deploy", "test", 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := levenshteinDistance(tc.a, tc.b); got != tc.want {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
