package provisioning

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phillarmonic/drun/v2/internal/cache"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

type fakeGitHubFetcher struct {
	calls   int
	content map[string][]byte
}

func (f *fakeGitHubFetcher) Fetch(_ context.Context, path, ref string) ([]byte, error) {
	f.calls++
	key := path + "@" + ref
	if content, ok := f.content[key]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

type fakeHTTPSFetcher struct {
	calls   int
	content map[string][]byte
}

func (f *fakeHTTPSFetcher) Fetch(_ context.Context, path, _ string) ([]byte, error) {
	f.calls++
	if content, ok := f.content[path]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

type fakeGitFetcher struct {
	calls   int
	content map[string][]byte
}

func (f *fakeGitFetcher) FetchManifest(_ context.Context, repoURL, manifestPath, ref string) ([]byte, error) {
	f.calls++
	key := repoURL + "::" + manifestPath + "::" + ref
	if content, ok := f.content[key]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func TestResolverResolveRequirement_ProjectSourceWinsAndAutoResolvesDir(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	projectCatalog := filepath.Join(projectDir, "tooling")
	if err := os.MkdirAll(projectCatalog, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	projectManifest := `version: "1"
provisionings:
  golangci-lint:
    aliases: ["golangci"]
    targets:
      - os: darwin
        arch: arm64
        install: "brew install golangci-lint"
        install_versioned: "brew install golangci-lint@{version}"
`
	if err := os.WriteFile(filepath.Join(projectCatalog, "provisionings.yaml"), []byte(projectManifest), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resolver := NewResolver(projectDir, WithCurrentPlatform("darwin", "arm64"))
	resolution, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{
		Name: "golangci",
		Constraints: []statement.VersionConstraint{
			{Operator: ">=", Version: "1.64"},
			{Operator: "<=", Version: "1.64"},
		},
	}, SourceSet{
		Project: []string{"./tooling"},
		User:    []string{"/does/not/matter.yaml"},
	})
	if err != nil {
		t.Fatalf("ResolveRequirement() error = %v", err)
	}

	if resolution.Source != filepath.Join(projectCatalog, "provisionings.yaml") {
		t.Fatalf("resolution.Source = %q", resolution.Source)
	}
	if resolution.Entry.Name != "golangci-lint" {
		t.Fatalf("resolution.Entry.Name = %q", resolution.Entry.Name)
	}
	if resolution.MatchedName != "golangci" {
		t.Fatalf("resolution.MatchedName = %q", resolution.MatchedName)
	}
	if resolution.ExactVersion != "1.64" {
		t.Fatalf("resolution.ExactVersion = %q", resolution.ExactVersion)
	}
	if !resolution.UsesVersionedInstall {
		t.Fatalf("expected versioned install to be selected")
	}
	if got := resolution.InstallCommand(); got != "brew install golangci-lint@1.64" {
		t.Fatalf("InstallCommand() = %q", got)
	}
}

func TestResolverResolveRequirement_FallsBackToUserThenEmbedded(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	userManifest := filepath.Join(projectDir, "user-provisionings.yaml")
	if err := os.WriteFile(userManifest, []byte(`version: "1"
provisionings:
  govulncheck:
    targets:
      - os: linux
        install: "go install govulncheck@latest"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resolver := NewResolver(projectDir,
		WithCurrentPlatform("linux", "amd64"),
		WithEmbeddedSources([]EmbeddedSource{{
			Name: "embedded-defaults",
			Content: []byte(`version: "1"
provisionings:
  staticcheck:
    targets:
      - install: "go install honnef.co/go/tools/cmd/staticcheck@latest"
`),
		}}),
	)

	userResolution, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "govulncheck"}, SourceSet{
		User: []string{userManifest},
	})
	if err != nil {
		t.Fatalf("ResolveRequirement(user) error = %v", err)
	}
	if userResolution.Source != userManifest {
		t.Fatalf("user resolution source = %q", userResolution.Source)
	}

	embeddedResolution, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "staticcheck"}, SourceSet{})
	if err != nil {
		t.Fatalf("ResolveRequirement(embedded) error = %v", err)
	}
	if embeddedResolution.Source != "embedded-defaults" {
		t.Fatalf("embedded resolution source = %q", embeddedResolution.Source)
	}
}

func TestResolverResolveRequirement_RemoteSourcesAndCache(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	withEnv(t, "HOME", homeDir, func() {
		cacheManager, err := cache.NewManager(time.Minute, false)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() { _ = cacheManager.Close() })

		github := &fakeGitHubFetcher{
			content: map[string][]byte{
				"acme/catalog/provisionings.yaml@main": []byte(`version: "1"
provisionings:
  gosec:
    targets:
      - os: linux
        install: "brew install gosec"
`),
			},
		}

		resolver1 := NewResolver(t.TempDir(),
			WithCurrentPlatform("linux", "amd64"),
			WithCacheManager(cacheManager),
			WithGitHubFetcher(github),
		)
		if _, err := resolver1.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "gosec"}, SourceSet{
			Project: []string{"github:acme/catalog/provisionings.yaml@main"},
		}); err != nil {
			t.Fatalf("first ResolveRequirement() error = %v", err)
		}

		resolver2 := NewResolver(t.TempDir(),
			WithCurrentPlatform("linux", "amd64"),
			WithCacheManager(cacheManager),
			WithGitHubFetcher(github),
		)
		if _, err := resolver2.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "gosec"}, SourceSet{
			Project: []string{"github:acme/catalog/provisionings.yaml@main"},
		}); err != nil {
			t.Fatalf("second ResolveRequirement() error = %v", err)
		}

		if github.calls != 1 {
			t.Fatalf("github fetch calls = %d, want 1", github.calls)
		}
	})
}

func TestResolverResolveRequirement_HTTPSAndSSHSources(t *testing.T) {
	t.Parallel()

	https := &fakeHTTPSFetcher{
		content: map[string][]byte{
			"https://example.com/provisionings.yaml": []byte(`version: "1"
provisionings:
  golangci-lint:
    targets:
      - os: linux
        install: "curl https://example.com/install.sh | sh"
`),
		},
	}
	git := &fakeGitFetcher{
		content: map[string][]byte{
			"ssh://git@github.com/acme/internal-tooling.git::catalog/provisionings.yaml::main": []byte(`version: "1"
provisionings:
  gosec:
    targets:
      - os: linux
        install: "make install-gosec"
`),
		},
	}

	resolver := NewResolver(t.TempDir(),
		WithCurrentPlatform("linux", "amd64"),
		WithHTTPSFetcher(https),
		WithGitFetcher(git),
	)

	if _, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "golangci-lint"}, SourceSet{
		Project: []string{"https://example.com/provisionings.yaml"},
	}); err != nil {
		t.Fatalf("ResolveRequirement(https) error = %v", err)
	}

	if _, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "gosec"}, SourceSet{
		Project: []string{"ssh://git@github.com/acme/internal-tooling.git//catalog/provisionings.yaml?ref=main"},
	}); err != nil {
		t.Fatalf("ResolveRequirement(ssh) error = %v", err)
	}

	if https.calls != 1 {
		t.Fatalf("https calls = %d, want 1", https.calls)
	}
	if git.calls != 1 {
		t.Fatalf("git calls = %d, want 1", git.calls)
	}
}

func TestResolverResolveRequirement_FirstMatchingSourceDoesNotFallThrough(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	projectManifest := filepath.Join(projectDir, "project.yaml")
	if err := os.WriteFile(projectManifest, []byte(`version: "1"
provisionings:
  gosec:
    targets:
      - os: windows
        install: "choco install gosec"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	userManifest := filepath.Join(projectDir, "user.yaml")
	if err := os.WriteFile(userManifest, []byte(`version: "1"
provisionings:
  gosec:
    targets:
      - os: linux
        install: "go install gosec@latest"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resolver := NewResolver(projectDir, WithCurrentPlatform("linux", "amd64"))
	_, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "gosec"}, SourceSet{
		Project: []string{projectManifest},
		User:    []string{userManifest},
	})
	if err == nil {
		t.Fatal("expected error when first matching source has no compatible target")
	}
	if strings.Contains(err.Error(), userManifest) {
		t.Fatalf("unexpected fallback to user source: %v", err)
	}
}

func TestResolverResolveRequirement_RejectsAmbiguousTargetsAndDuplicateAliases(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	ambiguous := filepath.Join(projectDir, "ambiguous.yaml")
	if err := os.WriteFile(ambiguous, []byte(`version: "1"
provisionings:
  gosec:
    targets:
      - os: linux
        install: "one"
      - os: linux
        install: "two"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resolver := NewResolver(projectDir, WithCurrentPlatform("linux", "amd64"))
	_, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "gosec"}, SourceSet{
		Project: []string{ambiguous},
	})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}

	duplicate := filepath.Join(projectDir, "duplicate.yaml")
	if err := os.WriteFile(duplicate, []byte(`version: "1"
provisionings:
  golangci-lint:
    aliases: ["lint"]
    targets:
      - install: "first"
  lint:
    targets:
      - install: "second"
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{Name: "golangci-lint"}, SourceSet{
		Project: []string{duplicate},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate provisioning name or alias") {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func TestDeriveExactVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		constraints []statement.VersionConstraint
		exact       string
		ok          bool
	}{
		{
			name: "closed range exact version",
			constraints: []statement.VersionConstraint{
				{Operator: ">=", Version: "2.22"},
				{Operator: "<=", Version: "2.22"},
			},
			exact: "2.22",
			ok:    true,
		},
		{
			name: "open range does not become exact",
			constraints: []statement.VersionConstraint{
				{Operator: ">=", Version: "2.22"},
			},
		},
		{
			name: "strict range does not become exact",
			constraints: []statement.VersionConstraint{
				{Operator: ">", Version: "2.22"},
				{Operator: "<", Version: "2.23"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exact, ok, err := deriveExactVersion(tt.constraints)
			if err != nil {
				t.Fatalf("deriveExactVersion() error = %v", err)
			}
			if exact != tt.exact || ok != tt.ok {
				t.Fatalf("deriveExactVersion() = (%q, %t), want (%q, %t)", exact, ok, tt.exact, tt.ok)
			}
		})
	}
}

func withEnv(t *testing.T, key, value string, fn func()) {
	t.Helper()

	originalValue, hadValue := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Setenv(%q) error = %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if hadValue {
			err = os.Setenv(key, originalValue)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("restore env %q error = %v", key, err)
		}
	})

	fn()
}
