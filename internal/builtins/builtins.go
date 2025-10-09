package builtins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Context provides access to execution context for builtins
type Context interface {
	GetProjectName() string
	GetSecretsManager() SecretsManager
}

// SecretsManager provides access to secrets
type SecretsManager interface {
	Get(namespace, key string) (string, error)
	Set(namespace, key, value string) error
	Exists(namespace, key string) (bool, error)
}

// BuiltinFunction represents a built-in function with optional context
type BuiltinFunction func(ctx Context, args ...string) (string, error)

// ProgressState holds the state of a progress indicator
type ProgressState struct {
	Name       string
	Message    string
	Percentage int
	StartTime  time.Time
	IsActive   bool
}

// TimerState holds the state of a timer
type TimerState struct {
	Name      string
	StartTime time.Time
	EndTime   *time.Time
	IsRunning bool
}

// Global state for progress and timers
var (
	progressStates = make(map[string]*ProgressState)
	timerStates    = make(map[string]*TimerState)
	stateMutex     sync.RWMutex
)

// Registry holds all built-in functions
var Registry = map[string]BuiltinFunction{
	"current git commit":    getCurrentGitCommit,
	"current git branch":    getCurrentGitBranch,
	"now.format":            formatCurrentTime,
	"file exists":           checkFileExists,
	"dir exists":            checkDirExists,
	"env":                   getEnvironmentVariable,
	"pwd":                   getCurrentDirectory,
	"hostname":              getHostname,
	"start progress":        startProgress,
	"update progress":       updateProgress,
	"finish progress":       finishProgress,
	"start timer":           startTimer,
	"stop timer":            stopTimer,
	"show elapsed time":     showElapsedTime,
	"docker compose status": checkDockerComposeStatus,
	"secret":                getSecret,
}

// getCurrentGitCommit returns the current git commit hash
func getCurrentGitCommit(ctx Context, args ...string) (string, error) {
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

// getCurrentGitBranch returns the current git branch name
func getCurrentGitBranch(ctx Context, args ...string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	return branch, nil
}

// formatCurrentTime formats the current time
func formatCurrentTime(ctx Context, args ...string) (string, error) {
	now := time.Now()

	// Default format if no args
	format := "2006-01-02 15:04:05"
	if len(args) > 0 {
		format = args[0]
	}

	return now.Format(format), nil
}

// checkFileExists checks if a file exists
func checkFileExists(ctx Context, args ...string) (string, error) {
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
func checkDirExists(ctx Context, args ...string) (string, error) {
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
func getEnvironmentVariable(ctx Context, args ...string) (string, error) {
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
func getCurrentDirectory(ctx Context, args ...string) (string, error) {
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
func getHostname(ctx Context, args ...string) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	return hostname, nil
}

// CallBuiltin calls a built-in function by name with optional context
func CallBuiltin(name string, ctx Context, args ...string) (string, error) {
	fn, exists := Registry[name]
	if !exists {
		return "", fmt.Errorf("unknown built-in function: %s", name)
	}

	return fn(ctx, args...)
}

// CallBuiltinLegacy calls a built-in function without context (for backward compatibility)
func CallBuiltinLegacy(name string, args ...string) (string, error) {
	return CallBuiltin(name, nil, args...)
}

// IsBuiltin checks if a function name is a built-in
func IsBuiltin(name string) bool {
	_, exists := Registry[name]
	return exists
}

// startProgress starts a new progress indicator
func startProgress(ctx Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("progress message required")
	}

	message := args[0]
	name := "default"
	if len(args) > 1 {
		name = args[1]
	}

	stateMutex.Lock()
	defer stateMutex.Unlock()

	progressStates[name] = &ProgressState{
		Name:       name,
		Message:    message,
		Percentage: 0,
		StartTime:  time.Now(),
		IsActive:   true,
	}

	return fmt.Sprintf("📋 %s", message), nil
}

// updateProgress updates an existing progress indicator
func updateProgress(ctx Context, args ...string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("percentage and message required")
	}

	percentageStr := args[0]
	message := args[1]
	name := "default"
	if len(args) > 2 {
		name = args[2]
	}

	percentage, err := strconv.Atoi(percentageStr)
	if err != nil {
		return "", fmt.Errorf("invalid percentage: %s", percentageStr)
	}

	if percentage < 0 || percentage > 100 {
		return "", fmt.Errorf("percentage must be between 0 and 100")
	}

	stateMutex.Lock()
	defer stateMutex.Unlock()

	progress, exists := progressStates[name]
	if !exists {
		return "", fmt.Errorf("progress indicator '%s' not found", name)
	}

	if !progress.IsActive {
		return "", fmt.Errorf("progress indicator '%s' is not active", name)
	}

	progress.Percentage = percentage
	progress.Message = message

	// Create progress bar
	progressBar := createProgressBar(percentage)

	return fmt.Sprintf("📋 %s %s (%d%%)", message, progressBar, percentage), nil
}

// finishProgress completes a progress indicator
func finishProgress(ctx Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("completion message required")
	}

	message := args[0]
	name := "default"
	if len(args) > 1 {
		name = args[1]
	}

	stateMutex.Lock()
	defer stateMutex.Unlock()

	progress, exists := progressStates[name]
	if !exists {
		return "", fmt.Errorf("progress indicator '%s' not found", name)
	}

	if !progress.IsActive {
		return "", fmt.Errorf("progress indicator '%s' is not active", name)
	}

	progress.IsActive = false
	progress.Percentage = 100
	progress.Message = message

	elapsed := time.Since(progress.StartTime)

	return fmt.Sprintf("✅ %s (completed in %v)", message, elapsed.Round(time.Millisecond)), nil
}

// startTimer starts a new timer
func startTimer(ctx Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("timer name required")
	}

	name := args[0]

	stateMutex.Lock()
	defer stateMutex.Unlock()

	// Check if timer already exists and is running
	if timer, exists := timerStates[name]; exists && timer.IsRunning {
		return "", fmt.Errorf("timer '%s' is already running", name)
	}

	timerStates[name] = &TimerState{
		Name:      name,
		StartTime: time.Now(),
		EndTime:   nil,
		IsRunning: true,
	}

	return fmt.Sprintf("⏱️  Started timer '%s'", name), nil
}

