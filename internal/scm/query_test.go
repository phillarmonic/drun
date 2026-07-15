package scm

import (
	"context"
	"testing"
	"time"
)

type querySession struct{ refs []GitRef }

func (s *querySession) Tags(context.Context, bool) ([]GitRef, error) { return s.refs, nil }
func (s *querySession) Close() error                                 { return nil }

func TestGitTagQuerySelectsPHPSeries(t *testing.T) {
	contract, err := NewGitVersionTagContract("", []string{"php-{version}"}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	source := &GitSource{Alias: "php", VersionTags: contract}
	session := &querySession{refs: []GitRef{
		{Name: "php-8.5.9RC1"}, {Name: "php-8.4.24RC"}, {Name: "php-8.6.0-alpha2"},
		{Name: "php-8.5.8"}, {Name: "php-8.3.32"}, {Name: "php-8.4.23"},
	}}
	result, err := (GitTagQuery{Result: "version", Series: "8.4"}).Execute(context.Background(), session, source)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "php-8.4.23" || result.Value("version") != "8.4.23" || result.Value("tag") != "php-8.4.23" {
		t.Fatalf("result = %#v", result)
	}
}

func TestGitTagQueryInlineOverrideDoesNotMutateSource(t *testing.T) {
	sourceContract, _ := NewGitVersionTagContract("semver", nil, "", nil)
	source := &GitSource{VersionTags: sourceContract}
	session := &querySession{refs: []GitRef{{Name: "runtime-1.2.3"}, {Name: "v9.9.9"}}}
	result, err := (GitTagQuery{TagFormat: "runtime-{version}"}).Execute(context.Background(), session, source)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version.Raw != "1.2.3" {
		t.Fatalf("result = %#v", result)
	}
	version, matched, err := source.VersionTags.Extract("v9.9.9")
	if err != nil || !matched || version.Raw != "9.9.9" {
		t.Fatalf("source contract changed: version=%#v matched=%v err=%v", version, matched, err)
	}
}

func TestGitTagQueryRawPatternOverride(t *testing.T) {
	sourceContract, _ := NewGitVersionTagContract("semver", nil, "", nil)
	source := &GitSource{VersionTags: sourceContract}
	session := &querySession{refs: []GitRef{{Name: "runtime-8.4.23"}, {Name: "runtime-8.4.24RC"}, {Name: "v9.9.9"}}}
	result, err := (GitTagQuery{
		TagPattern: `^runtime-(?P<version>[0-9]+\.[0-9]+\.[0-9]+)$`,
		Series:     "8.4",
	}).Execute(context.Background(), session, source)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tag != "runtime-8.4.23" {
		t.Fatalf("result = %#v", result)
	}
}

func TestGitTagQueryOrderingAndTieBreaks(t *testing.T) {
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	session := &querySession{refs: []GitRef{
		{Name: "v2.0.0", Date: date.Add(-time.Hour)},
		{Name: "v1.0.0", Date: date},
		{Name: "1.0.0", Date: date},
	}}
	byVersion, err := (GitTagQuery{}).Execute(context.Background(), session, nil)
	if err != nil || byVersion.Tag != "v2.0.0" {
		t.Fatalf("version result = %#v, err = %v", byVersion, err)
	}
	byDate, err := (GitTagQuery{OrderBy: "date"}).Execute(context.Background(), session, nil)
	if err != nil || byDate.Tag != "v1.0.0" {
		t.Fatalf("date result = %#v, err = %v", byDate, err)
	}
}
