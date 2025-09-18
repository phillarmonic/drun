package fileops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileOperation represents a file system operation
type FileOperation struct {
	Type    string // "create", "copy", "move", "delete", "read", "write", "append"
	Target  string // target file/directory path
	Source  string // source path (for copy/move operations)
	Content string // content (for write/append operations)
	IsDir   bool   // whether the operation is on a directory
}

// Execute performs the file operation
func (op *FileOperation) Execute(dryRun bool) (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
		Source:    op.Source,
	}

	if dryRun {
		result.DryRun = true
		result.Message = fmt.Sprintf("[DRY RUN] Would %s %s", op.Type, op.getDescription())
		return result, nil
	}

	switch op.Type {
	case "create":
		return op.executeCreate()
	case "copy":
		return op.executeCopy()
	case "move":
		return op.executeMove()
	case "delete":
		return op.executeDelete()
	case "read":
		return op.executeRead()
	case "write":
		return op.executeWrite()
	case "append":
		return op.executeAppend()
	default:
		return nil, fmt.Errorf("unknown file operation: %s", op.Type)
	}
}

// Result represents the result of a file operation
type Result struct {
	Operation string // operation type
	Target    string // target path
	Source    string // source path (if applicable)
	Content   string // content (for read operations)
	Message   string // operation message
	Success   bool   // whether operation succeeded
	DryRun    bool   // whether this was a dry run
}

// getDescription returns a human-readable description of the operation
func (op *FileOperation) getDescription() string {
	switch op.Type {
	case "create":
		if op.IsDir {
			return fmt.Sprintf("directory '%s'", op.Target)
		}
		return fmt.Sprintf("file '%s'", op.Target)
	case "copy":
		return fmt.Sprintf("'%s' to '%s'", op.Source, op.Target)
	case "move":
		return fmt.Sprintf("'%s' to '%s'", op.Source, op.Target)
	case "delete":
		if op.IsDir {
			return fmt.Sprintf("directory '%s'", op.Target)
		}
		return fmt.Sprintf("file '%s'", op.Target)
	case "read":
		return fmt.Sprintf("file '%s'", op.Target)
	case "write":
		return fmt.Sprintf("content to file '%s'", op.Target)
	case "append":
		return fmt.Sprintf("content to file '%s'", op.Target)
	default:
		return op.Target
	}
}

// executeCreate creates a file or directory
func (op *FileOperation) executeCreate() (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
	}

	if op.IsDir {
		// Create directory
		err := os.MkdirAll(op.Target, 0755)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to create directory '%s': %v", op.Target, err)
			return result, err
		}
		result.Success = true
		result.Message = fmt.Sprintf("Created directory '%s'", op.Target)
	} else {
		// Create file (and parent directories if needed)
		dir := filepath.Dir(op.Target)
		if dir != "." {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				result.Message = fmt.Sprintf("Failed to create parent directory for '%s': %v", op.Target, err)
				return result, err
			}
		}

		file, err := os.Create(op.Target)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to create file '%s': %v", op.Target, err)
			return result, err
		}
		defer func() { _ = file.Close() }()

		// Write initial content if provided
		if op.Content != "" {
			_, err = file.WriteString(op.Content)
			if err != nil {
				result.Message = fmt.Sprintf("Failed to write content to file '%s': %v", op.Target, err)
				return result, err
			}
		}

		result.Success = true
		result.Message = fmt.Sprintf("Created file '%s'", op.Target)
	}

	return result, nil
}

// executeCopy copies a file or directory
func (op *FileOperation) executeCopy() (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
		Source:    op.Source,
	}

	// Check if source exists
	srcInfo, err := os.Stat(op.Source)
	if err != nil {
		result.Message = fmt.Sprintf("Source '%s' does not exist: %v", op.Source, err)
		return result, err
	}

	if srcInfo.IsDir() {
		// Copy directory recursively
		err = copyDir(op.Source, op.Target)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to copy directory '%s' to '%s': %v", op.Source, op.Target, err)
			return result, err
		}
		result.Success = true
		result.Message = fmt.Sprintf("Copied directory '%s' to '%s'", op.Source, op.Target)
	} else {
		// Copy file
		err = copyFile(op.Source, op.Target)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to copy file '%s' to '%s': %v", op.Source, op.Target, err)
			return result, err
		}
		result.Success = true
		result.Message = fmt.Sprintf("Copied file '%s' to '%s'", op.Source, op.Target)
	}

	return result, nil
}

// executeMove moves/renames a file or directory
func (op *FileOperation) executeMove() (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
		Source:    op.Source,
	}

	// Check if source exists
	_, err := os.Stat(op.Source)
	if err != nil {
		result.Message = fmt.Sprintf("Source '%s' does not exist: %v", op.Source, err)
		return result, err
	}

	// Create parent directory for target if needed
	dir := filepath.Dir(op.Target)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to create parent directory for '%s': %v", op.Target, err)
			return result, err
		}
	}

	err = os.Rename(op.Source, op.Target)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to move '%s' to '%s': %v", op.Source, op.Target, err)
		return result, err
	}

	result.Success = true
	result.Message = fmt.Sprintf("Moved '%s' to '%s'", op.Source, op.Target)
	return result, nil
}

