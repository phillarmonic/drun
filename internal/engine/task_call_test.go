package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
)

func TestTaskCall(t *testing.T) {
	input := `
version: 2.0

task "hello":
  info "Hello from hello task"

task "greet":
  requires $name
  info "Hello, {$name}!"

task "main":
  info "Starting main task"
  call task "hello"
  call task "greet" with name="World"
  info "Main task completed"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	err := engine.Execute(program, "main")
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output := buf.String()
	expected := []string{
		"Starting main task",
		"Hello from hello task",
		"Hello, World!",
		"Main task completed",
	}

	for _, exp := range expected {
		if !contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got: %s", exp, output)
		}
	}
}

func TestTaskCallWithParameters(t *testing.T) {
	input := `
version: 2.0

task "multiply":
  requires $a
  requires $b
  info "Result: {$a} * {$b}"

task "calculator":
  call task "multiply" with a="5" b="3"
  info "Calculation done"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	err := engine.Execute(program, "calculator")
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output := buf.String()
	if !contains(output, "Result: 5 * 3") {
		t.Errorf("Expected output to contain 'Result: 5 * 3', got: %s", output)
	}
	if !contains(output, "Calculation done") {
		t.Errorf("Expected output to contain 'Calculation done', got: %s", output)
	}
}

func TestTaskCallWithNumericParameterLiteral(t *testing.T) {
	input := `
version: 2.0

task "fuzz":
  requires $iterations
  info "Iterations: {$iterations}"

task "main":
  call task "fuzz" with iterations=100
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	err := engine.Execute(program, "main")
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output := buf.String()
	if !contains(output, "Iterations: 100") {
		t.Errorf("Expected output to contain 'Iterations: 100', got: %s", output)
	}
}

func TestTaskCallNotFound(t *testing.T) {
	input := `
version: 2.0

task "main":
  call task "nonexistent"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	err := engine.Execute(program, "main")
	if err == nil {
		t.Fatal("Expected error for nonexistent task, got nil")
	}

	if !contains(err.Error(), "task 'nonexistent' not found") {
		t.Errorf("Expected error about task not found, got: %v", err)
	}
}

func TestTaskCallKebabNameUnquoted(t *testing.T) {
	input := `
version: 2.0

task "run-action":
  info "running action"

task "main":
  call task run-action
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngine(&buf)

	if err := engine.Execute(program, "main"); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output := buf.String()
	if !contains(output, "running action") {
		t.Errorf("Expected output to contain 'running action', got: %s", output)
	}
}

func TestCITaskModeBuffersSuccessfulShellOutput(t *testing.T) {
	input := `
version: 2.0

task "ci" mode "ci":
  info "Starting"
  run "echo hidden output"
  success "Done"
`

	var buf bytes.Buffer
	if err := ExecuteString(input, "ci", &buf); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Starting") {
		t.Fatalf("Expected action output to remain visible, got: %s", output)
	}
	if strings.Contains(output, "hidden output") {
		t.Fatalf("Expected successful shell output to stay buffered in ci mode, got: %s", output)
	}
	if !strings.Contains(output, "Done") {
		t.Fatalf("Expected success action to remain visible, got: %s", output)
	}
}

func TestCITaskModePrintsBufferedShellOutputOnFailure(t *testing.T) {
	input := `
version: 2.0

task "ci" mode "ci":
  run "echo useful output; echo noisy error >&2; exit 1"
`

	var buf bytes.Buffer
	err := ExecuteString(input, "ci", &buf)
	if err == nil {
		t.Fatal("Expected ci task to fail")
	}

	output := buf.String()
	if !strings.Contains(output, "stdout:\nuseful output") {
		t.Fatalf("Expected buffered stdout to be printed on failure, got: %s", output)
	}
	if !strings.Contains(output, "stderr:\nnoisy error") {
		t.Fatalf("Expected buffered stderr to be printed on failure, got: %s", output)
	}
	if !strings.Contains(output, "summary: command `echo useful output; echo noisy error >&2; exit 1` failed with exit code 1") {
		t.Fatalf("Expected buffered failure summary to be printed, got: %s", output)
	}
}

func TestCITaskModeIsInheritedByCalledTasks(t *testing.T) {
	input := `
version: 2.0

task "lint":
  run "echo inherited hidden output"

task "ci" mode "ci":
  call task "lint"
`

	var buf bytes.Buffer
	if err := ExecuteString(input, "ci", &buf); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if strings.Contains(buf.String(), "inherited hidden output") {
		t.Fatalf("Expected called task shell output to inherit ci buffering, got: %s", buf.String())
	}
}

func TestRuntimeTaskModeOverrideDisablesCIBuffering(t *testing.T) {
	input := `
version: 2.0

task "ci" mode "ci":
  run "echo visible output"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngineWithOptions(WithOutput(&buf), WithTaskModeOverride("normal"))

	if err := engine.Execute(program, "ci"); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if !strings.Contains(buf.String(), "visible output") {
		t.Fatalf("Expected runtime normal mode to disable ci buffering, got: %s", buf.String())
	}
}

func TestRuntimeTaskModeOverrideEnablesCIBuffering(t *testing.T) {
	input := `
version: 2.0

task "build":
  run "echo hidden output"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngineWithOptions(WithOutput(&buf), WithTaskModeOverride("ci"))

	if err := engine.Execute(program, "build"); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if strings.Contains(buf.String(), "hidden output") {
		t.Fatalf("Expected runtime ci mode to buffer shell output, got: %s", buf.String())
	}
}

func TestRuntimeTaskModeOverrideAppliesToCalledTasks(t *testing.T) {
	input := `
version: 2.0

task "lint":
  run "echo hidden called output"

task "build":
  call task "lint"
`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var buf bytes.Buffer
	engine := NewEngineWithOptions(WithOutput(&buf), WithTaskModeOverride("ci"))

	if err := engine.Execute(program, "build"); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if strings.Contains(buf.String(), "hidden called output") {
		t.Fatalf("Expected runtime ci mode to propagate to called tasks, got: %s", buf.String())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
