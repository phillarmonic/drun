package engine

import (
	"bytes"
	"strings"
	"testing"
)

func TestSemanticVersionConditions(t *testing.T) {
	input := `version: 2.0

task "compare":
  requires $candidate
  requires $reference
  if $candidate is older than version "{$reference}":
    info "OLDER_TRUE"
  else:
    info "OLDER_FALSE"
  if $candidate is newer than version "{$reference}":
    info "NEWER_TRUE"
  else:
    info "NEWER_FALSE"`

	tests := []struct {
		name          string
		candidate     string
		reference     string
		expected      []string
		notExpected   []string
		expectedError string
	}{
		{name: "older", candidate: "1.0.4", reference: "1.1.0", expected: []string{"OLDER_TRUE", "NEWER_FALSE"}, notExpected: []string{"OLDER_FALSE", "NEWER_TRUE"}},
		{name: "equal", candidate: "1.0.4", reference: "1.0.4", expected: []string{"OLDER_FALSE", "NEWER_FALSE"}, notExpected: []string{"OLDER_TRUE", "NEWER_TRUE"}},
		{name: "newer", candidate: "2.0.0", reference: "1.9.9", expected: []string{"OLDER_FALSE", "NEWER_TRUE"}, notExpected: []string{"OLDER_TRUE", "NEWER_FALSE"}},
		{name: "invalid candidate", candidate: "v1.0.4", reference: "1.0.4", expectedError: `left side of semantic version comparison: "v1.0.4" is not a stable MAJOR.MINOR.PATCH version`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := ParseString(input)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			var output bytes.Buffer
			engine := NewEngine(&output)
			err = engine.ExecuteWithParams(program, "compare", map[string]string{
				"candidate": tt.candidate,
				"reference": tt.reference,
			})
			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Fatalf("error = %v, want containing %q", err, tt.expectedError)
				}
				return
			}
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			for _, expected := range tt.expected {
				if !strings.Contains(output.String(), expected) {
					t.Errorf("output %q does not contain %q", output.String(), expected)
				}
			}
			for _, unexpected := range tt.notExpected {
				if strings.Contains(output.String(), unexpected) {
					t.Errorf("output %q unexpectedly contains %q", output.String(), unexpected)
				}
			}
		})
	}
}

func TestSemanticVersionConditionAllowsDryRunQueryPlaceholder(t *testing.T) {
	engine := NewEngineWithOptions(WithDryRun(true))
	ctx := &ExecutionContext{Variables: map[string]string{
		"candidate": "1.0.4",
		"latest":    "[DRY RUN] latest version from app",
	}}

	result, handled, err := engine.evaluateSemanticVersionCondition(
		`$candidate is older than version "{$latest}"`, ctx,
	)
	if err != nil {
		t.Fatalf("dry-run placeholder returned an error: %v", err)
	}
	if !handled {
		t.Fatal("semantic version condition was not handled")
	}
	if result {
		t.Fatal("a dry-run placeholder must not satisfy version ordering")
	}
}
