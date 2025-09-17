package tmpl

import (
	"strings"
	"testing"

	"github.com/phillarmonic/drun/internal/v1/model"
)

func TestEngine_RenderStep_WithRecipePrerun(t *testing.T) {
	recipePrerun := []string{
		"# Setup colors",
		"RED='\\033[0;31m'",
		"GREEN='\\033[0;32m'",
		"NC='\\033[0m'",
	}

	engine := NewEngine(nil, recipePrerun, nil)

	step := model.Step{
		Lines: []string{
			"echo Starting build",
			"echo -e \"${GREEN}Success${NC}\"",
		},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"name": "test",
		},
	}

	result, err := engine.RenderStep(step, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that recipe-prerun snippets are prepended
	if len(result.Lines) < 6 {
		t.Fatalf("Expected at least 6 lines (4 recipe-prerun + 2 original), got %d", len(result.Lines))
	}

	// Verify recipe-prerun content is at the beginning
	if !strings.Contains(result.Lines[0], "Setup colors") {
		t.Errorf("Expected first line to contain recipe-prerun comment, got: %s", result.Lines[0])
	}

	// Verify original content is preserved
	found := false
	for _, line := range result.Lines {
		if strings.Contains(line, "Starting build") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find original step content")
	}
}

func TestEngine_RenderStep_WithRecipePrerunTemplating(t *testing.T) {
	recipePrerun := []string{
		"# Project: {{ .project_name }}",
		"echo Starting {{ .project_name }}",
	}

	engine := NewEngine(nil, recipePrerun, nil)

	step := model.Step{
		Lines: []string{
			"echo Main task",
		},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{
			"project_name": "myapp",
		},
	}

	result, err := engine.RenderStep(step, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that templating worked in prerun
	found := false
	for _, line := range result.Lines {
		if strings.Contains(line, "Project: myapp") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected recipe-prerun template to be rendered with project name")
	}

	// Check that templating worked in recipe-prerun echo
	found = false
	for _, line := range result.Lines {
		if strings.Contains(line, "Starting myapp") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected recipe-prerun echo to be rendered with project name")
	}
}

func TestEngine_RenderStep_EmptyRecipePrerun(t *testing.T) {
	engine := NewEngine(nil, []string{}, nil)

	step := model.Step{
		Lines: []string{
			"echo test",
		},
	}

	ctx := &model.ExecutionContext{}

	result, err := engine.RenderStep(step, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(result.Lines))
	}

	if result.Lines[0] != "echo test" {
		t.Errorf("Expected 'echo test', got %s", result.Lines[0])
	}
}

func TestEngine_RenderStep_NilRecipePrerun(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	step := model.Step{
		Lines: []string{
			"echo test",
		},
	}

	ctx := &model.ExecutionContext{}

	result, err := engine.RenderStep(step, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(result.Lines))
	}

	if result.Lines[0] != "echo test" {
		t.Errorf("Expected 'echo test', got %s", result.Lines[0])
	}
}

func TestEngine_RenderStep_WithFlags(t *testing.T) {
	engine := NewEngine(nil, nil, nil)

	step := model.Step{
		Lines: []string{
			"{{ if .flags.verbose }}echo Verbose mode{{ end }}",
			"{{ if .coverage }}echo Coverage enabled{{ end }}",
		},
	}

	ctx := &model.ExecutionContext{
		Flags: map[string]any{
			"verbose":  true,
			"coverage": true,
		},
	}

	result, err := engine.RenderStep(step, ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have 2 lines since both conditions are true
	if len(result.Lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(result.Lines))
	}

	// Check that both .flags.verbose and direct .coverage access work
	if result.Lines[0] != "echo Verbose mode" {
		t.Errorf("Expected 'echo Verbose mode', got %s", result.Lines[0])
	}

	if result.Lines[1] != "echo Coverage enabled" {
		t.Errorf("Expected 'echo Coverage enabled', got %s", result.Lines[1])
	}
}

func TestEngine_RenderStep_WithRecipePostrun(t *testing.T) {
	recipePostrun := []string{
		"echo Recipe completed",
		"echo Cleanup finished",
	}

	engine := NewEngine(nil, nil, recipePostrun)

	step := model.Step{
		Lines: []string{
			"echo Main task",
		},
	}

	ctx := &model.ExecutionContext{
		Vars: map[string]any{},
		Env:  map[string]string{},
	}

	result, err := engine.RenderStep(step, ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have original + postrun lines
	if len(result.Lines) < 3 {
		t.Fatalf("Expected at least 3 lines, got %d", len(result.Lines))
	}

	// Verify original content comes first
	if !strings.Contains(result.Lines[0], "Main task") {
		t.Errorf("Expected first line to contain original content, got: %s", result.Lines[0])
	}

	// Verify recipe-postrun content is at the end
	foundCompleted := false
	foundCleanup := false
	for _, line := range result.Lines {
		if strings.Contains(line, "Recipe completed") {
			foundCompleted = true
		}
		if strings.Contains(line, "Cleanup finished") {
			foundCleanup = true
		}
	}
	if !foundCompleted {
		t.Error("Expected to find 'Recipe completed' in recipe-postrun")
	}
	if !foundCleanup {
		t.Error("Expected to find 'Cleanup finished' in recipe-postrun")
	}
}
