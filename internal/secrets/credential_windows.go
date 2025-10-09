//go:build windows

package secrets

import (
	"strings"

	"github.com/danieljoos/wincred"
)

// CredentialBackend provides Windows Credential Manager storage
type CredentialBackend struct {
	prefix string
}

// NewCredentialBackend creates a new Windows Credential Manager backend
func NewCredentialBackend() (Backend, error) {
	return &CredentialBackend{
		prefix: "drun:",
	}, nil
}

// Set stores a secret in Credential Manager
func (c *CredentialBackend) Set(key, value string) error {
	cred := wincred.NewGenericCredential(c.prefix + key)
	cred.CredentialBlob = []byte(value)
	cred.Persist = wincred.PersistLocalMachine

	return cred.Write()
}

// Get retrieves a secret from Credential Manager
func (c *CredentialBackend) Get(key string) (string, error) {
	cred, err := wincred.GetGenericCredential(c.prefix + key)
	if err != nil {
		if err == wincred.ErrElementNotFound {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	return string(cred.CredentialBlob), nil
}

// Delete removes a secret from Credential Manager
func (c *CredentialBackend) Delete(key string) error {
	cred, err := wincred.GetGenericCredential(c.prefix + key)
	if err != nil {
		if err == wincred.ErrElementNotFound {
			return nil // Already deleted
		}
		return err
	}

	return cred.Delete()
}

// Exists checks if a secret exists in Credential Manager
func (c *CredentialBackend) Exists(key string) (bool, error) {
	_, err := wincred.GetGenericCredential(c.prefix + key)
	if err != nil {
		if err == wincred.ErrElementNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns all secret keys
func (c *CredentialBackend) List() ([]string, error) {
	creds, err := wincred.List()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0)
	for _, cred := range creds {
		if strings.HasPrefix(cred.TargetName, c.prefix) {
			key := strings.TrimPrefix(cred.TargetName, c.prefix)
			keys = append(keys, key)
		}
	}

	return keys, nil
}

