package app

import "testing"

func TestAllowToolVersionChangesFlagParses(t *testing.T) {
	app := NewApp("test", "test", "test")
	if err := app.rootCmd.ParseFlags([]string{"--allow-tool-version-changes"}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	if !app.allowToolVersionChanges {
		t.Fatalf("allowToolVersionChanges = false, want true")
	}
}
