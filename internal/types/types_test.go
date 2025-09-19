package types

import (
	"testing"
)

func TestParseParameterType(t *testing.T) {
	tests := []struct {
		input    string
		expected ParameterType
		hasError bool
	}{
		{"string", StringType, false},
		{"number", NumberType, false},
		{"boolean", BooleanType, false},
		{"bool", BooleanType, false},
		{"list", ListType, false},
		{"STRING", StringType, false}, // case insensitive
		{"NUMBER", NumberType, false},
		{"invalid", StringType, true},
	}

	for _, test := range tests {
		result, err := ParseParameterType(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("Expected %v for input %q, got %v", test.expected, test.input, result)
			}
		}
	}
}

func TestNewValue_String(t *testing.T) {
	value, err := NewValue(StringType, "hello world")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if value.Type != StringType {
		t.Errorf("Expected StringType, got %v", value.Type)
	}

	if value.AsString() != "hello world" {
		t.Errorf("Expected 'hello world', got %q", value.AsString())
	}
}

func TestNewValue_Number(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		{"42", 42.0, false},
		{"3.14", 3.14, false},
		{"-10", -10.0, false},
		{"0", 0.0, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, test := range tests {
		value, err := NewValue(NumberType, test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", test.input, err)
			}

			num, err := value.AsNumber()
			if err != nil {
				t.Errorf("Error getting number value: %v", err)
			}

			if num != test.expected {
				t.Errorf("Expected %f for input %q, got %f", test.expected, test.input, num)
			}
		}
	}
}

func TestNewValue_Boolean(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		hasError bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"yes", true, false},
		{"no", false, false},
		{"1", true, false},
		{"0", false, false},
		{"on", true, false},
		{"off", false, false},
		{"enabled", true, false},
		{"disabled", false, false},
		{"", false, false},
		{"TRUE", true, false}, // case insensitive
		{"invalid", false, true},
	}

	for _, test := range tests {
		value, err := NewValue(BooleanType, test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", test.input, err)
			}

			b, err := value.AsBoolean()
			if err != nil {
				t.Errorf("Error getting boolean value: %v", err)
			}

			if b != test.expected {
				t.Errorf("Expected %t for input %q, got %t", test.expected, test.input, b)
			}
		}
	}
}

func TestNewValue_List(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"one, two, three", []string{"one", "two", "three"}},
		{"single", []string{"single"}},
		{"", []string{}},
		{"  a  ,  b  ,  c  ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}}, // empty items filtered out
	}

	for _, test := range tests {
		value, err := NewValue(ListType, test.input)
		if err != nil {
			t.Errorf("Unexpected error for input %q: %v", test.input, err)
			continue
		}

		list, err := value.AsList()
		if err != nil {
			t.Errorf("Error getting list value: %v", err)
			continue
		}

		if len(list) != len(test.expected) {
			t.Errorf("Expected %d items for input %q, got %d", len(test.expected), test.input, len(list))
			continue
		}

		for i, expected := range test.expected {
			if list[i] != expected {
				t.Errorf("Expected item %d to be %q for input %q, got %q", i, expected, test.input, list[i])
			}
		}
	}
}

func TestValue_TypeConversion(t *testing.T) {
	// Test string to number conversion
	stringValue, _ := NewValue(StringType, "42")
	num, err := stringValue.AsNumber()
	if err != nil {
		t.Errorf("Error converting string to number: %v", err)
	}
	if num != 42.0 {
		t.Errorf("Expected 42.0, got %f", num)
	}

	// Test number to string conversion
	numberValue, _ := NewValue(NumberType, "3.14")
	str := numberValue.AsString()
	if str != "3.14" {
		t.Errorf("Expected '3.14', got %q", str)
	}

	// Test boolean to string conversion
	boolValue, _ := NewValue(BooleanType, "true")
	str = boolValue.AsString()
	if str != "true" {
		t.Errorf("Expected 'true', got %q", str)
	}

	// Test list to string conversion
	listValue, _ := NewValue(ListType, "a,b,c")
	str = listValue.AsString()
	if str != "a,b,c" {
		t.Errorf("Expected 'a,b,c', got %q", str)
	}
}

func TestValue_ValidateConstraints(t *testing.T) {
	value, _ := NewValue(StringType, "dev")

	// Valid constraint
	err := value.ValidateConstraints([]string{"dev", "staging", "prod"})
	if err != nil {
		t.Errorf("Unexpected error for valid constraint: %v", err)
	}

	// Invalid constraint
	err = value.ValidateConstraints([]string{"staging", "prod"})
	if err == nil {
		t.Error("Expected error for invalid constraint, but got none")
	}

	// No constraints (should pass)
	err = value.ValidateConstraints([]string{})
	if err != nil {
		t.Errorf("Unexpected error for no constraints: %v", err)
	}
}

func TestInferType(t *testing.T) {
	tests := []struct {
		input    string
		expected ParameterType
	}{
		{"hello", StringType},
		{"42", NumberType},
		{"3.14", NumberType},
		{"true", BooleanType},
		{"false", BooleanType},
		{"yes", BooleanType},
		{"no", BooleanType},
		{"a,b,c", ListType},
		{"one, two, three", ListType},
		{"", StringType}, // empty defaults to string
	}

	for _, test := range tests {
		result := InferType(test.input)
		if result != test.expected {
			t.Errorf("Expected %v for input %q, got %v", test.expected, test.input, result)
		}
	}
}

func TestParameterType_String(t *testing.T) {
	tests := []struct {
		paramType ParameterType
		expected  string
	}{
		{StringType, "string"},
		{NumberType, "number"},
		{BooleanType, "boolean"},
		{ListType, "list"},
	}

	for _, test := range tests {
		result := test.paramType.String()
		if result != test.expected {
			t.Errorf("Expected %q for type %v, got %q", test.expected, test.paramType, result)
		}
	}
}
