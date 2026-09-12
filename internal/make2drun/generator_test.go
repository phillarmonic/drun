package make2drun

import (
	"strings"
	"testing"
)

// generateCommand collapses two formerly duplicated branches: the "@" (silent)
// prefix no longer changes the emitted echo/run statement, and commands with
// shell features or embedded newlines share the plain run branch. These cases
// pin the generated output so that collapse stays behaviour-preserving.
func TestGenerateCommandPrefixesAndShellFeatures(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "atsign prefix is stripped and echo is converted",
			cmd:  "@echo hello",
			want: "\techo \"hello\"\n",
		},
		{
			name: "atsign prefix does not change the run statement",
			cmd:  "@go build",
			want: "\trun \"go build\"\n",
		},
		{
			name: "dash prefix wraps the command in try/ignore",
			cmd:  "-false",
			want: "\ttry:\n\t\trun \"false\"\n\tignore:\n\t\twarn \"Command failed but continuing\"\n",
		},
		{
			name: "mkdir becomes create dir",
			cmd:  "mkdir -p build",
			want: "\tcreate dir \"build\"\n",
		},
		{
			name: "rm becomes delete",
			cmd:  "rm -rf build",
			want: "\tdelete \"build\"\n",
		},
		{
			name: "shell features use the plain run branch",
			cmd:  "a && b",
			want: "\trun \"a && b\"\n",
		},
		{
			name: "embedded newline uses the plain run branch",
			cmd:  "line one\nline two",
			want: "\trun \"line one\nline two\"\n",
		},
		{
			name: "blank command emits nothing",
			cmd:  "   ",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var sb strings.Builder
			generateCommand(&sb, tc.cmd, &Makefile{})

			if got := sb.String(); got != tc.want {
				t.Errorf("generateCommand(%q) = %q, want %q", tc.cmd, got, tc.want)
			}
		})
	}
}

func TestGenerateCommandConvertsMakeVariables(t *testing.T) {
	var sb strings.Builder
	generateCommand(&sb, "go build -o bin/$(APP)", &Makefile{})

	want := "\trun \"go build -o bin/{$app}\"\n"
	if got := sb.String(); got != want {
		t.Errorf("generateCommand() = %q, want %q", got, want)
	}
}
