package lsp

import (
	"strings"
	"testing"
)

func TestFilesystemAliasHover(t *testing.T) {
	for _, source := range []string{
		`create folder "cache"`,
		`create directory "cache"`,
		`delete folder "cache"`,
		`delete directory "cache"`,
	} {
		got := hoverForSource(source, position{Line: 0, Character: strings.Index(source, "folder") + 1})
		if strings.Contains(source, "directory") {
			got = hoverForSource(source, position{Line: 0, Character: strings.Index(source, "directory") + 1})
		}
		if got == nil {
			t.Errorf("expected hover for %q", source)
		}
	}
}
