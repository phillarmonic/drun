package model

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements custom YAML unmarshaling for Step
// Handles both string and []string formats
func (s *Step) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Single string - split by newlines to create multiple lines
		s.Lines = strings.Split(strings.TrimSpace(node.Value), "\n")
		// Remove empty lines
		var filtered []string
		for _, line := range s.Lines {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				filtered = append(filtered, trimmed)
			}
		}
		s.Lines = filtered
		return nil
	case yaml.SequenceNode:
		// Array of strings
		var lines []string
		if err := node.Decode(&lines); err != nil {
			return fmt.Errorf("failed to decode step as string array: %w", err)
		}
		s.Lines = lines
		return nil
	default:
		return fmt.Errorf("step must be either a string or array of strings, got %v", node.Kind)
	}
}

// MarshalYAML implements custom YAML marshaling for Step
func (s Step) MarshalYAML() (interface{}, error) {
	if len(s.Lines) == 1 {
		return s.Lines[0], nil
	}
	return s.Lines, nil
}

// String returns the step as a single script string
func (s Step) String() string {
	return strings.Join(s.Lines, "\n")
}

// IsEmpty returns true if the step has no commands
func (s Step) IsEmpty() bool {
	return len(s.Lines) == 0
}
