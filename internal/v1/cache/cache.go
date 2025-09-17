package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phillarmonic/drun/internal/v1/model"
	"github.com/phillarmonic/drun/internal/v1/tmpl"
)

// Manager handles caching operations
type Manager struct {
	cacheDir       string
	templateEngine *tmpl.Engine
	disabled       bool
}

// NewManager creates a new cache manager
func NewManager(cacheDir string, templateEngine *tmpl.Engine, disabled bool) *Manager {
	return &Manager{
		cacheDir:       cacheDir,
		templateEngine: templateEngine,
		disabled:       disabled,
	}
}

// IsValid checks if a cache entry is valid for the given recipe and context
func (m *Manager) IsValid(recipe *model.Recipe, ctx *model.ExecutionContext) (bool, error) {
	if m.disabled || recipe.CacheKey == "" {
		return false, nil
	}

	// Render the cache key template
	cacheKey, err := m.templateEngine.Render(recipe.CacheKey, ctx)
	if err != nil {
		return false, fmt.Errorf("failed to render cache key: %w", err)
	}

	// Generate cache file path
	cacheFile := m.getCacheFilePath(cacheKey)

	// Check if cache file exists and is not too old
	info, err := os.Stat(cacheFile)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check cache file: %w", err)
	}

	// For now, consider cache valid if file exists and is less than 1 hour old
	// In a more sophisticated implementation, this could be configurable
	maxAge := time.Hour
	if time.Since(info.ModTime()) > maxAge {
		return false, nil
	}

	return true, nil
}

// MarkComplete marks a recipe as successfully completed in the cache
func (m *Manager) MarkComplete(recipe *model.Recipe, ctx *model.ExecutionContext) error {
	if m.disabled || recipe.CacheKey == "" {
		return nil
	}

	// Render the cache key template
	cacheKey, err := m.templateEngine.Render(recipe.CacheKey, ctx)
	if err != nil {
		return fmt.Errorf("failed to render cache key: %w", err)
	}

	// Generate cache file path
	cacheFile := m.getCacheFilePath(cacheKey)

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create cache marker file with timestamp and metadata
	content := fmt.Sprintf("cached_at: %s\ncache_key: %s\n", time.Now().Format(time.RFC3339), cacheKey)
	if err := os.WriteFile(cacheFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Clear removes all cache entries
func (m *Manager) Clear() error {
	if m.disabled {
		return nil
	}

	if err := os.RemoveAll(m.cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// getCacheFilePath generates a cache file path from a cache key
func (m *Manager) getCacheFilePath(cacheKey string) string {
	// Hash the cache key to create a safe filename
	hash := sha256.Sum256([]byte(cacheKey))
	hashStr := fmt.Sprintf("%x", hash)

	// Use first 16 characters of hash for the filename
	filename := hashStr[:16] + ".cache"

	return filepath.Join(m.cacheDir, filename)
}

// GetStats returns cache statistics
func (m *Manager) GetStats() (CacheStats, error) {
	stats := CacheStats{}

	if m.disabled {
		return stats, nil
	}

	// Count cache files
	err := filepath.Walk(m.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".cache") {
			stats.TotalEntries++
			if time.Since(info.ModTime()) < time.Hour {
				stats.ValidEntries++
			}
		}
		return nil
	})

	return stats, err
}

// CacheStats contains cache statistics
type CacheStats struct {
	TotalEntries int
	ValidEntries int
}
