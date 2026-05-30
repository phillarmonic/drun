package secrets

import (
	"testing"
)

func TestManagerSetGet(t *testing.T) {
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	namespace := "test-ns"
	key := "test_key"
	value := "test_value"

	// Set a secret
	err = mgr.Set(namespace, key, value)
	if err != nil {
		t.Fatalf("Failed to set secret: %v", err)
	}

	// Get the secret
	retrieved, err := mgr.Get(namespace, key)
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected %q, got %q", value, retrieved)
	}
}

func TestManagerNotFound(t *testing.T) {
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	_, err = mgr.Get("test-ns", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent secret")
	}
}

func TestNamespaceIsolation(t *testing.T) {
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	key := "shared_key"
	value1 := "value_ns1"
	value2 := "value_ns2"

	// Set same key in different namespaces
	_ = mgr.Set("ns1", key, value1)
	_ = mgr.Set("ns2", key, value2)

	// Retrieve from each namespace
	val1, err := mgr.Get("ns1", key)
	if err != nil {
		t.Fatalf("Failed to get from ns1: %v", err)
	}

	val2, err := mgr.Get("ns2", key)
	if err != nil {
		t.Fatalf("Failed to get from ns2: %v", err)
	}

	// Verify isolation
	if val1 != value1 {
		t.Errorf("ns1: expected %q, got %q", value1, val1)
	}
	if val2 != value2 {
		t.Errorf("ns2: expected %q, got %q", value2, val2)
	}
}

func TestManagerDelete(t *testing.T) {
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	namespace := "test-ns"
	key := "delete_me"

	// Set and verify
	_ = mgr.Set(namespace, key, "value")
	exists, _ := mgr.Exists(namespace, key)
	if !exists {
		t.Error("Secret should exist after set")
	}

	// Delete
	err = mgr.Delete(namespace, key)
	if err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}

	// Verify deletion
	exists, _ = mgr.Exists(namespace, key)
	if exists {
		t.Error("Secret should not exist after delete")
	}
}

func TestManagerExists(t *testing.T) {
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	namespace := "test-exists-ns"
	key := "unique_exists_key_123"

	// Clean up first in case of previous test runs
	_ = mgr.Delete(namespace, key)

	// Should not exist initially
	exists, _ := mgr.Exists(namespace, key)
	if exists {
		t.Error("Secret should not exist initially")
	}

	// Set and check
	_ = mgr.Set(namespace, key, "value")
	exists, _ = mgr.Exists(namespace, key)
	if !exists {
		t.Error("Secret should exist after set")
	}

	// Clean up
	_ = mgr.Delete(namespace, key)
}

func TestManagerList(t *testing.T) {
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	namespace := "test-list"

	// Set multiple secrets
	_ = mgr.Set(namespace, "key1", "value1")
	_ = mgr.Set(namespace, "key2", "value2")
	_ = mgr.Set(namespace, "key3", "value3")

	// List keys
	keys, err := mgr.List(namespace)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Verify keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	for _, expectedKey := range []string{"key1", "key2", "key3"} {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key %q not found in list", expectedKey)
		}
	}
}

func TestManagerListNamespaces(t *testing.T) {
	mgr, err := NewManager(WithFallback())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Set secrets in different namespaces
	_ = mgr.Set("ns1", "key1", "value1")
	_ = mgr.Set("ns2", "key2", "value2")
	_ = mgr.Set("ns3", "key3", "value3")

	// List namespaces
	namespaces, err := mgr.ListNamespaces()
	if err != nil {
		t.Fatalf("Failed to list namespaces: %v", err)
	}

	if len(namespaces) < 3 {
		t.Errorf("Expected at least 3 namespaces, got %d", len(namespaces))
	}

	// Verify namespaces are present
	nsMap := make(map[string]bool)
	for _, ns := range namespaces {
		nsMap[ns] = true
	}

	for _, expectedNs := range []string{"ns1", "ns2", "ns3"} {
		if !nsMap[expectedNs] {
			t.Errorf("Expected namespace %q not found", expectedNs)
		}
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"valid_key", true},
		{"validKey123", true},
		{"valid-key", true},
		{"a", true},
		{"", false},
		{"123invalid", false},
		{"invalid!key", false},
		{"invalid key", false},
		{"_invalid", false},
		{"-invalid", false},
	}

	for _, tt := range tests {
		err := validateKey(tt.key)
		isValid := err == nil

		if isValid != tt.valid {
			t.Errorf("validateKey(%q): expected valid=%v, got valid=%v", tt.key, tt.valid, isValid)
		}
	}
}

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		namespace string
		valid     bool
	}{
		{"valid-namespace", true},
		{"validNamespace123", true},
		{"valid_namespace", true},
		{"a", true},
		{"", false},
		{"123invalid", false},
		{"invalid!namespace", false},
		{"invalid namespace", false},
	}

	for _, tt := range tests {
		err := validateNamespace(tt.namespace)
		isValid := err == nil

		if isValid != tt.valid {
			t.Errorf("validateNamespace(%q): expected valid=%v, got valid=%v", tt.namespace, tt.valid, isValid)
		}
	}
}

func TestClearString(t *testing.T) {
	secret := "sensitive_data_12345"
	ClearString(&secret)

	if secret != "" {
		t.Errorf("Expected empty string after clear, got %q", secret)
	}

	// Test with nil
	var nilString *string
	ClearString(nilString) // Should not panic
}
