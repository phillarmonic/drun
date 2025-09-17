package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStep_UnmarshalYAML_SingleString(t *testing.T) {
	yamlContent := `echo "hello world"`

	var step Step
	err := yaml.Unmarshal([]byte(yamlContent), &step)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(step.Lines) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(step.Lines))
	}

	if step.Lines[0] != `echo "hello world"` {
		t.Errorf("Expected 'echo \"hello world\"', got %q", step.Lines[0])
	}
}

func TestStep_UnmarshalYAML_MultipleStrings(t *testing.T) {
	yamlContent := `
- echo "line 1"
- echo "line 2"
- echo "line 3"
`

	var step Step
	err := yaml.Unmarshal([]byte(yamlContent), &step)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(step.Lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(step.Lines))
	}

	expected := []string{`echo "line 1"`, `echo "line 2"`, `echo "line 3"`}
	for i, line := range step.Lines {
		if line != expected[i] {
			t.Errorf("Expected %q, got %q", expected[i], line)
		}
	}
}

func TestStep_UnmarshalYAML_InvalidType(t *testing.T) {
	yamlContent := `{key: value}` // object instead of string/array

	var step Step
	err := yaml.Unmarshal([]byte(yamlContent), &step)

	if err == nil {
		t.Fatal("Expected error for invalid type, got nil")
	}
}

func TestStep_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		step     Step
		expected bool
	}{
		{
			name:     "empty step",
			step:     Step{Lines: []string{}},
			expected: true,
		},
		{
			name:     "nil lines",
			step:     Step{Lines: nil},
			expected: true,
		},
		{
			name:     "non-empty step",
			step:     Step{Lines: []string{"echo hello"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.step.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStep_String(t *testing.T) {
	tests := []struct {
		name     string
		step     Step
		expected string
	}{
		{
			name:     "empty step",
			step:     Step{Lines: []string{}},
			expected: "",
		},
		{
			name:     "single line",
			step:     Step{Lines: []string{"echo hello"}},
			expected: "echo hello",
		},
		{
			name:     "multiple lines",
			step:     Step{Lines: []string{"echo line1", "echo line2", "echo line3"}},
			expected: "echo line1\necho line2\necho line3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.step.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
