package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const fileValueReleaseSpec = `version: 2.0

project "drun-intellij" version "%s":

task "set-version" means "Synchronize the IntelliJ plugin release version":
    requires $version as string matching pattern "^[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"
    get property "pluginVersion" from "gradle.properties" as $plugin_version
    check project version equals "{$plugin_version}"

    update property "pluginVersion" in "gradle.properties" to "{$version}" or fail
    update project version to "{$version}"

task "package" means "Create the production plugin distribution":
    given $version defaults to ""
    given $previous_version defaults to ""
    given $first_release defaults to "false" from ["false", "true"]

    get property "pluginVersion" from "gradle.properties" as $plugin_version
    capture from shell "git tag --points-at HEAD --list 'v*' | head -n 1 | sed 's/^v//'" as $tag_version
    if $version is not empty:
        set $release_version to "{$version}"
    else if $tag_version is not empty:
        set $release_version to "{$tag_version}"
    else:
        set $release_version to "{$plugin_version}"

    set $resolved_previous_version to "{$previous_version}"
    if $first_release is "false":
        if $resolved_previous_version is empty:
            capture from shell "curl -fsSL 'https://plugins.jetbrains.com/plugins/list?pluginId=com.phillarmonic.drun' 2>/dev/null | grep -o '<version>[^<]*</version>' | head -n 1 | sed 's/<[^>]*>//g'" as $resolved_previous_version
        if $resolved_previous_version is empty:
            fail "Could not resolve the previous Marketplace version; pass previous_version or use first_release=true"

    check property "pluginVersion" in "gradle.properties" equals "{$release_version}"
    check project version equals "{$plugin_version}"
    check property "pluginVersion" in "gradle.properties" differs from "{$resolved_previous_version}"

    run "./gradlew buildPlugin"
`

type fileValueReleaseFixture struct {
	workspace    string
	homeDir      string
	binDir       string
	gradleMarker string
}

func TestFileValueReleaseWorkflowRejectsUnsafePackagingBeforeGradle(t *testing.T) {
	t.Run("reused version", func(t *testing.T) {
		fixture := newFileValueReleaseFixture(t, "1.0.1")
		result := fixture.run(t, "package", "version=1.0.1", "previous_version=1.0.1")

		if result.err == nil {
			t.Fatalf("expected reused version to fail\nstdout:\n%s\nstderr:\n%s", result.stdout, result.stderr)
		}
		if combined := result.stdout + result.stderr; !strings.Contains(combined, "expected to differ") {
			t.Fatalf("expected reused-version diagnostic, got:\n%s", combined)
		}
		if fixture.gradleWasInvoked() {
			t.Fatal("Gradle was invoked after the reused-version preflight failed")
		}
	})

	t.Run("canonical mismatch", func(t *testing.T) {
		fixture := newFileValueReleaseFixture(t, "1.0.1")
		result := fixture.run(t, "package", "version=1.0.2", "previous_version=1.0.1")

		if result.err == nil {
			t.Fatalf("expected canonical mismatch to fail\nstdout:\n%s\nstderr:\n%s", result.stdout, result.stderr)
		}
		if combined := result.stdout + result.stderr; !strings.Contains(combined, `expected to equal "1.0.2"`) {
			t.Fatalf("expected canonical-mismatch diagnostic, got:\n%s", combined)
		}
		if fixture.gradleWasInvoked() {
			t.Fatal("Gradle was invoked after the canonical-version preflight failed")
		}
	})
}

