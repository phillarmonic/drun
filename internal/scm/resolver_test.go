package scm

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

type fakeAdapter struct {
	opened  *GitAccessProfile
	session *fakeSession
}

func (f *fakeAdapter) Capabilities() GitProviderCapabilities {
	return GitProviderCapabilities{RemoteRefs: true}
}
func (f *fakeAdapter) Open(_ context.Context, _ *GitSource, profile *GitAccessProfile, _ bool) (GitRepositorySession, error) {
	f.opened = profile
	if f.session == nil {
		return nil, errors.New("missing session")
	}
	return f.session, nil
}

type fakeSession struct{ closed bool }

func (f *fakeSession) Tags(context.Context, bool) ([]GitRef, error) {
	return []GitRef{{Name: "v1.2.3", Date: time.Unix(1, 0)}}, nil
}
func (f *fakeSession) Close() error { f.closed = true; return nil }

func TestResolverUsesExplicitOrDefaultMethodWithoutFallback(t *testing.T) {
	filesystem := &fakeAdapter{session: &fakeSession{}}
	resolver := GitSourceResolver{
		Registry: &GitRegistry{Sources: map[string]*GitSource{
			"app": {
				Alias: "app", Provider: "generic", Default: "filesystem",
				Access: map[string]*GitAccessProfile{
					"filesystem": {Method: "filesystem", Path: "{$repo}/app"},
					"https":      {Method: "https", URL: "https://example.test/app.git"},
				},
			},
		}},
		Adapters: map[string]GitProviderAdapter{AdapterKey("*", "filesystem"): filesystem},
		BaseDir:  "/workspace",
		Expand: func(value string) (string, error) {
			if value == "{$repo}/app" {
				return "sources/app", nil
			}
			return value, nil
		},
	}
	resolved, err := resolver.Resolve("app", "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Access.Path != filepath.Clean("/workspace/sources/app") {
		t.Fatalf("path = %q", resolved.Access.Path)
	}
	if _, err := resolver.Resolve("app", "https"); err == nil {
		t.Fatal("expected missing HTTPS adapter error instead of method fallback")
	}
	session, err := resolver.Open(context.Background(), "app", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if !filesystem.session.closed {
		t.Fatal("session was not closed")
	}
}
