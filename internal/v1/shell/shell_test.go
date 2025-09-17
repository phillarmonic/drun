package shell

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v1/model"
)

func TestNewSelector(t *testing.T) {
	shellConfigs := map[string]model.ShellConfig{
		"linux": {
			Cmd:  "/bin/bash",
			Args: []string{"-c"},
		},
	}

	selector := NewSelector(shellConfigs)

	if selector == nil {
		t.Fatal("Expected selector to be created, got nil")
	}
}

func TestSelector_GetShell_CustomConfig(t *testing.T) {
	shellConfigs := map[string]model.ShellConfig{
		"linux": {
			Cmd:  "/bin/bash",
			Args: []string{"-c"},
		},
		"darwin": {
			Cmd:  "/bin/zsh",
			Args: []string{"-c"},
		},
	}

	selector := NewSelector(shellConfigs)

	shell, err := selector.Select("", "linux")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if shell.Cmd != "/bin/bash" {
		t.Errorf("Expected cmd '/bin/bash', got %q", shell.Cmd)
	}

	if len(shell.Args) != 1 || shell.Args[0] != "-c" {
		t.Errorf("Expected args ['-c'], got %v", shell.Args)
	}

	if shell.OS != "linux" {
		t.Errorf("Expected OS 'linux', got %q", shell.OS)
	}
}

func TestSelector_GetShell_DefaultConfig(t *testing.T) {
	selector := NewSelector(nil) // No custom configs

	// Test default Linux config
	shell, err := selector.Select("", "linux")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if shell.Cmd != "/bin/sh" {
		t.Errorf("Expected default Linux cmd '/bin/sh', got %q", shell.Cmd)
	}

	// Test default Darwin config
	shell, err = selector.Select("", "darwin")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if shell.Cmd != "/bin/zsh" {
		t.Errorf("Expected default Darwin cmd '/bin/zsh', got %q", shell.Cmd)
	}

	// Test default Windows config
	shell, err = selector.Select("", "windows")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if shell.Cmd != "pwsh" {
		t.Errorf("Expected default Windows cmd 'pwsh', got %q", shell.Cmd)
	}
}

func TestSelector_GetShell_UnknownOS(t *testing.T) {
	selector := NewSelector(nil)

	shell, err := selector.Select("", "unknown-os")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should fall back to Linux default
	if shell.Cmd != "/bin/sh" {
		t.Errorf("Expected fallback to Linux default '/bin/sh', got %q", shell.Cmd)
	}

	if shell.OS != "unknown-os" {
		t.Errorf("Expected OS to be preserved as 'unknown-os', got %q", shell.OS)
	}
}

func TestShell_BuildCommand_Linux(t *testing.T) {
	shell := &Shell{
		Cmd:  "/bin/bash",
		Args: []string{"-c"},
		OS:   "linux",
	}

	script := "echo hello world"
	command := shell.BuildCommand(script)

	expected := []string{"/bin/bash", "-c", "echo hello world"}

	if len(command) != len(expected) {
		t.Fatalf("Expected %d args, got %d", len(expected), len(command))
	}

	for i, arg := range command {
		if arg != expected[i] {
			t.Errorf("Arg %d: expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestShell_BuildCommand_Windows(t *testing.T) {
	shell := &Shell{
		Cmd:  "pwsh",
		Args: []string{"-NoLogo", "-Command"},
		OS:   "windows",
	}

	script := "echo hello world"
	command := shell.BuildCommand(script)

	expected := []string{"pwsh", "-NoLogo", "-Command", "echo hello world"}

	if len(command) != len(expected) {
		t.Fatalf("Expected %d args, got %d", len(expected), len(command))
	}

	for i, arg := range command {
		if arg != expected[i] {
			t.Errorf("Arg %d: expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestShell_BuildCommand_WithMultipleArgs(t *testing.T) {
	shell := &Shell{
		Cmd:  "/bin/bash",
		Args: []string{"-c", "-e", "-u"},
		OS:   "linux",
	}

	script := "echo test"
	command := shell.BuildCommand(script)

	expected := []string{"/bin/bash", "-c", "-e", "-u", "echo test"}

	if len(command) != len(expected) {
		t.Fatalf("Expected %d args, got %d", len(expected), len(command))
	}

	for i, arg := range command {
		if arg != expected[i] {
			t.Errorf("Arg %d: expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestShell_convertShellIdioms_Windows(t *testing.T) {
	shell := &Shell{
		Cmd:  "pwsh",
		Args: []string{"-Command"},
		OS:   "windows",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple command",
			input:    "echo hello",
			expected: "echo hello",
		},
		{
			name:     "export command",
			input:    "export VAR=value",
			expected: "$env:VAR='value'",
		},
		{
			name:     "multiple exports",
			input:    "export A=1\nexport B=2",
			expected: "$env:A='1'\n$env:B='2'",
		},
		{
			name:     "mixed commands",
			input:    "echo start\nexport VAR=test\necho end",
			expected: "echo start\n$env:VAR='test'\necho end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shell.convertShellIdioms(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestShell_convertShellIdioms_NonWindows(t *testing.T) {
	shell := &Shell{
		Cmd:  "/bin/bash",
		Args: []string{"-c"},
		OS:   "linux",
	}

	input := "export VAR=value\necho hello"
	result := shell.convertShellIdioms(input)

	// Should not modify non-Windows scripts
	if result != input {
		t.Errorf("Expected script to be unchanged for non-Windows OS, got %q", result)
	}
}

func TestDefaultShellConfigs(t *testing.T) {
	// Test that selector with nil configs uses reasonable defaults
	selector := NewSelector(nil)

	// Check that all expected OS configs work
	expectedOS := []string{"linux", "darwin", "windows"}

	for _, os := range expectedOS {
		shell, err := selector.Select("", os)
		if err != nil {
			t.Errorf("Expected no error for %s, got %v", os, err)
		}
		if shell == nil {
			t.Errorf("Expected shell for %s to be created", os)
		}
	}
}

func TestShell_BuildCommand_EmptyScript(t *testing.T) {
	shell := &Shell{
		Cmd:  "/bin/bash",
		Args: []string{"-c"},
		OS:   "linux",
	}

	command := shell.BuildCommand("")

	expected := []string{"/bin/bash", "-c", ""}

	if len(command) != len(expected) {
		t.Fatalf("Expected %d args, got %d", len(expected), len(command))
	}

	for i, arg := range command {
		if arg != expected[i] {
			t.Errorf("Arg %d: expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestShell_BuildCommand_MultilineScript(t *testing.T) {
	shell := &Shell{
		Cmd:  "/bin/bash",
		Args: []string{"-c"},
		OS:   "linux",
	}

	script := "echo line1\necho line2\necho line3"
	command := shell.BuildCommand(script)

	expected := []string{"/bin/bash", "-c", "echo line1\necho line2\necho line3"}

	if len(command) != len(expected) {
		t.Fatalf("Expected %d args, got %d", len(expected), len(command))
	}

	if command[2] != script {
		t.Errorf("Expected script to be preserved as-is, got %q", command[2])
	}
}
