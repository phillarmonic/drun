package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ParameterType represents the type of a parameter
type ParameterType int

const (
	StringType ParameterType = iota
	NumberType
	BooleanType
	ListType
)

// String returns the string representation of the parameter type
func (pt ParameterType) String() string {
	switch pt {
	case StringType:
		return "string"
	case NumberType:
		return "number"
	case BooleanType:
		return "boolean"
	case ListType:
		return "list"
	default:
		return "unknown"
	}
}

// ParseParameterType parses a string into a ParameterType
func ParseParameterType(s string) (ParameterType, error) {
	switch strings.ToLower(s) {
	case "string":
		return StringType, nil
	case "number":
		return NumberType, nil
	case "boolean", "bool":
		return BooleanType, nil
	case "list":
		return ListType, nil
	default:
		return StringType, fmt.Errorf("unknown parameter type: %s", s)
	}
}

// Value represents a typed value
type Value struct {
	Type  ParameterType
	Raw   string // Original string value
	Value any    // Parsed typed value
}

// NewValue creates a new typed value
func NewValue(paramType ParameterType, raw string) (*Value, error) {
	v := &Value{
		Type: paramType,
		Raw:  raw,
	}

	var err error
	switch paramType {
	case StringType:
		v.Value = raw
	case NumberType:
		v.Value, err = parseNumber(raw)
	case BooleanType:
		v.Value, err = parseBoolean(raw)
	case ListType:
		v.Value, err = parseList(raw)
	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", paramType)
	}

	return v, err
}

// String returns the string representation of the value
func (v *Value) String() string {
	return v.Raw
}

// AsString returns the value as a string
func (v *Value) AsString() string {
	switch v.Type {
	case StringType:
		return v.Value.(string)
	case NumberType:
		if f, ok := v.Value.(float64); ok {
			// Check if it's a whole number
			if f == float64(int64(f)) {
				return fmt.Sprintf("%.0f", f)
			}
			return fmt.Sprintf("%g", f)
		}
		return fmt.Sprintf("%v", v.Value)
	case BooleanType:
		return fmt.Sprintf("%t", v.Value.(bool))
	case ListType:
		list := v.Value.([]string)
		return strings.Join(list, ",")
	default:
		return v.Raw
	}
}

// AsNumber returns the value as a number (float64)
func (v *Value) AsNumber() (float64, error) {
	switch v.Type {
	case NumberType:
		return v.Value.(float64), nil
	case StringType:
		return parseNumber(v.Value.(string))
	case BooleanType:
		if v.Value.(bool) {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, fmt.Errorf("cannot convert %s to number", v.Type)
	}
}

// AsBoolean returns the value as a boolean
func (v *Value) AsBoolean() (bool, error) {
	switch v.Type {
	case BooleanType:
		return v.Value.(bool), nil
	case StringType:
		return parseBoolean(v.Value.(string))
	case NumberType:
		num := v.Value.(float64)
		return num != 0, nil
	case ListType:
		list := v.Value.([]string)
		return len(list) > 0, nil
	default:
		return false, fmt.Errorf("cannot convert %s to boolean", v.Type)
	}
}

// AsList returns the value as a list of strings
func (v *Value) AsList() ([]string, error) {
	switch v.Type {
	case ListType:
		return v.Value.([]string), nil
	case StringType:
		return parseList(v.Value.(string))
	default:
		return []string{v.AsString()}, nil
	}
}

// parseNumber parses a string into a float64
func parseNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}

	// Try parsing as float
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", s)
	}

	return f, nil
}

// parseBoolean parses a string into a boolean
func parseBoolean(s string) (bool, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	switch s {
	case "true", "yes", "1", "on", "enabled":
		return true, nil
	case "false", "no", "0", "off", "disabled", "":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean: %s", s)
	}
}

// parseList parses a string into a list of strings
func parseList(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}, nil
	}

	// Split by comma and trim each item
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result, nil
}

// ValidateConstraints validates a value against constraints
func (v *Value) ValidateConstraints(constraints []string) error {
	if len(constraints) == 0 {
		return nil
	}

	// For list types, validate each item in the list
	if v.Type == ListType {
		list, err := v.AsList()
		if err != nil {
			return err
		}

		for _, item := range list {
			found := false
			for _, constraint := range constraints {
				if item == constraint {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("list item '%s' is not in allowed values: %v", item, constraints)
			}
		}
		return nil
	}

	// For non-list types, validate the string representation
	valueStr := v.AsString()

	for _, constraint := range constraints {
		if valueStr == constraint {
			return nil
		}
	}

	return fmt.Errorf("value '%s' is not in allowed values: %v", valueStr, constraints)
}

// InferType attempts to infer the type from a string value
func InferType(s string) ParameterType {
	s = strings.TrimSpace(s)

	// Empty string defaults to string type
	if s == "" {
		return StringType
	}

	// Check for list first (contains comma)
	if strings.Contains(s, ",") {
		return ListType
	}

	// Check for number
	if _, err := parseNumber(s); err == nil {
		return NumberType
	}

	// Check for boolean (but not empty string)
	if _, err := parseBoolean(s); err == nil {
		return BooleanType
	}

	// Default to string
	return StringType
}
