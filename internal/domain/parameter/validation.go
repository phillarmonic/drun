package parameter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/internal/patterns"
	"github.com/phillarmonic/drun/internal/types"
)

// Validator validates parameters
type Validator struct {
	// No state needed - patterns package provides functions
}

// NewValidator creates a new parameter validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a parameter value
func (v *Validator) Validate(param *Parameter, value *types.Value) error {
	// Check data type
	if err := v.validateDataType(param, value); err != nil {
		return err
	}

	// Check constraints
	if err := v.validateConstraints(param, value); err != nil {
		return err
	}

	// Check advanced constraints
	if err := v.validateAdvancedConstraints(param, value); err != nil {
		return err
	}

	return nil
}

// validateDataType validates the parameter data type
func (v *Validator) validateDataType(param *Parameter, value *types.Value) error {
	if param.DataType == "" || param.DataType == "string" {
		return nil // Strings are always valid
	}

	switch param.DataType {
	case "number":
		if value.Type != types.NumberType {
			_, err := strconv.ParseFloat(value.String(), 64)
			if err != nil {
				return &ValidationError{
					Parameter: param.Name,
					Message:   "must be a number",
					Value:     value.String(),
				}
			}
		}

	case "boolean":
		if value.Type != types.BooleanType {
			val := strings.ToLower(value.String())
			if val != "true" && val != "false" && val != "yes" && val != "no" {
				return &ValidationError{
					Parameter: param.Name,
					Message:   "must be a boolean (true/false, yes/no)",
					Value:     value.String(),
				}
			}
		}

	case "list":
		if value.Type != types.ListType {
			return &ValidationError{
				Parameter: param.Name,
				Message:   "must be a list",
				Value:     value.String(),
			}
		}

	default:
		return &ValidationError{
			Parameter: param.Name,
			Message:   fmt.Sprintf("unknown data type: %s", param.DataType),
		}
	}

	return nil
}

// validateConstraints validates parameter constraints
func (v *Validator) validateConstraints(param *Parameter, value *types.Value) error {
	if len(param.Constraints) == 0 {
		return nil
	}

	valueStr := value.String()
	for _, constraint := range param.Constraints {
		if valueStr == constraint {
			return nil // Value matches constraint
		}
	}

	return &ValidationError{
		Parameter: param.Name,
		Message:   fmt.Sprintf("must be one of: %s", strings.Join(param.Constraints, ", ")),
		Value:     valueStr,
	}
}

// validateAdvancedConstraints validates advanced constraints
func (v *Validator) validateAdvancedConstraints(param *Parameter, value *types.Value) error {
	// Validate number range
	if param.MinValue != nil || param.MaxValue != nil {
		if err := v.validateNumberRange(param, value); err != nil {
			return err
		}
	}

	// Validate pattern
	if param.Pattern != "" {
		if err := v.validatePattern(param, value); err != nil {
			return err
		}
	}

	// Validate pattern macro
	if param.PatternMacro != "" {
		if err := v.validatePatternMacro(param, value); err != nil {
			return err
		}
	}

	// Validate email format
	if param.EmailFormat {
		if err := v.validateEmail(param, value); err != nil {
			return err
		}
	}

	return nil
}

// validateNumberRange validates number is within range
func (v *Validator) validateNumberRange(param *Parameter, value *types.Value) error {
	numValue, err := strconv.ParseFloat(value.String(), 64)
	if err != nil {
		return &ValidationError{
			Parameter: param.Name,
			Message:   "must be a number for range validation",
			Value:     value.String(),
		}
	}

	if param.MinValue != nil && numValue < *param.MinValue {
		return &ValidationError{
			Parameter: param.Name,
			Message:   fmt.Sprintf("must be >= %.2f", *param.MinValue),
			Value:     value.String(),
		}
	}

	if param.MaxValue != nil && numValue > *param.MaxValue {
		return &ValidationError{
			Parameter: param.Name,
			Message:   fmt.Sprintf("must be <= %.2f", *param.MaxValue),
			Value:     value.String(),
		}
	}

	return nil
}

// validatePattern validates against regex pattern
func (v *Validator) validatePattern(param *Parameter, value *types.Value) error {
	matched, err := regexp.MatchString(param.Pattern, value.String())
	if err != nil {
		return &ValidationError{
			Parameter: param.Name,
			Message:   fmt.Sprintf("invalid pattern: %v", err),
		}
	}

	if !matched {
		return &ValidationError{
			Parameter: param.Name,
			Message:   fmt.Sprintf("must match pattern: %s", param.Pattern),
			Value:     value.String(),
		}
	}

	return nil
}

// validatePatternMacro validates against pattern macro
func (v *Validator) validatePatternMacro(param *Parameter, value *types.Value) error {
	err := patterns.ValidatePattern(value.String(), param.PatternMacro)
	if err != nil {
		return &ValidationError{
			Parameter: param.Name,
			Message:   fmt.Sprintf("must match %s format", param.PatternMacro),
			Value:     value.String(),
		}
	}

	return nil
}

// validateEmail validates email format
func (v *Validator) validateEmail(param *Parameter, value *types.Value) error {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(emailRegex, value.String())
	if err != nil || !matched {
		return &ValidationError{
			Parameter: param.Name,
			Message:   "must be a valid email address",
			Value:     value.String(),
		}
	}

	return nil
}

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string
	Message   string
	Value     string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("parameter '%s' validation failed: %s (value: '%s')", e.Parameter, e.Message, e.Value)
}
