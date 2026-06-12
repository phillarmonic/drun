package engine

import (
	"io"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/detection"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

func TestEngine_checkToolRequirements(t *testing.T) {
	// Simple test using a known tool that will either exist or not,
	// but we can mock the detector if we wanted. Since we can't easily mock the
	// internal detector in the engine checkToolRequirements without interface injection,
	// we'll test the formatting and logic for known behaviors.

	e := NewEngine(io.Discard)

	detector := detection.NewDetector()

	tests := []struct {
		name        string
		tools       []statement.ToolRequirement
		expectError bool
		errorMsg    string
	}{
		{
			name: "Missing tool",
			tools: []statement.ToolRequirement{
				{Name: "this-tool-definitely-does-not-exist-12345"},
			},
			expectError: true,
			errorMsg:    "required tool 'this-tool-definitely-does-not-exist-12345' is not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.checkToolRequirements(detector, tt.tools)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestFormatConstraints(t *testing.T) {
	constraints := []statement.VersionConstraint{
		{Operator: ">=", Version: "1.0"},
		{Operator: "<", Version: "2.0"},
	}
	expected := ">= 1.0, < 2.0"
	actual := formatConstraints(constraints)
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}
