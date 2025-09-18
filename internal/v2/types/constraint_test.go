package types

import (
	"testing"
)

func TestValue_ListConstraintValidation(t *testing.T) {
	// Test valid list items
	listValue, err := NewValue(ListType, "dev,staging")
	if err != nil {
		t.Fatalf("Failed to create list value: %v", err)
	}

	constraints := []string{"dev", "staging", "production"}
	err = listValue.ValidateConstraints(constraints)
	if err != nil {
		t.Errorf("Expected valid list constraint validation to pass, got error: %v", err)
	}

	// Test invalid list item
	invalidListValue, err := NewValue(ListType, "dev,invalid,staging")
	if err != nil {
		t.Fatalf("Failed to create invalid list value: %v", err)
	}

	err = invalidListValue.ValidateConstraints(constraints)
	if err == nil {
		t.Error("Expected invalid list constraint validation to fail, but it passed")
	}

	if !contains(err.Error(), "list item 'invalid' is not in allowed values") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestValue_StringConstraintValidation(t *testing.T) {
	// Test valid string
	stringValue, err := NewValue(StringType, "dev")
	if err != nil {
		t.Fatalf("Failed to create string value: %v", err)
	}

	constraints := []string{"dev", "staging", "production"}
	err = stringValue.ValidateConstraints(constraints)
	if err != nil {
		t.Errorf("Expected valid string constraint validation to pass, got error: %v", err)
	}

	// Test invalid string
	invalidStringValue, err := NewValue(StringType, "invalid")
	if err != nil {
		t.Fatalf("Failed to create invalid string value: %v", err)
	}

	err = invalidStringValue.ValidateConstraints(constraints)
	if err == nil {
		t.Error("Expected invalid string constraint validation to fail, but it passed")
	}

	if !contains(err.Error(), "value 'invalid' is not in allowed values") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
