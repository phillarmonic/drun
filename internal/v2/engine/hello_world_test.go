package engine

import (
	"bytes"
	"os"
	"testing"
)

func TestHelloWorldExample(t *testing.T) {
	// Read the actual hello world example file
	content, err := os.ReadFile("../../../examples/01-hello-world.drun")
	if err != nil {
		t.Fatalf("Failed to read hello world example: %v", err)
	}

	input := string(content)

	// Test executing the "hello" task
	var output bytes.Buffer
	err = ExecuteString(input, "hello", &output)
	if err != nil {
		t.Fatalf("ExecuteString failed: %v", err)
	}

	expectedOutput := "ℹ️  Hello from drun v2! 👋\n"
	if output.String() != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output.String())
	}

	// Test executing the "hello world" task
	output.Reset()
	err = ExecuteString(input, "hello world", &output)
	if err != nil {
		t.Fatalf("ExecuteString failed: %v", err)
	}

	expectedLines := []string{
		"🚀 Starting hello world example",
		"ℹ️  Welcome to the semantic task runner!",
		"✅ Hello world completed successfully!",
	}

	outputStr := output.String()
	for _, expected := range expectedLines {
		if !contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, outputStr)
		}
	}

	t.Logf("Hello world example executed successfully!")
	t.Logf("Output:\n%s", outputStr)
}

func TestHelloWorldDryRun(t *testing.T) {
	// Read the actual hello world example file
	content, err := os.ReadFile("../../../examples/01-hello-world.drun")
	if err != nil {
		t.Fatalf("Failed to read hello world example: %v", err)
	}

	program, err := ParseString(string(content))
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	err = engine.Execute(program, "hello world")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	outputStr := output.String()
	expectedParts := []string{
		"[DRY RUN] Would execute task: hello world",
		"[DRY RUN] step: Starting hello world example",
		"[DRY RUN] info: Welcome to the semantic task runner!",
		"[DRY RUN] success: Hello world completed successfully!",
	}

	for _, part := range expectedParts {
		if !contains(outputStr, part) {
			t.Errorf("Expected output to contain %q, got %q", part, outputStr)
		}
	}

	t.Logf("Dry run output:\n%s", outputStr)
}

func TestListHelloWorldTasks(t *testing.T) {
	// Read the actual hello world example file
	content, err := os.ReadFile("../../../examples/01-hello-world.drun")
	if err != nil {
		t.Fatalf("Failed to read hello world example: %v", err)
	}

	program, err := ParseString(string(content))
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	engine := NewEngine(nil)
	tasks := engine.ListTasks(program)

	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	// Check tasks
	expectedTasks := []struct {
		name        string
		description string
	}{
		{"hello", "No description"},
		{"hello world", "No description"},
	}

	for i, expected := range expectedTasks {
		if tasks[i].Name != expected.name {
			t.Errorf("Task %d: expected name %q, got %q", i, expected.name, tasks[i].Name)
		}
		if tasks[i].Description != expected.description {
			t.Errorf("Task %d: expected description %q, got %q", i, expected.description, tasks[i].Description)
		}
	}

	t.Logf("Available tasks:")
	for _, task := range tasks {
		t.Logf("  %s - %s", task.Name, task.Description)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
