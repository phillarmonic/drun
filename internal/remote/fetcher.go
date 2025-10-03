// Package remote provides fetchers for remote includes (GitHub, HTTPS)
package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Fetcher defines the interface for remote content fetchers
type Fetcher interface {
	Fetch(ctx context.Context, path, ref string) ([]byte, error)
	Protocol() string
}

// ParseRemoteURL parses a remote URL and extracts protocol, path, and ref
// Examples:
//
//	github:owner/repo/path/file.drun@main -> "github", "owner/repo/path/file.drun", "main"
//	https://example.com/file.drun -> "https", "https://example.com/file.drun", ""
//	drunhub:ops/docker -> "drunhub", "ops/docker", ""
func ParseRemoteURL(url string) (protocol, path, ref string, err error) {
	// Check for drunhub protocol
	if strings.HasPrefix(url, "drunhub:") {
		protocol = "drunhub"
		rest := strings.TrimPrefix(url, "drunhub:")

		// Check for @ref (optional)
		if idx := strings.Index(rest, "@"); idx != -1 {
			path = rest[:idx]
			ref = rest[idx+1:]
		} else {
			path = rest
			ref = ""
		}
		return protocol, path, ref, nil
	}

	// Check for GitHub protocol
	if strings.HasPrefix(url, "github:") {
		protocol = "github"
		rest := strings.TrimPrefix(url, "github:")

		// Check for @ref
		if idx := strings.Index(rest, "@"); idx != -1 {
			path = rest[:idx]
			ref = rest[idx+1:]
		} else {
			path = rest
			ref = ""
		}
		return protocol, path, ref, nil
	}

	// Check for HTTPS
	if strings.HasPrefix(url, "https://") {
		protocol = "https"
		path = url
		ref = "" // HTTPS doesn't support refs
		return protocol, path, ref, nil
	}

	// Check for HTTP (reject for security)
	if strings.HasPrefix(url, "http://") {
		return "", "", "", fmt.Errorf("insecure HTTP URLs are not allowed, use HTTPS")
	}

	return "", "", "", fmt.Errorf("unsupported protocol in URL: %s", url)
}

// IsRemoteURL checks if a URL is a remote include
func IsRemoteURL(url string) bool {
	return strings.HasPrefix(url, "github:") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "drunhub:")
}

// GitHubFetcher fetches content from GitHub repositories
type GitHubFetcher struct {
	token         string
	client        *http.Client
	branchCache   map[string]string // Cache for default branches
	cacheExpiry   map[string]time.Time
	cacheDuration time.Duration
}

// NewGitHubFetcher creates a new GitHub fetcher
func NewGitHubFetcher() *GitHubFetcher {
	return &GitHubFetcher{
		token:         os.Getenv("GITHUB_TOKEN"),
		client:        &http.Client{Timeout: 30 * time.Second},
		branchCache:   make(map[string]string),
		cacheExpiry:   make(map[string]time.Time),
		cacheDuration: 1 * time.Hour, // Cache default branches for 1 hour
	}
}

// Protocol returns the protocol identifier
func (g *GitHubFetcher) Protocol() string {
	return "github"
}

// Fetch retrieves content from a GitHub repository
func (g *GitHubFetcher) Fetch(ctx context.Context, path, ref string) ([]byte, error) {
	// Parse: owner/repo/path/to/file.drun
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid GitHub path format, expected owner/repo/path/file.drun, got: %s", path)
	}

	owner, repo, filePath := parts[0], parts[1], parts[2]

	// If no ref provided, detect default branch
	if ref == "" {
		var err error
		ref, err = g.getDefaultBranch(ctx, owner, repo, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to detect default branch: %w", err)
		}
	}

	// Use raw.githubusercontent.com for content
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/%s",
		owner, repo, ref, filePath,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add GitHub token if available
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	req.Header.Set("User-Agent", "drun-remote-includes")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GitHub: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// Check for rate limiting
		if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
			resetTime := resp.Header.Get("X-RateLimit-Reset")
			return nil, fmt.Errorf("GitHub rate limit exceeded (resets at %s)", resetTime)
		}
		return nil, fmt.Errorf("GitHub returned status %d for %s", resp.StatusCode, url)
	}

	// Read content with size limit (10 MB max)
	const maxSize = 10 * 1024 * 1024
	limited := io.LimitReader(resp.Body, maxSize)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return content, nil
}

