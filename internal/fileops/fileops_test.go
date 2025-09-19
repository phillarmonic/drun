package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test creating a file with content
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"

	result, err := CreateFile(testFile, content)
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	// Verify file was created
	if !FileExists(testFile) {
		t.Errorf("File was not created")
	}

	// Verify content
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(readContent))
	}
}

func TestCreateDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test creating a directory
	testDir := filepath.Join(tmpDir, "subdir", "nested")

	result, err := CreateDir(testDir)
	if err != nil {
		t.Fatalf("CreateDir failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	// Verify directory was created
	if !DirExists(testDir) {
		t.Errorf("Directory was not created")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	content := "Test content for copying"
	_, err = CreateFile(srcFile, content)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstFile := filepath.Join(tmpDir, "destination.txt")
	result, err := CopyFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	// Verify destination file exists and has correct content
	if !FileExists(dstFile) {
		t.Errorf("Destination file was not created")
	}

	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(dstContent))
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	content := "Test content for moving"
	_, err = CreateFile(srcFile, content)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Move file
	dstFile := filepath.Join(tmpDir, "moved.txt")
	result, err := MoveFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	// Verify source file no longer exists
	if FileExists(srcFile) {
		t.Errorf("Source file still exists after move")
	}

	// Verify destination file exists and has correct content
	if !FileExists(dstFile) {
		t.Errorf("Destination file was not created")
	}

	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(dstContent))
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create file to delete
	testFile := filepath.Join(tmpDir, "to_delete.txt")
	_, err = CreateFile(testFile, "Delete me")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify file exists before deletion
	if !FileExists(testFile) {
		t.Fatalf("Test file was not created")
	}

	// Delete file
	result, err := DeleteFile(testFile)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	// Verify file no longer exists
	if FileExists(testFile) {
		t.Errorf("File still exists after deletion")
	}
}

func TestReadFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create file with content
	testFile := filepath.Join(tmpDir, "read_test.txt")
	content := "Content to read\nLine 2\nLine 3"
	_, err = CreateFile(testFile, content)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read file
	result, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	if result.Content != content {
		t.Errorf("Expected content %q, got %q", content, result.Content)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Write to new file
	testFile := filepath.Join(tmpDir, "write_test.txt")
	content := "Written content"

	result, err := WriteFile(testFile, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	// Verify content was written
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("Expected content %q, got %q", content, string(readContent))
	}
}

func TestAppendToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create initial file
	testFile := filepath.Join(tmpDir, "append_test.txt")
	initialContent := "Initial content\n"
	_, err = CreateFile(testFile, initialContent)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Append to file
	appendContent := "Appended content\n"
	result, err := AppendToFile(testFile, appendContent)
	if err != nil {
		t.Fatalf("AppendToFile failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Message)
	}

	// Verify final content
	finalContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}

	expected := initialContent + appendContent
	if string(finalContent) != expected {
		t.Errorf("Expected content %q, got %q", expected, string(finalContent))
	}
}

func TestFileExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test non-existent file
	nonExistentFile := filepath.Join(tmpDir, "does_not_exist.txt")
	if FileExists(nonExistentFile) {
		t.Errorf("FileExists returned true for non-existent file")
	}

	// Create file and test
	testFile := filepath.Join(tmpDir, "exists.txt")
	_, err = CreateFile(testFile, "content")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !FileExists(testFile) {
		t.Errorf("FileExists returned false for existing file")
	}

	// Test that it returns false for directories
	testDir := filepath.Join(tmpDir, "testdir")
	_, err = CreateDir(testDir)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	if FileExists(testDir) {
		t.Errorf("FileExists returned true for directory")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test non-existent directory
	nonExistentDir := filepath.Join(tmpDir, "does_not_exist")
	if DirExists(nonExistentDir) {
		t.Errorf("DirExists returned true for non-existent directory")
	}

	// Create directory and test
	testDir := filepath.Join(tmpDir, "exists")
	_, err = CreateDir(testDir)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	if !DirExists(testDir) {
		t.Errorf("DirExists returned false for existing directory")
	}

	// Test that it returns false for files
	testFile := filepath.Join(tmpDir, "testfile.txt")
	_, err = CreateFile(testFile, "content")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if DirExists(testFile) {
		t.Errorf("DirExists returned true for file")
	}
}

func TestDryRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fileops_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test dry run create file
	testFile := filepath.Join(tmpDir, "dry_run_test.txt")
	op := &FileOperation{
		Type:    "create",
		Target:  testFile,
		Content: "test content",
		IsDir:   false,
	}

	result, err := op.Execute(true) // dry run
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	if !result.DryRun {
		t.Errorf("Expected DryRun to be true")
	}

	if result.Success {
		t.Errorf("Expected Success to be false for dry run")
	}

	// Verify file was not actually created
	if FileExists(testFile) {
		t.Errorf("File was created during dry run")
	}

	// Verify message contains dry run indicator
	if !strings.Contains(result.Message, "[DRY RUN]") {
		t.Errorf("Expected dry run message to contain '[DRY RUN]', got: %s", result.Message)
	}
}
