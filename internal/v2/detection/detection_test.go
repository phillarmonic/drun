package detection

import (
	"os"
	"testing"
)

func TestDetector_IsToolAvailable(t *testing.T) {
	detector := NewDetector()

	// Test with a tool that should be available on most systems
	if !detector.IsToolAvailable("ls") {
		t.Errorf("Expected 'ls' to be available")
	}

	// Test with a tool that likely doesn't exist
	if detector.IsToolAvailable("nonexistent-tool-12345") {
		t.Errorf("Expected 'nonexistent-tool-12345' to not be available")
	}

	// Test caching - second call should use cache
	if !detector.IsToolAvailable("ls") {
		t.Errorf("Expected cached result for 'ls' to be available")
	}
}

func TestDetector_DetectEnvironment(t *testing.T) {
	detector := NewDetector()

	// Test default environment
	env := detector.DetectEnvironment()
	if env == "" {
		t.Errorf("Expected environment to be detected, got empty string")
	}

	// Test CI environment detection
	_ = os.Setenv("CI", "true")
	detector = NewDetector() // Reset cache
	env = detector.DetectEnvironment()
	if env != "ci" {
		t.Errorf("Expected 'ci' environment, got %q", env)
	}
	_ = os.Unsetenv("CI")

	// Test production environment detection
	_ = os.Setenv("NODE_ENV", "production")
	detector = NewDetector() // Reset cache
	env = detector.DetectEnvironment()
	if env != "production" {
		t.Errorf("Expected 'production' environment, got %q", env)
	}
	_ = os.Unsetenv("NODE_ENV")
}

func TestDetector_CompareVersion(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		version1 string
		operator string
		version2 string
		expected bool
	}{
		{"1.0.0", ">=", "1.0.0", true},
		{"1.1.0", ">", "1.0.0", true},
		{"1.0.0", "<", "1.1.0", true},
		{"1.0.0", "<=", "1.0.0", true},
		{"1.0.0", "==", "1.0.0", true},
		{"1.0.0", "!=", "1.1.0", true},
		{"2.0.0", ">", "1.9.9", true},
		{"1.0.0", ">", "2.0.0", false},
		{"16.14.0", ">=", "16.0.0", true},
		{"14.18.0", "<", "16.0.0", true},
	}

	for _, test := range tests {
		result := detector.CompareVersion(test.version1, test.operator, test.version2)
		if result != test.expected {
			t.Errorf("CompareVersion(%q, %q, %q) = %v, expected %v",
				test.version1, test.operator, test.version2, result, test.expected)
		}
	}
}

func TestDetector_DetectProjectType(t *testing.T) {
	detector := NewDetector()

	// Test with current directory
	types := detector.DetectProjectType()

	// Just verify the method works and returns a slice
	t.Logf("Detected project types: %v", types)

	// The result can be empty if no project files are found in the test directory
	// This is acceptable behavior
}

func TestDetector_GetToolVersion(t *testing.T) {
	detector := NewDetector()

	// Test with Go (should be available in test environment)
	version := detector.GetToolVersion("go")
	if version == "" {
		// Go might not be available in all test environments, so we just check the method works
		t.Logf("Go version not detected (this is okay if Go is not installed)")
	} else {
		t.Logf("Detected Go version: %s", version)
	}

	// Test caching
	version2 := detector.GetToolVersion("go")
	if version != version2 {
		t.Errorf("Expected cached version result to be consistent")
	}
}

func TestDetector_parseVersion(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		version  string
		expected []int
	}{
		{"1.0.0", []int{1, 0, 0}},
		{"16.14.2", []int{16, 14, 2}},
		{"2.7", []int{2, 7}},
		{"1", []int{1}},
		{"", []int{}},
	}

	for _, test := range tests {
		result := detector.parseVersion(test.version)
		if len(result) != len(test.expected) {
			t.Errorf("parseVersion(%q) length = %d, expected %d", test.version, len(result), len(test.expected))
			continue
		}

		for i, v := range result {
			if v != test.expected[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, expected %d", test.version, i, v, test.expected[i])
			}
		}
	}
}

func TestDetector_compareVersions(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		v1       []int
		v2       []int
		expected int
	}{
		{[]int{1, 0, 0}, []int{1, 0, 0}, 0},
		{[]int{1, 1, 0}, []int{1, 0, 0}, 1},
		{[]int{1, 0, 0}, []int{1, 1, 0}, -1},
		{[]int{2, 0}, []int{1, 9, 9}, 1},
		{[]int{1, 0}, []int{1, 0, 0}, 0},
	}

	for _, test := range tests {
		result := detector.compareVersions(test.v1, test.v2)
		if result != test.expected {
			t.Errorf("compareVersions(%v, %v) = %d, expected %d", test.v1, test.v2, result, test.expected)
		}
	}
}
