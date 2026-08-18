package lsp

import (
	"strings"
	"testing"
)

func TestNetworkHover(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		column int
		want   []string
	}{
		{
			name:   "test connection action",
			line:   `  test connection to "localhost" on port 5432 timeout "5s"`,
			column: 5,
			want:   []string{"TCP port check", "test connection to \"<host>\" on port <port>", "database.example.com"},
		},
		{
			name:   "check if port action",
			line:   `  check if port 6379 is open on "redis.local"`,
			column: 9,
			want:   []string{"TCP port check", "check if port <port> is open on \"<host>\""},
		},
		{
			name:   "if port condition wins over plain if",
			line:   `  if port 5432 is open on "localhost":`,
			column: 5,
			want:   []string{"TCP port condition", "is [not] open on \"<host>\"", "dry-run"},
		},
		{
			name:   "if port condition on the port keyword",
			line:   `  if port 5432 is open on "localhost":`,
			column: 8,
			want:   []string{"TCP port condition"},
		},
		{
			name:   "when port condition wins over plain when",
			line:   `  when port 8080 is open on "localhost":`,
			column: 6,
			want:   []string{"TCP port condition", "otherwise"},
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
					t.Errorf("hover contents missing %q:\n%s", want, got.Contents.Value)
				}
			}
		})
	}
}