func TestFileValueReleaseWorkflowSetsBothCanonicalVersions(t *testing.T) {
	fixture := newFileValueReleaseFixture(t, "1.0.1")
	result := fixture.run(t, "set-version", "version=1.0.2")
	if result.err != nil {
		t.Fatalf("set-version failed: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
	}

	spec := readFixtureFile(t, filepath.Join(fixture.workspace, ".drun", "spec.drun"))
	if !strings.Contains(spec, `project "drun-intellij" version "1.0.2":`) {
		t.Fatalf("project declaration was not updated:\n%s", spec)
	}

	properties := readFixtureFile(t, filepath.Join(fixture.workspace, "gradle.properties"))
	if properties != "pluginVersion=1.0.2\n" {
		t.Fatalf("gradle.properties = %q, want pluginVersion=1.0.2", properties)
	}
	if fixture.gradleWasInvoked() {
		t.Fatal("set-version unexpectedly invoked Gradle")
	}
}

func TestFileValueReleaseWorkflowPackagesANewerVersion(t *testing.T) {
	fixture := newFileValueReleaseFixture(t, "1.0.2")
	result := fixture.run(t, "package", "version=1.0.2", "previous_version=1.0.1")
	if result.err != nil {
		t.Fatalf("package failed: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
	}
	if !fixture.gradleWasInvoked() {
		t.Fatal("Gradle was not invoked after all package preflights passed")
	}
}

func TestFileValueReleaseWorkflowResolvesTagAndFirstReleaseInputs(t *testing.T) {
	t.Run("exact Git tag", func(t *testing.T) {
		fixture := newFileValueReleaseFixture(t, "1.0.2")
		fixture.tag(t, "v1.0.2")

		result := fixture.run(t, "package", "previous_version=1.0.1")
		if result.err != nil {
			t.Fatalf("tag-derived package failed: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
		}
		if !fixture.gradleWasInvoked() {
			t.Fatal("Gradle was not invoked for a matching tag-derived version")
		}
	})

	t.Run("missing Git tag falls back to canonical version", func(t *testing.T) {
		fixture := newFileValueReleaseFixture(t, "1.0.2")
		result := fixture.run(t, "package", "previous_version=1.0.1")

		if result.err != nil {
			t.Fatalf("untagged package failed: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
		}
		if !fixture.gradleWasInvoked() {
			t.Fatal("Gradle was not invoked after the canonical fallback passed")
		}
	})

	t.Run("first release bypasses previous-version lookup", func(t *testing.T) {
		fixture := newFileValueReleaseFixture(t, "1.0.0")
		result := fixture.run(t, "package", "first_release=true")

		if result.err != nil {
			t.Fatalf("first-release package failed: %v\nstdout:\n%s\nstderr:\n%s", result.err, result.stdout, result.stderr)
		}
		if !fixture.gradleWasInvoked() {
			t.Fatal("Gradle was not invoked for the first-release override")
		}
	})
}

func newFileValueReleaseFixture(t *testing.T, version string) fileValueReleaseFixture {
	t.Helper()

	workspace := t.TempDir()
	fixture := fileValueReleaseFixture{
		workspace:    workspace,
		homeDir:      filepath.Join(workspace, "home"),
		binDir:       filepath.Join(workspace, "bin"),
		gradleMarker: filepath.Join(workspace, "gradle-invoked"),
	}

	mustMkdirAll(t, filepath.Join(workspace, ".drun"))
	mustMkdirAll(t, fixture.homeDir)
	mustMkdirAll(t, fixture.binDir)
	writeFile(t, filepath.Join(workspace, ".drun", "spec.drun"), fmt.Sprintf(fileValueReleaseSpec, version))
	writeFile(t, filepath.Join(workspace, "gradle.properties"), "pluginVersion="+version+"\n")
	writeExecutable(t, filepath.Join(workspace, "gradlew"), "#!/bin/sh\nset -eu\ntouch gradle-invoked\n")

	runGitFixtureCommand(t, workspace, "init", "-q")
	runGitFixtureCommand(t, workspace, "config", "user.email", "fixture@example.invalid")
	runGitFixtureCommand(t, workspace, "config", "user.name", "Fixture")
	runGitFixtureCommand(t, workspace, "add", ".drun/spec.drun", "gradle.properties", "gradlew")
	runGitFixtureCommand(t, workspace, "commit", "-qm", "fixture")

	return fixture
}

func (f fileValueReleaseFixture) run(t *testing.T, args ...string) xdrunRunResult {
	t.Helper()
	return runXdrun(t, f.workspace, f.homeDir, f.binDir, append([]string{"--file", ".drun/spec.drun"}, args...)...)
}

func (f fileValueReleaseFixture) gradleWasInvoked() bool {
	_, err := os.Stat(f.gradleMarker)
	return err == nil
}

func (f fileValueReleaseFixture) tag(t *testing.T, version string) {
	t.Helper()
	runGitFixtureCommand(t, f.workspace, "tag", version)
}

func runGitFixtureCommand(t *testing.T, workspace string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = workspace
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func readFixtureFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(contents)
}
