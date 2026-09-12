package parameter

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/patterns"
	"github.com/phillarmonic/drun/v2/internal/types"
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

	// The parser emits typed lists as "list of <elem>" (parser_parameter.go).
	// Require a list value, then validate each element against the element type.
	if after, ok := strings.CutPrefix(param.DataType, "list of "); ok {
		return v.validateTypedList(param, value, after)
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
			if _, err := types.NewValue(types.BooleanType, value.String()); err != nil {
				return &ValidationError{
					Parameter: param.Name,
					Message:   "must be a valid boolean value",
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
			Message:   "unknown data type: " + param.DataType,
		}
	}

	return nil
}

// validateTypedList validates a "list of <elem>" parameter: the value must be a
// list, and each element must be usable as the declared element type. Unknown
// element types error clearly rather than silently accepting anything.
func (v *Validator) validateTypedList(param *Parameter, value *types.Value, elemType string) error {
	if value.Type != types.ListType {
		return &ValidationError{
			Parameter: param.Name,
			Message:   "must be a list",
			Value:     value.String(),
		}
	}

	parsedElem, err := types.ParseParameterType(normalizeElementType(elemType))
	if err != nil {
		return &ValidationError{
			Parameter: param.Name,
			Message:   "unknown data type: " + param.DataType,
		}
	}

	elements, err := value.AsList()
	if err != nil {
		return &ValidationError{
			Parameter: param.Name,
			Message:   "must be a list",
			Value:     value.String(),
		}
	}

	for i, element := range elements {
		if _, err := types.NewValue(parsedElem, element); err != nil {
			return &ValidationError{
				Parameter: param.Name,
				Message:   fmt.Sprintf("element %d must be a %s", i+1, elemType),
				Value:     element,
			}
		}
	}

	return nil
}

// normalizeElementType maps the plural type names used after "list of"
// ("strings", "numbers", "booleans") to the singulars ParseParameterType
// recognizes. Unknown names pass through unchanged so they can fail with a
// clear "unknown data type" error.
func normalizeElementType(name string) string {
	switch name {
	case "strings":
		return "string"
	case "numbers":
		return "number"
	case "booleans":
		return "boolean"
	default:
		return name
	}
}

// validateConstraints validates parameter constraints
func (v *Validator) validateConstraints(param *Parameter, value *types.Value) error {
	if len(param.Constraints) == 0 {
		return nil
	}

	valueStr := value.String()
	if slices.Contains(param.Constraints, valueStr) {
		return nil // Value matches constraint
	}

	return &ValidationError{
		Parameter: param.Name,
		Message:   "must be one of: " + strings.Join(param.Constraints, ", "),
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
			Message:   "must match pattern: " + param.Pattern,
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
