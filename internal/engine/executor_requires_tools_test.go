package engine

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/detection"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/provisioning"
)

func TestEngine_checkToolRequirements(t *testing.T) {
	e := NewEngine(io.Discard)
	projectCtx := &ProjectContext{}

	tests := []struct {
		name        string
		detector    toolDetector
		tools       []statement.ToolRequirement
		expectError bool
		errorMsg    string
	}{
		{
			name: "Missing tool",
			detector: fakeToolDetector{
				available: map[string]bool{"this-tool-definitely-does-not-exist-12345": false},
			},
			tools: []statement.ToolRequirement{
				{Name: "this-tool-definitely-does-not-exist-12345"},
			},
			expectError: true,
			errorMsg:    "required tool 'this-tool-definitely-does-not-exist-12345' is not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.checkToolRequirements(tt.detector, tt.tools, projectCtx, nil)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestEngine_checkToolRequirements_AutoProvisionMissingTool(t *testing.T) {
	e := NewEngine(io.Discard)
	provisioned := false
	e.newProvisioningResolver = func(string) provisioningResolver {
		return fakeProvisioningResolver{
			resolve: func(_ statement.ToolRequirement, _ provisioning.SourceSet) (*provisioning.Resolution, error) {
				return &provisioning.Resolution{
					Source: "project.yaml",
					Target: provisioning.Target{Install: "install missing-tool"},
				}, nil
			},
		}
	}
	e.provisionCommandRunner = func(command string, _ *ExecutionContext) error {
		if command != "install missing-tool" {
			t.Fatalf("unexpected provisioning command %q", command)
		}
		provisioned = true
		return nil
	}
	e.newToolDetector = sequenceDetectorFactory(
		fakeToolDetector{available: map[string]bool{"missing-tool": false}},
		fakeToolDetector{available: map[string]bool{"missing-tool": true}},
	)

	err := e.checkToolRequirements(
		e.newToolDetector(),
		[]statement.ToolRequirement{{Name: "missing-tool", AutoProvision: true}},
		&ProjectContext{ProvisioningSources: []string{"./project.yaml"}},
		&ExecutionContext{},
	)
	if err != nil {
		t.Fatalf("checkToolRequirements() error = %v", err)
	}
	if !provisioned {
		t.Fatalf("expected provisioning command to run")
	}
}

func TestEngine_checkToolRequirements_VersionMismatchRequiresFlag(t *testing.T) {
	e := NewEngine(io.Discard)

	err := e.checkToolRequirements(
		fakeToolDetector{
			available: map[string]bool{"gosec": true},
			versions:  map[string]string{"gosec": "2.21.0"},
		},
		[]statement.ToolRequirement{{
			Name:          "gosec",
			AutoProvision: true,
			Constraints: []statement.VersionConstraint{
				{Operator: ">=", Version: "2.22.0"},
			},
		}},
		&ProjectContext{},
		nil,
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--allow-tool-version-changes") {
		t.Fatalf("expected allow-tool-version-changes guidance, got %q", err.Error())
	}
}

func TestEngine_checkToolRequirements_VersionMismatchAutoProvisionRechecks(t *testing.T) {
	e := NewEngine(io.Discard)
	e.allowToolVersionChanges = true
	e.newProvisioningResolver = func(string) provisioningResolver {
		return fakeProvisioningResolver{
			resolve: func(_ statement.ToolRequirement, _ provisioning.SourceSet) (*provisioning.Resolution, error) {
				return &provisioning.Resolution{
					Source:               "project.yaml",
					ExactVersion:         "2.22.0",
					UsesVersionedInstall: true,
					Target: provisioning.Target{
						Install:          "install latest",
						InstallVersioned: "install {version}",
					},
				}, nil
			},
		}
	}
	e.provisionCommandRunner = func(command string, _ *ExecutionContext) error {
		if command != "install 2.22.0" {
			t.Fatalf("unexpected provisioning command %q", command)
		}
		return nil
	}
	e.newToolDetector = sequenceDetectorFactory(
		fakeToolDetector{
			available: map[string]bool{"gosec": true},
			versions:  map[string]string{"gosec": "2.21.0"},
		},
		fakeToolDetector{
			available: map[string]bool{"gosec": true},
			versions:  map[string]string{"gosec": "2.22.0"},
		},
	)

	err := e.checkToolRequirements(
		e.newToolDetector(),
		[]statement.ToolRequirement{{
			Name:          "gosec",
			AutoProvision: true,
			Constraints: []statement.VersionConstraint{
				{Operator: ">=", Version: "2.22.0"},
				{Operator: "<=", Version: "2.22.0"},
			},
		}},
		&ProjectContext{},
		nil,
	)
	if err != nil {
		t.Fatalf("checkToolRequirements() error = %v", err)
	}
}

func TestEngine_checkToolRequirements_PostProvisionRecheckFailure(t *testing.T) {
	e := NewEngine(io.Discard)
	e.newProvisioningResolver = func(string) provisioningResolver {
		return fakeProvisioningResolver{
			resolve: func(_ statement.ToolRequirement, _ provisioning.SourceSet) (*provisioning.Resolution, error) {
				return &provisioning.Resolution{
					Source: "project.yaml",
					Target: provisioning.Target{Install: "install missing-tool"},
				}, nil
			},
		}
	}
	e.provisionCommandRunner = func(command string, _ *ExecutionContext) error {
		if command != "install missing-tool" {
			t.Fatalf("unexpected provisioning command %q", command)
		}
		return nil
	}
	e.newToolDetector = sequenceDetectorFactory(
		fakeToolDetector{available: map[string]bool{"missing-tool": false}},
		fakeToolDetector{available: map[string]bool{"missing-tool": false}},
	)

	err := e.checkToolRequirements(
		e.newToolDetector(),
		[]statement.ToolRequirement{{Name: "missing-tool", AutoProvision: true}},
		&ProjectContext{ProvisioningSources: []string{"./project.yaml"}},
		&ExecutionContext{},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "post-provision check for tool 'missing-tool' failed") {
		t.Fatalf("expected post-provision failure, got %q", err.Error())
	}
}

func TestEngine_ExecuteTaskRequiresToolsInheritanceChecksInheritedTools(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "build":
  requires tools:
    go >= "1.21"

task "lint":
  requires tools:
    golangci-lint
    from tasks:
      build

task "security":
  requires tools:
    gosec
    from tasks:
      lint
  info "security ok"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var checked []string
	engine := NewEngine(&bytes.Buffer{})
	engine.newToolDetector = func() toolDetector {
		return &recordingToolDetector{
			available: map[string]bool{
				"go":            true,
				"golangci-lint": true,
				"gosec":         true,
			},
			versions: map[string]string{"go": "1.22.0"},
			checked:  &checked,
		}
	}

	if err := engine.Execute(program, "security"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertCheckedTools(t, checked, []string{"gosec", "golangci-lint", "go"})
}

func TestEngine_ProjectRequiresToolsInheritanceChecksDirectThenMultipleTaskSources(t *testing.T) {
	program, err := ParseString(`version: 2.0

project "quality":
  requires tools:
    project-tool
    from tasks:
      build
    from tasks:
      lint

task "build":
  requires tools:
    go

task "lint":
  requires tools:
    golangci-lint

task "default":
  info "ok"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var checked []string
	engine := NewEngine(&bytes.Buffer{})
	engine.newToolDetector = func() toolDetector {
		return &recordingToolDetector{
			available: map[string]bool{
				"project-tool":  true,
				"go":            true,
				"golangci-lint": true,
			},
			checked: &checked,
		}
	}

	if err := engine.Execute(program, "default"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertCheckedTools(t, checked, []string{"project-tool", "go", "golangci-lint"})
}

func TestEngine_ProjectRequiresToolsDirectRequirementOverridesInheritedConstraint(t *testing.T) {
	program, err := ParseString(`version: 2.0

project "quality":
  requires tools:
    shared-tool >= "2.0"
    from tasks:
      lint

task "lint":
  requires tools:
    shared-tool >= "1.0"

task "default":
  info "ok"
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var checked []string
	engine := NewEngine(&bytes.Buffer{})
	engine.newToolDetector = func() toolDetector {
		return &recordingToolDetector{
			available: map[string]bool{"shared-tool": true},
			versions:  map[string]string{"shared-tool": "1.5.0"},
			checked:   &checked,
		}
	}

	err = engine.Execute(program, "default")
	if err == nil {
		t.Fatal("Execute() error = nil, want direct constraint failure")
	}
	if !strings.Contains(err.Error(), "constraint >= 2.0") {
		t.Fatalf("expected direct constraint failure, got %v", err)
	}
	assertCheckedTools(t, checked, []string{"shared-tool"})
}

func TestEngine_ProjectRequiresToolsInheritanceUsesIncludedPlatformVariant(t *testing.T) {
	current := currentPlatformLabel()
	other := "linux"
	if current == other {
		other = "mac"
	}

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.drun")
	sharedPath := filepath.Join(dir, "shared.drun")

	if err := os.WriteFile(sharedPath, []byte(`version: 2.0

project "shared":
  set label to "shared"

@platform("`+current+`")
task "lint":
  requires tools:
    current-platform-tool

@platform("`+other+`")
task "lint":
  requires tools:
    other-platform-tool
`), 0o600); err != nil {
		t.Fatalf("WriteFile(shared) error = %v", err)
	}

	mainSource := `version: 2.0

project "app":
  include "shared.drun"
  requires tools:
    from tasks:
      "shared.lint"

task "default":
  info "ok"
`
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o600); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	program, err := ParseStringWithFilename(mainSource, mainPath)
	if err != nil {
		t.Fatalf("ParseStringWithFilename() error = %v", err)
	}

	var checked []string
	var out bytes.Buffer
	engine := NewEngineWithOptions(WithOutput(&out), WithVerbose(true))
	engine.newToolDetector = func() toolDetector {
		return &recordingToolDetector{
			available: map[string]bool{
				"current-platform-tool": true,
				"other-platform-tool":   false,
			},
			checked: &checked,
		}
	}

	if err := engine.ExecuteWithParamsAndFile(program, "default", nil, mainPath); err != nil {
		t.Fatalf("ExecuteWithParamsAndFile() error = %v\noutput:\n%s", err, out.String())
	}

	assertCheckedTools(t, checked, []string{"current-platform-tool"})
}

func TestEngine_RequiresToolsInheritanceCycleFailsBeforeToolDetection(t *testing.T) {
	program, err := ParseString(`version: 2.0

task "a":
  requires tools:
    from tasks:
      b

task "b":
  requires tools:
    from tasks:
      a
`)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	var checked []string
	engine := NewEngine(&bytes.Buffer{})
	engine.newToolDetector = func() toolDetector {
		return &recordingToolDetector{checked: &checked}
	}

	err = engine.Execute(program, "a")
	if err == nil {
		t.Fatal("Execute() error = nil, want cycle failure")
	}
	if !strings.Contains(err.Error(), "circular requires-tools inheritance detected") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(checked) != 0 {
		t.Fatalf("tool detection ran before cycle failure: %#v", checked)
	}
}

func TestFormatConstraints(t *testing.T) {
	constraints := []statement.VersionConstraint{
		{Operator: ">=", Version: "1.0"},
		{Operator: "<", Version: "2.0"},
	}
	expected := ">= 1.0, < 2.0"
	actual := formatConstraints(constraints)
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

type fakeToolDetector struct {
	available map[string]bool
	versions  map[string]string
}

func (f fakeToolDetector) IsToolAvailable(tool string) bool {
	return f.available[tool]
}

func (f fakeToolDetector) GetToolVersion(tool string) string {
	return f.versions[tool]
}

func (f fakeToolDetector) CompareVersion(version1, operator, version2 string) bool {
	return detection.NewDetector().CompareVersion(version1, operator, version2)
}

type fakeProvisioningResolver struct {
	resolve func(req statement.ToolRequirement, sources provisioning.SourceSet) (*provisioning.Resolution, error)
}

func (f fakeProvisioningResolver) ResolveRequirement(_ context.Context, req statement.ToolRequirement, sources provisioning.SourceSet) (*provisioning.Resolution, error) {
	return f.resolve(req, sources)
}

func sequenceDetectorFactory(detectors ...toolDetector) func() toolDetector {
	index := 0
	return func() toolDetector {
		if index >= len(detectors) {
			return detectors[len(detectors)-1]
		}
		detector := detectors[index]
		index++
		return detector
	}
}

type recordingToolDetector struct {
	available map[string]bool
	versions  map[string]string
	checked   *[]string
}

func (r *recordingToolDetector) IsToolAvailable(tool string) bool {
	if r.checked != nil {
		*r.checked = append(*r.checked, tool)
	}
	return r.available[tool]
}

func (r *recordingToolDetector) GetToolVersion(tool string) string {
	return r.versions[tool]
}

func (r *recordingToolDetector) CompareVersion(version1, operator, version2 string) bool {
	return detection.NewDetector().CompareVersion(version1, operator, version2)
}

func assertCheckedTools(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("checked tools = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("checked tools = %#v, want %#v", got, want)
		}
	}
}
