package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Domain: Filesystem Helpers
// This file contains helper methods for filesystem operations

func (e *Engine) resolveFilesystemPath(path string, ctx *ExecutionContext) string {
	if path == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	base := ""
	if ctx != nil {
		if ctx.WorkingDir != "" {
			base = ctx.WorkingDir
		} else if ctx.OriginalWorkingDir != "" {
			base = ctx.OriginalWorkingDir
		}
	}
	if base == "" {
		if cwd, err := os.Getwd(); err == nil {
			base = cwd
		}
	}
	if base == "" {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}

// fileExists checks if a file exists
func (e *Engine) fileExists(path string, ctx *ExecutionContext) bool {
	info, err := os.Stat(e.resolveFilesystemPath(path, ctx))
	return err == nil && !info.IsDir()
}

// getFileSize returns the size of a file in bytes
func (e *Engine) getFileSize(path string, ctx *ExecutionContext) (int64, error) {
	info, err := os.Stat(e.resolveFilesystemPath(path, ctx))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// dirExists checks if a directory exists
func (e *Engine) dirExists(path string, ctx *ExecutionContext) bool {
	info, err := os.Stat(e.resolveFilesystemPath(path, ctx))
	return err == nil && info.IsDir()
}

// isDirEmpty checks if a directory is empty
func (e *Engine) isDirEmpty(path string, ctx *ExecutionContext) (bool, error) {
	entries, err := os.ReadDir(e.resolveFilesystemPath(path, ctx))
	if err != nil {
		return false, err
	}

	// Count only visible entries (filter out hidden files on Windows)
	visibleCount := 0
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files (those starting with . on Unix, or system files on Windows)
		if strings.HasPrefix(name, ".") {
			continue
		}
		// Skip common Windows system files
		if runtime.GOOS == "windows" {
			lowerName := strings.ToLower(name)
			if lowerName == "desktop.ini" || lowerName == "thumbs.db" || lowerName == "$recycle.bin" {
				continue
			}
		}
		visibleCount++
	}

	return visibleCount == 0, nil
}
