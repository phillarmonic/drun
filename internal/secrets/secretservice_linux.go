//go:build linux

package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/zalando/go-keyring"
)

// SecretServiceBackend provides Linux Secret Service storage (GNOME Keyring, KWallet)
// with an index file for listing keys
type SecretServiceBackend struct {
	service   string
	indexPath string
	mu        sync.RWMutex
}

// NewSecretServiceBackend creates a new Linux Secret Service backend
func NewSecretServiceBackend() (Backend, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	secretsDir := filepath.Join(homeDir, ".drun")
	_ = os.MkdirAll(secretsDir, 0700)

	indexPath := filepath.Join(secretsDir, "secrets-index.json")

	return &SecretServiceBackend{
		service:   "drun",
		indexPath: indexPath,
	}, nil
}

// Set stores a secret in the secret service
func (s *SecretServiceBackend) Set(key, value string) error {
	if err := keyring.Set(s.service, key, value); err != nil {
		return err
	}
	// Update index
	return s.addToIndex(key)
}

// Get retrieves a secret from the secret service
func (s *SecretServiceBackend) Get(key string) (string, error) {
	value, err := keyring.Get(s.service, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", ErrSecretNotFound
		}
		return "", err
	}
	return value, nil
}

// Delete removes a secret from the secret service
func (s *SecretServiceBackend) Delete(key string) error {
	err := keyring.Delete(s.service, key)
	if err != nil && err != keyring.ErrNotFound {
		return err
	}
	// Update index
	return s.removeFromIndex(key)
}

// Exists checks if a secret exists in the secret service
func (s *SecretServiceBackend) Exists(key string) (bool, error) {
	_, err := keyring.Get(s.service, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns all secret keys
func (s *SecretServiceBackend) List() ([]string, error) {
	return s.loadIndex()
}

// addToIndex adds a key to the index file
func (s *SecretServiceBackend) addToIndex(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys, err := s.loadIndexUnsafe()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if key already exists
	for _, k := range keys {
		if k == key {
			return nil // Already in index
		}
	}

	keys = append(keys, key)
	return s.saveIndexUnsafe(keys)
}

// removeFromIndex removes a key from the index file
func (s *SecretServiceBackend) removeFromIndex(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys, err := s.loadIndexUnsafe()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Filter out the key
	newKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != key {
			newKeys = append(newKeys, k)
		}
	}

	return s.saveIndexUnsafe(newKeys)
}

// loadIndex loads the index file (thread-safe)
func (s *SecretServiceBackend) loadIndex() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadIndexUnsafe()
}

// loadIndexUnsafe loads the index file (not thread-safe, caller must lock)
func (s *SecretServiceBackend) loadIndexUnsafe() ([]string, error) {
	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// saveIndexUnsafe saves the index file (not thread-safe, caller must lock)
func (s *SecretServiceBackend) saveIndexUnsafe(keys []string) error {
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	return os.WriteFile(s.indexPath, data, 0600)
}
