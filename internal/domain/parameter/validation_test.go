package parameter

import (
	"testing"

	"github.com/phillarmonic/drun/internal/types"
)

// Helper function to create Value for tests
func mustNewValue(paramType types.ParameterType, raw string) *types.Value {
	v, err := types.NewValue(paramType, raw)
	if err != nil {
		panic(err)
	}
	return v
}

func TestValidator_ValidateDataType(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		param   *Parameter
		value   *types.Value
		wantErr bool
	}{
		{
			name:    "valid string",
			param:   &Parameter{Name: "test", DataType: "string"},
			value:   mustNewValue(types.StringType, "hello"),
			wantErr: false,
		},
		{
			name:    "valid number",
			param:   &Parameter{Name: "test", DataType: "number"},
			value:   mustNewValue(types.NumberType, "42"),
			wantErr: false,
		},
		{
			name:    "invalid number",
			param:   &Parameter{Name: "test", DataType: "number"},
			value:   mustNewValue(types.StringType, "not a number"),
			wantErr: true,
		},
		{
			name:    "valid boolean",
			param:   &Parameter{Name: "test", DataType: "boolean"},
			value:   mustNewValue(types.BooleanType, "true"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.param, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateConstraints(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		param   *Parameter
		value   *types.Value
		wantErr bool
	}{
		{
			name: "valid constraint",
			param: &Parameter{
				Name:        "env",
				Constraints: []string{"dev", "staging", "prod"},
			},
			value:   mustNewValue(types.StringType, "dev"),
			wantErr: false,
		},
		{
			name: "invalid constraint",
			param: &Parameter{
				Name:        "env",
				Constraints: []string{"dev", "staging", "prod"},
			},
			value:   mustNewValue(types.StringType, "invalid"),
			wantErr: true,
		},
		{
			name: "no constraints",
			param: &Parameter{
				Name: "test",
			},
			value:   mustNewValue(types.StringType, "anything"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.param, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateNumberRange(t *testing.T) {
	validator := NewValidator()

	minVal := 0.0
	maxVal := 100.0

	tests := []struct {
		name    string
		param   *Parameter
		value   *types.Value
		wantErr bool
	}{
		{
			name: "valid range",
			param: &Parameter{
				Name:     "port",
				MinValue: &minVal,
				MaxValue: &maxVal,
			},
			value:   mustNewValue(types.NumberType, "50"),
			wantErr: false,
		},
		{
			name: "below minimum",
			param: &Parameter{
				Name:     "port",
				MinValue: &minVal,
			},
			value:   mustNewValue(types.NumberType, "-1"),
			wantErr: true,
		},
		{
			name: "above maximum",
			param: &Parameter{
				Name:     "port",
				MaxValue: &maxVal,
			},
			value:   mustNewValue(types.NumberType, "101"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.param, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidatePattern(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		param   *Parameter
		value   *types.Value
		wantErr bool
	}{
		{
			name: "valid pattern",
			param: &Parameter{
				Name:    "code",
				Pattern: "^[A-Z]{3}$",
			},
			value:   mustNewValue(types.StringType, "ABC"),
			wantErr: false,
		},
		{
			name: "invalid pattern",
			param: &Parameter{
				Name:    "code",
				Pattern: "^[A-Z]{3}$",
			},
			value:   mustNewValue(types.StringType, "abc"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.param, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateEmail(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		param   *Parameter
		value   *types.Value
		wantErr bool
	}{
		{
			name: "valid email",
			param: &Parameter{
				Name:        "email",
				EmailFormat: true,
			},
			value:   mustNewValue(types.StringType, "test@example.com"),
			wantErr: false,
		},
		{
			name: "invalid email",
			param: &Parameter{
				Name:        "email",
				EmailFormat: true,
			},
			value:   mustNewValue(types.StringType, "not-an-email"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.param, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParameter_Methods(t *testing.T) {
	param := &Parameter{
		Name:     "test",
		Type:     "requires",
		Required: true,
		MinValue: new(float64),
	}

	if !param.IsRequired() {
		t.Error("IsRequired() should return true for required parameter")
	}

	if !param.HasConstraints() {
		t.Error("HasConstraints() should return true when MinValue is set")
	}

	param2 := &Parameter{
		Name: "optional",
		Type: "given",
	}

	if param2.IsRequired() {
		t.Error("IsRequired() should return false for optional parameter")
	}
}
