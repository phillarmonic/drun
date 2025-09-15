package dag

import (
	"fmt"
	"testing"

	"github.com/phillarmonic/drun/internal/model"
)

// BenchmarkBuilder_Build benchmarks DAG building performance
func BenchmarkBuilder_Build(b *testing.B) {
	spec := createTestSpec(10) // 10 recipes
	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.Build("final", ctx)
		if err != nil {
			b.Fatalf("Failed to build DAG: %v", err)
		}
	}
}

// BenchmarkBuilder_Build_Large benchmarks DAG building with many recipes
func BenchmarkBuilder_Build_Large(b *testing.B) {
	spec := createTestSpec(100) // 100 recipes
	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.Build("final", ctx)
		if err != nil {
			b.Fatalf("Failed to build large DAG: %v", err)
		}
	}
}

// BenchmarkBuilder_Build_Complex benchmarks complex dependency patterns
func BenchmarkBuilder_Build_Complex(b *testing.B) {
	spec := createComplexSpec()
	builder := NewBuilder(spec)
	ctx := &model.ExecutionContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.Build("deploy", ctx)
		if err != nil {
			b.Fatalf("Failed to build complex DAG: %v", err)
		}
	}
}

// BenchmarkBuilder_topologicalSort benchmarks the topological sort algorithm
func BenchmarkBuilder_topologicalSort(b *testing.B) {
	builder := NewBuilder(&model.Spec{})

	// Create a large graph for sorting
	nodes := make([]model.PlanNode, 100)
	edges := make([][2]int, 0)

	for i := 0; i < 100; i++ {
		nodes[i] = model.PlanNode{
			ID: fmt.Sprintf("node%d", i),
			Recipe: &model.Recipe{
				Run: model.Step{Lines: []string{fmt.Sprintf("echo 'node %d'", i)}},
			},
		}
		// Create dependencies: each node depends on the previous 2-3 nodes
		if i > 0 {
			edges = append(edges, [2]int{i - 1, i})
		}
		if i > 1 {
			edges = append(edges, [2]int{i - 2, i})
		}
		if i > 2 && i%3 == 0 {
			edges = append(edges, [2]int{i - 3, i})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := builder.topologicalSort(nodes, edges)
		if err != nil {
			b.Fatalf("Failed to sort: %v", err)
		}
	}
}

// createTestSpec creates a test specification with the given number of recipes
func createTestSpec(numRecipes int) *model.Spec {
	recipes := make(map[string]model.Recipe)

	// Create a chain of dependencies
	for i := 0; i < numRecipes; i++ {
		name := fmt.Sprintf("task%d", i)
		var deps []string

		if i > 0 {
			deps = append(deps, fmt.Sprintf("task%d", i-1))
		}

		// Add some parallel dependencies
		if i > 2 && i%3 == 0 {
			deps = append(deps, fmt.Sprintf("task%d", i-2))
		}

		recipes[name] = model.Recipe{
			Help: fmt.Sprintf("Task %d", i),
			Deps: deps,
			Run: model.Step{
				Lines: []string{fmt.Sprintf("echo 'Running task %d'", i)},
			},
		}
	}

	// Add a final task that depends on the last few tasks
	finalDeps := []string{}
	for i := max(0, numRecipes-3); i < numRecipes; i++ {
		finalDeps = append(finalDeps, fmt.Sprintf("task%d", i))
	}

	recipes["final"] = model.Recipe{
		Help: "Final task",
		Deps: finalDeps,
		Run: model.Step{
			Lines: []string{"echo 'Final task complete'"},
		},
	}

	return &model.Spec{
		Recipes: recipes,
	}
}

// createComplexSpec creates a more realistic complex specification
func createComplexSpec() *model.Spec {
	return &model.Spec{
		Recipes: map[string]model.Recipe{
			"deps": {
				Help: "Install dependencies",
				Run:  model.Step{Lines: []string{"npm install"}},
			},
			"lint": {
				Help: "Run linting",
				Deps: []string{"deps"},
				Run:  model.Step{Lines: []string{"npm run lint"}},
			},
			"test-unit": {
				Help: "Run unit tests",
				Deps: []string{"deps"},
				Run:  model.Step{Lines: []string{"npm run test:unit"}},
			},
			"test-integration": {
				Help: "Run integration tests",
				Deps: []string{"deps"},
				Run:  model.Step{Lines: []string{"npm run test:integration"}},
			},
			"test-e2e": {
				Help: "Run e2e tests",
				Deps: []string{"deps"},
				Run:  model.Step{Lines: []string{"npm run test:e2e"}},
			},
			"security": {
				Help: "Security audit",
				Deps: []string{"deps"},
				Run:  model.Step{Lines: []string{"npm audit"}},
			},
			"build-frontend": {
				Help: "Build frontend",
				Deps: []string{"deps", "lint", "test-unit"},
				Run:  model.Step{Lines: []string{"npm run build:frontend"}},
			},
			"build-backend": {
				Help: "Build backend",
				Deps: []string{"deps", "lint", "test-unit"},
				Run:  model.Step{Lines: []string{"npm run build:backend"}},
			},
			"build-docs": {
				Help: "Build documentation",
				Deps: []string{"deps"},
				Run:  model.Step{Lines: []string{"npm run build:docs"}},
			},
			"package": {
				Help: "Package application",
				Deps: []string{"build-frontend", "build-backend", "test-integration"},
				Run:  model.Step{Lines: []string{"npm run package"}},
			},
			"docker-build": {
				Help: "Build Docker image",
				Deps: []string{"package"},
				Run:  model.Step{Lines: []string{"docker build -t myapp ."}},
			},
			"deploy-staging": {
				Help: "Deploy to staging",
				Deps: []string{"docker-build", "test-e2e", "security"},
				Run:  model.Step{Lines: []string{"kubectl apply -f staging/"}},
			},
			"deploy-prod": {
				Help: "Deploy to production",
				Deps: []string{"deploy-staging"},
				Run:  model.Step{Lines: []string{"kubectl apply -f production/"}},
			},
			"deploy": {
				Help: "Full deployment pipeline",
				Deps: []string{"deploy-prod", "build-docs"},
				Run:  model.Step{Lines: []string{"echo 'Deployment complete'"}},
			},
		},
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
