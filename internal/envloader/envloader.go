package envloader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnvFileInfo holds information about a loaded .env file
type EnvFileInfo struct {
	Path   string            // Path to the .env file
	Exists bool              // Whether the file exists
	Loaded bool              // Whether the file was successfully loaded
	Vars   map[string]string // Variables loaded from this file
	Error  error             // Any error encountered while loading
}

// LoadResult contains the results of loading .env files
type LoadResult struct {
	Files           []EnvFileInfo     // Information about all .env files processed
	FinalEnv        map[string]string // Final merged environment variables
	Environment     string            // The environment name used
	HostEnvIncluded bool              // Whether host environment variables were included
}

// Loader handles loading .env files hierarchically
type Loader struct {
	workingDir  string
	environment string
	debugMode   bool
	output      io.Writer
}

// NewLoader creates a new .env file loader
func NewLoader(workingDir string, environment string, debugMode bool, output io.Writer) *Loader {
	if output == nil {
		output = os.Stdout
	}
	if workingDir == "" {
		workingDir = "."
	}
	return &Loader{
		workingDir:  workingDir,
		environment: environment,
		debugMode:   debugMode,
		output:      output,
	}
}

// Load loads .env files in hierarchical order and returns the merged environment
// Priority (lowest to highest):
// 1. Host OS environment variables
// 2. .env (base configuration)
// 3. .env.local (local overrides, typically gitignored)
// 4. .env.{environment} (environment-specific, e.g., .env.production)
// 5. .env.{environment}.local (environment-specific local overrides)
func (l *Loader) Load() (*LoadResult, error) {
	result := &LoadResult{
		Files:           make([]EnvFileInfo, 0, 4),
		FinalEnv:        make(map[string]string),
		Environment:     l.environment,
		HostEnvIncluded: true,
	}

	// Start with host OS environment variables
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			result.FinalEnv[parts[0]] = parts[1]
		}
	}

	if l.debugMode {
		_, _ = fmt.Fprintf(l.output, "🔧 [ENV DEBUG] Starting environment loading...\n")
		_, _ = fmt.Fprintf(l.output, "🔧 [ENV DEBUG] Working directory: %s\n", l.workingDir)
		_, _ = fmt.Fprintf(l.output, "🔧 [ENV DEBUG] Environment: %s\n", l.environment)
		_, _ = fmt.Fprintf(l.output, "🔧 [ENV DEBUG] Host environment variables: %d\n", len(result.FinalEnv))
	}

	// Define the hierarchy of .env files to load
	envFiles := []string{
		".env",
		".env.local",
	}

	// Add environment-specific files if environment is specified
	if l.environment != "" {
		envFiles = append(envFiles,
			fmt.Sprintf(".env.%s", l.environment),
			fmt.Sprintf(".env.%s.local", l.environment),
		)
	}

	// Load each .env file in order
	for _, filename := range envFiles {
		filepath := filepath.Join(l.workingDir, filename)
		info := l.loadFile(filepath, result.FinalEnv)
		result.Files = append(result.Files, info)

		if l.debugMode {
			if info.Exists {
				if info.Loaded {
					_, _ = fmt.Fprintf(l.output, "✅ [ENV DEBUG] Loaded: %s (%d variables)\n", info.Path, len(info.Vars))
					for key, value := range info.Vars {
						// Mask sensitive values in debug output
						maskedValue := maskSensitiveValue(key, value)
						_, _ = fmt.Fprintf(l.output, "   %s=%s\n", key, maskedValue)
					}
				} else {
					_, _ = fmt.Fprintf(l.output, "❌ [ENV DEBUG] Failed to load: %s (error: %v)\n", info.Path, info.Error)
				}
			} else {
				_, _ = fmt.Fprintf(l.output, "⏭️  [ENV DEBUG] Skipped (not found): %s\n", info.Path)
			}
		}
	}

	if l.debugMode {
		_, _ = fmt.Fprintf(l.output, "🔧 [ENV DEBUG] Final environment: %d variables\n", len(result.FinalEnv))
		_, _ = fmt.Fprintf(l.output, "🔧 [ENV DEBUG] Environment loading complete\n\n")
	}

	return result, nil
}

