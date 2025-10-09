package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFallbackBackend(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	
	backend := &FallbackBackend{
		filepath: filepath.Join(tmpDir, "test_secrets.enc"),
		key:      deriveKey(),
		secrets:  make(map[string]string),
	}

	// Test Set
	err := backend.Set("test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set secret: %v", err)
	}

	// Test Get
	value, err := backend.Get("test_key")
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %q", value)
	}

	// Test Exists
	exists, err := backend.Exists("test_key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Secret should exist")
	}

	// Test List
	backend.Set("key1", "value1")
	backend.Set("key2", "value2")
	
	keys, err := backend.List()
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Test Delete
	err = backend.Delete("test_key")
	if err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}

	exists, _ = backend.Exists("test_key")
	if exists {
		t.Error("Secret should not exist after deletion")
	}
}

func TestFallbackEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	
	backend := &FallbackBackend{
		filepath: filepath.Join(tmpDir, "test_secrets.enc"),
		key:      deriveKey(),
		secrets:  make(map[string]string),
	}

	sensitiveValue := "super_secret_password_123"
	backend.Set("password", sensitiveValue)

	// Read the file directly
	data, err := os.ReadFile(backend.filepath)
	if err != nil {
		t.Fatalf("Failed to read encrypted file: %v", err)
	}

	// The file should NOT contain the plaintext secret
	fileContent := string(data)
	if len(fileContent) > 0 && fileContent == sensitiveValue {
		t.Error("Plaintext secret found in encrypted file!")
	}
}

func TestFallbackPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	filepath := filepath.Join(tmpDir, "test_secrets.enc")
	key := deriveKey()

	// Create first backend and store secrets
	backend1 := &FallbackBackend{
		filepath: filepath,
		key:      key,
		secrets:  make(map[string]string),
	}
	backend1.Set("key1", "value1")
	backend1.Set("key2", "value2")

	// Create second backend with same filepath and key
	backend2 := &FallbackBackend{
		filepath: filepath,
		key:      key,
		secrets:  make(map[string]string),
	}
	backend2.load()

	// Verify secrets were persisted
	value1, err := backend2.Get("key1")
	if err != nil {
		t.Fatalf("Failed to get key1: %v", err)
	}
	if value1 != "value1" {
		t.Errorf("Expected 'value1', got %q", value1)
	}

	value2, err := backend2.Get("key2")
	if err != nil {
		t.Fatalf("Failed to get key2: %v", err)
	}
	if value2 != "value2" {
		t.Errorf("Expected 'value2', got %q", value2)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	backend := &FallbackBackend{
		key:     deriveKey(),
		secrets: make(map[string]string),
	}

	plaintext := []byte("sensitive data to encrypt")

	encrypted, err := backend.encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := backend.decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Expected %q, got %q", plaintext, decrypted)
	}
}

func TestSecureRandom(t *testing.T) {
	bytes, err := SecureRandom(32)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	if len(bytes) != 32 {
		t.Errorf("Expected 32 bytes, got %d", len(bytes))
	}

	// Generate another set and verify they're different
	bytes2, _ := SecureRandom(32)
	
	same := true
	for i := range bytes {
		if bytes[i] != bytes2[i] {
			same = false
			break
		}
	}
	
	if same {
		t.Error("Two random generations should not be identical")
	}
}

