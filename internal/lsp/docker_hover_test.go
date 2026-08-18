package lsp

import (
	"strings"
	"testing"
)

func TestDockerNetworkHover(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		column int
		want   []string
	}{
		{
			name:   "if docker network condition wins over plain if",
			line:   `  if docker network "proxy" exists:`,
			column: 5,
			want:   []string{"Docker network condition", "if docker network \"<name>\" [not] exists:", "dry-run"},
		},
		{
			name:   "if docker network condition on the network keyword",
			line:   `  if docker network "proxy" exists:`,
			column: 12,
			want:   []string{"Docker network condition"},
		},
		{
			name:   "if docker network not exists",
			line:   `  if docker network "{$app_network}" not exists:`,
			column: 6,
			want:   []string{"Docker network condition", "[not] exists"},
		},
		{
			name:   "when docker network condition wins over plain when",
			line:   `  when docker network "legacy-bridge" exists:`,
			column: 6,
			want:   []string{"Docker network condition", "otherwise"},
		},
		{
			name:   "plain if still gets generic hover",
			line:   `  if $env is "production":`,
			column: 4,
			want:   []string{"Conditional execution"},
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
					t.Errorf("hover contents missing %q, got:\n%s", want, got.Contents.Value)
				}
			}
		})
	}
}
