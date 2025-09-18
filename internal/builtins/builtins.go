package builtins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BuiltinFunction represents a built-in function
type BuiltinFunction func(args ...string) (string, error)

// Registry holds all built-in functions
var Registry = map[string]BuiltinFunction{
	"current git commit": getCurrentGitCommit,
	"now.format":         formatCurrentTime,
	"file exists":        checkFileExists,
	"dir exists":         checkDirExists,
	"env":                getEnvironmentVariable,
	"pwd":                getCurrentDirectory,
	"hostname":           getHostname,
}

// getCurrentGitCommit returns the current git commit hash
func getCurrentGitCommit(args ...string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git commit: %w", err)
	}

	commit := strings.TrimSpace(string(output))

	// If args provided, return short version
	if len(args) > 0 && args[0] == "short" {
		if len(commit) > 7 {
			return commit[:7], nil
		}
	}

	return commit, nil
}

// formatCurrentTime formats the current time
func formatCurrentTime(args ...string) (string, error) {
	now := time.Now()

	// Default format if no args
	format := "2006-01-02 15:04:05"
	if len(args) > 0 {
		format = args[0]
	}

	return now.Format(format), nil
}

// checkFileExists checks if a file exists
func checkFileExists(args ...string) (string, error) {
	if len(args) == 0 {
		return "false", fmt.Errorf("file path required")
	}

	path := args[0]
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "false", nil
	} else if err != nil {
		return "false", err
	}

	return "true", nil
}

// checkDirExists checks if a directory exists
func checkDirExists(args ...string) (string, error) {
	if len(args) == 0 {
		return "false", fmt.Errorf("directory path required")
	}

	path := args[0]
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "false", nil
	} else if err != nil {
		return "false", err
	}

	if info.IsDir() {
		return "true", nil
	}

	return "false", nil
}

// getEnvironmentVariable gets an environment variable
func getEnvironmentVariable(args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("environment variable name required")
	}

	name := args[0]
	value := os.Getenv(name)

	// Return default value if provided and env var is empty
	if value == "" && len(args) > 1 {
		return args[1], nil
	}

	return value, nil
}

// getCurrentDirectory returns the current working directory
func getCurrentDirectory(args ...string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// If args provided, return just the basename
	if len(args) > 0 && args[0] == "basename" {
		return filepath.Base(dir), nil
	}

	return dir, nil
}

// getHostname returns the system hostname
func getHostname(args ...string) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	return hostname, nil
}

// CallBuiltin calls a built-in function by name
func CallBuiltin(name string, args ...string) (string, error) {
	fn, exists := Registry[name]
	if !exists {
		return "", fmt.Errorf("unknown built-in function: %s", name)
	}

	return fn(args...)
}

// IsBuiltin checks if a function name is a built-in
func IsBuiltin(name string) bool {
	_, exists := Registry[name]
	return exists
}
