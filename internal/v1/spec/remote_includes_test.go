package spec

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/drun/internal/v1/model"
)

func TestLoader_isRemoteURL(t *testing.T) {
	loader := NewLoader(".")

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"HTTP URL", "http://example.com/file.yml", true},
		{"HTTPS URL", "https://example.com/file.yml", true},
		{"Git URL", "git+https://github.com/org/repo.git", true},
		{"Local file", "local/file.yml", false},
		{"Absolute path", "/absolute/path/file.yml", false},
		{"Relative path", "../relative/file.yml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.isRemoteURL(tt.url)
			if result != tt.expected {
				t.Errorf("isRemoteURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestLoader_parseHTTPInclude(t *testing.T) {
	loader := NewLoader(".")

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"Valid HTTP URL", "http://example.com/file.yml", false},
		{"Valid HTTPS URL", "https://example.com/file.yml", false},
		{"Invalid URL", "://invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote, err := loader.parseHTTPInclude(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseHTTPInclude(%q) expected error, got nil", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("parseHTTPInclude(%q) unexpected error: %v", tt.url, err)
				return
			}

			if remote.Type != "http" {
				t.Errorf("parseHTTPInclude(%q) type = %q, want %q", tt.url, remote.Type, "http")
			}

			if remote.URL != tt.url {
				t.Errorf("parseHTTPInclude(%q) URL = %q, want %q", tt.url, remote.URL, tt.url)
			}

			if remote.CacheKey == "" {
				t.Errorf("parseHTTPInclude(%q) CacheKey is empty", tt.url)
			}
		})
	}
}

func TestLoader_parseGitInclude(t *testing.T) {
	loader := NewLoader(".")

	tests := []struct {
		name         string
		url          string
		expectedURL  string
		expectedRef  string
		expectedPath string
	}{
		{
			"Git URL with branch and path",
			"git+https://github.com/org/repo.git@main:path/file.yml",
			"https://github.com/org/repo.git",
			"main",
			"path/file.yml",
		},
		{
			"Git URL with tag and path",
			"git+https://github.com/org/repo.git@v1.0.0:drun.yml",
			"https://github.com/org/repo.git",
			"v1.0.0",
			"drun.yml",
		},
		{
			"Git URL with branch only",
			"git+https://github.com/org/repo.git@develop",
			"https://github.com/org/repo.git",
			"develop",
			"drun.yml",
		},
		{
			"Git URL with defaults",
			"git+https://github.com/org/repo.git",
			"https://github.com/org/repo.git",
			"main",
			"drun.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote, err := loader.parseGitInclude(tt.url)
			if err != nil {
				t.Errorf("parseGitInclude(%q) unexpected error: %v", tt.url, err)
				return
			}

			if remote.Type != "git" {
				t.Errorf("parseGitInclude(%q) type = %q, want %q", tt.url, remote.Type, "git")
			}

			if remote.URL != tt.expectedURL {
				t.Errorf("parseGitInclude(%q) URL = %q, want %q", tt.url, remote.URL, tt.expectedURL)
			}

			if remote.Ref != tt.expectedRef {
				t.Errorf("parseGitInclude(%q) Ref = %q, want %q", tt.url, remote.Ref, tt.expectedRef)
			}

			if remote.Path != tt.expectedPath {
				t.Errorf("parseGitInclude(%q) Path = %q, want %q", tt.url, remote.Path, tt.expectedPath)
			}

			if remote.CacheKey == "" {
				t.Errorf("parseGitInclude(%q) CacheKey is empty", tt.url)
			}
		})
	}
}

