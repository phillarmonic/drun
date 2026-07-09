package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	buildXdrunOnce sync.Once
	builtXdrunPath string
	buildXdrunErr  error
)

func TestXdrunProvisioningE2E_ProjectSourceOverridesUserSource(t *testing.T) {
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	binDir := filepath.Join(workspace, "bin")

	mustMkdirAll(t, filepath.Join(workspace, ".drun"))
	mustMkdirAll(t, filepath.Join(homeDir, ".drun"))
	mustMkdirAll(t, binDir)

	writeExecutable(t, filepath.Join(workspace, ".drun", "install-project-tool.sh"), installerScript(binDir, "demo-tool", "project-source", "1.2.3"))
	writeExecutable(t, filepath.Join(homeDir, ".drun", "install-user-tool.sh"), installerScript(binDir, "demo-tool", "user-source", "9.9.9"))

	writeFile(t, filepath.Join(workspace, ".drun", "spec.drun"), `version: 2.0

project "provisioning-e2e" version "1.0":
	provisioning sources:
		"./.drun/provisionings.yaml"

	requires tools:
		demo-tool >= "1.2.3" <= "1.2.3" provision

task "demo" means "Use the provisioned tool":
	run "demo-tool"
`)

	writeFile(t, filepath.Join(workspace, ".drun", "provisionings.yaml"), `version: "1"
provisionings:
  demo-tool:
    targets:
      - install: "./.drun/install-project-tool.sh"
        install_versioned: "./.drun/install-project-tool.sh {version}"
`)

	writeFile(t, filepath.Join(homeDir, ".drun", "config.yml"), fmt.Sprintf(`provisioningSources:
  - %q
`, filepath.Join(homeDir, ".drun", "user-provisionings.yaml")))

	writeFile(t, filepath.Join(homeDir, ".drun", "user-provisionings.yaml"), `version: "1"
provisionings:
  demo-tool:
    targets:
      - install: "./install-user-tool.sh"
        install_versioned: "./install-user-tool.sh {version}"
`)

	result := runXdrun(t, workspace, homeDir, binDir, "--file", ".drun/spec.drun", "demo")
	if result.err != nil {
		t.Fatalf("xdrun demo failed: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
	}

	if !strings.Contains(result.stdout, "project-source 1.2.3") {
		t.Fatalf("expected project installer output, got stdout:\n%s", result.stdout)
	}
	if strings.Contains(result.stdout, "user-source") {
		t.Fatalf("expected project source to win over user source, got stdout:\n%s", result.stdout)
	}
}

func TestXdrunProvisioningE2E_ExactVersionRequiresFlagAndThenSucceeds(t *testing.T) {
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	binDir := filepath.Join(workspace, "bin")

	mustMkdirAll(t, filepath.Join(workspace, ".drun"))
	mustMkdirAll(t, filepath.Join(homeDir, ".drun"))
	mustMkdirAll(t, binDir)

	writeExecutable(t, filepath.Join(binDir, "exact-tool"), toolScript("exact-tool", "stale-install", "1.0.0"))
	writeExecutable(t, filepath.Join(workspace, ".drun", "install-exact-tool.sh"), versionedInstallerScript(binDir, "exact-tool", "project-updated"))

	writeFile(t, filepath.Join(workspace, ".drun", "spec.drun"), `version: 2.0

project "provisioning-e2e" version "1.0":
	provisioning sources:
		"./.drun/provisionings.yaml"

	requires tools:
		exact-tool >= "2.2.0" <= "2.2.0" provision

task "demo" means "Use the exact versioned tool":
	run "exact-tool version"
`)

	writeFile(t, filepath.Join(workspace, ".drun", "provisionings.yaml"), `version: "1"
provisionings:
  exact-tool:
    targets:
      - install: "./.drun/install-exact-tool.sh"
        install_versioned: "./.drun/install-exact-tool.sh {version}"
`)

	result := runXdrun(t, workspace, homeDir, binDir, "--file", ".drun/spec.drun", "demo")
	if result.err == nil {
		t.Fatalf("expected version mismatch to fail without flag\nstdout:\n%s\nstderr:\n%s", result.stdout, result.stderr)
	}
	if !strings.Contains(result.stderr, "--allow-tool-version-changes") {
		t.Fatalf("expected guidance about --allow-tool-version-changes, got stderr:\n%s", result.stderr)
	}

	initialVersion := runInstalledTool(t, binDir, "exact-tool", "version")
	if strings.TrimSpace(initialVersion) != "1.0.0" {
		t.Fatalf("expected tool to remain at 1.0.0 after failed run, got %q", initialVersion)
	}

	result = runXdrun(t, workspace, homeDir, binDir, "--allow-tool-version-changes", "--file", ".drun/spec.drun", "demo")
	if result.err != nil {
		t.Fatalf("expected versioned provisioning run to succeed: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
	}
	if !strings.Contains(result.stdout, "2.2.0") {
		t.Fatalf("expected exact version in task output, got stdout:\n%s", result.stdout)
	}

	updatedVersion := runInstalledTool(t, binDir, "exact-tool", "version")
	if strings.TrimSpace(updatedVersion) != "2.2.0" {
		t.Fatalf("expected installed tool to update to 2.2.0, got %q", updatedVersion)
	}
}

func TestXdrunProvisioningE2E_EmbeddedFallbackReportsPostProvisionFailure(t *testing.T) {
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	binDir := filepath.Join(workspace, "bin")

	mustMkdirAll(t, filepath.Join(workspace, ".drun"))
	mustMkdirAll(t, filepath.Join(homeDir, ".drun"))
	mustMkdirAll(t, binDir)

	writeFile(t, filepath.Join(workspace, ".drun", "spec.drun"), `version: 2.0

project "provisioning-e2e" version "1.0":
	requires tools:
		dummy-tool >= "1.2.3" <= "1.2.3" provision

task "demo" means "Trigger embedded fallback":
	info "embedded fallback should be attempted before execution continues"
`)

	result := runXdrun(t, workspace, homeDir, binDir, "--verbose", "--file", ".drun/spec.drun", "demo")
	if result.err == nil {
		t.Fatalf("expected embedded fallback run to fail\nstdout:\n%s\nstderr:\n%s", result.stdout, result.stderr)
	}

	combined := result.stdout + "\n" + result.stderr
	if !strings.Contains(combined, "source: embedded:drun-defaults") {
		t.Fatalf("expected embedded source in verbose output, got:\n%s", combined)
	}
	if !strings.Contains(combined, "embedded dummy provisioner 1.2.3") {
		t.Fatalf("expected embedded provisioning command to run, got:\n%s", combined)
	}
	if !strings.Contains(combined, "post-provision check for tool 'dummy-tool' failed") {
		t.Fatalf("expected post-provision recheck failure, got:\n%s", combined)
	}
}

type xdrunRunResult struct {
	stdout string
	stderr string
	err    error
}

func runXdrun(t *testing.T, workspace, homeDir, binDir string, args ...string) xdrunRunResult {
	t.Helper()

	binary := buildXdrunBinary(t)
	cmd := exec.Command(binary, args...)
	cmd.Dir = workspace
	cmd.Env = withPathEnv(os.Environ(), homeDir, binDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return xdrunRunResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
		err:    err,
	}
}

func buildXdrunBinary(t *testing.T) string {
	t.Helper()

	buildXdrunOnce.Do(func() {
		repoRoot, err := repoRoot()
		if err != nil {
			buildXdrunErr = err
			return
		}

		buildDir, err := os.MkdirTemp("", "xdrun-e2e-*")
		if err != nil {
			buildXdrunErr = err
			return
		}

		builtXdrunPath = filepath.Join(buildDir, "xdrun")
		cmd := exec.Command("go", "build", "-o", builtXdrunPath, "./cmd/xdrun")
		cmd.Dir = repoRoot
		cmd.Env = os.Environ()
		buildXdrunErr = cmd.Run()
	})

	if buildXdrunErr != nil {
		t.Fatalf("build xdrun binary: %v", buildXdrunErr)
	}
	return builtXdrunPath
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate repo root from %q", dir)
		}
		dir = parent
	}
}

