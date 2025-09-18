package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngine_FileOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "drun_file_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	input := `version: 2.0

task "file operations":
  create file "` + filepath.Join(tmpDir, "test.txt") + `"
  write "Hello World" to file "` + filepath.Join(tmpDir, "test.txt") + `"
  read file "` + filepath.Join(tmpDir, "test.txt") + `" as content
  info "File content: {content}"
  copy "` + filepath.Join(tmpDir, "test.txt") + `" to "` + filepath.Join(tmpDir, "copy.txt") + `"
  append "\nNew line" to file "` + filepath.Join(tmpDir, "test.txt") + `"
  create dir "` + filepath.Join(tmpDir, "subdir") + `"
  success "File operations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "file operations")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that file operations were executed
	expectedParts := []string{
		"📄 Creating file:",
		"✏️  Writing to file:",
		"📖 Reading file:",
		"ℹ️  File content: Hello World",
		"📋 Copying:",
		"➕ Appending to file:",
		"📁 Creating directory:",
		"✅ File operations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Verify files were actually created
	testFile := filepath.Join(tmpDir, "test.txt")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("Test file was not created")
	}

	copyFile := filepath.Join(tmpDir, "copy.txt")
	if _, err := os.Stat(copyFile); os.IsNotExist(err) {
		t.Errorf("Copy file was not created")
	}

	subdir := filepath.Join(tmpDir, "subdir")
	if _, err := os.Stat(subdir); os.IsNotExist(err) {
		t.Errorf("Subdirectory was not created")
	}

	// Verify file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	expectedContent := "Hello World\\nNew line"
	if string(content) != expectedContent {
		t.Errorf("Expected file content %q, got %q", expectedContent, string(content))
	}
}

func TestEngine_FileOperationsDryRun(t *testing.T) {
	input := `version: 2.0

task "file dry run":
  create file "/tmp/dryrun_test.txt"
  write "Test content" to file "/tmp/dryrun_test.txt"
  read file "/tmp/dryrun_test.txt" as content
  delete file "/tmp/dryrun_test.txt"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.Execute(program, "file dry run")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that dry run messages are shown
	expectedParts := []string{
		"[DRY RUN] Would create file",
		"[DRY RUN] Would write",
		"[DRY RUN] Would read",
		"[DRY RUN] Would capture file content in variable 'content'",
		"[DRY RUN] Would delete file",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected dry run output to contain %q, got %q", part, outputStr)
		}
	}

	// Verify no actual files were created
	if _, err := os.Stat("/tmp/dryrun_test.txt"); !os.IsNotExist(err) {
		t.Errorf("File should not exist in dry run mode")
	}
}

func TestEngine_FileOperationsWithParameters(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "drun_file_param_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	input := `version: 2.0

task "file with params":
  requires filename as string
  given content as string defaults to "Default content"
  
  create file "{filename}"
  write "{content}" to file "{filename}"
  read file "{filename}" as file_content
  info "Read from {filename}: {file_content}"
  success "File operations with parameters completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"filename": filepath.Join(tmpDir, "param_test.txt"),
		"content":  "Parameterized content",
	}

	err = engine.ExecuteWithParams(program, "file with params", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that parameters were interpolated correctly
	expectedParts := []string{
		"📄 Creating file: " + filepath.Join(tmpDir, "param_test.txt"),
		"✏️  Writing to file: " + filepath.Join(tmpDir, "param_test.txt"),
		"📖 Reading file: " + filepath.Join(tmpDir, "param_test.txt"),
		"ℹ️  Read from " + filepath.Join(tmpDir, "param_test.txt") + ": Parameterized content",
		"✅ File operations with parameters completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Verify file content
	content, err := os.ReadFile(filepath.Join(tmpDir, "param_test.txt"))
	if err != nil {
		t.Fatalf("Failed to read param test file: %v", err)
	}

	if string(content) != "Parameterized content" {
		t.Errorf("Expected file content 'Parameterized content', got %q", string(content))
	}
}

func TestEngine_FileOperationsWithConditionals(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "drun_file_cond_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	input := `version: 2.0

task "conditional files":
  requires create_backup as boolean
  
  create file "` + filepath.Join(tmpDir, "main.txt") + `"
  write "Main content" to file "` + filepath.Join(tmpDir, "main.txt") + `"
  
  when create_backup is "true":
    copy "` + filepath.Join(tmpDir, "main.txt") + `" to "` + filepath.Join(tmpDir, "backup.txt") + `"
    info "Backup created"
  
  when create_backup is "false":
    info "No backup created"
  
  success "Conditional file operations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	// Test with backup enabled
	var output bytes.Buffer
	engine := NewEngine(&output)

	params := map[string]string{
		"create_backup": "true",
	}

	err = engine.ExecuteWithParams(program, "conditional files", params)
	if err != nil {
		t.Fatalf("ExecuteWithParams failed: %v", err)
	}

	outputStr := output.String()

	// Check that backup was created
	expectedParts := []string{
		"📄 Creating file:",
		"✏️  Writing to file:",
		"📋 Copying:",
		"ℹ️  Backup created",
		"✅ Conditional file operations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Should not contain the false branch
	if strings.Contains(outputStr, "No backup created") {
		t.Errorf("Should not contain false branch output")
	}

	// Verify backup file was created
	backupFile := filepath.Join(tmpDir, "backup.txt")
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Errorf("Backup file was not created")
	}
}

func TestEngine_DeleteOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "drun_delete_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files and directories
	testFile := filepath.Join(tmpDir, "to_delete.txt")
	testDir := filepath.Join(tmpDir, "to_delete_dir")

	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	input := `version: 2.0

task "delete operations":
  delete file "` + testFile + `"
  delete dir "` + testDir + `"
  success "Delete operations completed!"`

	program, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)

	err = engine.Execute(program, "delete operations")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()

	// Check that delete operations were executed
	expectedParts := []string{
		"🗑️  Deleting file:",
		"🗑️  Deleting directory:",
		"✅ Delete operations completed!",
	}

	for _, part := range expectedParts {
		if !strings.Contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	// Verify files were actually deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("Test file was not deleted")
	}

	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Errorf("Test directory was not deleted")
	}
}
