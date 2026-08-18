package changelog

import (
	"strings"
	"testing"
	"time"
)

func mustDate(t *testing.T, raw string) time.Time {
	t.Helper()
	date, err := ParseDate(raw)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", raw, err)
	}
	return date
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"1.5.0", "1.5.0", false},
		{"v1.5.0", "1.5.0", false},
		{" 1.5.0 ", "1.5.0", false},
		{"1.5", "", true},
		{"1.5.0-beta", "", true},
		{"banana", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := NormalizeVersion(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeVersion(%q): expected error, got %q", tt.raw, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeVersion(%q): %v", tt.raw, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	if _, err := ParseDate("2026-08-10"); err != nil {
		t.Errorf("ParseDate valid date: %v", err)
	}
	for _, raw := range []string{"2026-02-30", "2026-13-01", "10-08-2026", "banana", ""} {
		if _, err := ParseDate(raw); err == nil {
			t.Errorf("ParseDate(%q): expected error, got none", raw)
		}
	}
}

func TestPromoteWithSubsectionsAndCompareLinks(t *testing.T) {
	content := `# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]
Upstream support for Drun v2.27
### Added

- Added syntax highlighting support.

### Changed

### Deprecated

### Removed

### Fixed

### Security

[Unreleased]: https://github.com/acme/widget/compare/v1.2.0...HEAD
`

	want := `# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [1.5.0] - 2026-08-10

Upstream support for Drun v2.27
### Added

- Added syntax highlighting support.

### Changed

### Deprecated

### Removed

### Fixed

### Security

[Unreleased]: https://github.com/acme/widget/compare/v1.5.0...HEAD
[1.5.0]: https://github.com/acme/widget/compare/v1.2.0...v1.5.0
`

	got, err := Promote(content, "v1.5.0", mustDate(t, "2026-08-10"))
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if got != want {
		t.Errorf("Promote mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPromoteBeforeExistingReleases(t *testing.T) {
	content := `# Changelog

## [Unreleased]

### Added

- New thing.

## [1.2.0] - 2026-07-15

### Added

- Old thing.

[Unreleased]: https://github.com/acme/widget/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/acme/widget/releases/tag/v1.2.0
`

	want := `# Changelog

## [Unreleased]

### Added

## [1.3.0] - 2026-08-10

### Added

- New thing.

## [1.2.0] - 2026-07-15

### Added

- Old thing.

[Unreleased]: https://github.com/acme/widget/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/acme/widget/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/acme/widget/releases/tag/v1.2.0
`

	got, err := Promote(content, "1.3.0", mustDate(t, "2026-08-10"))
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if got != want {
		t.Errorf("Promote mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPromoteWithoutSubsectionsOrLinks(t *testing.T) {
	content := `# Changelog

## [Unreleased]

- Loose entry without subsections.
`

	want := `# Changelog

## [Unreleased]

## [0.2.0] - 2026-08-10

- Loose entry without subsections.
`

	got, err := Promote(content, "0.2.0", mustDate(t, "2026-08-10"))
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if got != want {
		t.Errorf("Promote mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPromoteMergesIntoExistingReleaseSection(t *testing.T) {
	content := `# Changelog

## [Unreleased]

### Added

- Another new thing.

### Fixed

- A fix.

## [1.3.0] - 2026-08-10

### Added

- New thing.

[Unreleased]: https://github.com/acme/widget/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/acme/widget/compare/v1.2.0...v1.3.0
`

	want := `# Changelog

## [Unreleased]

### Added

### Fixed

## [1.3.0] - 2026-08-10

### Added

- New thing.
- Another new thing.

### Fixed

- A fix.

[Unreleased]: https://github.com/acme/widget/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/acme/widget/compare/v1.2.0...v1.3.0
`

	got, err := Promote(content, "1.3.0", mustDate(t, "2026-08-15"))
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if got != want {
		t.Errorf("Promote mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestPromoteWithExistingSectionAndEmptyUnreleasedIsNoOp(t *testing.T) {
	content := `# Changelog

## [Unreleased]

### Added

### Changed

## [1.3.0] - 2026-08-10

### Added

- New thing.

[Unreleased]: https://github.com/acme/widget/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/acme/widget/compare/v1.2.0...v1.3.0
`

	got, err := Promote(content, "v1.3.0", mustDate(t, "2026-08-15"))
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if got != content {
		t.Errorf("expected a byte-identical no-op\ngot:\n%s\nwant:\n%s", got, content)
	}
}

func TestPromoteErrors(t *testing.T) {
	date := mustDate(t, "2026-08-10")

	if _, err := Promote("# Changelog\n\n## [1.0.0] - 2026-01-01\n", "1.1.0", date); err == nil ||
		!strings.Contains(err.Error(), "Unreleased") {
		t.Errorf("missing Unreleased section: %v", err)
	}

	existing := "# Changelog\n\n## [Unreleased]\n\n- x\n\n## [1.0.0] - 2026-01-01\n\n- y\n"
	if _, err := Promote(existing, "banana", date); err == nil {
		t.Error("invalid version: expected error, got none")
	}
}
