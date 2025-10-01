package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDirEmpty(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	
	// Test 1: Empty directory
	emptyDir := filepath.Join(tempDir, "empty")
	err := os.MkdirAll(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}
	
	engine := NewEngine(os.Stdout)
	isEmpty, err := engine.isDirEmpty(emptyDir)
	if err != nil {
		t.Fatalf("Error checking if directory is empty: %v", err)
	}
	if !isEmpty {
		t.Errorf("Expected empty directory to be detected as empty, but it was not")
	}
	
	// Test 2: Non-empty directory with a file
	nonEmptyDir := filepath.Join(tempDir, "nonempty")
	err = os.MkdirAll(nonEmptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create non-empty directory: %v", err)
	}
	
	testFile := filepath.Join(nonEmptyDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Verify the file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Test file was not created: %v", err)
	}
	
	// Check if the directory has entries
	entries, err := os.ReadDir(nonEmptyDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}
	t.Logf("Directory %s has %d entries", nonEmptyDir, len(entries))
	for _, entry := range entries {
		t.Logf("  - %s (IsDir: %v)", entry.Name(), entry.IsDir())
	}
	
	isEmpty, err = engine.isDirEmpty(nonEmptyDir)
	if err != nil {
		t.Fatalf("Error checking if directory is empty: %v", err)
	}
	if isEmpty {
		t.Errorf("Expected non-empty directory to be detected as non-empty, but it was detected as empty")
	}
	
	// Test 3: Directory with hidden file (should still be considered empty)
	hiddenDir := filepath.Join(tempDir, "hidden")
	err = os.MkdirAll(hiddenDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create hidden directory: %v", err)
	}
	
	hiddenFile := filepath.Join(hiddenDir, ".hidden")
	err = os.WriteFile(hiddenFile, []byte("hidden"), 0644)
	if err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}
	
	isEmpty, err = engine.isDirEmpty(hiddenDir)
	if err != nil {
		t.Fatalf("Error checking if directory with hidden file is empty: %v", err)
	}
	if !isEmpty {
		t.Errorf("Expected directory with only hidden files to be detected as empty, but it was not")
	}
}
