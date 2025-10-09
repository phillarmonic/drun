//go:build !darwin

package secrets

// NewKeychainBackend is not available on non-Darwin platforms
func NewKeychainBackend() (Backend, error) {
	return nil, ErrBackendNotAvail
}