// loadFile loads a single .env file and merges variables into the environment map
func (l *Loader) loadFile(filepath string, envMap map[string]string) EnvFileInfo {
	info := EnvFileInfo{
		Path:   filepath,
		Exists: false,
		Loaded: false,
		Vars:   make(map[string]string),
		Error:  nil,
	}

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return info
	}
	info.Exists = true

	// Open and read the file
	file, err := os.Open(filepath)
	if err != nil {
		info.Error = err
		return info
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// If we don't have an error yet, set this close error
			if info.Error == nil {
				info.Error = closeErr
			}
		}
	}()

	// Parse the file
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			// Invalid format, but don't fail - just skip
			if l.debugMode {
				_, _ = fmt.Fprintf(l.output, "⚠️  [ENV DEBUG] Skipping invalid line %d in %s: %s\n", lineNum, filepath, line)
			}
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = removeQuotes(value)

		// Store in both the file-specific vars and merge into main environment
		info.Vars[key] = value
		envMap[key] = value
	}

	if err := scanner.Err(); err != nil {
		info.Error = err
		return info
	}

	info.Loaded = true
	return info
}

// removeQuotes removes surrounding single or double quotes from a string
func removeQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// maskSensitiveValue masks potentially sensitive environment variable values
func maskSensitiveValue(key, value string) string {
	lowerKey := strings.ToLower(key)

	// List of common sensitive key patterns
	sensitivePatterns := []string{
		"password", "secret", "key", "token", "auth", "api",
		"credential", "private", "access", "salt",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerKey, pattern) {
			if len(value) <= 4 {
				return "***"
			}
			// Show first 4 characters and mask the rest
			return value[:4] + strings.Repeat("*", len(value)-4)
		}
	}

	return value
}

// PrintDebugInfo prints detailed debug information about the load result
func PrintDebugInfo(result *LoadResult, output io.Writer) {
	_, _ = fmt.Fprintf(output, "\n╔═══════════════════════════════════════════════════════════════╗\n")
	_, _ = fmt.Fprintf(output, "║             Environment Variables Debug Info                  ║\n")
	_, _ = fmt.Fprintf(output, "╚═══════════════════════════════════════════════════════════════╝\n\n")

	_, _ = fmt.Fprintf(output, "Environment: %s\n", result.Environment)
	if result.Environment == "" {
		_, _ = fmt.Fprintf(output, "  (no environment specified, using default files only)\n")
	}
	_, _ = fmt.Fprintf(output, "\n")

	_, _ = fmt.Fprintf(output, "Files Processed:\n")
	for i, file := range result.Files {
		_, _ = fmt.Fprintf(output, "  %d. %s\n", i+1, file.Path)
		if !file.Exists {
			_, _ = fmt.Fprintf(output, "     Status: Not found (skipped)\n")
		} else if !file.Loaded {
			_, _ = fmt.Fprintf(output, "     Status: Error loading (%v)\n", file.Error)
		} else {
			_, _ = fmt.Fprintf(output, "     Status: Loaded successfully\n")
			_, _ = fmt.Fprintf(output, "     Variables: %d\n", len(file.Vars))
		}
		_, _ = fmt.Fprintf(output, "\n")
	}

	_, _ = fmt.Fprintf(output, "Final Environment: %d total variables\n", len(result.FinalEnv))
	_, _ = fmt.Fprintf(output, "  (includes host OS environment + loaded .env files)\n\n")

	// List all final environment variables (sorted for readability)
	_, _ = fmt.Fprintf(output, "Final Variables (sample):\n")
	count := 0
	maxDisplay := 20
	for key, value := range result.FinalEnv {
		if count >= maxDisplay {
			_, _ = fmt.Fprintf(output, "  ... and %d more\n", len(result.FinalEnv)-maxDisplay)
			break
		}
		maskedValue := maskSensitiveValue(key, value)
		_, _ = fmt.Fprintf(output, "  %s=%s\n", key, maskedValue)
		count++
	}

	_, _ = fmt.Fprintf(output, "\n")
}
