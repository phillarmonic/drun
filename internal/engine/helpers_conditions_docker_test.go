package engine

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/phillarmonic/drun/v2/internal/lexer"
	"github.com/phillarmonic/drun/v2/internal/parser"
	"github.com/phillarmonic/drun/v2/internal/types"
)

// requireDockerDaemon skips the test when the docker CLI or daemon is
// unavailable, so CI machines without Docker still run the rest of the suite.
func requireDockerDaemon(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker CLI not available")
	}
	if err := exec.Command("docker", "network", "ls").Run(); err != nil {
		t.Skip("docker daemon not available")
	}
}

// requireExistingDockerNetworkName returns one network currently known by the
// daemon. Windows Docker hosts often use "nat" instead of "bridge", so tests
// should avoid hardcoding a Linux-specific network.
func requireExistingDockerNetworkName(t *testing.T) string {
	t.Helper()
	output, err := exec.Command("docker", "network", "ls", "--format", "{{.Name}}").Output()
	if err != nil {
		t.Skip("docker daemon not available")
	}
	for _, line := range strings.Split(string(output), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			return name
		}
	}
	t.Skip("no docker networks available")
	return ""
}

func executeDockerConditionTask(t *testing.T, condition string, dryRun bool) (string, error) {
	t.Helper()
	input := `version: 2.0

task "probe":
  if ` + condition + `:
    info "branch: taken"
  else:
    info "branch: skipped"`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var output bytes.Buffer
	e := NewEngine(&output)
	e.SetDryRun(dryRun)
	err := e.ExecuteWithParams(program, "probe", nil)
	return output.String(), err
}

func TestDockerNetworkConditionRouting(t *testing.T) {
	e := NewEngine(&bytes.Buffer{})
	ctx := &ExecutionContext{
		Parameters: map[string]*types.Value{},
		Variables:  map[string]string{},
	}

	// Non-docker conditions must not be claimed by the docker evaluator.
	for _, condition := range []string{
		`environment is "production"`,
		`port 5432 is open on "localhost"`,
		`file "./README.md" exists`,
		`docker compose status`,
	} {
		if _, handled, err := e.evaluateDockerCondition(condition, ctx); handled || err != nil {
			t.Errorf("evaluateDockerCondition(%q) = handled %v, err %v; want handled false, nil err", condition, handled, err)
		}
	}
}

func TestDockerNetworkCondition(t *testing.T) {
	requireDockerDaemon(t)
	existingNetworkName := requireExistingDockerNetworkName(t)

	tests := []struct {
		name           string
		condition      string
		expectedOutput []string
		notExpected    []string
	}{
		{
			name:           "existing network takes if branch",
			condition:      `docker network "` + existingNetworkName + `" exists`,
			expectedOutput: []string{"branch: taken"},
			notExpected:    []string{"branch: skipped"},
		},
		{
			name:           "missing network takes else branch",
			condition:      `docker network "drun-test-missing-network" exists`,
			expectedOutput: []string{"branch: skipped"},
			notExpected:    []string{"branch: taken"},
		},
		{
			name:           "not exists is true for missing network",
			condition:      `docker network "drun-test-missing-network" not exists`,
			expectedOutput: []string{"branch: taken"},
			notExpected:    []string{"branch: skipped"},
		},
		{
			name:           "not exists is false for existing network",
			condition:      `docker network "` + existingNetworkName + `" not exists`,
			expectedOutput: []string{"branch: skipped"},
			notExpected:    []string{"branch: taken"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputStr, err := executeDockerConditionTask(t, tt.condition, false)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, outputStr)
				}
			}
			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain %q, but got:\n%s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestDockerNetworkConditionInterpolation(t *testing.T) {
	requireDockerDaemon(t)
	existingNetworkName := requireExistingDockerNetworkName(t)

	input := `version: 2.0

task "probe":
  given $network_name defaults to "` + existingNetworkName + `"
  if docker network "{$network_name}" exists:
    info "network is up"
  else:
    info "network is down"`

	l := lexer.NewLexer(input)
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	var output bytes.Buffer
	e := NewEngine(&output)
	if err := e.ExecuteWithParams(program, "probe", nil); err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	if !strings.Contains(output.String(), "network is up") {
		t.Errorf("Expected interpolated docker network condition to be true, got:\n%s", output.String())
	}
}

func TestDockerNetworkConditionDryRun(t *testing.T) {
	// Dry run must not query the daemon: even an existing network evaluates as
	// missing, and no docker CLI is required for this test.
	outputStr, err := executeDockerConditionTask(t, `docker network "any-network-name" exists`, true)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
	if !strings.Contains(outputStr, "branch: skipped") {
		t.Errorf("Expected docker network condition to be false in dry run, got:\n%s", outputStr)
	}
}