// executeDelete deletes a file or directory
func (op *FileOperation) executeDelete() (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
	}

	// Check if target exists
	info, err := os.Stat(op.Target)
	if err != nil {
		if os.IsNotExist(err) {
			result.Success = true
			result.Message = fmt.Sprintf("Target '%s' does not exist (already deleted)", op.Target)
			return result, nil
		}
		result.Message = fmt.Sprintf("Failed to check target '%s': %v", op.Target, err)
		return result, err
	}

	if info.IsDir() {
		err = os.RemoveAll(op.Target)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to delete directory '%s': %v", op.Target, err)
			return result, err
		}
		result.Success = true
		result.Message = fmt.Sprintf("Deleted directory '%s'", op.Target)
	} else {
		err = os.Remove(op.Target)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to delete file '%s': %v", op.Target, err)
			return result, err
		}
		result.Success = true
		result.Message = fmt.Sprintf("Deleted file '%s'", op.Target)
	}

	return result, nil
}

// executeRead reads the content of a file
func (op *FileOperation) executeRead() (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
	}

	content, err := os.ReadFile(op.Target)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to read file '%s': %v", op.Target, err)
		return result, err
	}

	result.Success = true
	result.Content = string(content)
	result.Message = fmt.Sprintf("Read %d bytes from file '%s'", len(content), op.Target)
	return result, nil
}

// executeWrite writes content to a file (overwrites existing content)
func (op *FileOperation) executeWrite() (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
	}

	// Create parent directory if needed
	dir := filepath.Dir(op.Target)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to create parent directory for '%s': %v", op.Target, err)
			return result, err
		}
	}

	err := os.WriteFile(op.Target, []byte(op.Content), 0644)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to write to file '%s': %v", op.Target, err)
		return result, err
	}

	result.Success = true
	result.Message = fmt.Sprintf("Wrote %d bytes to file '%s'", len(op.Content), op.Target)
	return result, nil
}

// executeAppend appends content to a file
func (op *FileOperation) executeAppend() (*Result, error) {
	result := &Result{
		Operation: op.Type,
		Target:    op.Target,
	}

	// Create parent directory if needed
	dir := filepath.Dir(op.Target)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to create parent directory for '%s': %v", op.Target, err)
			return result, err
		}
	}

	file, err := os.OpenFile(op.Target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to open file '%s' for appending: %v", op.Target, err)
		return result, err
	}
	defer func() { _ = file.Close() }()

	_, err = file.WriteString(op.Content)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to append to file '%s': %v", op.Target, err)
		return result, err
	}

	result.Success = true
	result.Message = fmt.Sprintf("Appended %d bytes to file '%s'", len(op.Content), op.Target)
	return result, nil
}

// Helper functions

// copyFile copies a single file
func copyFile(src, dst string) error {
	// Create parent directory for destination if needed
	dir := filepath.Dir(dst)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy file
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Convenience functions for common operations

// CreateFile creates a new file with optional content
func CreateFile(path, content string) (*Result, error) {
	op := &FileOperation{
		Type:    "create",
		Target:  path,
		Content: content,
		IsDir:   false,
	}
	return op.Execute(false)
}

// CreateDir creates a new directory
func CreateDir(path string) (*Result, error) {
	op := &FileOperation{
		Type:   "create",
		Target: path,
		IsDir:  true,
	}
	return op.Execute(false)
}

// CopyFile copies a file from source to destination
func CopyFile(src, dst string) (*Result, error) {
	op := &FileOperation{
		Type:   "copy",
		Source: src,
		Target: dst,
		IsDir:  false,
	}
	return op.Execute(false)
}

// MoveFile moves/renames a file
func MoveFile(src, dst string) (*Result, error) {
	op := &FileOperation{
		Type:   "move",
		Source: src,
		Target: dst,
		IsDir:  false,
	}
	return op.Execute(false)
}

// DeleteFile deletes a file
func DeleteFile(path string) (*Result, error) {
	op := &FileOperation{
		Type:   "delete",
		Target: path,
		IsDir:  false,
	}
	return op.Execute(false)
}

// ReadFile reads the content of a file
func ReadFile(path string) (*Result, error) {
	op := &FileOperation{
		Type:   "read",
		Target: path,
	}
	return op.Execute(false)
}

// WriteFile writes content to a file
func WriteFile(path, content string) (*Result, error) {
	op := &FileOperation{
		Type:    "write",
		Target:  path,
		Content: content,
	}
	return op.Execute(false)
}

// AppendToFile appends content to a file
func AppendToFile(path, content string) (*Result, error) {
	op := &FileOperation{
		Type:    "append",
		Target:  path,
		Content: content,
	}
	return op.Execute(false)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// PathExists checks if a path (file or directory) exists
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsEmpty checks if a directory is empty
func IsEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileLines returns the number of lines in a file
func GetFileLines(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strings.Count(string(content), "\n") + 1, nil
}
