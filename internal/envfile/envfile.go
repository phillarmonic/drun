package envfile

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/orchestration"
)

// Manager manages environment file operations
type Manager struct {
	workDir string
}

// NewManager creates a new environment file manager
func NewManager(workDir string) *Manager {
	return &Manager{
		workDir: workDir,
	}
}

// Ensure ensures an environment file exists for a service
func (m *Manager) Ensure(ctx context.Context, config *orchestration.EnvFileConfig, servicePath string) error {
	envFilePath := filepath.Join(m.workDir, servicePath, ".env")

	// Check if .env file exists
	if _, err := os.Stat(envFilePath); err == nil {
		// File exists
		if config.Required {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check env file: %w", err)
	}

	// File doesn't exist
	if config.Required {
		// Look for .env.example or .env.template
		examplePath := filepath.Join(m.workDir, servicePath, ".env.example")
		templatePath := filepath.Join(m.workDir, servicePath, ".env.template")

		var sourcePath string
		if _, err := os.Stat(examplePath); err == nil {
			sourcePath = examplePath
		} else if _, err := os.Stat(templatePath); err == nil {
			sourcePath = templatePath
		} else {
			return fmt.Errorf("env file required but not found and no example/template available at %s", servicePath)
		}

		// Copy example/template to .env
		if err := m.CopyFile(sourcePath, envFilePath); err != nil {
			return fmt.Errorf("failed to copy env file: %w", err)
		}
	}

	return nil
}

// Read reads an environment file
func (m *Manager) Read(filePath string) (map[string]string, error) {
	fullPath := filepath.Join(m.workDir, filePath)

	// #nosec G304 -- env files are intentionally resolved relative to the configured workDir.
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	return env, nil
}

// Write writes an environment file
func (m *Manager) Write(filePath string, env map[string]string) error {
	fullPath := filepath.Join(m.workDir, filePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// #nosec G304 -- env files are intentionally created relative to the configured workDir.
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create env file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	for key, value := range env {
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, value); err != nil {
			return fmt.Errorf("failed to write env file: %w", err)
		}
	}

	return nil
}

// Replace replaces values in an environment file
func (m *Manager) Replace(filePath string, replacements map[string]string) error {
	_ = filepath.Join(m.workDir, filePath) // fullPath not needed, Read/Write handle paths

	// Read current env file
	env, err := m.Read(filePath)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	// Apply replacements
	for key, value := range replacements {
		env[key] = value
	}

	// Write updated env file
	if err := m.Write(filePath, env); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	return nil
}

// Validate validates that required variables exist in an environment file
func (m *Manager) Validate(filePath string, requiredVars []string) error {
	env, err := m.Read(filePath)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	missing := []string{}
	for _, varName := range requiredVars {
		if _, exists := env[varName]; !exists {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// Contains checks if an environment file contains a specific variable
func (m *Manager) Contains(filePath, varName string) (bool, error) {
	env, err := m.Read(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read env file: %w", err)
	}

	_, exists := env[varName]
	return exists, nil
}

// CopyFile copies a file from source to destination
func (m *Manager) CopyFile(source, destination string) error {
	// #nosec G304 -- source is selected by the caller for explicit env-file management.
	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		_ = srcFile.Close()
	}()

	// #nosec G304 -- destination is selected by the caller for explicit env-file management.
	dstFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		_ = dstFile.Close()
	}()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// Backup creates a backup of an environment file
func (m *Manager) Backup(filePath string) (string, error) {
	fullPath := filepath.Join(m.workDir, filePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("env file not found: %s", filePath)
	}

	// Create backup path with timestamp
	backupPath := fullPath + ".backup"

	if err := m.CopyFile(fullPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// Restore restores an environment file from a backup
func (m *Manager) Restore(filePath, backupPath string) error {
	fullPath := filepath.Join(m.workDir, filePath)

	if err := m.CopyFile(backupPath, fullPath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}

// SetPermissions sets secure permissions on an environment file
func (m *Manager) SetPermissions(filePath string, mode os.FileMode) error {
	fullPath := filepath.Join(m.workDir, filePath)

	if err := os.Chmod(fullPath, mode); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// Merge merges multiple environment files
func (m *Manager) Merge(outputPath string, inputPaths ...string) error {
	merged := make(map[string]string)

	// Read all input files
	for _, inputPath := range inputPaths {
		env, err := m.Read(inputPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", inputPath, err)
		}

		// Merge into combined map (later files override earlier ones)
		for key, value := range env {
			merged[key] = value
		}
	}

	// Write merged file
	if err := m.Write(outputPath, merged); err != nil {
		return fmt.Errorf("failed to write merged file: %w", err)
	}

	return nil
}

// Interpolate interpolates variables in an environment file
func (m *Manager) Interpolate(filePath string, variables map[string]string) error {
	fullPath := filepath.Join(m.workDir, filePath)

	// Read file content
	// #nosec G304 -- env files are intentionally resolved relative to the configured workDir.
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	// Interpolate variables
	result := string(content)
	for key, value := range variables {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)

		// Also support $VAR syntax
		placeholder = fmt.Sprintf("$%s", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Write interpolated content
	cleanPath := filepath.Clean(fullPath)
	// #nosec G703 -- env files are intentionally written under the configured workDir.
	if err := os.WriteFile(cleanPath, []byte(result), 0600); err != nil {
		return fmt.Errorf("failed to write interpolated file: %w", err)
	}

	return nil
}
