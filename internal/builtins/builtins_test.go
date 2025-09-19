package builtins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	builtins := []string{
		"hostname", "pwd", "env", "file exists", "now.format",
		"start progress", "update progress", "finish progress",
		"start timer", "stop timer", "show elapsed time",
	}

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

func TestProgressFunctions(t *testing.T) {
	// Clear any existing progress states
	stateMutex.Lock()
	progressStates = make(map[string]*ProgressState)
	stateMutex.Unlock()

	// Test start progress
	result, err := startProgress("Starting task")
	if err != nil {
		t.Fatalf("startProgress() failed: %v", err)
	}

	expected := "📋 Starting task"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test start progress with custom name
	result, err = startProgress("Custom task", "custom")
	if err != nil {
		t.Fatalf("startProgress() with custom name failed: %v", err)
	}

	expected = "📋 Custom task"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test update progress
	result, err = updateProgress("50", "Halfway done")
	if err != nil {
		t.Fatalf("updateProgress() failed: %v", err)
	}

	if !strings.Contains(result, "📋 Halfway done") {
		t.Errorf("Expected result to contain progress message, got %q", result)
	}
	if !strings.Contains(result, "(50%)") {
		t.Errorf("Expected result to contain percentage, got %q", result)
	}
	if !strings.Contains(result, "[") {
		t.Errorf("Expected result to contain progress bar, got %q", result)
	}

	// Test update progress with custom name
	result, err = updateProgress("75", "Almost done", "custom")
	if err != nil {
		t.Fatalf("updateProgress() with custom name failed: %v", err)
	}

	if !strings.Contains(result, "📋 Almost done") {
		t.Errorf("Expected result to contain progress message, got %q", result)
	}
	if !strings.Contains(result, "(75%)") {
		t.Errorf("Expected result to contain percentage, got %q", result)
	}

	// Test finish progress
	result, err = finishProgress("Task completed")
	if err != nil {
		t.Fatalf("finishProgress() failed: %v", err)
	}

	if !strings.Contains(result, "✅ Task completed") {
		t.Errorf("Expected result to contain completion message, got %q", result)
	}
	if !strings.Contains(result, "(completed in") {
		t.Errorf("Expected result to contain elapsed time, got %q", result)
	}

	// Test finish progress with custom name
	result, err = finishProgress("Custom task completed", "custom")
	if err != nil {
		t.Fatalf("finishProgress() with custom name failed: %v", err)
	}

	if !strings.Contains(result, "✅ Custom task completed") {
		t.Errorf("Expected result to contain completion message, got %q", result)
	}

	// Test error cases
	_, err = startProgress()
	if err == nil {
		t.Error("Expected error when no message provided to startProgress")
	}

	_, err = updateProgress("50")
	if err == nil {
		t.Error("Expected error when no message provided to updateProgress")
	}

	_, err = updateProgress("invalid", "message")
	if err == nil {
		t.Error("Expected error when invalid percentage provided to updateProgress")
	}

	_, err = updateProgress("150", "message")
	if err == nil {
		t.Error("Expected error when percentage > 100 provided to updateProgress")
	}

	_, err = finishProgress()
	if err == nil {
		t.Error("Expected error when no message provided to finishProgress")
	}

	_, err = updateProgress("50", "message", "nonexistent")
	if err == nil {
		t.Error("Expected error when updating non-existent progress")
	}

	_, err = finishProgress("message", "nonexistent")
	if err == nil {
		t.Error("Expected error when finishing non-existent progress")
	}
}

func TestTimerFunctions(t *testing.T) {
	// Clear any existing timer states
	stateMutex.Lock()
	timerStates = make(map[string]*TimerState)
	stateMutex.Unlock()

	// Test start timer
	result, err := startTimer("test_timer")
	if err != nil {
		t.Fatalf("startTimer() failed: %v", err)
	}

	expected := "⏱️  Started timer 'test_timer'"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}

	// Test show elapsed time for running timer
	result, err = showElapsedTime("test_timer")
	if err != nil {
		t.Fatalf("showElapsedTime() for running timer failed: %v", err)
	}

	if !strings.Contains(result, "⏱️  Timer 'test_timer' (running):") {
		t.Errorf("Expected result to contain running timer info, got %q", result)
	}

	// Wait a bit to ensure measurable elapsed time
	time.Sleep(10 * time.Millisecond)

	// Test stop timer
	result, err = stopTimer("test_timer")
	if err != nil {
		t.Fatalf("stopTimer() failed: %v", err)
	}

	if !strings.Contains(result, "⏹️  Stopped timer 'test_timer'") {
		t.Errorf("Expected result to contain stopped timer info, got %q", result)
	}
	if !strings.Contains(result, "(elapsed:") {
		t.Errorf("Expected result to contain elapsed time, got %q", result)
	}

	// Test show elapsed time for stopped timer
	result, err = showElapsedTime("test_timer")
	if err != nil {
		t.Fatalf("showElapsedTime() for stopped timer failed: %v", err)
	}

	if !strings.Contains(result, "⏱️  Timer 'test_timer' (stopped):") {
		t.Errorf("Expected result to contain stopped timer info, got %q", result)
	}

	// Test error cases
	_, err = startTimer()
	if err == nil {
		t.Error("Expected error when no timer name provided to startTimer")
	}

	// Try to start the same timer again while it's stopped (should work)
	_, err = startTimer("test_timer")
	if err != nil {
		t.Errorf("Expected to be able to restart stopped timer, got error: %v", err)
	}

	// Now try to start it again while it's running (should fail)
	_, err = startTimer("test_timer")
	if err == nil {
		t.Error("Expected error when starting already running timer")
	}

	_, err = stopTimer()
	if err == nil {
		t.Error("Expected error when no timer name provided to stopTimer")
	}

	_, err = stopTimer("nonexistent")
	if err == nil {
		t.Error("Expected error when stopping non-existent timer")
	}

	// Stop the timer again (should work since we restarted it)
	_, err = stopTimer("test_timer")
	if err != nil {
		t.Errorf("Expected to be able to stop running timer, got error: %v", err)
	}

	// Now try to stop it again while it's already stopped (should fail)
	_, err = stopTimer("test_timer")
	if err == nil {
		t.Error("Expected error when stopping already stopped timer")
	}

	_, err = showElapsedTime()
	if err == nil {
		t.Error("Expected error when no timer name provided to showElapsedTime")
	}

	_, err = showElapsedTime("nonexistent")
	if err == nil {
		t.Error("Expected error when showing elapsed time for non-existent timer")
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		percentage int
		expected   string
	}{
		{0, "[░░░░░░░░░░░░░░░░░░░░]"},
		{25, "[█████░░░░░░░░░░░░░░░]"},
		{50, "[██████████░░░░░░░░░░]"},
		{75, "[███████████████░░░░░]"},
		{100, "[████████████████████]"},
	}

	for _, test := range tests {
		result := createProgressBar(test.percentage)
		if result != test.expected {
			t.Errorf("For percentage %d, expected %q, got %q", test.percentage, test.expected, result)
		}
	}
}
