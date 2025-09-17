package dag

import (
	"testing"

	"github.com/phillarmonic/drun/internal/v1/model"
)

func TestBuilder_Build_SimpleRecipe(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"simple": {
				Help: "Simple recipe",
				Run:  model.Step{Lines: []string{"echo hello"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	plan, err := builder.Build("simple", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plan.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(plan.Nodes))
	}

	if plan.Nodes[0].ID != "simple" {
		t.Errorf("Expected node ID 'simple', got %q", plan.Nodes[0].ID)
	}

	if len(plan.Edges) != 0 {
		t.Errorf("Expected 0 edges, got %d", len(plan.Edges))
	}
}

func TestBuilder_Build_WithDependencies(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"dep1": {
				Help: "Dependency 1",
				Run:  model.Step{Lines: []string{"echo dep1"}},
			},
			"dep2": {
				Help: "Dependency 2",
				Run:  model.Step{Lines: []string{"echo dep2"}},
			},
			"main": {
				Help: "Main recipe",
				Deps: []string{"dep1", "dep2"},
				Run:  model.Step{Lines: []string{"echo main"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	plan, err := builder.Build("main", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plan.Nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(plan.Nodes))
	}

	if len(plan.Edges) != 2 {
		t.Fatalf("Expected 2 edges, got %d", len(plan.Edges))
	}

	// Check that dependencies come before main
	nodeNames := make([]string, len(plan.Nodes))
	for i, node := range plan.Nodes {
		nodeNames[i] = node.ID
	}

	mainIndex := -1
	dep1Index := -1
	dep2Index := -1

	for i, name := range nodeNames {
		switch name {
		case "main":
			mainIndex = i
		case "dep1":
			dep1Index = i
		case "dep2":
			dep2Index = i
		}
	}

	if mainIndex == -1 || dep1Index == -1 || dep2Index == -1 {
		t.Fatal("Not all expected nodes found")
	}

	// Dependencies should come before main
	if dep1Index >= mainIndex {
		t.Error("dep1 should come before main in execution order")
	}

	if dep2Index >= mainIndex {
		t.Error("dep2 should come before main in execution order")
	}
}

func TestBuilder_Build_CircularDependency(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"a": {
				Help: "Recipe A",
				Deps: []string{"b"},
				Run:  model.Step{Lines: []string{"echo a"}},
			},
			"b": {
				Help: "Recipe B",
				Deps: []string{"c"},
				Run:  model.Step{Lines: []string{"echo b"}},
			},
			"c": {
				Help: "Recipe C",
				Deps: []string{"a"}, // Circular dependency
				Run:  model.Step{Lines: []string{"echo c"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	_, err := builder.Build("a", ctx)

	if err == nil {
		t.Fatal("Expected error for circular dependency, got nil")
	}
}

func TestBuilder_Build_MissingDependency(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"main": {
				Help: "Main recipe",
				Deps: []string{"nonexistent"},
				Run:  model.Step{Lines: []string{"echo main"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	_, err := builder.Build("main", ctx)

	if err == nil {
		t.Fatal("Expected error for missing dependency, got nil")
	}
}

func TestBuilder_Build_MissingRecipe(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	_, err := builder.Build("nonexistent", ctx)

	if err == nil {
		t.Fatal("Expected error for missing recipe, got nil")
	}
}

func TestBuilder_Build_ExecutionLevels(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"dep1": {
				Help: "Dependency 1",
				Run:  model.Step{Lines: []string{"echo dep1"}},
			},
			"dep2": {
				Help: "Dependency 2",
				Run:  model.Step{Lines: []string{"echo dep2"}},
			},
			"dep3": {
				Help: "Dependency 3",
				Run:  model.Step{Lines: []string{"echo dep3"}},
			},
			"main": {
				Help: "Main recipe",
				Deps: []string{"dep1", "dep2", "dep3"},
				Run:  model.Step{Lines: []string{"echo main"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	plan, err := builder.Build("main", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plan.Levels) != 2 {
		t.Fatalf("Expected 2 execution levels, got %d", len(plan.Levels))
	}

	// First level should have 3 dependencies (can run in parallel)
	if len(plan.Levels[0]) != 3 {
		t.Errorf("Expected 3 tasks in first level, got %d", len(plan.Levels[0]))
	}

	// Second level should have 1 main task
	if len(plan.Levels[1]) != 1 {
		t.Errorf("Expected 1 task in second level, got %d", len(plan.Levels[1]))
	}
}

func TestBuilder_Build_ComplexDependencyGraph(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"base": {
				Help: "Base recipe",
				Run:  model.Step{Lines: []string{"echo base"}},
			},
			"lint": {
				Help: "Lint code",
				Deps: []string{"base"},
				Run:  model.Step{Lines: []string{"echo lint"}},
			},
			"test": {
				Help: "Run tests",
				Deps: []string{"base"},
				Run:  model.Step{Lines: []string{"echo test"}},
			},
			"build": {
				Help: "Build application",
				Deps: []string{"lint", "test"},
				Run:  model.Step{Lines: []string{"echo build"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	plan, err := builder.Build("build", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plan.Nodes) != 4 {
		t.Fatalf("Expected 4 nodes, got %d", len(plan.Nodes))
	}

	if len(plan.Levels) != 3 {
		t.Fatalf("Expected 3 execution levels, got %d", len(plan.Levels))
	}

	// Level 0: base (1 task)
	if len(plan.Levels[0]) != 1 {
		t.Errorf("Expected 1 task in level 0, got %d", len(plan.Levels[0]))
	}

	// Level 1: lint, test (2 tasks in parallel)
	if len(plan.Levels[1]) != 2 {
		t.Errorf("Expected 2 tasks in level 1, got %d", len(plan.Levels[1]))
	}

	// Level 2: build (1 task)
	if len(plan.Levels[2]) != 1 {
		t.Errorf("Expected 1 task in level 2, got %d", len(plan.Levels[2]))
	}
}

func TestBuilder_Build_RecipeWithEnvironment(t *testing.T) {
	spec := &model.Spec{
		Env: map[string]string{
			"GLOBAL_VAR": "global_value",
		},
		Recipes: map[string]model.Recipe{
			"test": {
				Help: "Test recipe with env",
				Env: map[string]string{
					"LOCAL_VAR": "local_value",
				},
				Run: model.Step{Lines: []string{"echo test"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{
		Env: map[string]string{
			"CONTEXT_VAR": "context_value",
		},
	}

	plan, err := builder.Build("test", ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(plan.Nodes) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(plan.Nodes))
	}

	node := plan.Nodes[0]

	// Check that recipe-specific environment is merged
	if node.Context.Env["LOCAL_VAR"] != "local_value" {
		t.Errorf("Expected LOCAL_VAR='local_value', got %q", node.Context.Env["LOCAL_VAR"])
	}

	// Check that context environment is preserved
	if node.Context.Env["CONTEXT_VAR"] != "context_value" {
		t.Errorf("Expected CONTEXT_VAR='context_value', got %q", node.Context.Env["CONTEXT_VAR"])
	}
}

func TestBuilder_topologicalSort(t *testing.T) {
	builder := &Builder{}

	// Create a simple DAG: A -> C, B -> C
	nodes := []model.PlanNode{
		{ID: "A"},
		{ID: "B"},
		{ID: "C"},
	}

	edges := [][2]int{
		{0, 2}, // A -> C
		{1, 2}, // B -> C
	}

	sorted, err := builder.topologicalSort(nodes, edges)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 nodes in sorted order, got %d", len(sorted))
	}

	// C should come after both A and B
	cIndex := -1
	aIndex := -1
	bIndex := -1

	for i, nodeIdx := range sorted {
		switch nodes[nodeIdx].ID {
		case "A":
			aIndex = i
		case "B":
			bIndex = i
		case "C":
			cIndex = i
		}
	}

	if aIndex >= cIndex {
		t.Error("A should come before C in topological order")
	}

	if bIndex >= cIndex {
		t.Error("B should come before C in topological order")
	}
}
