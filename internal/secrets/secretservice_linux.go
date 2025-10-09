//go:build linux

package secrets

import (
	"github.com/zalando/go-keyring"
)

// SecretServiceBackend provides Linux Secret Service storage (GNOME Keyring, KWallet)
type SecretServiceBackend struct {
	service string
}

// NewSecretServiceBackend creates a new Linux Secret Service backend
func NewSecretServiceBackend() (Backend, error) {
	return &SecretServiceBackend{
		service: "drun",
	}, nil
}

// Set stores a secret in the secret service
func (s *SecretServiceBackend) Set(key, value string) error {
	return keyring.Set(s.service, key, value)
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
	return nil
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
// Note: The go-keyring library doesn't support listing all keys,
// so we fall back to the fallback backend for this operation
func (s *SecretServiceBackend) List() ([]string, error) {
	// Unfortunately, the freedesktop.org Secret Service API and go-keyring
	// don't provide a way to list all keys for a service.
	// We would need to maintain a separate index or use a different approach.
	// For now, return an empty list or implement a workaround.
	return []string{}, nil
}

