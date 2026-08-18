package engine

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/domain/orchestration"
	"github.com/phillarmonic/drun/v2/internal/domain/statement"
	"github.com/phillarmonic/drun/v2/internal/healthcheck"
)

// Domain: Network Operations Execution
// This file contains executors for:
// - Network connectivity checks (ping, port check)
// - File downloads (HTTP/HTTPS)

// defaultPortCheckTimeout is the dial timeout when no timeout option is given.
const defaultPortCheckTimeout = 5 * time.Second

// executeNetwork executes network operations (health checks, port testing, ping)
func (e *Engine) executeNetwork(networkStmt *statement.Network, ctx *ExecutionContext) error {
	// Interpolate variables in network statement
	target := e.interpolateVariables(networkStmt.Target, ctx)
	port := e.interpolateVariables(networkStmt.Port, ctx)
	condition := e.interpolateVariables(networkStmt.Condition, ctx)

	// Interpolate options
	options := make(map[string]string, len(networkStmt.Options))
	for key, value := range networkStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	if e.dryRun && networkStmt.Action != "port_check" {
		return e.buildNetworkCommand(networkStmt.Action, target, port, condition, options, true)
	}

	// Show what we're about to do with appropriate emoji
	switch networkStmt.Action {
	case "health_check":
		_, _ = fmt.Fprintf(e.output, "🏥  Health check: %s\n", target)
	case "wait_for_service":
		_, _ = fmt.Fprintf(e.output, "⏳  Waiting for service: %s\n", target)
	case "port_check":
		if port != "" {
			_, _ = fmt.Fprintf(e.output, "🔌 Port check: %s:%s\n", target, port)
		} else {
			_, _ = fmt.Fprintf(e.output, "🔌 Connection test: %s\n", target)
		}
	case "ping":
		_, _ = fmt.Fprintf(e.output, "🏓 Ping: %s\n", target)
	default:
		_, _ = fmt.Fprintf(e.output, "🌐  Network operation: %s on %s\n", networkStmt.Action, target)
	}

	// Port checks run natively (no external tools); other network actions
	// still go through the shell command builder.
	if networkStmt.Action == "port_check" {
		return e.executePortCheck(target, port, options)
	}

	// Build and execute the actual network command
	return e.buildNetworkCommand(networkStmt.Action, target, port, condition, options, false)
}

// executePortCheck natively probes a TCP endpoint using the health check
// subsystem instead of shelling out to netcat. A successful dial means the
// port is open (something is listening); a dial error fails the statement.
func (e *Engine) executePortCheck(target, port string, options map[string]string) error {
	endpoint, err := portCheckEndpoint(target, port)
	if err != nil {
		return err
	}

	timeout, err := parsePortCheckTimeout(options["timeout"])
	if err != nil {
		return fmt.Errorf("port check: %w", err)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would probe TCP port: %s (timeout %s)\n", endpoint, timeout)
		return nil
	}

	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Probe: dial tcp %s (timeout %s)\n", endpoint, timeout)
	}

	if err := probeTCPPort(endpoint, timeout); err != nil {
		_, _ = fmt.Fprintf(e.output, "❌  Port check failed: %v\n", err)
		return err
	}

	_, _ = fmt.Fprintf(e.output, "✅  Port is open: %s\n", endpoint)
	return nil
}

// portCheckEndpoint builds a host:port endpoint for the dialer. When port is
// empty the target itself must already carry a port (e.g. "db.internal:5432").
func portCheckEndpoint(target, port string) (string, error) {
	target = strings.TrimSpace(target)
	port = strings.TrimSpace(port)

	if target == "" {
		return "", fmt.Errorf("port check: host is empty")
	}
	if port != "" {
		return net.JoinHostPort(target, port), nil
	}
	if _, _, err := net.SplitHostPort(target); err == nil {
		return target, nil
	}
	return "", fmt.Errorf("port check: no port given for host %q", target)
}

// parsePortCheckTimeout accepts Go durations ("5s", "500ms") or bare seconds
// ("5"). Empty means the default timeout.
func parsePortCheckTimeout(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultPortCheckTimeout, nil
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second, nil
	}
	return 0, fmt.Errorf("invalid timeout %q (use e.g. \"5s\" or \"5\")", raw)
}

