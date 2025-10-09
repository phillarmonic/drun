package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// Default iterations for PBKDF2
	pbkdf2Iterations = 100000
	// Salt size in bytes
	saltSize = 32
	// Key size for AES-256
	keySize = 32
)

// FallbackBackend provides encrypted file-based secret storage
type FallbackBackend struct {
	filepath string
	key      []byte
	secrets  map[string]string
	mu       sync.RWMutex
}

type encryptedData struct {
	Salt   []byte `json:"salt"`
	Nonce  []byte `json:"nonce"`
	Cipher []byte `json:"cipher"`
}

// NewFallbackBackend creates a new fallback backend with encrypted file storage
func NewFallbackBackend() Backend {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	secretsDir := filepath.Join(homeDir, ".drun")
	os.MkdirAll(secretsDir, 0700)

	storagePath := filepath.Join(secretsDir, "secrets.enc")

	return NewFallbackBackendWithPath(storagePath)
}

// NewFallbackBackendWithPath creates a new fallback backend with a custom storage path
func NewFallbackBackendWithPath(storagePath string) Backend {
	// Ensure directory exists
	dir := filepath.Dir(storagePath)
	os.MkdirAll(dir, 0700)

	// Generate or load encryption key
	key := deriveKey()

	backend := &FallbackBackend{
		filepath: storagePath,
		key:      key,
		secrets:  make(map[string]string),
	}

	// Try to load existing secrets
	backend.load()

	return backend
}

// Set stores a secret value
func (f *FallbackBackend) Set(key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.secrets[key] = value
	return f.save()
}

// Get retrieves a secret value
func (f *FallbackBackend) Get(key string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	value, ok := f.secrets[key]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

// Delete removes a secret
func (f *FallbackBackend) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.secrets, key)
	return f.save()
}

// Exists checks if a secret exists
func (f *FallbackBackend) Exists(key string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, ok := f.secrets[key]
	return ok, nil
}

// List returns all secret keys
func (f *FallbackBackend) List() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keys := make([]string, 0, len(f.secrets))
	for key := range f.secrets {
		keys = append(keys, key)
	}
	return keys, nil
}

// save encrypts and saves secrets to disk
func (f *FallbackBackend) save() error {
	data, err := json.Marshal(f.secrets)
	if err != nil {
		return err
	}

	encrypted, err := f.encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(f.filepath, encrypted, 0600)
}

// load decrypts and loads secrets from disk
func (f *FallbackBackend) load() error {
	data, err := os.ReadFile(f.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No secrets file yet, that's okay
		}
		return err
	}

	decrypted, err := f.decrypt(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(decrypted, &f.secrets)
}

// encrypt encrypts data using AES-256-GCM
func (f *FallbackBackend) encrypt(plaintext []byte) ([]byte, error) {
	// Generate salt for this encryption
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Derive key from password with salt
	key := pbkdf2.Key(f.key, salt, pbkdf2Iterations, keySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Package salt, nonce, and ciphertext together
	envelope := encryptedData{
		Salt:   salt,
		Nonce:  nonce,
		Cipher: ciphertext,
	}

	return json.Marshal(envelope)
}

// decrypt decrypts data using AES-256-GCM
func (f *FallbackBackend) decrypt(data []byte) ([]byte, error) {
	var envelope encryptedData
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}

	// Derive key from password with stored salt
	key := pbkdf2.Key(f.key, envelope.Salt, pbkdf2Iterations, keySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(envelope.Nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid nonce size")
	}

	plaintext, err := gcm.Open(nil, envelope.Nonce, envelope.Cipher, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// deriveKey creates a deterministic key from machine-specific data
func deriveKey() []byte {
	// In a real implementation, this would derive from:
	// - User's home directory path
	// - Machine ID
	// - Or prompt user for a password
	//
	// For now, we use a simple deterministic approach
	homeDir, _ := os.UserHomeDir()
	hostname, _ := os.Hostname()

	seed := homeDir + ":" + hostname + ":drun-secrets"
	return pbkdf2.Key([]byte(seed), []byte("drun-salt"), pbkdf2Iterations, keySize, sha256.New)
}

// SecureRandom generates cryptographically secure random bytes
func SecureRandom(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}
