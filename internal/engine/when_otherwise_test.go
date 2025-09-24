package engine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
	"github.com/phillarmonic/drun/internal/parser"
)

func TestEngine_WhenOtherwiseExecution(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskName       string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "when condition true",
			input: `version: 2.0
task "test":
	let $platform = "windows"
	when $platform is "windows":
		info "Building for Windows OS"
		step "Windows build process"
	otherwise:
		info "Building for Unix-like OS"
		step "Building for other platform"`,
			taskName: "test",
			expectedOutput: []string{
				"Building for Windows OS",
				"Windows build process",
			},
			notExpected: []string{
				"Building for Unix-like OS",
				"Building for other platform",
			},
		},
		{
			name: "when condition false",
			input: `version: 2.0
task "test":
	let $platform = "linux"
	when $platform is "windows":
		info "Building for Windows OS"
		step "Windows build process"
	otherwise:
		info "Building for Unix-like OS"
		step "Building for other platform"`,
			taskName: "test",
			expectedOutput: []string{
				"Building for Unix-like OS",
				"Building for other platform",
			},
			notExpected: []string{
				"Building for Windows OS",
				"Windows build process",
			},
		},
		{
			name: "when without otherwise",
			input: `version: 2.0
task "test":
	let $env = "development"
	when $env is "production":
		info "Production mode"
		step "Deploy to production"
	info "Task completed"`,
			taskName: "test",
			expectedOutput: []string{
				"Task completed",
			},
			notExpected: []string{
				"Production mode",
				"Deploy to production",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)

			lexer := lexer.NewLexer(tt.input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, tt.taskName)
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()

			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestEngine_NestedWhenOtherwise(t *testing.T) {
	input := `version: 2.0
task "nested":
	let $platform = "windows"
	let $arch = "amd64"
	
	when $platform is "windows":
		info "Windows platform"
		when $arch is "amd64":
			info "x64 architecture"
			step "Build Windows x64"
		otherwise:
			info "ARM architecture"
			step "Build Windows ARM"
	otherwise:
		info "Non-Windows platform"
		step "Build for other platform"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "nested")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	expectedOutputs := []string{
		"Windows platform",
		"x64 architecture",
		"Build Windows x64",
	}

	notExpectedOutputs := []string{
		"ARM architecture",
		"Build Windows ARM",
		"Non-Windows platform",
		"Build for other platform",
	}

	for _, expected := range expectedOutputs {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, outputStr)
		}
	}

	for _, notExpected := range notExpectedOutputs {
		if strings.Contains(outputStr, notExpected) {
			t.Errorf("Expected output to NOT contain %q, got:\n%s", notExpected, outputStr)
		}
	}
}

func TestEngine_WhenOtherwiseInLoop(t *testing.T) {
	input := `version: 2.0
task "matrix":
	for each $platform in ["windows", "linux", "darwin"]:
		info "Processing {$platform}"
		when $platform is "windows":
			step "Building {$platform} with .exe extension"
		otherwise:
			step "Building {$platform} without extension"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "matrix")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that all platforms are processed
	if !strings.Contains(outputStr, "Processing windows") {
		t.Error("Expected windows processing")
	}
	if !strings.Contains(outputStr, "Processing linux") {
		t.Error("Expected linux processing")
	}
	if !strings.Contains(outputStr, "Processing darwin") {
		t.Error("Expected darwin processing")
	}

	// Check that Windows gets .exe extension
	if !strings.Contains(outputStr, "Building windows with .exe extension") {
		t.Error("Expected Windows to get .exe extension")
	}

	// Check that non-Windows platforms don't get .exe extension
	if !strings.Contains(outputStr, "Building linux without extension") {
		t.Error("Expected Linux to build without extension")
	}
	if !strings.Contains(outputStr, "Building darwin without extension") {
		t.Error("Expected Darwin to build without extension")
	}
}

func TestEngine_WhenOtherwiseConditionTypes(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		varValue  string
		expected  bool
	}{
		{
			name:      "string equality true",
			condition: "$var is \"test\"",
			varValue:  "test",
			expected:  true,
		},
		{
			name:      "string equality false",
			condition: "$var is \"test\"",
			varValue:  "other",
			expected:  false,
		},
		{
			name:      "string inequality true",
			condition: "$var is not \"test\"",
			varValue:  "other",
			expected:  true,
		},
		{
			name:      "string inequality false",
			condition: "$var is not \"test\"",
			varValue:  "test",
			expected:  false,
		},
		{
			name:      "empty check true",
			condition: "$var is empty",
			varValue:  "",
			expected:  true,
		},
		{
			name:      "empty check false",
			condition: "$var is empty",
			varValue:  "value",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `version: 2.0
task "test":
	let $var = "` + tt.varValue + `"
	when ` + tt.condition + `:
		info "Condition matched"
	otherwise:
		info "Condition did not match"`

			var output bytes.Buffer
			engine := NewEngine(&output)
			engine.SetDryRun(true)

			lexer := lexer.NewLexer(input)
			parser := parser.NewParser(lexer)
			program := parser.ParseProgram()

			if len(parser.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", parser.Errors())
			}

			err := engine.Execute(program, "test")
			if err != nil {
				t.Fatalf("Execution error: %v", err)
			}

			outputStr := output.String()

			if tt.expected {
				if !strings.Contains(outputStr, "Condition matched") {
					t.Errorf("Expected condition to match, got:\n%s", outputStr)
				}
				if strings.Contains(outputStr, "Condition did not match") {
					t.Errorf("Expected condition to match, but otherwise executed:\n%s", outputStr)
				}
			} else {
				if strings.Contains(outputStr, "Condition matched") {
					t.Errorf("Expected condition to not match, but when executed:\n%s", outputStr)
				}
				if !strings.Contains(outputStr, "Condition did not match") {
					t.Errorf("Expected condition to not match, got:\n%s", outputStr)
				}
			}
		})
	}
}

func TestEngine_WhenOtherwiseVariableScoping(t *testing.T) {
	input := `version: 2.0
task "scoping":
	let $global = "global_value"
	
	when $global is "global_value":
		let $local = "local_value"
		info "In when: global = {$global}, local = {$local}"
	otherwise:
		info "In otherwise: should not execute"
	
	info "After when: global = {$global}, local = {$local}"`

	var output bytes.Buffer
	engine := NewEngine(&output)
	engine.SetDryRun(true)

	lexer := lexer.NewLexer(input)
	parser := parser.NewParser(lexer)
	program := parser.ParseProgram()

	if len(parser.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", parser.Errors())
	}

	err := engine.Execute(program, "scoping")
	if err != nil {
		t.Fatalf("Execution error: %v", err)
	}

	outputStr := output.String()

	// Check that global variable is accessible throughout
	if !strings.Contains(outputStr, "In when: global = global_value, local = local_value") {
		t.Error("Expected variables to be accessible in when block")
	}

	// Check that local variable from when block is accessible after (same context)
	if !strings.Contains(outputStr, "After when: global = global_value, local = local_value") {
		t.Error("Expected local variable to be accessible after when block")
	}

	// Check that otherwise block doesn't execute
	if strings.Contains(outputStr, "In otherwise: should not execute") {
		t.Error("Otherwise block should not execute when condition is true")
	}
}