// probeTCPPort performs a single native TCP dial through the health check
// subsystem, so port probing shares one implementation with service health
// checks.
func probeTCPPort(endpoint string, timeout time.Duration) error {
	checker := healthcheck.NewChecker()
	return checker.Check(context.Background(), &orchestration.HealthCheck{
		Type:     "tcp",
		Endpoint: endpoint,
		Timeout:  timeout,
	})
}

// executeDownload executes file download operations using native Go HTTP client
func (e *Engine) executeDownload(downloadStmt *statement.Download, ctx *ExecutionContext) error {
	// Interpolate variables in download statement
	url := e.interpolateVariables(downloadStmt.URL, ctx)
	path := e.interpolateVariables(downloadStmt.Path, ctx)

	// Interpolate headers
	headers := make(map[string]string, len(downloadStmt.Headers))
	for key, value := range downloadStmt.Headers {
		headers[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate auth
	auth := make(map[string]string, len(downloadStmt.Auth))
	for key, value := range downloadStmt.Auth {
		auth[key] = e.interpolateVariables(value, ctx)
	}

	// Interpolate options
	options := make(map[string]string, len(downloadStmt.Options))
	for key, value := range downloadStmt.Options {
		options[key] = e.interpolateVariables(value, ctx)
	}

	// Check if file exists and handle overwrite
	if !downloadStmt.AllowOverwrite && e.fileExists(path, ctx) {
		errMsg := fmt.Sprintf("file already exists: %s (use 'allow overwrite' to replace)", path)
		_, _ = fmt.Fprintf(e.output, "❌  %s\n", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	if e.dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would download %s to %s", url, path)
		if downloadStmt.AllowOverwrite {
			_, _ = fmt.Fprintf(e.output, " (overwrite allowed)")
		}
		if len(downloadStmt.AllowPermissions) > 0 {
			_, _ = fmt.Fprintf(e.output, " with permissions: ")
			for i, perm := range downloadStmt.AllowPermissions {
				if i > 0 {
					_, _ = fmt.Fprintf(e.output, ", ")
				}
				_, _ = fmt.Fprintf(e.output, "%v to %v", perm.Permissions, perm.Targets)
			}
		}
		_, _ = fmt.Fprintf(e.output, "\n")
		return nil
	}

	// Show what we're about to do
	_, _ = fmt.Fprintf(e.output, "⬇️  Downloading: %s\n", url)
	_, _ = fmt.Fprintf(e.output, "   → %s\n", path)

	// Perform the download with progress tracking
	err := e.downloadFileWithProgress(url, path, headers, auth, options)
	if err != nil {
		_, _ = fmt.Fprintf(e.output, "❌  Download failed: %v\n", err)
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract archive if requested
	if downloadStmt.ExtractTo != "" {
		extractTo := e.interpolateVariables(downloadStmt.ExtractTo, ctx)
		_, _ = fmt.Fprintf(e.output, "📦  Extracting archive to: %s\n", extractTo)

		err = e.extractArchive(path, extractTo)
		if err != nil {
			_, _ = fmt.Fprintf(e.output, "❌  Extraction failed: %v\n", err)
			return fmt.Errorf("extraction failed: %w", err)
		}

		_, _ = fmt.Fprintf(e.output, "✅  Extraction completed\n")

		// Remove archive if requested
		if downloadStmt.RemoveArchive {
			_, _ = fmt.Fprintf(e.output, "🗑️  Removing archive: %s\n", path)
			err = os.Remove(path)
			if err != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Warning: Failed to remove archive: %v\n", err)
			} else {
				_, _ = fmt.Fprintf(e.output, "✅  Archive removed\n")
			}
		}
	} else {
		// Apply file permissions if specified (only for non-extracted files)
		if len(downloadStmt.AllowPermissions) > 0 {
			// Convert domain PermissionSpec to ast.PermissionSpec
			var astPerms []ast.PermissionSpec
			for _, perm := range downloadStmt.AllowPermissions {
				astPerms = append(astPerms, ast.PermissionSpec{
					Permissions: perm.Permissions,
					Targets:     perm.Targets,
				})
			}
			err = e.applyFilePermissions(path, astPerms)
			if err != nil {
				_, _ = fmt.Fprintf(e.output, "⚠️  Warning: Failed to set permissions: %v\n", err)
				// Don't fail the download, just warn
			}
		}
	}

	_, _ = fmt.Fprintf(e.output, "✅  Downloaded successfully to: %s\n", path)
	return nil
}
