package types

import (
	"testing"
)

func TestValue_ValidateAdvancedConstraints_RangeValidation(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		valueType ParameterType
		minValue  *float64
		maxValue  *float64
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid number in range",
			value:     "5000",
			valueType: NumberType,
			minValue:  floatPtr(1000),
			maxValue:  floatPtr(9999),
			wantError: false,
		},
		{
			name:      "number below minimum",
			value:     "500",
			valueType: NumberType,
			minValue:  floatPtr(1000),
			maxValue:  floatPtr(9999),
			wantError: true,
			errorMsg:  "less than minimum",
		},
		{
			name:      "number above maximum",
			value:     "15000",
			valueType: NumberType,
			minValue:  floatPtr(1000),
			maxValue:  floatPtr(9999),
			wantError: true,
			errorMsg:  "greater than maximum",
		},
		{
			name:      "range constraint on non-number type",
			value:     "hello",
			valueType: StringType,
			minValue:  floatPtr(1000),
			maxValue:  floatPtr(9999),
			wantError: true,
			errorMsg:  "range constraints can only be applied to number types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewValue(tt.valueType, tt.value)
			if err != nil {
				t.Fatalf("Failed to create value: %v", err)
			}

			err = v.ValidateAdvancedConstraints(tt.minValue, tt.maxValue, "", "", false)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValue_ValidateAdvancedConstraints_PatternValidation(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		valueType ParameterType
		pattern   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid version pattern",
			value:     "v1.2.3",
			valueType: StringType,
			pattern:   `v\d+\.\d+\.\d+`,
			wantError: false,
		},
		{
			name:      "invalid version pattern",
			value:     "1.2.3",
			valueType: StringType,
			pattern:   `v\d+\.\d+\.\d+`,
			wantError: true,
			errorMsg:  "does not match pattern",
		},
		{
			name:      "pattern constraint on non-string type",
			value:     "123",
			valueType: NumberType,
			pattern:   `\d+`,
			wantError: true,
			errorMsg:  "pattern constraints can only be applied to string types",
		},
		{
			name:      "invalid regex pattern",
			value:     "test",
			valueType: StringType,
			pattern:   `[`,
			wantError: true,
			errorMsg:  "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewValue(tt.valueType, tt.value)
			if err != nil {
				t.Fatalf("Failed to create value: %v", err)
			}

			err = v.ValidateAdvancedConstraints(nil, nil, tt.pattern, "", false)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValue_ValidateAdvancedConstraints_EmailValidation(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		valueType   ParameterType
		emailFormat bool
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "valid email",
			value:       "test@example.com",
			valueType:   StringType,
			emailFormat: true,
			wantError:   false,
		},
		{
			name:        "valid email with subdomain",
			value:       "user@mail.example.com",
			valueType:   StringType,
			emailFormat: true,
			wantError:   false,
		},
		{
			name:        "invalid email - no @",
			value:       "invalid-email",
			valueType:   StringType,
			emailFormat: true,
			wantError:   true,
			errorMsg:    "not a valid email address",
		},
		{
			name:        "invalid email - no domain",
			value:       "test@",
			valueType:   StringType,
			emailFormat: true,
			wantError:   true,
			errorMsg:    "not a valid email address",
		},
		{
			name:        "email validation on non-string type",
			value:       "123",
			valueType:   NumberType,
			emailFormat: true,
			wantError:   true,
			errorMsg:    "email format validation can only be applied to string types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewValue(tt.valueType, tt.value)
			if err != nil {
				t.Fatalf("Failed to create value: %v", err)
			}

			err = v.ValidateAdvancedConstraints(nil, nil, "", "", tt.emailFormat)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"user@subdomain.example.com", true},
		{"invalid-email", false},
		{"@example.com", false},
		{"test@", false},
		{"test@.com", false},
		{"test@example", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.valid {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, result, tt.valid)
			}
		})
	}
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
