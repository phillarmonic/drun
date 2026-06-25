package task

import (
	"strings"
	"testing"
)

func TestRegistry_RegisterAllowsDisjointPlatformVariants(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(&Task{Name: "shell", Platforms: []string{"linux"}}); err != nil {
		t.Fatalf("register linux variant: %v", err)
	}
	if err := registry.Register(&Task{Name: "shell", Platforms: []string{"mac"}}); err != nil {
		t.Fatalf("register mac variant: %v", err)
	}
}

func TestRegistry_RegisterRejectsOverlappingPlatformVariants(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(&Task{Name: "shell", Platforms: []string{"mac"}}); err != nil {
		t.Fatalf("register mac variant: %v", err)
	}
	err := registry.Register(&Task{Name: "shell", Platforms: []string{"mac", "windows"}})
	if err == nil {
		t.Fatal("expected overlap registration to fail")
	}
	if !strings.Contains(err.Error(), "overlapping platform variants") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistry_GetSelectsCurrentPlatformVariant(t *testing.T) {
	registry := NewRegistry()
	registry.SetCurrentPlatform("mac")

	if err := registry.Register(&Task{Name: "shell", Platforms: []string{"linux"}}); err != nil {
		t.Fatalf("register linux variant: %v", err)
	}
	macVariant := &Task{Name: "shell", Platforms: []string{"mac"}}
	if err := registry.Register(macVariant); err != nil {
		t.Fatalf("register mac variant: %v", err)
	}

	got, err := registry.Get("shell")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != macVariant {
		t.Fatalf("expected mac variant, got %#v", got)
	}
}

func TestRegistry_RegisterAllowsSingleUnannotatedFallbackVariant(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(&Task{Name: "shell", Platforms: []string{"linux"}}); err != nil {
		t.Fatalf("register linux variant: %v", err)
	}
	if err := registry.Register(&Task{Name: "shell"}); err != nil {
		t.Fatalf("register fallback variant: %v", err)
	}
}

func TestRegistry_RegisterRejectsMultipleUnannotatedFallbackVariants(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(&Task{Name: "shell"}); err != nil {
		t.Fatalf("register first fallback variant: %v", err)
	}
	err := registry.Register(&Task{Name: "shell"})
	if err == nil {
		t.Fatal("expected second unannotated fallback registration to fail")
	}
	if !strings.Contains(err.Error(), "one unannotated fallback variant") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistry_GetFallsBackToUnannotatedVariant(t *testing.T) {
	registry := NewRegistry()
	registry.SetCurrentPlatform("windows")

	if err := registry.Register(&Task{Name: "shell", Platforms: []string{"linux"}}); err != nil {
		t.Fatalf("register linux variant: %v", err)
	}
	fallbackVariant := &Task{Name: "shell"}
	if err := registry.Register(fallbackVariant); err != nil {
		t.Fatalf("register fallback variant: %v", err)
	}

	got, err := registry.Get("shell")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != fallbackVariant {
		t.Fatalf("expected fallback variant, got %#v", got)
	}
}
