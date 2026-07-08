package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadUserConfigNormalizesProvisioningSources(t *testing.T) {
	homeDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(homeDir, ".drun"), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	config := `extraTaskFileSearchPaths:
  - automation/project.drun
  - " automation/project.drun "
provisioningSources:
  - "~/.drun/provisionings.yaml"
  - " github:acme/shared/catalog/provisionings.yaml@stable "
  - "~/.drun/provisionings.yaml"
`
	if err := os.WriteFile(filepath.Join(homeDir, ".drun", "config.yml"), []byte(config), 0o600); err != nil {
		t.Fatalf("WriteFile(config.yml) error = %v", err)
	}

	withEnv(t, "HOME", homeDir, func() {
		got, err := loadUserConfig()
		if err != nil {
			t.Fatalf("loadUserConfig() error = %v", err)
		}

		wantSearchPaths := []string{"automation/project.drun"}
		if !reflect.DeepEqual(got.ExtraTaskFileSearchPaths, wantSearchPaths) {
			t.Fatalf("ExtraTaskFileSearchPaths = %#v, want %#v", got.ExtraTaskFileSearchPaths, wantSearchPaths)
		}

		wantSources := []string{
			"~/.drun/provisionings.yaml",
			"github:acme/shared/catalog/provisionings.yaml@stable",
		}
		if !reflect.DeepEqual(got.ProvisioningSources, wantSources) {
			t.Fatalf("ProvisioningSources = %#v, want %#v", got.ProvisioningSources, wantSources)
		}
	})
}

func TestNormalizeStringListFiltersEmptyAndDuplicateValues(t *testing.T) {
	got := normalizeStringList([]string{" alpha ", "", "alpha", "beta", " beta "})
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeStringList() = %#v, want %#v", got, want)
	}
}
