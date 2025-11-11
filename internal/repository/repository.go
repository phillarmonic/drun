package repository

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/drun/internal/domain/orchestration"
)

// Manager manages Git repository operations
type Manager struct {
	workDir string
}

// NewManager creates a new repository manager
func NewManager(workDir string) *Manager {
	return &Manager{
		workDir: workDir,
	}
}

// resolvePath resolves a target path, handling both absolute and relative paths correctly
func (m *Manager) resolvePath(targetPath string) string {
	// If path is already absolute, use it directly
	if filepath.IsAbs(targetPath) {
		return targetPath
	}
	// Otherwise, join with workDir
	return filepath.Join(m.workDir, targetPath)
}

// Clone clones a repository if it doesn't exist
func (m *Manager) Clone(ctx context.Context, config *orchestration.Repository, targetPath string) error {
	fullPath := m.resolvePath(targetPath)

	// Check if target path already exists (as a directory or git repository)
	if _, err := os.Stat(fullPath); err == nil {
		// Directory exists, check if it's already a git repository
		if _, err := os.Stat(filepath.Join(fullPath, ".git")); err == nil {
			// Already a git repository, skip cloning
			return nil
		}
		// Directory exists but is not a git repository
		// If Clone is false, don't clone into existing directory
		if !config.Clone {
			return fmt.Errorf("target path '%s' exists but is not a git repository and clone is false", targetPath)
		}
		// Clone is true, proceed with cloning (will overwrite or fail)
	}

	// If Clone is false and directory doesn't exist, don't clone
	if !config.Clone {
		return fmt.Errorf("repository at '%s' does not exist and clone is false", targetPath)
	}

	// Prepare clone command
	args := []string{"clone"}

	// Add branch or tag if specified
	if config.Branch != "" {
		args = append(args, "--branch", config.Branch)
	} else if config.Tag != "" {
		args = append(args, "--branch", config.Tag)
	}

	// Add URL and target path
	args = append(args, config.URL, fullPath)

	// Execute clone command
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = m.workDir

	// Set up SSH key if specified
	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err != nil {
			return fmt.Errorf("failed to expand SSH key path: %w", err)
		}

		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Update updates a repository by pulling latest changes
func (m *Manager) Update(ctx context.Context, config *orchestration.Repository, targetPath string) error {
	fullPath := m.resolvePath(targetPath)

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(fullPath, ".git")); os.IsNotExist(err) {
		// Repository doesn't exist, clone it
		return m.Clone(ctx, config, targetPath)
	}

	// Fetch latest changes
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	fetchCmd.Dir = fullPath

	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err != nil {
			return fmt.Errorf("failed to expand SSH key path: %w", err)
		}

		fetchCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
	}

	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch repository: %w\nOutput: %s", err, string(output))
	}

	// Pull latest changes for current branch
	pullCmd := exec.CommandContext(ctx, "git", "pull")
	pullCmd.Dir = fullPath

	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err != nil {
			return fmt.Errorf("failed to expand SSH key path: %w", err)
		}

		pullCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
	}

	if output, err := pullCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull repository: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Checkout checks out a specific branch or tag
func (m *Manager) Checkout(ctx context.Context, targetPath, ref string) error {
	fullPath := m.resolvePath(targetPath)

	cmd := exec.CommandContext(ctx, "git", "checkout", ref)
	cmd.Dir = fullPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to checkout %s: %w\nOutput: %s", ref, err, string(output))
	}

	return nil
}