// stopTimer stops a running timer
func stopTimer(ctx Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("timer name required")
	}

	name := args[0]

	stateMutex.Lock()
	defer stateMutex.Unlock()

	timer, exists := timerStates[name]
	if !exists {
		return "", fmt.Errorf("timer '%s' not found", name)
	}

	if !timer.IsRunning {
		return "", fmt.Errorf("timer '%s' is not running", name)
	}

	now := time.Now()
	timer.EndTime = &now
	timer.IsRunning = false

	elapsed := now.Sub(timer.StartTime)

	return fmt.Sprintf("⏹️  Stopped timer '%s' (elapsed: %v)", name, elapsed.Round(time.Millisecond)), nil
}

// showElapsedTime shows the elapsed time for a timer
func showElapsedTime(ctx Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("timer name required")
	}

	name := args[0]

	stateMutex.RLock()
	defer stateMutex.RUnlock()

	timer, exists := timerStates[name]
	if !exists {
		return "", fmt.Errorf("timer '%s' not found", name)
	}

	var elapsed time.Duration
	if timer.IsRunning {
		elapsed = time.Since(timer.StartTime)
	} else if timer.EndTime != nil {
		elapsed = timer.EndTime.Sub(timer.StartTime)
	} else {
		return "", fmt.Errorf("timer '%s' has no valid end time", name)
	}

	status := "stopped"
	if timer.IsRunning {
		status = "running"
	}

	return fmt.Sprintf("⏱️  Timer '%s' (%s): %v", name, status, elapsed.Round(time.Millisecond)), nil
}

// createProgressBar creates a visual progress bar
func createProgressBar(percentage int) string {
	const barLength = 20
	filled := (percentage * barLength) / 100

	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	return bar
}

// checkDockerComposeStatus checks if a Docker Compose project is in a usable state
func checkDockerComposeStatus(ctx Context, args ...string) (string, error) {
	// Determine project name/path - default to current directory
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Detect which Docker Compose command to use (prioritize "docker compose")
	composeCmd, err := detectDockerComposeCommand()
	if err != nil {
		return "unavailable", fmt.Errorf("docker compose not available: %w", err)
	}

	// Get container status for the project
	status, err := getComposeProjectStatus(composeCmd, projectPath)
	if err != nil {
		return "error", fmt.Errorf("failed to get compose status: %w", err)
	}

	return status, nil
}

// detectDockerComposeCommand detects which Docker Compose command to use
func detectDockerComposeCommand() ([]string, error) {
	// First try "docker compose" (Docker Compose V2)
	if isCommandAvailable("docker") {
		cmd := exec.Command("docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}

	// Fallback to standalone "docker-compose" command (V1)
	if isCommandAvailable("docker-compose") {
		return []string{"docker-compose"}, nil
	}

	return nil, fmt.Errorf("neither 'docker compose' nor 'docker-compose' is available")
}

// getComposeProjectStatus gets the status of containers in a compose project
func getComposeProjectStatus(composeCmd []string, projectPath string) (string, error) {
	// Change to project directory if specified
	originalDir, err := os.Getwd()
	if err != nil {
		return "error", err
	}

	if projectPath != "." {
		if err := os.Chdir(projectPath); err != nil {
			return "error", fmt.Errorf("failed to change to project directory: %w", err)
		}
		defer func() {
			_ = os.Chdir(originalDir)
		}()
	}

	// Run "docker compose ps" to get container status
	psCmd := append(composeCmd, "ps", "--format", "table")
	cmd := exec.Command(psCmd[0], psCmd[1:]...)
	output, err := cmd.Output()
	if err != nil {
		// If ps fails, the project is likely down or not initialized
		return "down", nil
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return "down", nil
	}

	// Parse the output to determine status
	lines := strings.Split(outputStr, "\n")
	if len(lines) <= 1 {
		// Only header line or empty, project is down
		return "down", nil
	}

	// Count containers by status
	var running, restarting, exited, unhealthy int
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Look for status indicators in the line
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "up") || strings.Contains(lineLower, "running") {
			running++
		} else if strings.Contains(lineLower, "restarting") {
			restarting++
		} else if strings.Contains(lineLower, "exited") {
			exited++
		} else if strings.Contains(lineLower, "unhealthy") {
			unhealthy++
		}
	}

	// Determine overall status
	totalContainers := running + restarting + exited + unhealthy
	if totalContainers == 0 {
		return "down", nil
	}

	// If any containers are restarting or unhealthy, project is unusable
	if restarting > 0 || unhealthy > 0 {
		return "unusable", nil
	}

	// If all containers are running, project is usable
	if running == totalContainers {
		return "usable", nil
	}

	// If some containers are exited but others are running, it's partially up
	if running > 0 {
		return "partial", nil
	}

	// All containers are exited
	return "down", nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
