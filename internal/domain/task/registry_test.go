package task

import (
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	task1 := &Task{Name: "task1", Description: "First task"}
	task2 := &Task{Name: "task2", Description: "Second task"}

	// Register first task
	err := registry.Register(task1)
	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}

	// Register second task
	err = registry.Register(task2)
	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}

	// Try to register duplicate
	err = registry.Register(task1)
	if err == nil {
		t.Error("Register() should fail for duplicate task")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	task := &Task{Name: "test", Description: "Test task"}

	_ = registry.Register(task)

	// Get existing task
	got, err := registry.Get("test")
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}
	if got.Name != "test" {
		t.Errorf("Get() name = %v, want test", got.Name)
	}

	// Get non-existing task
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Get() should return error for non-existing task")
	}
}

func TestRegistry_Exists(t *testing.T) {
	registry := NewRegistry()
	task := &Task{Name: "test"}

	_ = registry.Register(task)

	if !registry.Exists("test") {
		t.Error("Exists() should return true for registered task")
	}

	if registry.Exists("nonexistent") {
		t.Error("Exists() should return false for non-existing task")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	task1 := &Task{Name: "task1"}
	task2 := &Task{Name: "task2"}
	task3 := &Task{Name: "task3"}

	_ = registry.Register(task1)
	_ = registry.Register(task2)
	_ = registry.Register(task3)

	tasks := registry.List()

	if len(tasks) != 3 {
		t.Errorf("List() length = %v, want 3", len(tasks))
	}

	// Verify order is preserved
	if tasks[0].Name != "task1" {
		t.Errorf("First task = %v, want task1", tasks[0].Name)
	}
	if tasks[1].Name != "task2" {
		t.Errorf("Second task = %v, want task2", tasks[1].Name)
	}
	if tasks[2].Name != "task3" {
		t.Errorf("Third task = %v, want task3", tasks[2].Name)
	}
}

func TestRegistry_ListByNamespace(t *testing.T) {
	registry := NewRegistry()

	task1 := &Task{Name: "task1", Namespace: "ns1"}
	task2 := &Task{Name: "task2", Namespace: "ns2"}
	task3 := &Task{Name: "task3", Namespace: "ns1"}

	_ = registry.Register(task1)
	_ = registry.Register(task2)
	_ = registry.Register(task3)

	ns1Tasks := registry.ListByNamespace("ns1")
	if len(ns1Tasks) != 2 {
		t.Errorf("ListByNamespace('ns1') length = %v, want 2", len(ns1Tasks))
	}

	ns2Tasks := registry.ListByNamespace("ns2")
	if len(ns2Tasks) != 1 {
		t.Errorf("ListByNamespace('ns2') length = %v, want 1", len(ns2Tasks))
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	task1 := &Task{Name: "task1"}
	task2 := &Task{Name: "task2"}

	_ = registry.Register(task1)
	_ = registry.Register(task2)

	if registry.Count() != 2 {
		t.Errorf("Count() before clear = %v, want 2", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count() after clear = %v, want 0", registry.Count())
	}

	if registry.Exists("task1") {
		t.Error("Task should not exist after clear")
	}
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Errorf("Initial count = %v, want 0", registry.Count())
	}

	_ = registry.Register(&Task{Name: "task1"})
	if registry.Count() != 1 {
		t.Errorf("Count after one register = %v, want 1", registry.Count())
	}

	_ = registry.Register(&Task{Name: "task2"})
	if registry.Count() != 2 {
		t.Errorf("Count after two registers = %v, want 2", registry.Count())
	}
}

func TestRegistry_NamespacedTasks(t *testing.T) {
	registry := NewRegistry()

	task := &Task{Name: "task1", Namespace: "myns"}
	_ = registry.Register(task)

	// Should be able to get by full name
	got, err := registry.Get("myns.task1")
	if err != nil {
		t.Errorf("Get('myns.task1') error = %v, want nil", err)
	}
	if got.Name != "task1" {
		t.Errorf("Got task name = %v, want task1", got.Name)
	}

	// Should also be able to get by simple name
	got2, err := registry.Get("task1")
	if err != nil {
		t.Errorf("Get('task1') error = %v, want nil", err)
	}
	if got2.Name != "task1" {
		t.Errorf("Got task name = %v, want task1", got2.Name)
	}
}
