package parameter

import (
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/types"
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
		param   *Parameter
		value   *types.Value
		name    string
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

// Regression: the parser emits DataType "list of <elem>" (parser_parameter.go),
// but validateDataType only knew bare "list", so typed list declarations
// (`accepts $items as list of strings`) failed at runtime with
// "unknown data type: list of strings".
func TestValidator_ValidateListOfTypes(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		param   *Parameter
		value   *types.Value
		name    string
		wantMsg string // substring expected in the error message when wantErr
		wantErr bool
	}{
		{
			name:    "list of strings accepts a list value",
			param:   &Parameter{Name: "items", DataType: "list of strings"},
			value:   mustNewValue(types.ListType, "alpha,beta"),
			wantErr: false,
		},
		{
			name:    "list of strings rejects a non-list value",
			param:   &Parameter{Name: "items", DataType: "list of strings"},
			value:   mustNewValue(types.StringType, "alpha"),
			wantErr: true,
			wantMsg: "must be a list",
		},
		{
			name:    "list of strings accepts digit strings as elements",
			param:   &Parameter{Name: "items", DataType: "list of strings"},
			value:   mustNewValue(types.ListType, "1,2,3"),
			wantErr: false,
		},
		{
			name:    "list of numbers accepts numeric elements",
			param:   &Parameter{Name: "ports", DataType: "list of numbers"},
			value:   mustNewValue(types.ListType, "80,443,8080"),
			wantErr: false,
		},
		{
			name:    "list of numbers rejects a non-numeric element",
			param:   &Parameter{Name: "ports", DataType: "list of numbers"},
			value:   mustNewValue(types.ListType, "80,https"),
			wantErr: true,
			wantMsg: "element 2 must be a number",
		},
		{
			name:    "list of booleans accepts boolean elements",
			param:   &Parameter{Name: "flags", DataType: "list of booleans"},
			value:   mustNewValue(types.ListType, "true,false"),
			wantErr: false,
		},
		{
			name:    "list of booleans rejects a non-boolean element",
			param:   &Parameter{Name: "flags", DataType: "list of booleans"},
			value:   mustNewValue(types.ListType, "true,maybe"),
			wantErr: true,
			wantMsg: "element 2 must be a boolean",
		},
		{
			name:    "unknown element type errors clearly",
			param:   &Parameter{Name: "items", DataType: "list of frobnicate"},
			value:   mustNewValue(types.ListType, "a,b"),
			wantErr: true,
			wantMsg: "unknown data type",
		},
		{
			name:    "plain list behavior is unchanged for a list value",
			param:   &Parameter{Name: "items", DataType: "list"},
			value:   mustNewValue(types.ListType, "a,b"),
			wantErr: false,
		},
		{
			name:    "plain list behavior is unchanged for a non-list value",
			param:   &Parameter{Name: "items", DataType: "list"},
			value:   mustNewValue(types.StringType, "a"),
			wantErr: true,
			wantMsg: "must be a list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.param, tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
					t.Errorf("Validate() error = %v, want message containing %q", err, tt.wantMsg)
				}
			}
		})
	}
}

func TestValidator_ValidateConstraints(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		param   *Parameter
		value   *types.Value
		name    string
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
		param   *Parameter
		value   *types.Value
		name    string
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
		param   *Parameter
		value   *types.Value
		name    string
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
		param   *Parameter
		value   *types.Value
		name    string
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
