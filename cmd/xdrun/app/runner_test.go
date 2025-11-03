package app

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
)

func TestFindDefaultTaskPrefersDefaultOverEarlierStart(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "start"},
			{Name: "default"},
		},
	}

	if got := FindDefaultTask(program); got != "default" {
		t.Fatalf("expected default task to be selected, got %q", got)
	}
}

func TestFindDefaultTaskFallsBackThroughPriorityList(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "build"},
			{Name: "help"},
			{Name: "deploy"},
		},
	}

	if got := FindDefaultTask(program); got != "help" {
		t.Fatalf("expected help task to be selected, got %q", got)
	}
}

func TestFindDefaultTaskReturnsEmptyWhenNoMatch(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "build"},
			{Name: "deploy"},
		},
	}

	if got := FindDefaultTask(program); got != "" {
		t.Fatalf("expected empty default task when no match, got %q", got)
	}
}

func TestFindDefaultTaskDoesNotAutoRunStart(t *testing.T) {
	program := &ast.Program{
		Tasks: []*ast.TaskStatement{
			{Name: "start"},
		},
	}

	if got := FindDefaultTask(program); got != "" {
		t.Fatalf("expected empty default task when only start is defined, got %q", got)
	}
}
