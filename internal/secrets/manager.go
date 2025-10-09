package secrets

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
)

// Manager provides secure secret storage and retrieval
type Manager interface {
	// Set stores a secret value
	Set(namespace, key, value string) error

	// Get retrieves a secret value
	Get(namespace, key string) (string, error)

	// Delete removes a secret
	Delete(namespace, key string) error

	// Exists checks if a secret exists
	Exists(namespace, key string) (bool, error)

	// List returns all secret keys (not values) in namespace
	List(namespace string) ([]string, error)

	// ListNamespaces returns all available namespaces
	ListNamespaces() ([]string, error)
}

// Backend is the platform-specific storage implementation
type Backend interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Exists(key string) (bool, error)
	List() ([]string, error)
}

// DefaultManager implements Manager using platform-specific backends
type DefaultManager struct {
	backend   Backend
	separator string // Separator for namespace:key format
}

// Logger interface for optional logging
type Logger interface {
	Info(msg string)
	Error(msg string)
}

// Option is a functional option for configuring the manager
type Option func(*DefaultManager)

var (
	// Valid key pattern: must start with letter, contain only alphanumeric, underscore, or dash
	validKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
)

// NewManager creates a new secrets manager with appropriate backend
func NewManager(opts ...Option) (Manager, error) {
	backend, err := detectBackend()
	if err != nil {
		return nil, err
	}

	mgr := &DefaultManager{
		backend:   backend,
		separator: ":",
	}

	for _, opt := range opts {
		opt(mgr)
	}

	return mgr, nil
}

// WithFallback forces fallback backend (for testing)
func WithFallback() Option {
	return func(m *DefaultManager) {
		m.backend = NewFallbackBackend()
	}
}

// detectBackend chooses the appropriate backend for the platform
func detectBackend() (Backend, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewKeychainBackend()
	case "windows":
		return NewCredentialBackend()
	case "linux":
		return NewSecretServiceBackend()
	default:
		return NewFallbackBackend(), nil
	}
}

// Set stores a secret value
func (m *DefaultManager) Set(namespace, key, value string) error {
	if err := validateNamespace(namespace); err != nil {
		return NewSecretError("set", namespace, key, err)
	}
	if err := validateKey(key); err != nil {
		return NewSecretError("set", namespace, key, err)
	}

	compositeKey := m.formatKey(namespace, key)
	if err := m.backend.Set(compositeKey, value); err != nil {
		return NewSecretError("set", namespace, key, err)
	}

	return nil
}

// Get retrieves a secret value
func (m *DefaultManager) Get(namespace, key string) (string, error) {
	if err := validateNamespace(namespace); err != nil {
		return "", NewSecretError("get", namespace, key, err)
	}
	if err := validateKey(key); err != nil {
		return "", NewSecretError("get", namespace, key, err)
	}

	compositeKey := m.formatKey(namespace, key)
	value, err := m.backend.Get(compositeKey)
	if err != nil {
		if err == ErrSecretNotFound {
			return "", NewSecretError("get", namespace, key, ErrSecretNotFound)
		}
		return "", NewSecretError("get", namespace, key, err)
	}

	return value, nil
}

// Delete removes a secret
func (m *DefaultManager) Delete(namespace, key string) error {
	if err := validateNamespace(namespace); err != nil {
		return NewSecretError("delete", namespace, key, err)
	}
	if err := validateKey(key); err != nil {
		return NewSecretError("delete", namespace, key, err)
	}

	compositeKey := m.formatKey(namespace, key)
	if err := m.backend.Delete(compositeKey); err != nil {
		return NewSecretError("delete", namespace, key, err)
	}

	return nil
}

// Exists checks if a secret exists
func (m *DefaultManager) Exists(namespace, key string) (bool, error) {
	if err := validateNamespace(namespace); err != nil {
		return false, NewSecretError("exists", namespace, key, err)
	}
	if err := validateKey(key); err != nil {
		return false, NewSecretError("exists", namespace, key, err)
	}

	compositeKey := m.formatKey(namespace, key)
	exists, err := m.backend.Exists(compositeKey)
	if err != nil {
		return false, NewSecretError("exists", namespace, key, err)
	}

	return exists, nil
}

// List returns all secret keys (not values) in namespace
func (m *DefaultManager) List(namespace string) ([]string, error) {
	if err := validateNamespace(namespace); err != nil {
		return nil, NewSecretError("list", namespace, "", err)
	}

	allKeys, err := m.backend.List()
	if err != nil {
		return nil, NewSecretError("list", namespace, "", err)
	}

	prefix := namespace + m.separator
	var keys []string
	for _, fullKey := range allKeys {
		if strings.HasPrefix(fullKey, prefix) {
			// Extract just the key part (after namespace:)
			key := strings.TrimPrefix(fullKey, prefix)
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// ListNamespaces returns all available namespaces
func (m *DefaultManager) ListNamespaces() ([]string, error) {
	allKeys, err := m.backend.List()
	if err != nil {
		return nil, NewSecretError("list-namespaces", "", "", err)
	}

	namespaceSet := make(map[string]bool)
	for _, fullKey := range allKeys {
		parts := strings.SplitN(fullKey, m.separator, 2)
		if len(parts) == 2 {
			namespaceSet[parts[0]] = true
		}
	}

	namespaces := make([]string, 0, len(namespaceSet))
	for ns := range namespaceSet {
		namespaces = append(namespaces, ns)
	}

	return namespaces, nil
}

// formatKey creates the composite key in format "namespace:key"
func (m *DefaultManager) formatKey(namespace, key string) string {
	return fmt.Sprintf("%s%s%s", namespace, m.separator, key)
}

// validateNamespace validates that a namespace is valid
func validateNamespace(namespace string) error {
	if namespace == "" {
		return ErrNamespaceInvalid
	}
	if !validKeyPattern.MatchString(namespace) {
		return ErrNamespaceInvalid
	}
	return nil
}

// validateKey validates that a key is valid
func validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	if !validKeyPattern.MatchString(key) {
		return ErrInvalidKey
	}
	return nil
}

// ClearString clears a string from memory (best effort)
func ClearString(s *string) {
	if s == nil {
		return
	}
	b := []byte(*s)
	for i := range b {
		b[i] = 0
	}
	*s = ""
}

