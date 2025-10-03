// Package cache provides a caching layer for remote includes using SoloDB
package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	solodb "github.com/phillarmonic/SoloDB"
)

// Manager handles caching of remote includes with expiration
type Manager struct {
	db         *solodb.DB
	expiration time.Duration
	disabled   bool
}

// Stats provides cache statistics
type Stats struct {
	Keys        int
	FileBytes   int64
	LiveRecords int64
	Hits        int64
	Misses      int64
}

// NewManager creates a new cache manager with SoloDB
func NewManager(expiration time.Duration, disabled bool) (*Manager, error) {
	if disabled {
		return &Manager{
			disabled:   true,
			expiration: expiration,
		}, nil
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create ~/.drun directory
	drunDir := filepath.Join(homeDir, ".drun")
	if err := os.MkdirAll(drunDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .drun directory: %w", err)
	}

	// Open cache database
	dbPath := filepath.Join(drunDir, "cache.solo")
	db, err := solodb.Open(solodb.Options{
		Path:       dbPath,
		Durability: solodb.SyncBatch, // Balance between safety and performance
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}

	return &Manager{
		db:         db,
		expiration: expiration,
		disabled:   false,
	}, nil
}

// GenerateKey creates a cache key from a URL and optional ref
func GenerateKey(url, ref string) string {
	h := sha256.New()
	h.Write([]byte(url))
	if ref != "" {
		h.Write([]byte(":"))
		h.Write([]byte(ref))
	}
	return "remote:" + hex.EncodeToString(h.Sum(nil))[:32] // Use first 32 chars
}

// Get retrieves content from cache
// Returns: content, hit (true if found and not expired), error
func (m *Manager) Get(key string) ([]byte, bool, error) {
	if m.disabled {
		return nil, false, nil
	}

	rc, _, _, err := m.db.GetBlob(key)
	if err == solodb.ErrNotFound {
		return nil, false, nil
	}
	if err == solodb.ErrExpired {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache read error: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, false, fmt.Errorf("cache read error: %w", err)
	}

	return content, true, nil
}

// GetStale retrieves content from cache even if expired (for fallback)
func (m *Manager) GetStale(key string) ([]byte, bool) {
	if m.disabled || m.db == nil {
		return nil, false
	}

	// Try to get the value, ignoring expiration
	rc, _, _, err := m.db.GetBlob(key)
	if err == solodb.ErrNotFound {
		return nil, false
	}
	if err == solodb.ErrExpired {
		// For expired entries, we can't get the content anymore
		// SoloDB's lazy GC removes them from the index
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, false
	}

	return content, true
}

// Set stores content in cache with expiration
func (m *Manager) Set(key string, content []byte, expiration time.Duration) error {
	if m.disabled {
		return nil
	}

	expiryTime := time.Now().Add(expiration)
	reader := bytes.NewReader(content)
	if err := m.db.SetBlob(key, reader, int64(len(content)), expiryTime); err != nil {
		return fmt.Errorf("cache write error: %w", err)
	}

	return nil
}

// Delete removes a key from cache
func (m *Manager) Delete(key string) error {
	if m.disabled {
		return nil
	}

	return m.db.Delete(key)
}

// Stats returns cache statistics
func (m *Manager) Stats() Stats {
	if m.disabled || m.db == nil {
		return Stats{}
	}

	dbStats := m.db.Stats()
	return Stats{
		Keys:        dbStats.Keys,
		FileBytes:   dbStats.FileBytes,
		LiveRecords: int64(dbStats.LiveRecords),
		// Hits and Misses would be tracked separately
	}
}

// Compact triggers manual compaction to reclaim disk space
func (m *Manager) Compact() error {
	if m.disabled || m.db == nil {
		return nil
	}

	return m.db.Compact()
}

// Close closes the cache database
func (m *Manager) Close() error {
	if m.disabled || m.db == nil {
		return nil
	}

	return m.db.Close()
}
