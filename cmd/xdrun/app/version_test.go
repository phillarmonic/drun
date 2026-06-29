package app

import "testing"

func TestFormatBuildDateRFC3339(t *testing.T) {
	got := formatBuildDate("2026-06-29T13:53:17Z")
	want := "29/06/2026 13:53 UTC"

	if got != want {
		t.Fatalf("formatBuildDate() = %q, want %q", got, want)
	}
}

func TestFormatBuildDatePreservesNonRFC3339(t *testing.T) {
	input := "29/06/2026 13:43:17 (dev build)"

	got := formatBuildDate(input)
	if got != input {
		t.Fatalf("formatBuildDate() = %q, want %q", got, input)
	}
}
