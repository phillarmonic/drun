//go:build !windows

package secrets

// NewCredentialBackend is not available on non-Windows platforms
func NewCredentialBackend() (Backend, error) {
	return nil, ErrBackendNotAvail
}
