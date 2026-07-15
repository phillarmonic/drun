package scm

import (
	"strings"
	"testing"
)

func TestPHPFormatSelectsStableSeries(t *testing.T) {
	contract, err := NewGitVersionTagContract("", []string{"php-{version}"}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	constraint, err := SeriesConstraint("8.4")
	if err != nil {
		t.Fatal(err)
	}
	tags := []string{
		"php-8.5.9RC1", "php-8.4.24RC", "php-8.6.0-alpha2",
		"php-8.5.8", "php-8.3.32", "php-8.4.23",
	}
	var selected string
	for _, tag := range tags {
		version, matched, extractErr := contract.Extract(tag)
		if extractErr != nil {
			t.Fatal(extractErr)
		}
		if matched && constraint.Matches(version) {
			selected = tag
		}
	}
	if selected != "php-8.4.23" {
		t.Fatalf("selected %q, want php-8.4.23", selected)
	}
}

func TestPresetTemplateAndPatternAreEquivalent(t *testing.T) {
	contracts := []*GitVersionTagContract{}
	for _, input := range []struct {
		preset  string
		formats []string
		pattern string
	}{
		{preset: "semver"},
		{formats: []string{"v{version}"}},
		{pattern: `^v(?P<version>[0-9]+\.[0-9]+\.[0-9]+)$`},
	} {
		contract, err := NewGitVersionTagContract(input.preset, input.formats, input.pattern, nil)
		if err != nil {
			t.Fatal(err)
		}
		contracts = append(contracts, contract)
	}
	for _, contract := range contracts {
		version, matched, err := contract.Extract("v1.22.3")
		if err != nil || !matched || version.Raw != "1.22.3" {
			t.Fatalf("got version=%#v matched=%v err=%v", version, matched, err)
		}
	}
}

func TestTagFormatEscapesLiteralsAndBraces(t *testing.T) {
	contract, err := NewGitVersionTagContract("", []string{"release.+{{{version}}}"}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	version, matched, err := contract.Extract("release.+{2.4.6}")
	if err != nil || !matched || version.Raw != "2.4.6" {
		t.Fatalf("got version=%#v matched=%v err=%v", version, matched, err)
	}
	if _, matched, _ := contract.Extract("releaseXX{2.4.6}"); matched {
		t.Fatal("format literal was treated as a regular expression")
	}
}

func TestTagFormatValidation(t *testing.T) {
	invalid := []string{"release", "{version}-{version}", "{unknown}", "{version}}"}
	for _, format := range invalid {
		if _, err := NewGitVersionTagContract("", []string{format}, "", nil); err == nil {
			t.Errorf("format %q should fail validation", format)
		}
	}
}

func TestRawPatternDiagnosticSuggestsSimpleFormat(t *testing.T) {
	_, err := NewGitVersionTagContract("", nil, `^php-([0-9]+\.[0-9]+\.[0-9]+)$`, nil)
	if err == nil || !strings.Contains(err.Error(), `version tags: "php-{version}"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestVersionConstraints(t *testing.T) {
	version, _ := ParseVersion("8.4.23")
	series, _ := SeriesConstraint("8.4")
	if !series.Matches(version) {
		t.Fatal("8.4.23 should match series 8.4")
	}
	rangeConstraint, err := ParseVersionConstraint(">=8.4.0 <8.5.0")
	if err != nil || !rangeConstraint.Matches(version) {
		t.Fatalf("range did not match: %v", err)
	}
	outside, _ := ParseVersion("8.5.0")
	if rangeConstraint.Matches(outside) {
		t.Fatal("8.5.0 should not match an upper-exclusive 8.5.0 bound")
	}
}

func TestConventionalDefaultAcceptsBareAndVTags(t *testing.T) {
	contract, err := NewGitVersionTagContract("", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"1.2.3", "v1.2.3"} {
		version, matched, extractErr := contract.Extract(tag)
		if extractErr != nil || !matched || version.Raw != "1.2.3" {
			t.Fatalf("tag %q: version=%#v matched=%v err=%v", tag, version, matched, extractErr)
		}
	}
	for _, tag := range []string{"v1.2.3-rc1", "1.2", "release-1.2.3"} {
		if _, matched, _ := contract.Extract(tag); matched {
			t.Fatalf("unstable or nonconventional tag %q matched", tag)
		}
	}
}

func TestMultipleFormatsRejectConflictingExtractions(t *testing.T) {
	contract, err := NewGitVersionTagContract("", []string{
		"{version}-release-2.0.0",
		"1.0.0-release-{version}",
	}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := contract.Extract("1.0.0-release-2.0.0"); err == nil || !strings.Contains(err.Error(), "conflicting versions") {
		t.Fatalf("error = %v", err)
	}
}
