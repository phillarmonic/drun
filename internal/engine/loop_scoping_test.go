package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
	"github.com/phillarmonic/drun/internal/types"
)

func TestEngine_LoopVariableScoping(t *testing.T) {
	input := `version: 2.0
task "scoping test":
	let $global_var = "global_value"
	info "Before loop: global = {$global_var}"
	
	for each $item in ["apple", "banana"]:
		info "Inside loop: item = {$item}, global = {$global_var}"
		let $loop_local = "local_{$item}"
		info "Inside loop: loop_local = {$loop_local}"
	
	info "After loop: global = {$global_var}"
	info "After loop: item = {$item}"
	info "After loop: loop_local = {$loop_local}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)
	engine.SetAllowUndefinedVars(true) // Allow undefined variables for scoping test

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "scoping test")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that global variable is accessible throughout
	if !strings.Contains(outputStr, "Before loop: global = global_value") {
		t.Error("Global variable should be accessible before loop")
	}
	if !strings.Contains(outputStr, "Inside loop: item = apple, global = global_value") {
		t.Error("Global variable should be accessible inside loop")
	}
	if !strings.Contains(outputStr, "After loop: global = global_value") {
		t.Error("Global variable should be accessible after loop")
	}

	// Check that loop variables are scoped correctly
	if !strings.Contains(outputStr, "Inside loop: item = apple") {
		t.Error("Loop variable should be accessible inside loop")
	}
	if !strings.Contains(outputStr, "Inside loop: item = banana") {
		t.Error("Loop variable should be accessible for all iterations")
	}

	// Check that loop variables are NOT accessible after loop (should remain as placeholders)
	if !strings.Contains(outputStr, "After loop: item = {$item}") {
		t.Error("Loop variable should NOT be accessible after loop")
	}
	if !strings.Contains(outputStr, "After loop: loop_local = {$loop_local}") {
		t.Error("Loop-local variable should NOT be accessible after loop")
	}
}

func TestEngine_NestedLoopScoping(t *testing.T) {
	input := `version: 2.0
task "nested scoping":
	let $global = "global"
	
	for each $outer in ["1", "2"]:
		info "Outer: outer = {$outer}, global = {$global}"
		
		for each $inner in ["a", "b"]:
			info "Inner: outer = {$outer}, inner = {$inner}, global = {$global}"
		
		info "Back to outer: outer = {$outer}, inner = {$inner}, global = {$global}"
	
	info "After all: outer = {$outer}, inner = {$inner}, global = {$global}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)
	engine.SetAllowUndefinedVars(true) // Allow undefined variables for scoping test

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "nested scoping")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that both variables are accessible in inner loop
	if !strings.Contains(outputStr, "Inner: outer = 1, inner = a, global = global") {
		t.Error("Both outer and inner variables should be accessible in inner loop")
	}

	// Check that inner variable is NOT accessible in outer loop after inner loop completes
	if !strings.Contains(outputStr, "Back to outer: outer = 1, inner = {$inner}, global = global") {
		t.Error("Inner loop variable should NOT be accessible in outer loop after inner loop")
	}

	// Check that neither loop variable is accessible after all loops
	if !strings.Contains(outputStr, "After all: outer = {$outer}, inner = {$inner}, global = global") {
		t.Error("No loop variables should be accessible after all loops complete")
	}
}

