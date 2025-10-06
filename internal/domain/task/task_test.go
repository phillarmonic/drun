package task

import (
	"testing"

	"github.com/phillarmonic/drun/internal/ast"
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

	task := NewTask(astTask, "testns", "test.drun")

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
