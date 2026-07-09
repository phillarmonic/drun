package engine

import (
	"context"
	"io"
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