// getDefaultBranch intelligently determines the repository's default branch
func (g *GitHubFetcher) getDefaultBranch(ctx context.Context, owner, repo, filePath string) (string, error) {
	cacheKey := fmt.Sprintf("%s/%s", owner, repo)

	// Check cache first
	if branch, ok := g.branchCache[cacheKey]; ok {
		if expiry, exists := g.cacheExpiry[cacheKey]; exists && time.Now().Before(expiry) {
			return branch, nil
		}
	}

	// Query GitHub API for repo info
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return g.tryDefaultBranchFallback(ctx, owner, repo, filePath)
	}

	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	req.Header.Set("User-Agent", "drun-remote-includes")

	resp, err := g.client.Do(req)
	if err != nil {
		// Fallback strategy: try main, then master
		return g.tryDefaultBranchFallback(ctx, owner, repo, filePath)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return g.tryDefaultBranchFallback(ctx, owner, repo, filePath)
	}

	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return g.tryDefaultBranchFallback(ctx, owner, repo, filePath)
	}

	// Cache the result
	g.branchCache[cacheKey] = repoInfo.DefaultBranch
	g.cacheExpiry[cacheKey] = time.Now().Add(g.cacheDuration)

	return repoInfo.DefaultBranch, nil
}

// tryDefaultBranchFallback tries common default branches
func (g *GitHubFetcher) tryDefaultBranchFallback(ctx context.Context, owner, repo, filePath string) (string, error) {
	// Try main first (modern default)
	if g.fileExists(ctx, owner, repo, "main", filePath) {
		return "main", nil
	}

	// Try master (legacy default)
	if g.fileExists(ctx, owner, repo, "master", filePath) {
		return "master", nil
	}

	// Last resort: default to main and let it fail with a clear error
	return "main", nil
}

// fileExists checks if a file exists at a specific ref
func (g *GitHubFetcher) fileExists(ctx context.Context, owner, repo, ref, filePath string) bool {
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/%s",
		owner, repo, ref, filePath,
	)
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false
	}

	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	req.Header.Set("User-Agent", "drun-remote-includes")

	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

// HTTPSFetcher fetches content from HTTPS URLs
type HTTPSFetcher struct {
	client *http.Client
}

// NewHTTPSFetcher creates a new HTTPS fetcher
func NewHTTPSFetcher() *HTTPSFetcher {
	return &HTTPSFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Protocol returns the protocol identifier
func (h *HTTPSFetcher) Protocol() string {
	return "https"
}

// Fetch retrieves content from an HTTPS URL
func (h *HTTPSFetcher) Fetch(ctx context.Context, url, _ string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "drun-remote-includes")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from HTTPS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTPS returned status %d for %s", resp.StatusCode, url)
	}

	// Read content with size limit (10 MB max)
	const maxSize = 10 * 1024 * 1024
	limited := io.LimitReader(resp.Body, maxSize)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return content, nil
}

// DrunhubFetcher fetches content from the drunhub standard library repository
type DrunhubFetcher struct {
	githubFetcher  *GitHubFetcher
	ignoredFolders map[string]bool
}

// NewDrunhubFetcher creates a new drunhub fetcher
func NewDrunhubFetcher(githubFetcher *GitHubFetcher) *DrunhubFetcher {
	// Default ignored folders
	ignoredFolders := map[string]bool{
		"docs":    true,
		".github": true,
	}

	return &DrunhubFetcher{
		githubFetcher:  githubFetcher,
		ignoredFolders: ignoredFolders,
	}
}

// SetIgnoredFolders sets the list of folders to ignore from drunhub
func (d *DrunhubFetcher) SetIgnoredFolders(folders []string) {
	d.ignoredFolders = make(map[string]bool, len(folders))
	for _, folder := range folders {
		d.ignoredFolders[folder] = true
	}
}

// AddIgnoredFolder adds a folder to the ignore list
func (d *DrunhubFetcher) AddIgnoredFolder(folder string) {
	d.ignoredFolders[folder] = true
}

// Protocol returns the protocol identifier
func (d *DrunhubFetcher) Protocol() string {
	return "drunhub"
}

// Fetch retrieves content from the drunhub repository
// path format: "ops/docker" -> translates to phillarmonic/drun-hub/ops/docker.drun
func (d *DrunhubFetcher) Fetch(ctx context.Context, path, ref string) ([]byte, error) {
	// Check if the path starts with an ignored folder
	for ignoredFolder := range d.ignoredFolders {
		if strings.HasPrefix(path, ignoredFolder+"/") || path == ignoredFolder {
			return nil, fmt.Errorf("access to folder '%s' is not allowed from drunhub", ignoredFolder)
		}
	}

	// Add .drun extension if not present
	if !strings.HasSuffix(path, ".drun") {
		path = path + ".drun"
	}

	// Convert to GitHub path: phillarmonic/drun-hub/{path}
	githubPath := fmt.Sprintf("phillarmonic/drun-hub/%s", path)

	// Use the GitHub fetcher to retrieve the content
	return d.githubFetcher.Fetch(ctx, githubPath, ref)
}
