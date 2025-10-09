//go:build darwin

package secrets

import (
	"github.com/keybase/go-keychain"
)

// KeychainBackend provides macOS Keychain storage
type KeychainBackend struct {
	service string
}

// NewKeychainBackend creates a new macOS Keychain backend
func NewKeychainBackend() (Backend, error) {
	return &KeychainBackend{
		service: "com.phillarmonic.drun",
	}, nil
}

// Set stores a secret in the keychain
func (k *KeychainBackend) Set(key, value string) error {
	// First try to delete any existing item
	k.Delete(key)

	item := keychain.NewItem()
	item.SetService(k.service)
	item.SetAccount(key)
	item.SetData([]byte(value))
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	return keychain.AddItem(item)
}

// Get retrieves a secret from the keychain
func (k *KeychainBackend) Get(key string) (string, error) {
	query := keychain.NewItem()
	query.SetService(k.service)
	query.SetAccount(key)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		if err == keychain.ErrorItemNotFound {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	if len(results) == 0 {
		return "", ErrSecretNotFound
	}

	return string(results[0].Data), nil
}

// Delete removes a secret from the keychain
func (k *KeychainBackend) Delete(key string) error {
	item := keychain.NewItem()
	item.SetService(k.service)
	item.SetAccount(key)

	err := keychain.DeleteItem(item)
	if err != nil && err != keychain.ErrorItemNotFound {
		return err
	}
	return nil
}

// Exists checks if a secret exists in the keychain
func (k *KeychainBackend) Exists(key string) (bool, error) {
	query := keychain.NewItem()
	query.SetService(k.service)
	query.SetAccount(key)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(false)

	results, err := keychain.QueryItem(query)
	if err != nil {
		if err == keychain.ErrorItemNotFound {
			return false, nil
		}
		return false, err
	}

	return len(results) > 0, nil
}

// List returns all secret keys
func (k *KeychainBackend) List() ([]string, error) {
	query := keychain.NewItem()
	query.SetService(k.service)
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		if err == keychain.ErrorItemNotFound {
			return []string{}, nil
		}
		return nil, err
	}

	keys := make([]string, 0, len(results))
	for _, item := range results {
		keys = append(keys, item.Account)
	}

	return keys, nil
}

