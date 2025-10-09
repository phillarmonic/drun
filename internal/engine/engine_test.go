package engine

import (
	"bytes"
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
	"github.com/phillarmonic/drun/internal/domain/task"
)

func TestNewEngineWithOptions(t *testing.T) {
	output := &bytes.Buffer{}
	registry := task.NewRegistry()

	// Create engine with custom options
	engine := NewEngineWithOptions(
		WithOutput(output),
		WithTaskRegistry(registry),
		WithDryRun(true),
		WithVerbose(true),
	)

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	if engine.output != output {
		t.Error("Expected output to be set from options")
	}

	if engine.taskRegistry != registry {
		t.Error("Expected task registry to be set from options")
	}

	if !engine.dryRun {
		t.Error("Expected dry-run to be true")
	}

	if !engine.verbose {
		t.Error("Expected verbose to be true")
	}
}

func TestNewEngine_BackwardCompatibility(t *testing.T) {
	output := &bytes.Buffer{}
	engine := NewEngine(output)

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	if engine.output != output {
		t.Error("Expected output to be set")
	}

	// Should have defaults
	if engine.taskRegistry == nil {
		t.Error("Expected task registry to be created by default")
	}

	if engine.paramValidator == nil {
		t.Error("Expected param validator to be created by default")
	}

	if engine.depResolver == nil {
		t.Error("Expected dependency resolver to be created by default")
	}
}

func TestEngineWithOptions_IsolatedTaskRegistry(t *testing.T) {
	// Create two engines with separate registries
	registry1 := task.NewRegistry()
	registry2 := task.NewRegistry()

	engine1 := NewEngineWithOptions(WithTaskRegistry(registry1))
	engine2 := NewEngineWithOptions(WithTaskRegistry(registry2))

	// Register task in engine1's registry
	task1, err := task.NewTask(&ast.TaskStatement{
		Name: "task1",
		Body: []ast.Statement{},
	}, "", "test.drun")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if err := registry1.Register(task1); err != nil {
		t.Fatalf("Failed to register task: %v", err)
	}

	// Verify task is in registry1 but not registry2
	if _, err := registry1.Get("task1"); err != nil {
		t.Error("Expected task1 to be in registry1")
	}

	if _, err := registry2.Get("task1"); err == nil {
		t.Error("Expected task1 NOT to be in registry2")
	}

	// Verify engines have separate registries
	if engine1.taskRegistry == engine2.taskRegistry {
		t.Error("Expected engines to have separate task registries")
	}
}
