package orchestration

import (
	"testing"
)

func TestServiceRegistryLifecycle(t *testing.T) {
	registry := NewServiceRegistry()
	api := &Service{Name: "api"}
	worker := &Service{Name: "worker"}

	if err := registry.Register(api); err != nil {
		t.Fatalf("register api: %v", err)
	}
	if err := registry.Register(api); err == nil || err.Error() != "service api already registered" {
		t.Fatalf("duplicate register error = %v", err)
	}

	got, err := registry.Get("api")
	if err != nil || got != api {
		t.Fatalf("get api = %v, %v", got, err)
	}
	if _, err := registry.Get("missing"); err == nil || err.Error() != "service missing not found" {
		t.Fatalf("missing get error = %v", err)
	}

	if err := registry.Update(worker); err == nil || err.Error() != "service worker not found" {
		t.Fatalf("unknown update error = %v", err)
	}
	api.Description = "updated"
	if err := registry.Update(api); err != nil {
		t.Fatalf("update api: %v", err)
	}
	if got, _ := registry.Get("api"); got.Description != "updated" {
		t.Fatalf("update did not replace the entry: %#v", got)
	}

	if !registry.Exists("api") || registry.Exists("missing") {
		t.Fatalf("exists = %v/%v, want true/false", registry.Exists("api"), registry.Exists("missing"))
	}
	if err := registry.Register(worker); err != nil {
		t.Fatalf("register worker: %v", err)
	}
	if all := registry.GetAll(); len(all) != 2 {
		t.Fatalf("get all = %d entries, want 2", len(all))
	}

	if err := registry.Delete("missing"); err == nil || err.Error() != "service missing not found" {
		t.Fatalf("missing delete error = %v", err)
	}
	if err := registry.Delete("api"); err != nil {
		t.Fatalf("delete api: %v", err)
	}
	if registry.Exists("api") {
		t.Fatal("api still registered after delete")
	}
}

// TestOrchestrationRegistryReusesTheGenericRegistry proves the second alias is
// the same generic type with a different entry type and message kind.
func TestOrchestrationRegistryReusesTheGenericRegistry(t *testing.T) {
	registry := NewOrchestrationRegistry()
	group := &Orchestration{Name: "stack"}

	if err := registry.Register(group); err != nil {
		t.Fatalf("register stack: %v", err)
	}
	if err := registry.Register(group); err == nil || err.Error() != "orchestration stack already registered" {
		t.Fatalf("duplicate register error = %v", err)
	}
	if _, err := registry.Get("missing"); err == nil || err.Error() != "orchestration missing not found" {
		t.Fatalf("missing get error = %v", err)
	}
	if got, err := registry.Get("stack"); err != nil || got != group {
		t.Fatalf("get stack = %v, %v", got, err)
	}
	if all := registry.GetAll(); len(all) != 1 || all[0] != group {
		t.Fatalf("get all = %#v", all)
	}
}
