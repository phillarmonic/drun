package secrets_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/engine"
	"github.com/phillarmonic/drun/internal/secrets"
)

// TestIntegration_SecretStatements tests the full flow from parsing to execution
func TestIntegration_SecretStatements(t *testing.T) {
	// Create temp directory for fallback storage
	tmpDir, err := os.MkdirTemp("", "drun-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Parse a simple drun file with secret operations
	input := `
version: 2.0

project "test-project" version "1.0":

task "test":
  secret set "api_key" to "secret123"
  secret set "db_pass" to "pass456"
  info "API Key is: {secret('api_key')}"
  secret delete "api_key"
  secret delete "db_pass"
`

	// Parse
	program, err := engine.ParseStringWithFilename(input, "test.drun")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Create secrets manager with fallback backend
	secretsMgr, err := secrets.NewManager(
		secrets.WithFallback(),
		secrets.WithStoragePath(filepath.Join(tmpDir, "secrets.enc")),
	)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}

	// Capture output
	var output strings.Builder

	// Create engine with secrets support
	eng := engine.NewEngineWithOptions(
		engine.WithOutput(&output),
		engine.WithSecretsManager(secretsMgr),
	)

	// Execute the task
	err = eng.Execute(program, "test")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	// Verify output contains the interpolated secret
	outputStr := output.String()
	if !strings.Contains(outputStr, "API Key is: secret123") {
		t.Errorf("Expected output to contain interpolated secret, got: %s", outputStr)
	}

	// Verify secrets are deleted (shouldn't exist)
	exists, _ := secretsMgr.Exists("test-project", "api_key")
	if exists {
		t.Error("Secret api_key should be deleted but still exists")
	}
}

// TestIntegration_SecretInterpolation tests secret() function in various contexts
func TestIntegration_SecretInterpolation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drun-secrets-interp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	input := `
version: 2.0

project "interp-test" version "1.0":

task "setup":
  secret set "key1" to "value1"
  secret set "key2" to "value2"

task "test_interpolation":
  info "{secret('key1')}"
  info "Combined: {secret('key1')} and {secret('key2')}"
  info "With default: {secret('missing', 'default_value')}"
  run "echo Secret value: {secret('key1')}"
`

	program, err := engine.ParseStringWithFilename(input, "test.drun")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	secretsMgr, err := secrets.NewManager(
		secrets.WithFallback(),
		secrets.WithStoragePath(filepath.Join(tmpDir, "secrets.enc")),
	)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}

	var output strings.Builder
	eng := engine.NewEngineWithOptions(
		engine.WithOutput(&output),
		engine.WithSecretsManager(secretsMgr),
	)

	// Run setup
	err = eng.Execute(program, "setup")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Clear output
	output.Reset()

	// Run interpolation tests
	err = eng.Execute(program, "test_interpolation")
	if err != nil {
		t.Fatalf("Test execution failed: %v", err)
	}

	outputStr := output.String()

	// Verify various interpolation patterns
	tests := []struct {
		name     string
		contains string
	}{
		{"simple interpolation", "value1"},
		{"combined interpolation", "Combined: value1 and value2"},
		{"default value", "With default: default_value"},
		{"shell interpolation", "Secret value: value1"},
	}

	for _, tc := range tests {
		if !strings.Contains(outputStr, tc.contains) {
			t.Errorf("%s: expected output to contain %q, got: %s", tc.name, tc.contains, outputStr)
		}
	}
}

// TestIntegration_NamespaceIsolation verifies namespace isolation works end-to-end
func TestIntegration_NamespaceIsolation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drun-secrets-ns-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Project 1
	input1 := `
version: 2.0

project "project-one" version "1.0":

task "setup":
  secret set "shared_key" to "project1_value"
`

	// Project 2
	input2 := `
version: 2.0

project "project-two" version "1.0":

task "setup":
  secret set "shared_key" to "project2_value"

task "test":
  info "{secret('shared_key')}"
`

	// Shared secrets manager
	secretsMgr, err := secrets.NewManager(
		secrets.WithFallback(),
		secrets.WithStoragePath(filepath.Join(tmpDir, "secrets.enc")),
	)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}

	// Execute project 1
	program1, _ := engine.ParseStringWithFilename(input1, "test1.drun")
	eng1 := engine.NewEngineWithOptions(
		engine.WithOutput(os.Stdout),
		engine.WithSecretsManager(secretsMgr),
	)
	eng1.Execute(program1, "setup")

	// Execute project 2
	program2, _ := engine.ParseStringWithFilename(input2, "test2.drun")
	var output2 strings.Builder
	eng2 := engine.NewEngineWithOptions(
		engine.WithOutput(&output2),
		engine.WithSecretsManager(secretsMgr),
	)
	eng2.Execute(program2, "setup")
	output2.Reset()
	eng2.Execute(program2, "test")

	// Verify project 2 sees its own value, not project 1's
	outputStr := output2.String()
	if !strings.Contains(outputStr, "project2_value") {
		t.Errorf("Expected project-two to see 'project2_value', got: %s", outputStr)
	}
	if strings.Contains(outputStr, "project1_value") {
		t.Errorf("Project-two should not see project-one's secret value")
	}
}

// TestIntegration_SecretExists tests the secret exists statement
func TestIntegration_SecretExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drun-secrets-exists-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	input := `
version: 2.0

project "exists-test" version "1.0":

task "test":
  secret set "test_key" to "test_value"
  secret exists "test_key"
  secret delete "test_key"
`

	program, err := engine.ParseStringWithFilename(input, "test.drun")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	secretsMgr, err := secrets.NewManager(
		secrets.WithFallback(),
		secrets.WithStoragePath(filepath.Join(tmpDir, "secrets.enc")),
	)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}

	var output strings.Builder
	eng := engine.NewEngineWithOptions(
		engine.WithOutput(&output),
		engine.WithSecretsManager(secretsMgr),
	)

	err = eng.Execute(program, "test")
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Secret test_key exists") {
		t.Errorf("Expected 'Secret test_key exists' in output, got: %s", outputStr)
	}
}

// TestIntegration_SecretList tests the secret list statement
func TestIntegration_SecretList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drun-secrets-list-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	input := `
version: 2.0

project "list-test" version "1.0":

task "test":
  secret set "key1" to "value1"
  secret set "key2" to "value2"
  secret set "key3" to "value3"
  secret list
  secret delete "key1"
  secret delete "key2"
  secret delete "key3"
`

	program, err := engine.ParseStringWithFilename(input, "test.drun")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	secretsMgr, err := secrets.NewManager(
		secrets.WithFallback(),
		secrets.WithStoragePath(filepath.Join(tmpDir, "secrets.enc")),
	)
	if err != nil {
		t.Fatalf("Failed to create secrets manager: %v", err)
	}

	var output strings.Builder
	eng := engine.NewEngineWithOptions(
		engine.WithOutput(&output),
		engine.WithSecretsManager(secretsMgr),
	)

	err = eng.Execute(program, "test")
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	outputStr := output.String()

	// Verify all three keys are listed
	for _, key := range []string{"key1", "key2", "key3"} {
		if !strings.Contains(outputStr, key) {
			t.Errorf("Expected to find %s in secret list, got: %s", key, outputStr)
		}
	}
}
