package engine

import (
	"bytes"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/domain/task"
	"github.com/phillarmonic/drun/v2/internal/provisioning"
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

func TestNewEngineWithOptions_ProvisioningControls(t *testing.T) {
	engine := NewEngineWithOptions(
		WithAllowToolVersionChanges(true),
		WithUserProvisioningSources([]string{"~/.drun/provisionings.yaml"}),
	)

	if !engine.allowToolVersionChanges {
		t.Fatal("expected allowToolVersionChanges to be enabled")
	}

	if len(engine.userProvisioningSources) != 1 || engine.userProvisioningSources[0] != "~/.drun/provisionings.yaml" {
		t.Fatalf("userProvisioningSources = %#v", engine.userProvisioningSources)
	}

	defaultSources := provisioning.DefaultEmbeddedSources()
	if len(defaultSources) == 0 {
		t.Fatal("expected built-in embedded provisioning sources")
	}
	if len(engine.embeddedProvisionings) < len(defaultSources) {
		t.Fatalf("embeddedProvisionings = %#v", engine.embeddedProvisionings)
	}
	if engine.embeddedProvisionings[len(engine.embeddedProvisionings)-1].Name != defaultSources[0].Name {
		t.Fatalf("last embedded source = %q", engine.embeddedProvisionings[len(engine.embeddedProvisionings)-1].Name)
	}
}

func TestNewEngineWithOptions_CustomEmbeddedSourcesPrecedeDefaults(t *testing.T) {
	engine := NewEngineWithOptions(
		WithEmbeddedProvisioningSources([]provisioning.EmbeddedSource{{
			Name:    "custom",
			Content: []byte("version: \"1\"\nprovisionings:\n  custom:\n    targets:\n      - install: \"echo custom\"\n"),
		}}),
	)

	if len(engine.embeddedProvisionings) < 2 {
		t.Fatalf("embeddedProvisionings = %#v", engine.embeddedProvisionings)
	}
	if engine.embeddedProvisionings[0].Name != "custom" {
		t.Fatalf("first embedded source = %q", engine.embeddedProvisionings[0].Name)
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
