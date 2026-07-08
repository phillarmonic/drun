package provisioning

import (
	"context"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

func TestDefaultEmbeddedSources_ResolveCommonGoTools(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(t.TempDir(), WithEmbeddedSources(DefaultEmbeddedSources()))

	testCases := []struct {
		name         string
		requirement  statement.ToolRequirement
		wantSource   string
		wantContains string
	}{
		{
			name:         "golangci-lint latest",
			requirement:  statement.ToolRequirement{Name: "golangci-lint"},
			wantSource:   defaultEmbeddedSourceName,
			wantContains: "github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
		},
		{
			name: "gosec exact version",
			requirement: statement.ToolRequirement{
				Name: "gosec",
				Constraints: []statement.VersionConstraint{
					{Operator: ">=", Version: "2.22.0"},
					{Operator: "<=", Version: "2.22.0"},
				},
			},
			wantSource:   defaultEmbeddedSourceName,
			wantContains: "github.com/securego/gosec/v2/cmd/gosec@v2.22.0",
		},
		{
			name:         "govulncheck latest",
			requirement:  statement.ToolRequirement{Name: "govulncheck"},
			wantSource:   defaultEmbeddedSourceName,
			wantContains: "golang.org/x/vuln/cmd/govulncheck@latest",
		},
		{
			name:         "staticcheck latest",
			requirement:  statement.ToolRequirement{Name: "staticcheck"},
			wantSource:   defaultEmbeddedSourceName,
			wantContains: "honnef.co/go/tools/cmd/staticcheck@latest",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolution, err := resolver.ResolveRequirement(context.Background(), tc.requirement, SourceSet{})
			if err != nil {
				t.Fatalf("ResolveRequirement() error = %v", err)
			}
			if resolution.Source != tc.wantSource {
				t.Fatalf("resolution.Source = %q", resolution.Source)
			}
			if !strings.Contains(resolution.InstallCommand(), tc.wantContains) {
				t.Fatalf("InstallCommand() = %q", resolution.InstallCommand())
			}
		})
	}
}
