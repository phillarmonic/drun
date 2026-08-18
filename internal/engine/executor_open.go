package engine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/platform"
)

// ciEnvVars lists environment variables that indicate a CI environment.
var ciEnvVars = []string{
	"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI",
	"JENKINS_URL", "TRAVIS", "CIRCLECI", "BUILDKITE", "TEAMCITY_VERSION",
}

func isCIEnvironment() bool {
	for _, v := range ciEnvVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// hasDesktopSession reports whether an OS opener is expected to work.
//
// Rules:
//   - CI env var set             -> headless (all OSes)
//   - SSH_CONNECTION/SSH_TTY set -> headless (all OSes; conservative for macOS too)
//   - Linux additionally needs DISPLAY or WAYLAND_DISPLAY
//   - macOS/Windows local sessions are assumed to have a desktop
func hasDesktopSession() bool {
	if isCIEnvironment() {
		return false
	}
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false
	}
	if platform.Current() == platform.Linux {
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
	return true
}

// urlSchemePattern matches scheme:// targets (http, https, file, ...). Anything
// without a scheme is treated as a local path.
var urlSchemePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)

// openerCommand returns the OS opener for the current platform.
func openerCommand(target string) (*exec.Cmd, error) {
	switch platform.Current() {
	case platform.Mac:
		return exec.Command("open", target), nil
	case platform.Windows:
		// The empty first quoted arg is the window title; without it a quoted
		// URL is misinterpreted by cmd.
		return exec.Command("cmd", "/c", "start", "", target), nil
	default: // linux and other unixes
		path, err := exec.LookPath("xdg-open")
		if err != nil {
			return nil, fmt.Errorf("no opener found (xdg-open not in PATH)")
		}
		return exec.Command(path, target), nil
	}
}

// executeOpen executes: open url "<target>"
func (e *Engine) executeOpen(stmt *statement.Open, ctx *ExecutionContext) error {
	if !e.folderTrusted {
		return fmt.Errorf("open url: this folder is not trusted for security-sensitive operations\n" +
			"Run 'xdrun cmd:trust' in the project directory to allow 'open url' statements")
	}

	target, err := e.interpolateVariablesWithError(stmt.URL, ctx)
	if err != nil {
		return err
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("open url: target is empty after interpolation")
	}

	// No scheme -> local path: resolve to absolute (openers imply file:// for paths)
	if !urlSchemePattern.MatchString(target) {
		abs, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("open url: cannot resolve path %q: %w", target, err)
		}
		target = abs
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would open %s\n", target)
		return nil
	}

	if !hasDesktopSession() {
		_, _ = fmt.Fprintf(e.output, "Warning: no desktop environment detected (headless/SSH/CI)\n    Open this URL manually: %s\n", target)
		return nil
	}

	cmd, err := openerCommand(target)
	if err != nil {
		// No opener installed: same non-fatal treatment as headless
		_, _ = fmt.Fprintf(e.output, "Warning: %v\n    Open this URL manually: %s\n", err, target)
		return nil
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open %q: %w", target, err)
	}

	_, _ = fmt.Fprintf(e.output, "Opened %s\n", target)
	return nil
}
