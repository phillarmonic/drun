package engine

import (
	"io"
	"testing"

	"github.com/phillarmonic/drun/internal/types"
)

// BenchmarkVariableInterpolation benchmarks the optimized variable interpolation
func BenchmarkVariableInterpolation(b *testing.B) {
	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{
		Parameters: map[string]*types.Value{
			"name":    mustCreateValue("test-app"),
			"version": mustCreateValue("1.2.3"),
			"env":     mustCreateValue("production"),
		},
		Variables: map[string]string{
			"branch": "main",
			"commit": "abc123",
		},
	}

	message := "Deploying {name} version {version} to {env} from branch {branch} (commit: {commit})"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.interpolateVariables(message, ctx)
	}
}

// BenchmarkVariableInterpolationSimple benchmarks simple variable interpolation
func BenchmarkVariableInterpolationSimple(b *testing.B) {
	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{
		Parameters: map[string]*types.Value{
			"name": mustCreateValue("test"),
		},
	}

	message := "Hello {name}!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.interpolateVariables(message, ctx)
	}
}

// BenchmarkVariableInterpolationComplex benchmarks complex variable interpolation
func BenchmarkVariableInterpolationComplex(b *testing.B) {
	engine := NewEngine(io.Discard)
	ctx := &ExecutionContext{
		Parameters: map[string]*types.Value{
			"app":     mustCreateValue("my-application"),
			"version": mustCreateValue("2.1.0"),
			"env":     mustCreateValue("staging"),
			"region":  mustCreateValue("us-west-2"),
		},
		Variables: map[string]string{
			"timestamp": "2024-01-15T10:30:00Z",
			"user":      "deploy-bot",
			"build":     "12345",
		},
	}

	message := "Deploying {app} v{version} to {env} in {region} at {timestamp} by {user} (build #{build})"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.interpolateVariables(message, ctx)
	}
}

// BenchmarkCreateExecutionContext benchmarks execution context creation with pre-allocated maps
func BenchmarkCreateExecutionContext(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &ExecutionContext{
			Parameters: make(map[string]*types.Value, 8),
			Variables:  make(map[string]string, 16),
		}

		// Simulate typical usage
		ctx.Parameters["name"] = mustCreateValue("test")
		ctx.Parameters["version"] = mustCreateValue("1.0.0")
		ctx.Variables["branch"] = "main"
		ctx.Variables["commit"] = "abc123"

		_ = ctx
	}
}

// BenchmarkCreateLoopContext benchmarks loop context creation
func BenchmarkCreateLoopContext(b *testing.B) {
	engine := NewEngine(io.Discard)
	parentCtx := &ExecutionContext{
		Parameters: map[string]*types.Value{
			"name":    mustCreateValue("test"),
			"version": mustCreateValue("1.0.0"),
		},
		Variables: map[string]string{
			"branch": "main",
			"commit": "abc123",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loopCtx := engine.createLoopContext(parentCtx, "item", "value")
		_ = loopCtx
	}
}

// Helper function to create typed values for benchmarks
func mustCreateValue(value string) *types.Value {
	v, err := types.NewValue(types.StringType, value)
	if err != nil {
		panic(err)
	}
	return v
}
