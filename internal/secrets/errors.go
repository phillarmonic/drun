package secrets

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrSecretNotFound   = errors.New("secret not found")
	ErrSecretExists     = errors.New("secret already exists")
	ErrInvalidKey       = errors.New("invalid secret key")
	ErrBackendNotAvail  = errors.New("secrets backend not available")
	ErrPermissionDenied = errors.New("permission denied")
	ErrNamespaceInvalid = errors.New("invalid namespace")
)

// SecretError wraps an error with additional context
type SecretError struct {
	Namespace string
	Key       string
	Op        string
	Err       error
}

func (e *SecretError) Error() string {
	if e.Namespace != "" && e.Key != "" {
		return fmt.Sprintf("secret operation '%s' failed for %s:%s: %v",
			e.Op, e.Namespace, e.Key, e.Err)
	} else if e.Namespace != "" {
		return fmt.Sprintf("secret operation '%s' failed for namespace %s: %v",
			e.Op, e.Namespace, e.Err)
	}
	return fmt.Sprintf("secret operation '%s' failed: %v", e.Op, e.Err)
}

func (e *SecretError) Unwrap() error {
	return e.Err
}

// NewSecretError creates a new SecretError
func NewSecretError(op, namespace, key string, err error) *SecretError {
	return &SecretError{
		Namespace: namespace,
		Key:       key,
		Op:        op,
		Err:       err,
	}
}

