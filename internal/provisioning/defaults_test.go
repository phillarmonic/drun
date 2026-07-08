package provisioning

import (
	"context"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
)

func TestDefaultEmbeddedSources_ResolveDummyProvisioning(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(t.TempDir(),
		WithBuiltinSources(nil),
		WithEmbeddedSources(DefaultEmbeddedSources()),
	)

	resolution, err := resolver.ResolveRequirement(context.Background(), statement.ToolRequirement{
		Name: "dummy-tool",
		Constraints: []statement.VersionConstraint{
			{Operator: ">=", Version: "1.2.3"},
			{Operator: "<=", Version: "1.2.3"},
		},
	}, SourceSet{})
	if err != nil {
		t.Fatalf("ResolveRequirement() error = %v", err)
	}
	if resolution.Source != defaultEmbeddedSourceName {
		t.Fatalf("resolution.Source = %q", resolution.Source)
	}
	if !strings.Contains(resolution.InstallCommand(), "embedded dummy provisioner 1.2.3") {
		t.Fatalf("InstallCommand() = %q", resolution.InstallCommand())
	}
}