func withPathEnv(base []string, homeDir, binDir string) []string {
	env := make([]string, 0, len(base)+2)
	for _, entry := range base {
		if strings.HasPrefix(entry, "HOME=") || strings.HasPrefix(entry, "PATH=") {
			continue
		}
		env = append(env, entry)
	}

	env = append(env, "HOME="+homeDir)
	env = append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return env
}

func runInstalledTool(t *testing.T, binDir, tool string, args ...string) string {
	t.Helper()

	commandArgs := append([]string{filepath.Join(binDir, tool)}, args...)
	cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run installed tool %q: %v\n%s", tool, err, string(output))
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func toolScript(toolName, identity, version string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu
if [ "${1:-}" = "version" ] || [ "${1:-}" = "--version" ]; then
	echo "%s"
	exit 0
fi
echo "%s %s"
`, version, identity, version)
}

func installerScript(binDir, toolName, identity, defaultVersion string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu
version="${1:-%s}"
cat > %q <<EOF
#!/bin/sh
set -eu
if [ "\${1:-}" = "version" ] || [ "\${1:-}" = "--version" ]; then
	echo "$version"
	exit 0
fi
echo "%s $version"
EOF
chmod +x %q
`, defaultVersion, filepath.Join(binDir, toolName), identity, filepath.Join(binDir, toolName))
}

func versionedInstallerScript(binDir, toolName, identity string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu
version="${1:?version is required}"
cat > %q <<EOF
#!/bin/sh
set -eu
if [ "\${1:-}" = "version" ] || [ "\${1:-}" = "--version" ]; then
	echo "$version"
	exit 0
fi
echo "%s $version"
EOF
chmod +x %q
`, filepath.Join(binDir, toolName), identity, filepath.Join(binDir, toolName))
}
