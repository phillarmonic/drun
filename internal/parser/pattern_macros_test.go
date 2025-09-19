package parser

import (
	"testing"

	"github.com/phillarmonic/drun/internal/lexer"
)

func TestParser_PatternMacros(t *testing.T) {
	input := `version: 2.0

task "pattern_macros":
  requires $version as string matching semver
  requires $id as string matching uuid
  requires $endpoint as string matching url
  requires $ip as string matching ipv4
  requires $name as string matching slug
  requires $tag as string matching docker_tag
  requires $branch as string matching git_branch
  given $extended_version as string matching semver_extended defaults to "v1.0.0-beta.1"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	if len(program.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(program.Tasks))
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 8 {
		t.Fatalf("Expected 8 parameters, got %d", len(task.Parameters))
	}

	// Test each pattern macro parameter
	expectedMacros := []struct {
		name  string
		macro string
	}{
		{"version", "semver"},
		{"id", "uuid"},
		{"endpoint", "url"},
		{"ip", "ipv4"},
		{"name", "slug"},
		{"tag", "docker_tag"},
		{"branch", "git_branch"},
		{"extended_version", "semver_extended"},
	}

	for i, expected := range expectedMacros {
		param := task.Parameters[i]
		if param.Name != expected.name {
			t.Errorf("Parameter %d: expected name %s, got %s", i, expected.name, param.Name)
		}
		if param.PatternMacro != expected.macro {
			t.Errorf("Parameter %d: expected macro %s, got %s", i, expected.macro, param.PatternMacro)
		}
		if param.DataType != "string" {
			t.Errorf("Parameter %d: expected type string, got %s", i, param.DataType)
		}
	}
}

func TestParser_PatternMacroVsRawPattern(t *testing.T) {
	input := `version: 2.0

task "mixed_patterns":
  requires $version as string matching semver
  requires $custom as string matching pattern "^custom-[0-9]+$"`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(task.Parameters))
	}

	// Test pattern macro parameter
	versionParam := task.Parameters[0]
	if versionParam.PatternMacro != "semver" {
		t.Errorf("Expected PatternMacro 'semver', got '%s'", versionParam.PatternMacro)
	}
	if versionParam.Pattern != "" {
		t.Errorf("Expected empty Pattern for macro, got '%s'", versionParam.Pattern)
	}

	// Test raw pattern parameter
	customParam := task.Parameters[1]
	if customParam.Pattern != "^custom-[0-9]+$" {
		t.Errorf("Expected Pattern '^custom-[0-9]+$', got '%s'", customParam.Pattern)
	}
	if customParam.PatternMacro != "" {
		t.Errorf("Expected empty PatternMacro for raw pattern, got '%s'", customParam.PatternMacro)
	}
}

func TestParser_PatternMacroWithConstraints(t *testing.T) {
	input := `version: 2.0

task "macro_with_constraints":
  given $version as string matching semver defaults to "v1.0.0"
  given $env as string defaults to "dev" from ["dev", "staging", "production"]`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	task := program.Tasks[0]
	if len(task.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(task.Parameters))
	}

	// Test pattern macro with default
	versionParam := task.Parameters[0]
	if versionParam.PatternMacro != "semver" {
		t.Errorf("Expected PatternMacro 'semver', got '%s'", versionParam.PatternMacro)
	}
	if versionParam.DefaultValue != "v1.0.0" {
		t.Errorf("Expected DefaultValue 'v1.0.0', got '%s'", versionParam.DefaultValue)
	}

	// Test list constraints with default
	envParam := task.Parameters[1]
	if envParam.DefaultValue != "dev" {
		t.Errorf("Expected DefaultValue 'dev', got '%s'", envParam.DefaultValue)
	}
	expectedConstraints := []string{"dev", "staging", "production"}
	if len(envParam.Constraints) != len(expectedConstraints) {
		t.Errorf("Expected %d constraints, got %d", len(expectedConstraints), len(envParam.Constraints))
	}
}

func TestParser_InvalidPatternMacro(t *testing.T) {
	input := `version: 2.0

task "invalid_macro":
  requires $version as string matching nonexistent_macro`

	l := lexer.NewLexer(input)
	p := NewParser(l)
	program := p.ParseProgram()

	// This should parse successfully (validation happens at runtime)
	if len(p.Errors()) != 0 {
		t.Fatalf("Parser should not error on unknown macro names: %v", p.Errors())
	}

	task := program.Tasks[0]
	param := task.Parameters[0]
	if param.PatternMacro != "nonexistent_macro" {
		t.Errorf("Expected PatternMacro 'nonexistent_macro', got '%s'", param.PatternMacro)
	}
}
