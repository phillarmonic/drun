package task

import (
	"testing"
)

func TestDependencyResolver_Resolve(t *testing.T) {
	registry := NewRegistry()

	// Create tasks with dependencies
	task1 := &Task{Name: "task1"}
	task2 := &Task{Name: "task2", Dependencies: []Dependency{{Name: "task1"}}}
	task3 := &Task{Name: "task3", Dependencies: []Dependency{{Name: "task2"}}}

	_ = registry.Register(task1)
	_ = registry.Register(task2)
	_ = registry.Register(task3)

	resolver := NewDependencyResolver(registry)

	// Resolve task3 - should get [task1, task2, task3]
	tasks, err := resolver.Resolve("task3")
	if err != nil {
		t.Errorf("Resolve() error = %v, want nil", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Resolve() length = %v, want 3", len(tasks))
	}

	// Verify order: dependencies before dependents
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

func TestDependencyResolver_NoDependencies(t *testing.T) {
	registry := NewRegistry()
	task := &Task{Name: "task1"}
	_ = registry.Register(task)

	resolver := NewDependencyResolver(registry)

	tasks, err := resolver.Resolve("task1")
	if err != nil {
		t.Errorf("Resolve() error = %v, want nil", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Resolve() length = %v, want 1", len(tasks))
	}
	if tasks[0].Name != "task1" {
		t.Errorf("Task name = %v, want task1", tasks[0].Name)
	}
}

func TestDependencyResolver_CircularDependency(t *testing.T) {
	registry := NewRegistry()

	// Create circular dependency: task1 -> task2 -> task1
	task1 := &Task{Name: "task1", Dependencies: []Dependency{{Name: "task2"}}}
	task2 := &Task{Name: "task2", Dependencies: []Dependency{{Name: "task1"}}}

	_ = registry.Register(task1)
	_ = registry.Register(task2)

	resolver := NewDependencyResolver(registry)

	_, err := resolver.Resolve("task1")
	if err == nil {
		t.Error("Resolve() should return error for circular dependency")
	}
}

func TestDependencyResolver_MissingDependency(t *testing.T) {
	registry := NewRegistry()

	task1 := &Task{Name: "task1", Dependencies: []Dependency{{Name: "nonexistent"}}}
	_ = registry.Register(task1)

	resolver := NewDependencyResolver(registry)

	_, err := resolver.Resolve("task1")
	if err == nil {
		t.Error("Resolve() should return error for missing dependency")
	}
}

func TestDependencyResolver_MissingTask(t *testing.T) {
	registry := NewRegistry()
	resolver := NewDependencyResolver(registry)

	_, err := resolver.Resolve("nonexistent")
	if err == nil {
		t.Error("Resolve() should return error for non-existent task")
	}
}

func TestDependencyResolver_GetParallelGroups(t *testing.T) {
	registry := NewRegistry()

	task := &Task{
		Name: "main",
		Dependencies: []Dependency{
			{Name: "dep1", Parallel: true},
			{Name: "dep2", Parallel: true},
			{Name: "dep3", Parallel: false, Sequential: true},
		},
	}

	dep1 := &Task{Name: "dep1"}
	dep2 := &Task{Name: "dep2"}
	dep3 := &Task{Name: "dep3"}

	_ = registry.Register(task)
	_ = registry.Register(dep1)
	_ = registry.Register(dep2)
	_ = registry.Register(dep3)

	resolver := NewDependencyResolver(registry)

	groups, err := resolver.GetParallelGroups(task)
	if err != nil {
		t.Errorf("GetParallelGroups() error = %v, want nil", err)
	}

	// Should have 2 groups: [dep1, dep2] (parallel), [dep3] (sequential)
	if len(groups) != 2 {
		t.Errorf("Groups length = %v, want 2", len(groups))
	}

	if len(groups[0]) != 2 {
		t.Errorf("First group length = %v, want 2", len(groups[0]))
	}

	if len(groups[1]) != 1 {
		t.Errorf("Second group length = %v, want 1", len(groups[1]))
	}
}

func TestDependencyResolver_ComplexDependencyTree(t *testing.T) {
	registry := NewRegistry()

	// Create a diamond dependency structure
	//       main
	//      /    \
	//   task1  task2
	//      \    /
	//      shared

	shared := &Task{Name: "shared"}
	task1 := &Task{Name: "task1", Dependencies: []Dependency{{Name: "shared"}}}
	task2 := &Task{Name: "task2", Dependencies: []Dependency{{Name: "shared"}}}
	main := &Task{Name: "main", Dependencies: []Dependency{{Name: "task1"}, {Name: "task2"}}}

	_ = registry.Register(shared)
	_ = registry.Register(task1)
	_ = registry.Register(task2)
	_ = registry.Register(main)

	resolver := NewDependencyResolver(registry)

	tasks, err := resolver.Resolve("main")
	if err != nil {
		t.Errorf("Resolve() error = %v, want nil", err)
	}

	// shared should appear first, then task1 and task2, then main
	if len(tasks) != 4 {
		t.Errorf("Resolve() length = %v, want 4", len(tasks))
	}

	if tasks[0].Name != "shared" {
		t.Errorf("First task = %v, want shared", tasks[0].Name)
	}

	if tasks[len(tasks)-1].Name != "main" {
		t.Errorf("Last task = %v, want main", tasks[len(tasks)-1].Name)
	}
}