// GetCurrentBranch returns the current branch name
func (m *Manager) GetCurrentBranch(ctx context.Context, targetPath string) (string, error) {
	fullPath := m.resolvePath(targetPath)

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = fullPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentTag returns the current tag if any
func (m *Manager) GetCurrentTag(ctx context.Context, targetPath string) (string, error) {
	fullPath := m.resolvePath(targetPath)

	cmd := exec.CommandContext(ctx, "git", "describe", "--exact-match", "--tags", "HEAD")
	cmd.Dir = fullPath

	output, err := cmd.Output()
	if err != nil {
		// No tag at HEAD is not an error
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// GetDefaultBranch returns the default branch name from the remote repository
// Falls back to checking origin/HEAD, then tries "main" and "master"
func (m *Manager) GetDefaultBranch(ctx context.Context, config *orchestration.Repository, targetPath string) (string, error) {
	fullPath := m.resolvePath(targetPath)

	// First, try to get the default branch from origin/HEAD
	cmd := exec.CommandContext(ctx, "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = fullPath

	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err != nil {
			return "", fmt.Errorf("failed to expand SSH key path: %w", err)
		}
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
	}

	output, err := cmd.Output()
	if err == nil {
		// Parse output like "refs/remotes/origin/main" or "refs/remotes/origin/master"
		ref := strings.TrimSpace(string(output))
		if strings.HasPrefix(ref, "refs/remotes/origin/") {
			branch := strings.TrimPrefix(ref, "refs/remotes/origin/")
			return branch, nil
		}
	}

	// If origin/HEAD is not set, try to fetch and set it
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	fetchCmd.Dir = fullPath

	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err != nil {
			return "", fmt.Errorf("failed to expand SSH key path: %w", err)
		}
		fetchCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
	}

	_ = fetchCmd.Run() // Ignore errors, just try to fetch

	// Try again after fetch
	cmd = exec.CommandContext(ctx, "git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = fullPath
	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err == nil {
			cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
		}
	}

	output, err = cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		if strings.HasPrefix(ref, "refs/remotes/origin/") {
			branch := strings.TrimPrefix(ref, "refs/remotes/origin/")
			return branch, nil
		}
	}

	// Fallback: check if main or master branches exist on remote
	checkBranchCmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", "origin", "main")
	checkBranchCmd.Dir = fullPath
	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err == nil {
			checkBranchCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
		}
	}

	output, err = checkBranchCmd.Output()
	if err == nil && len(output) > 0 {
		return "main", nil
	}

	// Try master
	checkBranchCmd = exec.CommandContext(ctx, "git", "ls-remote", "--heads", "origin", "master")
	checkBranchCmd.Dir = fullPath
	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err == nil {
			checkBranchCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
		}
	}

	output, err = checkBranchCmd.Output()
	if err == nil && len(output) > 0 {
		return "master", nil
	}

	// Last resort: default to main
	return "main", nil
}

// GetStatus returns the repository status
func (m *Manager) GetStatus(ctx context.Context, targetPath string) (string, error) {
	fullPath := m.resolvePath(targetPath)

	cmd := exec.CommandContext(ctx, "git", "status", "--short")
	cmd.Dir = fullPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	return string(output), nil
}

// IsClean returns true if the repository has no uncommitted changes
func (m *Manager) IsClean(ctx context.Context, targetPath string) (bool, error) {
	status, err := m.GetStatus(ctx, targetPath)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(status) == "", nil
}

// HasRemoteUpdates checks if there are updates available from the remote repository
// Returns true if the local branch is behind the remote branch
func (m *Manager) HasRemoteUpdates(ctx context.Context, config *orchestration.Repository, targetPath string) (bool, error) {
	fullPath := m.resolvePath(targetPath)

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(fullPath, ".git")); os.IsNotExist(err) {
		return false, fmt.Errorf("repository at '%s' does not exist", targetPath)
	}

	// Fetch latest changes from remote
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	fetchCmd.Dir = fullPath

	if config.SSHKey != "" {
		sshKeyPath, err := expandPath(config.SSHKey)
		if err != nil {
			return false, fmt.Errorf("failed to expand SSH key path: %w", err)
		}

		fetchCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKeyPath))
	}

	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("failed to fetch repository: %w\nOutput: %s", err, string(output))
	}

	// Get current branch
	branch := config.Branch
	if branch == "" {
		currentBranch, err := m.GetCurrentBranch(ctx, targetPath)
		if err != nil {
			return false, fmt.Errorf("failed to get current branch: %w", err)
		}
		branch = currentBranch
	}

	// Check if local branch is behind remote
	// git rev-list HEAD..origin/<branch> --count
	revListCmd := exec.CommandContext(ctx, "git", "rev-list", fmt.Sprintf("HEAD..origin/%s", branch), "--count")
	revListCmd.Dir = fullPath

	output, err := revListCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check remote updates: %w", err)
	}

	count := strings.TrimSpace(string(output))
	return count != "0", nil
}

// EnsureRepository ensures a repository is cloned and optionally updated
func (m *Manager) EnsureRepository(ctx context.Context, config *orchestration.Repository, targetPath string) error {
	fullPath := m.resolvePath(targetPath)

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(fullPath, ".git")); os.IsNotExist(err) {
		// Repository doesn't exist
		// Only clone if Clone is true (defaults to true)
		if !config.Clone {
			return fmt.Errorf("repository at '%s' does not exist and clone is false", targetPath)
		}

		// Clone the repository
		if err := m.Clone(ctx, config, targetPath); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}

		// Checkout specific branch or tag if specified
		if config.Branch != "" {
			if err := m.Checkout(ctx, targetPath, config.Branch); err != nil {
				return fmt.Errorf("failed to checkout branch: %w", err)
			}
		} else if config.Tag != "" {
			if err := m.Checkout(ctx, targetPath, config.Tag); err != nil {
				return fmt.Errorf("failed to checkout tag: %w", err)
			}
		}

		return nil
	}

	// Repository exists
	if config.UpdateOnStart {
		// Update repository
		if err := m.Update(ctx, config, targetPath); err != nil {
			return fmt.Errorf("failed to update repository: %w", err)
		}
	}

	return nil
}

// expandPath expands ~ in paths to the user's home directory
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}
