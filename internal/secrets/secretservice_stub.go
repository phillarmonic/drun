//go:build !linux

package secrets

// NewSecretServiceBackend is not available on non-Linux platforms
func NewSecretServiceBackend() (Backend, error) {
	return nil, ErrBackendNotAvail
}
