package secrets

import (
	"runtime"
	"testing"
)

// TestPlatformDetection tests that the correct backend is selected for the platform
func TestPlatformDetection(t *testing.T) {
	backend, err := detectBackend()
	if err != nil && runtime.GOOS != "darwin" && runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		// On unsupported platforms, fallback should be used
		if backend == nil {
			t.Fatal("Expected fallback backend, got nil")
		}
		if _, ok := backend.(*FallbackBackend); !ok {
			t.Errorf("Expected FallbackBackend on %s, got %T", runtime.GOOS, backend)
		}
		return
	}

	if err != nil {
		t.Fatalf("Failed to create backend for %s: %v", runtime.GOOS, err)
	}

	if backend == nil {
		t.Fatal("Backend should not be nil")
	}

	// Just verify we got a backend - type checking is platform-specific
	t.Logf("Platform %s using backend: %T", runtime.GOOS, backend)
}

// TestManagerWithPlatformBackend tests the manager with the platform-specific backend
func TestManagerWithPlatformBackend(t *testing.T) {
	// For this test, we'll use the fallback backend to avoid keychain permission issues
	// Platform-specific backends are tested separately when running on actual systems
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager with fallback: %v", err)
	}

	// Test basic operations
	namespace := "test-platform-ns"
	key := "unique_platform_test_key"
	value := "test_value"

	// Clean up first
	_ = mgr.Delete(namespace, key)

	// Set
	err = mgr.Set(namespace, key, value)
	if err != nil {
		t.Fatalf("Failed to set secret: %v", err)
	}

	// Get
	retrieved, err := mgr.Get(namespace, key)
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}
	if retrieved != value {
		t.Errorf("Expected %q, got %q", value, retrieved)
	}

	// Cleanup
	_ = mgr.Delete(namespace, key)
}
