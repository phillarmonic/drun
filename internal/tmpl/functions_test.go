package tmpl

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/drun/internal/model"
)

func TestDockerComposeFunc(t *testing.T) {
	// Test docker compose command detection
	result := dockerComposeFunc()

	// The result should be either "docker compose", "docker-compose", or empty
	if result != "" && result != "docker compose" && result != "docker-compose" {
		t.Errorf("dockerComposeFunc() = %q, expected 'docker compose', 'docker-compose', or empty", result)
	}
}

func TestDockerBuildxFunc(t *testing.T) {
	// Test docker buildx command detection
	result := dockerBuildxFunc()

	// The result should be either "docker buildx", "docker-buildx", or empty
	if result != "" && result != "docker buildx" && result != "docker-buildx" {
		t.Errorf("dockerBuildxFunc() = %q, expected 'docker buildx', 'docker-buildx', or empty", result)
	}
}

func TestHasCommandFunc(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"Existing command", "echo", true},
		{"Non-existing command", "definitely-not-a-command-12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCommandFunc(tt.command)
			if result != tt.expected {
				t.Errorf("hasCommandFunc(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestHasDockerAndSubcommand(t *testing.T) {
	// Only test if docker is available
	if !hasCommandFunc("docker") {
		t.Skip("Docker not available, skipping test")
	}

	tests := []struct {
		name       string
		subcommand string
		// We can't predict the result since it depends on the system
		// Just test that it doesn't panic
	}{
		{"Valid subcommand", "version"},
		{"Invalid subcommand", "definitely-not-a-subcommand"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure it doesn't panic
			_ = hasDockerAndSubcommand(tt.subcommand)
		})
	}
}

func TestStatusFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) string
		message  string
		expected string
	}{
		{"info", infoFunc, "test message", "echo \"ℹ️  test message\""},
		{"warn", warnFunc, "warning message", "echo \"⚠️  warning message\""},
		{"error", errorFunc, "error message", "echo \"❌ error message\""},
		{"success", successFunc, "success message", "echo \"✅ success message\""},
		{"step", stepFunc, "step message", "echo \"🚀 step message\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.message)
			if result != tt.expected {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.message, result, tt.expected)
			}
		})
	}
}

func TestGitBranchFunc(t *testing.T) {
	// Only test if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		t.Skip("Not in a git repository, skipping test")
	}

	result := gitBranchFunc()
	if result == "" {
		t.Error("gitBranchFunc() returned empty string in git repository")
	}

	// Should return a valid branch name, tag, or detached state
	if result != "unknown" {
		// Valid results include:
		// - Branch names (e.g., "main", "feature/test")
		// - Tag names (e.g., "v1.0.0")
		// - Detached state (e.g., "detached@abc1234")
		t.Logf("gitBranchFunc() returned: %s", result)
	} else {
		t.Error("gitBranchFunc() returned 'unknown' - Git commands failed")
	}
}

func TestGitCommitFunc(t *testing.T) {
	// Only test if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		t.Skip("Not in a git repository, skipping test")
	}

	result := gitCommitFunc()
	if result == "" {
		t.Error("gitCommitFunc() returned empty string in git repository")
	}

	if result == "unknown" {
		t.Error("gitCommitFunc() returned 'unknown' - Git command failed")
		return
	}

	// Should be a 40-character hex string
	if len(result) != 40 {
		t.Errorf("gitCommitFunc() returned %d characters, want 40", len(result))
	}

	t.Logf("gitCommitFunc() returned: %s", result)
}

func TestGitShortCommitFunc(t *testing.T) {
	// Only test if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		t.Skip("Not in a git repository, skipping test")
	}

	result := gitShortCommitFunc()
	if result == "" {
		t.Error("gitShortCommitFunc() returned empty string in git repository")
	}

	if result == "unknown" {
		t.Error("gitShortCommitFunc() returned 'unknown' - Git command failed")
		return
	}

	// Should be a 7-character hex string (or possibly shorter in some cases)
	if len(result) < 4 || len(result) > 12 {
		t.Errorf("gitShortCommitFunc() returned %d characters, expected 4-12", len(result))
	}

	t.Logf("gitShortCommitFunc() returned: %s", result)
}

