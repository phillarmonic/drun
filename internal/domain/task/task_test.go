package task

import (
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    *Task
		wantErr bool
	}{
		{
			name: "valid task",
			task: &Task{
				Name:        "test",
				Description: "Test task",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			task: &Task{
				Name:        "",
				Description: "Test task",
			},
			wantErr: true,
		},
		{
			name: "valid with parameters",
			task: &Task{
				Name: "test",
				Parameters: []Parameter{
					{Name: "param1", Type: "given"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Task.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTask_FullName(t *testing.T) {
	tests := []struct {
		name     string
		task     *Task
		wantName string
	}{
		{
			name: "no namespace",
			task: &Task{
				Name:      "mytask",
				Namespace: "",
			},
			wantName: "mytask",
		},
		{
			name: "with namespace",
			task: &Task{
				Name:      "mytask",
				Namespace: "myns",
			},
			wantName: "myns.mytask",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.FullName()
			if got != tt.wantName {
				t.Errorf("Task.FullName() = %v, want %v", got, tt.wantName)
			}
		})
	}
}

func TestTask_HasParameter(t *testing.T) {
	task := &Task{
		Name: "test",
		Parameters: []Parameter{
			{Name: "param1", Type: "given"},
			{Name: "param2", Type: "requires"},
		},
	}

	tests := []struct {
		name      string
		paramName string
		want      bool
	}{
		{"existing param", "param1", true},
		{"another existing", "param2", true},
		{"non-existing", "param3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := task.HasParameter(tt.paramName)
			if got != tt.want {
				t.Errorf("Task.HasParameter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_GetParameter(t *testing.T) {
	task := &Task{
		Name: "test",
		Parameters: []Parameter{
			{Name: "param1", Type: "given"},
		},
	}

	param, ok := task.GetParameter("param1")
	if !ok {
		t.Error("GetParameter() should find param1")
	}
	if param.Name != "param1" {
		t.Errorf("GetParameter() name = %v, want param1", param.Name)
	}

	_, ok = task.GetParameter("nonexistent")
	if ok {
		t.Error("GetParameter() should not find nonexistent param")
	}
}

func TestTask_HasDependencies(t *testing.T) {
	tests := []struct {
		name string
		task *Task
		want bool
	}{
		{
			name: "no dependencies",
			task: &Task{Name: "test"},
			want: false,
		},
		{
			name: "with dependencies",
			task: &Task{
				Name: "test",
				Dependencies: []Dependency{
					{Name: "dep1"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.HasDependencies()
			if got != tt.want {
				t.Errorf("Task.HasDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	astTask := &ast.TaskStatement{
		Name:        "test-task",
		Description: "Test description",
		Parameters: []ast.ParameterStatement{
			{
				Name: "param1",
				Type: "given",
			},
		},
		Dependencies: []ast.DependencyGroup{
			{
				Dependencies: []ast.DependencyItem{
					{Name: "dep1", Parallel: false},
				},
			},
		},
		Body: []ast.Statement{},
	}

	task, err := NewTask(astTask, "testns", "test.drun")
	if err != nil {
		t.Fatalf("NewTask() error = %v", err)
	}

	if task.Name != "test-task" {
		t.Errorf("Name = %v, want test-task", task.Name)
	}
	if task.Description != "Test description" {
		t.Errorf("Description = %v, want Test description", task.Description)
	}
	if task.Namespace != "testns" {
		t.Errorf("Namespace = %v, want testns", task.Namespace)
	}
	if task.Source != "test.drun" {
		t.Errorf("Source = %v, want test.drun", task.Source)
	}
	if len(task.Parameters) != 1 {
		t.Errorf("Parameters length = %v, want 1", len(task.Parameters))
	}
	if len(task.Dependencies) != 1 {
		t.Errorf("Dependencies length = %v, want 1", len(task.Dependencies))
	}
}

func TestResolveInheritedToolRequirements_FlattensNestedTaskRefs(t *testing.T) {
	registry := NewRegistry()
	for _, task := range []*Task{
		{
			Name: "build",
			Body: []statement.Statement{
				&statement.RequiresTools{Tools: []statement.ToolRequirement{{Name: "go"}}},
			},
		},
		{
			Name: "lint",
			Body: []statement.Statement{
				&statement.RequiresTools{
					Tools:    []statement.ToolRequirement{{Name: "golangci-lint"}},
					TaskRefs: []string{"build"},
				},
			},
		},
		{
			Name: "security",
			Body: []statement.Statement{
				&statement.RequiresTools{
					Tools:    []statement.ToolRequirement{{Name: "gosec"}},
					TaskRefs: []string{"lint"},
				},
			},
		},
	} {
		if err := registry.Register(task); err != nil {
			t.Fatalf("Register(%s) error = %v", task.Name, err)
		}
	}

	if err := ResolveInheritedToolRequirements(registry); err != nil {
		t.Fatalf("ResolveInheritedToolRequirements() error = %v", err)
	}

	security, err := registry.Get("security")
	if err != nil {
		t.Fatalf("Get(security) error = %v", err)
	}
	requiresTools := security.Body[0].(*statement.RequiresTools)
	wantTools := []string{"gosec", "golangci-lint", "go"}
	if len(requiresTools.Tools) != len(wantTools) {
		t.Fatalf("Tools length = %d, want %d: %#v", len(requiresTools.Tools), len(wantTools), requiresTools.Tools)
	}
	for i, want := range wantTools {
		if got := requiresTools.Tools[i].Name; got != want {
			t.Fatalf("Tools[%d] = %q, want %q", i, got, want)
		}
	}
	if len(requiresTools.TaskRefs) != 0 {
		t.Fatalf("TaskRefs = %#v, want flattened refs to be cleared", requiresTools.TaskRefs)
	}
}

func TestResolveInheritedProjectToolRequirements_AfterTaskFlattenDoesNotDuplicate(t *testing.T) {
	registry := NewRegistry()
	for _, task := range []*Task{
		{
			Name: "build",
			Body: []statement.Statement{
				&statement.RequiresTools{Tools: []statement.ToolRequirement{{Name: "go"}}},
			},
		},
		{
			Name: "lint",
			Body: []statement.Statement{
				&statement.RequiresTools{
					Tools:    []statement.ToolRequirement{{Name: "golangci-lint"}},
					TaskRefs: []string{"build"},
				},
			},
		},
	} {
		if err := registry.Register(task); err != nil {
			t.Fatalf("Register(%s) error = %v", task.Name, err)
		}
	}
	if err := ResolveInheritedToolRequirements(registry); err != nil {
		t.Fatalf("ResolveInheritedToolRequirements() error = %v", err)
	}

	tools, err := ResolveInheritedProjectToolRequirements(registry, []string{"lint"})
	if err != nil {
		t.Fatalf("ResolveInheritedProjectToolRequirements() error = %v", err)
	}

	wantTools := []string{"golangci-lint", "go"}
	if len(tools) != len(wantTools) {
		t.Fatalf("Tools length = %d, want %d: %#v", len(tools), len(wantTools), tools)
	}
	for i, want := range wantTools {
		if got := tools[i].Name; got != want {
			t.Fatalf("Tools[%d] = %q, want %q", i, got, want)
		}
	}
}

func TestResolveInheritedToolRequirements_MissingRef(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(&Task{
		Name: "security",
		Body: []statement.Statement{
			&statement.RequiresTools{TaskRefs: []string{"missing"}},
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	err := ResolveInheritedToolRequirements(registry)
	if err == nil {
		t.Fatal("ResolveInheritedToolRequirements() error = nil, want missing ref error")
	}
	if !strings.Contains(err.Error(), `requires tools from task "missing" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveInheritedToolRequirements_Cycle(t *testing.T) {
	registry := NewRegistry()
	for _, task := range []*Task{
		{
			Name: "a",
			Body: []statement.Statement{
				&statement.RequiresTools{TaskRefs: []string{"b"}},
			},
		},
		{
			Name: "b",
			Body: []statement.Statement{
				&statement.RequiresTools{TaskRefs: []string{"a"}},
			},
		},
	} {
		if err := registry.Register(task); err != nil {
			t.Fatalf("Register(%s) error = %v", task.Name, err)
		}
	}

	err := ResolveInheritedToolRequirements(registry)
	if err == nil {
		t.Fatal("ResolveInheritedToolRequirements() error = nil, want cycle error")
	}
	if !strings.Contains(err.Error(), "circular requires-tools inheritance detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}