func TestEngine_ParallelLoopScoping(t *testing.T) {
	input := `version: 2.0
task "parallel scoping":
	let $shared = "shared_value"
	
	for each $item in ["a", "b", "c"] in parallel:
		info "Parallel worker: item = {$item}, shared = {$shared}"
	
	info "After parallel: item = {$item}, shared = {$shared}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)
	engine.SetAllowUndefinedVars(true) // Allow undefined variables for scoping test

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "parallel scoping")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check parallel execution
	if !strings.Contains(outputStr, "Would execute 3 items in parallel") {
		t.Error("Should execute in parallel")
	}

	// Check that each worker has its own variable scope
	if !strings.Contains(outputStr, "$item = a") {
		t.Error("Worker should have access to item 'a'")
	}
	if !strings.Contains(outputStr, "$item = b") {
		t.Error("Worker should have access to item 'b'")
	}
	if !strings.Contains(outputStr, "$item = c") {
		t.Error("Worker should have access to item 'c'")
	}

	// Check that loop variable is not accessible after parallel loop
	if !strings.Contains(outputStr, "After parallel: item = {$item}, shared = shared_value") {
		t.Error("Loop variable should NOT be accessible after parallel loop")
	}
}

func TestEngine_LoopContextIsolation(t *testing.T) {
	// Test that loop contexts don't affect parent context
	input := `version: 2.0
task "context isolation":
	let $counter = "0"
	info "Initial counter: {$counter}"
	
	for each $item in ["test1", "test2"]:
		let $counter = "loop_{$item}"
		info "Loop counter: {$counter}"
	
	info "Final counter: {$counter}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)
	engine.SetAllowUndefinedVars(true) // Allow undefined variables for scoping test

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "context isolation")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that parent variable is not affected by loop variable shadowing
	if !strings.Contains(outputStr, "Initial counter: 0") {
		t.Error("Initial counter should be 0")
	}
	if !strings.Contains(outputStr, "Loop counter: loop_test1") {
		t.Error("Loop should shadow parent variable")
	}
	if !strings.Contains(outputStr, "Loop counter: loop_test2") {
		t.Error("Loop should shadow parent variable in each iteration")
	}
	if !strings.Contains(outputStr, "Final counter: 0") {
		t.Error("Parent variable should be unchanged after loop")
	}
}

func TestEngine_CreateLoopContext(t *testing.T) {
	engine := NewEngine(nil)

	// Create parent context
	parentCtx := &ExecutionContext{
		Parameters: make(map[string]*types.Value),
		Variables:  make(map[string]string),
	}
	parentCtx.Variables["parent_var"] = "parent_value"
	parentCtx.Variables["$global"] = "global_value"

	// Create loop context
	loopCtx := engine.createLoopContext(parentCtx, "$item", "test_item")

	// Check that parent variables are inherited
	if loopCtx.Variables["parent_var"] != "parent_value" {
		t.Error("Parent variables should be inherited in loop context")
	}
	if loopCtx.Variables["$global"] != "global_value" {
		t.Error("Parent variables should be inherited in loop context")
	}

	// Check that loop variable is added
	if loopCtx.Variables["$item"] != "test_item" {
		t.Error("Loop variable should be added to loop context")
	}

	// Check that parent context is not modified
	if _, exists := parentCtx.Variables["$item"]; exists {
		t.Error("Loop variable should NOT be added to parent context")
	}

	// Modify loop context and ensure parent is not affected
	loopCtx.Variables["loop_only"] = "loop_value"
	if _, exists := parentCtx.Variables["loop_only"]; exists {
		t.Error("Loop-only variables should NOT affect parent context")
	}
}

func TestEngine_LoopVariableInterpolation(t *testing.T) {
	input := `version: 2.0
task "interpolation test":
	for each $platform in ["linux", "darwin"]:
		for each $arch in ["amd64", "arm64"]:
			info "Building: {$platform}-{$arch}"
			step "Compile for {$platform}/{$arch}"
			let $target = "{$platform}-{$arch}-binary"
			info "Target: {$target}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)
	engine.SetAllowUndefinedVars(true) // Allow undefined variables for scoping test

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "interpolation test")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that loop variables are properly interpolated in nested contexts
	expectedOutputs := []string{
		"Building: linux-amd64",
		"Compile for linux/amd64",
		"Target: linux-amd64-binary",
		"Building: linux-arm64",
		"Compile for linux/arm64",
		"Target: linux-arm64-binary",
		"Building: darwin-amd64",
		"Compile for darwin/amd64",
		"Target: darwin-amd64-binary",
		"Building: darwin-arm64",
		"Compile for darwin/arm64",
		"Target: darwin-arm64-binary",
	}

	for _, expected := range expectedOutputs {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
		}
	}
}