func TestLoader_fetchHTTPInclude(t *testing.T) {
	// Create a test HTTP server
	testContent := `version: 0.1
recipes:
  test:
    help: "Test recipe"
    run: echo "Hello from HTTP include"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	loader := NewLoader(".")

	remote := &RemoteInclude{
		Type:     "http",
		URL:      server.URL,
		CacheKey: "test_key",
	}

	data, err := loader.fetchHTTPInclude(remote)
	if err != nil {
		t.Errorf("fetchHTTPInclude() unexpected error: %v", err)
		return
	}

	if string(data) != testContent {
		t.Errorf("fetchHTTPInclude() data = %q, want %q", string(data), testContent)
	}
}

func TestLoader_fetchHTTPInclude_NotFound(t *testing.T) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	loader := NewLoader(".")

	remote := &RemoteInclude{
		Type:     "http",
		URL:      server.URL,
		CacheKey: "test_key",
	}

	_, err := loader.fetchHTTPInclude(remote)
	if err == nil {
		t.Error("fetchHTTPInclude() expected error for 404, got nil")
	}
}

func TestLoader_processRemoteInclude_HTTP(t *testing.T) {
	// Create a test HTTP server
	testContent := `version: 0.1
env:
  REMOTE_VAR: "from_remote"
recipes:
  remote-test:
    help: "Remote test recipe"
    run: echo "Hello from remote"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	loader := NewLoader(".")

	// Create a base spec
	spec := &model.Spec{
		Version: 1.0,
		Env:     make(map[string]string),
		Recipes: make(map[string]model.Recipe),
	}

	// Process the remote include
	err := loader.processRemoteInclude(spec, server.URL)
	if err != nil {
		t.Errorf("processRemoteInclude() unexpected error: %v", err)
		return
	}

	// Check that the remote content was merged
	if spec.Env["REMOTE_VAR"] != "from_remote" {
		t.Errorf("processRemoteInclude() env var not merged: got %q, want %q", spec.Env["REMOTE_VAR"], "from_remote")
	}

	if _, exists := spec.Recipes["remote-test"]; !exists {
		t.Error("processRemoteInclude() remote recipe not merged")
	}
}

func TestLoader_cacheRemoteInclude(t *testing.T) {
	loader := NewLoader(".")

	testData := []byte("test data")
	cacheKey := "test_cache_key"

	// Cache the data
	loader.cacheRemoteInclude(cacheKey, testData)

	// Retrieve from cache
	cached, found := loader.getCachedRemoteInclude(cacheKey)
	if !found {
		t.Error("getCachedRemoteInclude() data not found in cache")
		return
	}

	if string(cached) != string(testData) {
		t.Errorf("getCachedRemoteInclude() data = %q, want %q", string(cached), string(testData))
	}
}

func TestLoader_ResolveSecrets(t *testing.T) {
	loader := NewLoader(".")

	// Set up test environment variables
	_ = os.Setenv("TEST_SECRET", "secret_value")
	_ = os.Setenv("TEST_OPTIONAL", "optional_value")
	defer func() {
		_ = os.Unsetenv("TEST_SECRET")
		_ = os.Unsetenv("TEST_OPTIONAL")
	}()

	secrets := map[string]model.Secret{
		"required_secret": {
			Source:   "env://TEST_SECRET",
			Required: true,
		},
		"optional_secret": {
			Source:   "env://TEST_OPTIONAL",
			Required: false,
		},
		"missing_optional": {
			Source:   "env://MISSING_VAR",
			Required: false,
		},
	}

	resolved, err := loader.ResolveSecrets(secrets)
	if err != nil {
		t.Errorf("ResolveSecrets() unexpected error: %v", err)
		return
	}

	if resolved["required_secret"] != "secret_value" {
		t.Errorf("ResolveSecrets() required_secret = %q, want %q", resolved["required_secret"], "secret_value")
	}

	if resolved["optional_secret"] != "optional_value" {
		t.Errorf("ResolveSecrets() optional_secret = %q, want %q", resolved["optional_secret"], "optional_value")
	}

	// Missing optional secret should not be in resolved map
	if _, exists := resolved["missing_optional"]; exists {
		t.Error("ResolveSecrets() missing optional secret should not be in resolved map")
	}
}

func TestLoader_ResolveSecrets_RequiredMissing(t *testing.T) {
	loader := NewLoader(".")

	secrets := map[string]model.Secret{
		"required_missing": {
			Source:   "env://MISSING_REQUIRED_VAR",
			Required: true,
		},
	}

	_, err := loader.ResolveSecrets(secrets)
	if err == nil {
		t.Error("ResolveSecrets() expected error for missing required secret, got nil")
	}
}

func TestLoader_resolveSecret_FileSource(t *testing.T) {
	loader := NewLoader(".")

	// Create a temporary file with secret content
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "secret.txt")
	secretContent := "file_secret_value"

	err := os.WriteFile(secretFile, []byte(secretContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	secret := model.Secret{
		Source:   "file://" + secretFile,
		Required: true,
	}

	value, err := loader.resolveSecret("test_secret", secret)
	if err != nil {
		t.Errorf("resolveSecret() unexpected error: %v", err)
		return
	}

	if value != secretContent {
		t.Errorf("resolveSecret() value = %q, want %q", value, secretContent)
	}
}

func TestLoader_resolveSecret_UnsupportedScheme(t *testing.T) {
	loader := NewLoader(".")

	secret := model.Secret{
		Source:   "unsupported://test",
		Required: true,
	}

	_, err := loader.resolveSecret("test_secret", secret)
	if err == nil {
		t.Error("resolveSecret() expected error for unsupported scheme, got nil")
	}
}
