package dag

import (
	"testing"

	"github.com/phillarmonic/drun/internal/model"
)

func TestBuilder_generateMatrixCombinations(t *testing.T) {
	builder := NewBuilder(&model.Spec{})

	tests := []struct {
		name     string
		matrix   map[string][]any
		expected int
	}{
		{
			"Empty matrix",
			map[string][]any{},
			0,
		},
		{
			"Single dimension",
			map[string][]any{
				"os": {"ubuntu", "macos", "windows"},
			},
			3,
		},
		{
			"Two dimensions",
			map[string][]any{
				"os":      {"ubuntu", "macos"},
				"version": {"16", "18", "20"},
			},
			6, // 2 * 3
		},
		{
			"Three dimensions",
			map[string][]any{
				"os":      {"ubuntu", "macos"},
				"version": {"16", "18"},
				"arch":    {"amd64", "arm64"},
			},
			8, // 2 * 2 * 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combinations := builder.generateMatrixCombinations(tt.matrix)

			if len(combinations) != tt.expected {
				t.Errorf("generateMatrixCombinations() generated %d combinations, want %d", len(combinations), tt.expected)
			}

			// Verify all combinations are unique
			seen := make(map[string]bool)
			for _, combo := range combinations {
				key := ""
				for k, v := range combo {
					key += k + "=" + v.(string) + ";"
				}
				if seen[key] {
					t.Errorf("generateMatrixCombinations() generated duplicate combination: %v", combo)
				}
				seen[key] = true
			}

			// Verify each combination has all matrix keys
			for _, combo := range combinations {
				for key := range tt.matrix {
					if _, exists := combo[key]; !exists {
						t.Errorf("generateMatrixCombinations() combination missing key %q: %v", key, combo)
					}
				}
			}
		})
	}
}

func TestBuilder_generateMatrixCombinations_Values(t *testing.T) {
	builder := NewBuilder(&model.Spec{})

	matrix := map[string][]any{
		"os":      {"ubuntu", "macos"},
		"version": {"16", "18"},
	}

	combinations := builder.generateMatrixCombinations(matrix)

	// Should generate 4 combinations: ubuntu/16, ubuntu/18, macos/16, macos/18
	if len(combinations) != 4 {
		t.Fatalf("generateMatrixCombinations() generated %d combinations, want 4", len(combinations))
	}

	// Check that all expected combinations exist
	expectedCombos := []map[string]any{
		{"os": "ubuntu", "version": "16"},
		{"os": "ubuntu", "version": "18"},
		{"os": "macos", "version": "16"},
		{"os": "macos", "version": "18"},
	}

	for _, expected := range expectedCombos {
		found := false
		for _, actual := range combinations {
			if actual["os"] == expected["os"] && actual["version"] == expected["version"] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("generateMatrixCombinations() missing expected combination: %v", expected)
		}
	}
}

func TestBuilder_Build_WithMatrix(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"matrix-test": {
				Help: "Matrix test recipe",
				Matrix: map[string][]any{
					"os":      {"ubuntu", "macos"},
					"version": {"16", "18"},
				},
				Run: model.Step{Lines: []string{"echo 'Testing {{ .matrix_os }}/{{ .matrix_version }}'"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{
		Vars:        make(map[string]any),
		Env:         make(map[string]string),
		Secrets:     make(map[string]string),
		Flags:       make(map[string]any),
		Positionals: make(map[string]any),
		OS:          "linux",
		Arch:        "amd64",
	}

	plan, err := builder.Build("matrix-test", ctx)
	if err != nil {
		t.Errorf("Build() unexpected error: %v", err)
		return
	}

	// Should generate 4 nodes (2 os * 2 versions)
	if len(plan.Nodes) != 4 {
		t.Errorf("Build() generated %d nodes, want 4", len(plan.Nodes))
	}

	// Check that all nodes have matrix variables in their context
	for i, node := range plan.Nodes {
		if node.Context.Vars["matrix_os"] == nil {
			t.Errorf("Build() node %d missing matrix_os variable", i)
		}
		if node.Context.Vars["matrix_version"] == nil {
			t.Errorf("Build() node %d missing matrix_version variable", i)
		}

		// Check that node ID includes matrix index
		expectedPrefix := "matrix-test["
		if !contains(node.ID, expectedPrefix) {
			t.Errorf("Build() node %d ID %q doesn't contain expected prefix %q", i, node.ID, expectedPrefix)
		}
	}
}

func TestBuilder_Build_MatrixWithDependencies(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"setup": {
				Help: "Setup recipe",
				Run:  model.Step{Lines: []string{"echo 'Setting up'"}},
			},
			"matrix-test": {
				Help: "Matrix test recipe",
				Deps: []string{"setup"},
				Matrix: map[string][]any{
					"arch": {"amd64", "arm64"},
				},
				Run: model.Step{Lines: []string{"echo 'Testing {{ .matrix_arch }}'"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{
		Vars:        make(map[string]any),
		Env:         make(map[string]string),
		Secrets:     make(map[string]string),
		Flags:       make(map[string]any),
		Positionals: make(map[string]any),
		OS:          "linux",
		Arch:        "amd64",
	}

	plan, err := builder.Build("matrix-test", ctx)
	if err != nil {
		t.Errorf("Build() unexpected error: %v", err)
		return
	}

	// Should have 3 nodes: 1 setup + 2 matrix nodes
	if len(plan.Nodes) != 3 {
		t.Errorf("Build() generated %d nodes, want 3", len(plan.Nodes))
	}

	// Check that setup node exists
	setupFound := false
	matrixNodes := 0
	for _, node := range plan.Nodes {
		if node.ID == "setup" {
			setupFound = true
		} else if contains(node.ID, "matrix-test[") {
			matrixNodes++
		}
	}

	if !setupFound {
		t.Error("Build() setup dependency node not found")
	}

	if matrixNodes != 2 {
		t.Errorf("Build() found %d matrix nodes, want 2", matrixNodes)
	}

	// Check that matrix nodes depend on setup
	for _, node := range plan.Nodes {
		if contains(node.ID, "matrix-test[") {
			if len(node.DependsOn) != 1 || node.DependsOn[0] != "setup" {
				t.Errorf("Build() matrix node %q dependencies = %v, want [setup]", node.ID, node.DependsOn)
			}
		}
	}
}

func TestBuilder_Build_NonMatrixRecipe(t *testing.T) {
	spec := &model.Spec{
		Recipes: map[string]model.Recipe{
			"regular-test": {
				Help: "Regular test recipe",
				Run:  model.Step{Lines: []string{"echo 'Regular test'"}},
			},
		},
	}

	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{
		Vars:        make(map[string]any),
		Env:         make(map[string]string),
		Secrets:     make(map[string]string),
		Flags:       make(map[string]any),
		Positionals: make(map[string]any),
		OS:          "linux",
		Arch:        "amd64",
	}

	plan, err := builder.Build("regular-test", ctx)
	if err != nil {
		t.Errorf("Build() unexpected error: %v", err)
		return
	}

	// Should have exactly 1 node
	if len(plan.Nodes) != 1 {
		t.Errorf("Build() generated %d nodes, want 1", len(plan.Nodes))
	}

	// Node should not have matrix variables
	node := plan.Nodes[0]
	if node.Context.Vars["matrix_os"] != nil {
		t.Error("Build() non-matrix node should not have matrix_os variable")
	}

	// Node ID should be the recipe name
	if node.ID != "regular-test" {
		t.Errorf("Build() node ID = %q, want %q", node.ID, "regular-test")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