func TestIsDirtyFunc(t *testing.T) {
	// Only test if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		t.Skip("Not in a git repository, skipping test")
	}

	// Just test that it doesn't panic - result depends on repository state
	_ = isDirtyFunc()
}

func TestPackageManagerFunc(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{"Go project", []string{"go.mod"}, "go"},
		{"Node.js with package-lock", []string{"package.json", "package-lock.json"}, "npm"},
		{"Node.js with yarn.lock", []string{"package.json", "yarn.lock"}, "yarn"},
		{"Python with requirements", []string{"requirements.txt"}, "pip"},
		{"No package manager", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tempDir, tt.name)
			_ = os.MkdirAll(testDir, 0755)
			_ = os.Chdir(testDir)

			// Create test files
			for _, file := range tt.files {
				f, err := os.Create(file)
				if err != nil {
					t.Fatalf("Failed to create test file %s: %v", file, err)
				}
				_ = f.Close()
			}

			result := packageManagerFunc()
			if result != tt.expected {
				t.Errorf("packageManagerFunc() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHasFileFunc(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(tempDir)

	// Create a test file
	testFile := "test-file.txt"
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_ = f.Close()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"Existing file", testFile, true},
		{"Non-existing file", "non-existent.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasFileFunc(tt.filename)
			if result != tt.expected {
				t.Errorf("hasFileFunc(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsCIFunc(t *testing.T) {
	// Save original environment
	originalCI := os.Getenv("CI")
	originalGithubActions := os.Getenv("GITHUB_ACTIONS")
	defer func() {
		_ = os.Setenv("CI", originalCI)
		_ = os.Setenv("GITHUB_ACTIONS", originalGithubActions)
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			"No CI environment",
			map[string]string{"CI": "", "GITHUB_ACTIONS": ""},
			false,
		},
		{
			"CI environment set",
			map[string]string{"CI": "true"},
			true,
		},
		{
			"GitHub Actions environment",
			map[string]string{"GITHUB_ACTIONS": "true"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment
			for key, value := range tt.envVars {
				if value == "" {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, value)
				}
			}

			result := isCIFunc()
			if result != tt.expected {
				t.Errorf("isCIFunc() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEngine_SecretFunctions(t *testing.T) {
	engine := NewEngine(map[string]string{})

	// Create test context with secrets
	ctx := &model.ExecutionContext{
		Vars: make(map[string]any),
		Env:  make(map[string]string),
		Secrets: map[string]string{
			"test_secret": "secret_value",
			"api_key":     "12345",
		},
		Flags:       make(map[string]any),
		Positionals: make(map[string]any),
		OS:          "linux",
		Arch:        "amd64",
	}

	// Set context for secret functions
	engine.currentCtx = ctx

	tests := []struct {
		name           string
		secretName     string
		expectedValue  string
		expectedExists bool
	}{
		{"Existing secret", "test_secret", "secret_value", true},
		{"Another existing secret", "api_key", "12345", true},
		{"Non-existing secret", "missing_secret", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test secretFunc
			value := engine.secretFunc(tt.secretName)
			if value != tt.expectedValue {
				t.Errorf("secretFunc(%q) = %q, want %q", tt.secretName, value, tt.expectedValue)
			}

			// Test hasSecretFunc
			exists := engine.hasSecretFunc(tt.secretName)
			if exists != tt.expectedExists {
				t.Errorf("hasSecretFunc(%q) = %v, want %v", tt.secretName, exists, tt.expectedExists)
			}
		})
	}
}

func TestEngine_SecretFunctions_NoContext(t *testing.T) {
	engine := NewEngine(map[string]string{})

	// Test without setting context
	value := engine.secretFunc("any_secret")
	if value != "" {
		t.Errorf("secretFunc() without context = %q, want empty string", value)
	}

	exists := engine.hasSecretFunc("any_secret")
	if exists {
		t.Error("hasSecretFunc() without context = true, want false")
	}
}
