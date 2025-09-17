package builtins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatCurrentTime(t *testing.T) {
	// Test default format
	result, err := formatCurrentTime()
	if err != nil {
		t.Fatalf("formatCurrentTime() failed: %v", err)
	}

	// Should be in format "2006-01-02 15:04:05"
	if len(result) != 19 {
		t.Errorf("Expected default format length 19, got %d: %s", len(result), result)
	}

	// Test custom format
	result, err = formatCurrentTime("2006-01-02")
	if err != nil {
		t.Fatalf("formatCurrentTime() with custom format failed: %v", err)
	}

	// Should be in format "2006-01-02"
	if len(result) != 10 {
		t.Errorf("Expected custom format length 10, got %d: %s", len(result), result)
	}
}

func TestCheckFileExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_file_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_ = tmpFile.Close()

	// Test existing file
	result, err := checkFileExists(tmpFile.Name())
	if err != nil {
		t.Fatalf("checkFileExists() failed: %v", err)
	}

	if result != "true" {
		t.Errorf("Expected 'true' for existing file, got %s", result)
	}

	// Test non-existing file
	result, err = checkFileExists("/non/existent/file.txt")
	if err != nil {
		t.Fatalf("checkFileExists() failed: %v", err)
	}

	if result != "false" {
		t.Errorf("Expected 'false' for non-existing file, got %s", result)
	}

	// Test no arguments
	_, err = checkFileExists()
	if err == nil {
		t.Error("Expected error when no arguments provided")
	}
}

func TestCheckDirExists(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test_dir_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test existing directory
	result, err := checkDirExists(tmpDir)
	if err != nil {
		t.Fatalf("checkDirExists() failed: %v", err)
	}

	if result != "true" {
		t.Errorf("Expected 'true' for existing directory, got %s", result)
	}

	// Test non-existing directory
	result, err = checkDirExists("/non/existent/directory")
	if err != nil {
		t.Fatalf("checkDirExists() failed: %v", err)
	}

	if result != "false" {
		t.Errorf("Expected 'false' for non-existing directory, got %s", result)
	}
}

func TestGetEnvironmentVariable(t *testing.T) {
	// Set a test environment variable
	testKey := "DRUN_TEST_VAR"
	testValue := "test_value_123"
	_ = os.Setenv(testKey, testValue)
	defer func() { _ = os.Unsetenv(testKey) }()

	// Test existing env var
	result, err := getEnvironmentVariable(testKey)
	if err != nil {
		t.Fatalf("getEnvironmentVariable() failed: %v", err)
	}

	if result != testValue {
		t.Errorf("Expected %s, got %s", testValue, result)
	}

	// Test non-existing env var with default
	result, err = getEnvironmentVariable("NON_EXISTENT_VAR", "default_value")
	if err != nil {
		t.Fatalf("getEnvironmentVariable() with default failed: %v", err)
	}

	if result != "default_value" {
		t.Errorf("Expected 'default_value', got %s", result)
	}

	// Test no arguments
	_, err = getEnvironmentVariable()
	if err == nil {
		t.Error("Expected error when no arguments provided")
	}
}

func TestGetCurrentDirectory(t *testing.T) {
	// Test default (full path)
	result, err := getCurrentDirectory()
	if err != nil {
		t.Fatalf("getCurrentDirectory() failed: %v", err)
	}

	// Should be an absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("Expected absolute path, got %s", result)
	}

	// Test basename
	result, err = getCurrentDirectory("basename")
	if err != nil {
		t.Fatalf("getCurrentDirectory('basename') failed: %v", err)
	}

	// Should not contain path separators
	if strings.Contains(result, string(filepath.Separator)) {
		t.Errorf("Expected basename without separators, got %s", result)
	}
}

func TestGetHostname(t *testing.T) {
	result, err := getHostname()
	if err != nil {
		t.Fatalf("getHostname() failed: %v", err)
	}

	// Should not be empty
	if result == "" {
		t.Error("Expected non-empty hostname")
	}
}

func TestCallBuiltin(t *testing.T) {
	// Test existing builtin
	result, err := CallBuiltin("hostname")
	if err != nil {
		t.Fatalf("CallBuiltin('hostname') failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result from hostname builtin")
	}

	// Test non-existing builtin
	_, err = CallBuiltin("non_existent_function")
	if err == nil {
		t.Error("Expected error for non-existent builtin")
	}
}

func TestIsBuiltin(t *testing.T) {
	// Test existing builtins
	builtins := []string{"hostname", "pwd", "env", "file exists", "now.format"}

	for _, builtin := range builtins {
		if !IsBuiltin(builtin) {
			t.Errorf("Expected %s to be recognized as builtin", builtin)
		}
	}

	// Test non-existing builtin
	if IsBuiltin("non_existent_function") {
		t.Error("Expected non_existent_function to not be recognized as builtin")
	}
}

func TestGetCurrentGitCommit(t *testing.T) {
	// This test might fail in environments without git or outside a git repo
	// So we'll make it more lenient
	result, err := getCurrentGitCommit()

	// If we're in a git repo, should get a commit hash
	if err == nil {
		if len(result) != 40 { // Full SHA-1 hash length
			t.Errorf("Expected 40-character commit hash, got %d characters: %s", len(result), result)
		}
	}

	// Test short version
	result, err = getCurrentGitCommit("short")
	if err == nil {
		if len(result) != 7 { // Short hash length
			t.Errorf("Expected 7-character short commit hash, got %d characters: %s", len(result), result)
		}
	}

	// Note: We don't fail the test if git is not available or we're not in a git repo
	// This makes the test more robust in different environments
}
