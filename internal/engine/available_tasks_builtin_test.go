package engine

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/platform"
)

func TestAvailableTasksBuiltinUsesProgramDeclarationOrder(t *testing.T) {
	currentPlatform := platform.Current()
	otherPlatform := platform.Linux
	if currentPlatform == otherPlatform {
		otherPlatform = platform.Mac
	}

	input := fmt.Sprintf(`version: 2.0

task "default":
  info "Comma-separated: {available tasks(', ')}"
  info "Without default: {available tasks(', ', 'default')}"
  info "One per line:\n{available tasks('\\n')}"

@platform(%q)
task "lint":
  info "lint"

@platform(%q)
task "other-os-only":
  info "other"

@platform(%q)
task "build":
  info "build"
`, currentPlatform, otherPlatform, currentPlatform)

	var output bytes.Buffer
	if err := ExecuteString(input, "default", &output); err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "Comma-separated: default, lint, build") {
		t.Fatalf("output missing comma-separated tasks:\n%s", got)
	}
	if !strings.Contains(got, "Without default: lint, build") {
		t.Fatalf("output missing filtered tasks:\n%s", got)
	}
	if !strings.Contains(got, "One per line:\ndefault\nlint\nbuild") {
		t.Fatalf("output missing line-separated tasks:\n%s", got)
	}
	if strings.Contains(got, "other-os-only") {
		t.Fatalf("output contains a task unavailable on %s:\n%s", currentPlatform, got)
	}
}

func TestBuiltinContextTaskNamesFiltersAndSortsIncludedTasks(t *testing.T) {
	currentPlatform := platform.Current()
	otherPlatform := platform.Linux
	if currentPlatform == otherPlatform {
		otherPlatform = platform.Mac
	}
	annotation := func(name string) []ast.Annotation {
		return []ast.Annotation{{Name: "platform", Args: []string{name}}}
	}

	ctx := &BuiltinContext{execCtx: &ExecutionContext{
		Program: &ast.Program{Tasks: []*ast.TaskStatement{
			{Name: "default", Annotations: annotation(currentPlatform)},
			{Name: "default", Annotations: annotation(otherPlatform)},
			{Name: "local"},
			{Name: "other-local", Annotations: annotation(otherPlatform)},
		}},
		Project: &ProjectContext{IncludedTasks: map[string][]*ast.TaskStatement{
			"zeta.deploy":  {{Name: "deploy"}},
			"alpha.check":  {{Name: "check", Annotations: annotation(currentPlatform)}},
			"alpha.hidden": {{Name: "hidden", Annotations: annotation(otherPlatform)}},
		}},
	}}

	want := []string{"default", "local", "alpha.check", "zeta.deploy"}
	if got := ctx.GetTaskNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("GetTaskNames() = %#v, want %#v", got, want)
	}
}
