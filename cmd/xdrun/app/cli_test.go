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

func TestYesFlagParses(t *testing.T) {
	app := NewApp("test", "test", "test")
	if err := app.rootCmd.ParseFlags([]string{"--yes"}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	if !app.assumeYes {
		t.Fatalf("assumeYes = false, want true")
	}
	if app.assumeNo {
		t.Fatalf("assumeNo = true, want false")
	}
}

func TestYesShortFlagParses(t *testing.T) {
	app := NewApp("test", "test", "test")
	if err := app.rootCmd.ParseFlags([]string{"-y"}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	if !app.assumeYes {
		t.Fatalf("assumeYes = false, want true")
	}
}

func TestNoFlagParses(t *testing.T) {
	app := NewApp("test", "test", "test")
	if err := app.rootCmd.ParseFlags([]string{"--no"}); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	if !app.assumeNo {
		t.Fatalf("assumeNo = false, want true")
	}
	if app.assumeYes {
		t.Fatalf("assumeYes = true, want false")
	}
}
