package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Domain: Self-Update
// This file contains logic for updating the drun binary

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// HandleSelfUpdate handles the --self-update flag
func HandleSelfUpdate(versionStr string) error {
	fmt.Println("🔄 Checking for drun updates...")

	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Check for latest version
	latestVersion, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Compare versions (normalize both for consistent comparison)
	currentVersion := normalizeVersion(versionStr)
	normalizedLatest := normalizeVersion(latestVersion)
	if currentVersion == normalizedLatest {
		fmt.Printf("✅ You're already running the latest version: %s\n", versionStr)
		return nil
	}

	fmt.Printf("📦 New version available: %s (current: %s)\n", latestVersion, versionStr)

	// Ask for user confirmation
	if !askForConfirmation("Do you want to update now?") {
		fmt.Println("Update cancelled.")
		return nil
	}

	// Create backup
	backupPath, err := createBackup(currentExe, versionStr)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Printf("💾 Created backup at: %s\n", backupPath)

	// Download and install new version
	if err := downloadAndInstall(latestVersion, currentExe); err != nil {
		// Restore backup on failure
		fmt.Printf("❌ Update failed: %v\n", err)
		fmt.Println("🔄 Restoring backup...")
		if restoreErr := restoreBackup(backupPath, currentExe); restoreErr != nil {
			return fmt.Errorf("update failed and backup restoration failed: %v (original error: %w)", restoreErr, err)
		}
		fmt.Println("✅ Backup restored successfully")
		return err
	}

	fmt.Printf("🎉 Successfully updated to version %s!\n", latestVersion)
	fmt.Printf("💾 Backup available at: %s\n", backupPath)

	// Display the actual version from the updated binary
	fmt.Println("\nVerifying updated binary:")
	cmd := exec.Command(currentExe, "--version")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to verify updated binary: %v\n", err)
	}

	return nil
}

// getLatestVersion fetches the latest version from GitHub
func getLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://api.github.com/repos/phillarmonic/drun/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch release information: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release information: %w", err)
	}

	return release.TagName, nil
}

// normalizeVersion removes 'v' prefix and '-dev' suffix for comparison
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimSuffix(v, "-dev")
	return v
}

// askForConfirmation asks the user for yes/no confirmation
func askForConfirmation(question string) bool {
	fmt.Printf("%s (y/N): ", question)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

// createBackup creates a backup of the current executable
func createBackup(currentExe, versionStr string) (string, error) {
	// Create backup directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	backupDir := filepath.Join(homeDir, ".drun")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create timestamped backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("xdrun_%s_backup_%s", normalizeVersion(versionStr), timestamp)
	if runtime.GOOS == "windows" {
		backupFilename += ".exe"
	}

	backupPath := filepath.Join(backupDir, backupFilename)

	// Copy current executable to backup location
	if err := copyFile(currentExe, backupPath); err != nil {
		return "", fmt.Errorf("failed to copy executable: %w", err)
	}

	// Make backup executable
	if err := os.Chmod(backupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make backup executable: %w", err)
	}

	// Clean up old backups (keep last 5)
	cleanupOldBackups(backupDir)

	return backupPath, nil
}

// downloadAndInstall downloads and installs the new version
func downloadAndInstall(version, targetPath string) error {
	// Determine platform and architecture
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go arch to release arch
	var arch string
	switch goarch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return fmt.Errorf("unsupported architecture: %s", goarch)
	}

	// Construct download URL
	binaryName := fmt.Sprintf("xdrun-%s-%s", goos, arch)
	if goos == "windows" {
		binaryName += ".exe"
	}

	downloadURL := fmt.Sprintf("https://github.com/phillarmonic/drun/releases/download/%s/%s", version, binaryName)

	fmt.Printf("📥 Downloading %s...\n", binaryName)

	// Download the binary
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "drun-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(tempFile.Name()); removeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove temporary file: %v\n", removeErr)
		}
	}()
	defer func() {
		if closeErr := tempFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close temporary file: %v\n", closeErr)
		}
	}()

	// Copy downloaded content to temp file
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write downloaded binary: %w", err)
	}

	// Make temp file executable
	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Verify the binary works
	fmt.Println("🔍 Verifying downloaded binary...")
	cmd := exec.Command(tempFile.Name(), "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("downloaded binary failed verification: %w", err)
	}

	// Install the binary (may require elevated permissions)
	fmt.Println("📦 Installing new version...")
	if err := installBinary(tempFile.Name(), targetPath); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	return nil
}

// installBinary installs the binary, handling permissions as needed
func installBinary(sourcePath, targetPath string) error {
	// Try direct copy first
	if err := copyFile(sourcePath, targetPath); err == nil {
		return nil
	}

	// If direct copy failed, try with elevated permissions
	fmt.Println("🔐 Requesting elevated permissions...")

	switch runtime.GOOS {
	case "darwin", "linux":
		// Use sudo on Unix-like systems
		cmd := exec.Command("sudo", "cp", sourcePath, targetPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case "windows":
		// On Windows, we need to use PowerShell with elevation
		// This is more complex and might require the user to run as administrator
		return copyFile(sourcePath, targetPath)

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close source file: %v\n", closeErr)
		}
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := destFile.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close destination file: %v\n", closeErr)
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// restoreBackup restores from backup
func restoreBackup(backupPath, targetPath string) error {
	return copyFile(backupPath, targetPath)
}

// cleanupOldBackups removes old backup files, keeping only the last 5
func cleanupOldBackups(backupDir string) {
	files, err := filepath.Glob(filepath.Join(backupDir, "xdrun*backup*"))
	if err != nil {
		return
	}

	if len(files) <= 5 {
		return
	}

	// Sort files by modification time (newest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var fileInfos []fileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			path:    file,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.Before(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove old files (keep first 5)
	for i := 5; i < len(fileInfos); i++ {
		if removeErr := os.Remove(fileInfos[i].path); removeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove old backup %s: %v\n", fileInfos[i].path, removeErr)
		}
	}
}
