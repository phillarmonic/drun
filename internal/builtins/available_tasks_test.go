package builtins

import (
	"reflect"
	"testing"
)

type taskListingContext struct {
	names []string
}

func (c taskListingContext) GetProjectName() string            { return "test" }
func (c taskListingContext) GetSecretsManager() SecretsManager { return nil }
func (c taskListingContext) IsDryRun() bool                    { return false }
func (c taskListingContext) GetTaskNames() []string            { return c.names }

func TestAvailableTasksBuiltin(t *testing.T) {
	ctx := taskListingContext{names: []string{"default", "lint", "build"}}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "default separator", want: "default, lint, build"},
		{name: "custom separator", args: []string{" | "}, want: "default | lint | build"},
		{name: "escaped newline", args: []string{`\n`}, want: "default\nlint\nbuild"},
		{name: "empty separator", args: []string{""}, want: "defaultlintbuild"},
		{name: "omitted tasks", args: []string{", ", "default", "build"}, want: "lint"},
		{name: "unknown omission ignored", args: []string{", ", "missing"}, want: "default, lint, build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CallBuiltin("available tasks", ctx, tt.args...)
			if err != nil {
				t.Fatalf("CallBuiltin() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CallBuiltin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAvailableTasksBuiltinRequiresTaskContext(t *testing.T) {
	if _, err := CallBuiltin("available tasks", nil); err == nil {
		t.Fatal("CallBuiltin() error = nil, want task listing support error")
	}
}
