package engine

import (
	"fmt"
	"strings"
)

// Domain: Command Builders
// This file contains helper methods for building shell commands for various operations

// buildDockerCommand builds and optionally executes the Docker command
func (e *Engine) buildDockerCommand(operation, resource, name string, options map[string]string, dryRun bool) error {
	var dockerCmd []string
	dockerCmd = append(dockerCmd, "docker")

	// Handle Docker Compose separately
	if operation == "compose" {
		dockerCmd = append(dockerCmd, "compose")
		if command, exists := options["command"]; exists {
			dockerCmd = append(dockerCmd, command)
		}
	} else if operation == "scale" && resource == "compose" {
		// Handle "docker compose scale service_name replicas"
		dockerCmd = append(dockerCmd, "compose", "scale")
		if name != "" {
			if replicas, exists := options["replicas"]; exists {
				dockerCmd = append(dockerCmd, fmt.Sprintf("%s=%s", name, replicas))
			}
		}
	} else {
		// Regular Docker commands
		dockerCmd = append(dockerCmd, operation)
		if resource != "" {
			dockerCmd = append(dockerCmd, resource)
		}
		if name != "" {
			dockerCmd = append(dockerCmd, name)
		}

		// Add options in a logical order
		if from, exists := options["from"]; exists {
			if operation == "build" {
				dockerCmd = append(dockerCmd, "--file", from)
			} else {
				dockerCmd = append(dockerCmd, from)
			}
		}
		if to, exists := options["to"]; exists {
			dockerCmd = append(dockerCmd, to)
		}
		if as, exists := options["as"]; exists {
			dockerCmd = append(dockerCmd, as)
		}
		if port, exists := options["port"]; exists {
			if operation == "run" {
				dockerCmd = append(dockerCmd, "-p", fmt.Sprintf("%s:%s", port, port))
			}
		}
	}

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute Docker command: %s\n", strings.Join(dockerCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(dockerCmd, " "))
	}

	// For now, we'll simulate the command execution
	// In a real implementation, you would use exec.Command to run the Docker command
	// cmd := exec.Command(dockerCmd[0], dockerCmd[1:]...)
	// return cmd.Run()

	return nil
}

// buildGitCommand builds and displays the git command
func (e *Engine) buildGitCommand(operation, resource, name string, options map[string]string, dryRun bool) error {
	var gitCmd []string
	gitCmd = append(gitCmd, "git")

	switch operation {
	case "create":
		switch resource {
		case "branch":
			// git checkout -b branch_name
			gitCmd = append(gitCmd, "checkout", "-b")
			if name != "" {
				gitCmd = append(gitCmd, name)
			}
		case "tag":
			// git tag tag_name
			gitCmd = append(gitCmd, "tag")
			if name != "" {
				gitCmd = append(gitCmd, name)
			}
		}

	case "checkout":
		// git checkout branch_name
		gitCmd = append(gitCmd, "checkout")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}

	case "merge":
		// git merge branch_name
		gitCmd = append(gitCmd, "merge")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}

	case "clone":
		// git clone repository "url" to "dir"
		gitCmd = append(gitCmd, "clone")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}
		if to, exists := options["to"]; exists {
			gitCmd = append(gitCmd, to)
		}

	case "init":
		// git init repository in "dir"
		gitCmd = append(gitCmd, "init")
		if in, exists := options["in"]; exists {
			gitCmd = append(gitCmd, in)
		}

	case "add":
		// git add files "pattern"
		gitCmd = append(gitCmd, "add")
		if name != "" {
			gitCmd = append(gitCmd, name)
		}

	case "commit":
		// git commit changes with message "msg"
		// git commit all changes with message "msg"
		gitCmd = append(gitCmd, "commit")
		if all, exists := options["all"]; exists && all == "true" {
			gitCmd = append(gitCmd, "-a")
		}
		if message, exists := options["message"]; exists {
			gitCmd = append(gitCmd, "-m", fmt.Sprintf("\"%s\"", message))
		}

	case "push":
		// git push to remote "origin" branch "main"
		// git push tag "v1.0.0" to remote "origin"
		gitCmd = append(gitCmd, "push")
		if resource == "tag" && name != "" {
			gitCmd = append(gitCmd, "origin", name)
		} else {
			if remote, exists := options["remote"]; exists {
				gitCmd = append(gitCmd, remote)
			}
			if branch, exists := options["branch"]; exists {
				gitCmd = append(gitCmd, branch)
			}
		}

	case "pull":
		// git pull from remote "origin" branch "main"
		gitCmd = append(gitCmd, "pull")
		if from, exists := options["from"]; exists {
			gitCmd = append(gitCmd, from)
		}
		if remote, exists := options["remote"]; exists {
			gitCmd = append(gitCmd, remote)
		}
		if branch, exists := options["branch"]; exists {
			gitCmd = append(gitCmd, branch)
		}

	case "fetch":
		// git fetch from remote "origin"
		gitCmd = append(gitCmd, "fetch")
		if from, exists := options["from"]; exists {
			gitCmd = append(gitCmd, from)
		}
		if remote, exists := options["remote"]; exists {
			gitCmd = append(gitCmd, remote)
		}

	case "status":
		// git status
		gitCmd = append(gitCmd, "status")

	case "log":
		// git log --oneline
		gitCmd = append(gitCmd, "log", "--oneline")

	case "show":
		// git show current branch
		// git show current commit
		if current, exists := options["current"]; exists && current == "true" {
			switch resource {
			case "branch":
				gitCmd = append(gitCmd, "branch", "--show-current")
			case "commit":
				gitCmd = append(gitCmd, "rev-parse", "HEAD")
			}
		} else {
			gitCmd = append(gitCmd, "show")
		}

	default:
		gitCmd = append(gitCmd, operation)
		if resource != "" {
			gitCmd = append(gitCmd, resource)
		}
		if name != "" {
			gitCmd = append(gitCmd, name)
		}
	}

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute Git command: %s\n", strings.Join(gitCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(gitCmd, " "))
	}

	// For now, we'll simulate the command execution
	// In a real implementation, you would use exec.Command to run the git command
	// cmd := exec.Command(gitCmd[0], gitCmd[1:]...)
	// return cmd.Run()

	return nil
}

// buildHTTPCommand builds and displays the HTTP request details
func (e *Engine) buildHTTPCommand(method, url, body string, headers, auth, options map[string]string, dryRun bool) error {
	var httpCmd []string
	httpCmd = append(httpCmd, "curl", "-X", method)

	// Add headers
	for key, value := range headers {
		httpCmd = append(httpCmd, "-H", fmt.Sprintf("\"%s: %s\"", key, value))
	}

	// Add authentication
	for authType, value := range auth {
		switch authType {
		case "bearer":
			httpCmd = append(httpCmd, "-H", fmt.Sprintf("\"Authorization: Bearer %s\"", value))
		case "basic":
			httpCmd = append(httpCmd, "--user", value)
		case "token":
			httpCmd = append(httpCmd, "-H", fmt.Sprintf("\"Authorization: Token %s\"", value))
		}
	}

	// Handle special operations
	if downloadPath, exists := options["download"]; exists {
		httpCmd = append(httpCmd, "-o", downloadPath)
	}

	if uploadPath, exists := options["upload"]; exists {
		httpCmd = append(httpCmd, "-T", uploadPath)
	}

	// Add body
	if body != "" {
		httpCmd = append(httpCmd, "-d", body)
	}

	// Add advanced options
	if timeout, exists := options["timeout"]; exists {
		httpCmd = append(httpCmd, "--max-time", timeout)
	}
	if retry, exists := options["retry"]; exists {
		httpCmd = append(httpCmd, "--retry", retry)
	}
	if followRedirects, exists := options["follow_redirects"]; exists && followRedirects == "true" {
		httpCmd = append(httpCmd, "-L")
	}
	if insecure, exists := options["insecure"]; exists && insecure == "true" {
		httpCmd = append(httpCmd, "-k")
	}
	if verbose, exists := options["verbose"]; exists && verbose == "true" {
		httpCmd = append(httpCmd, "-v")
	}
	if silent, exists := options["silent"]; exists && silent == "true" {
		httpCmd = append(httpCmd, "-s")
	}

	// Add URL last
	httpCmd = append(httpCmd, url)

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute HTTP command: %s\n", strings.Join(httpCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(httpCmd, " "))
	}

	// For now, we'll simulate the HTTP request execution
	// In a real implementation, you would use exec.Command to run the curl command
	// or use Go's http.Client for more advanced features
	// cmd := exec.Command(httpCmd[0], httpCmd[1:]...)
	// return cmd.Run()

	return nil
}

// buildNetworkCommand builds and executes network commands
func (e *Engine) buildNetworkCommand(action, target, port, condition string, options map[string]string, dryRun bool) error {
	var networkCmd []string

	switch action {
	case "health_check":
		// Use curl for health checks with status code validation
		networkCmd = append(networkCmd, "curl", "-f", "-s", "-S")

		// Add timeout if specified
		if timeout, exists := options["timeout"]; exists {
			networkCmd = append(networkCmd, "--max-time", timeout)
		} else {
			networkCmd = append(networkCmd, "--max-time", "10") // Default 10s timeout
		}

		// Add retry if specified
		if retry, exists := options["retry"]; exists {
			networkCmd = append(networkCmd, "--retry", retry)
		}

		// Add condition checking
		if condition != "" {
			if condition == "200" || strings.HasPrefix(condition, "20") {
				networkCmd = append(networkCmd, "-w", "%{http_code}")
			}
		}

		networkCmd = append(networkCmd, target)

	case "wait_for_service":
		// Create a retry loop for service waiting
		timeout := "60" // Default 60s timeout
		if t, exists := options["timeout"]; exists {
			timeout = t
		}

		retryInterval := "2" // Default 2s retry interval
		if r, exists := options["retry"]; exists {
			retryInterval = r
		}

		// Build a shell script for waiting
		script := fmt.Sprintf(`
timeout=%s
interval=%s
elapsed=0
while [ $elapsed -lt $timeout ]; do
  if curl -f -s -S --max-time 5 "%s" > /dev/null 2>&1; then
    echo "Service is ready"
    exit 0
  fi
  sleep $interval
  elapsed=$((elapsed + interval))
  echo "Waiting for service... ($elapsed/${timeout}s)"
done
echo "Timeout waiting for service"
exit 1`, timeout, retryInterval, target)

		networkCmd = []string{"sh", "-c", script}

	case "port_check":
		// Use netcat for port checking
		networkCmd = append(networkCmd, "nc", "-z")

		// Add timeout if specified
		if timeout, exists := options["timeout"]; exists {
			networkCmd = append(networkCmd, "-w", timeout)
		} else {
			networkCmd = append(networkCmd, "-w", "5") // Default 5s timeout
		}

		networkCmd = append(networkCmd, target, port)

	case "ping":
		// Use ping command
		networkCmd = append(networkCmd, "ping", "-c", "1")

		// Add timeout if specified (ping uses different timeout format)
		if timeout, exists := options["timeout"]; exists {
			networkCmd = append(networkCmd, "-W", timeout)
		}

		networkCmd = append(networkCmd, target)

	default:
		return fmt.Errorf("unknown network action: %s", action)
	}

	if dryRun {
		_, _ = fmt.Fprintf(e.output, "[DRY RUN] Would execute network command: %s\n", strings.Join(networkCmd, " "))
		return nil
	}

	// Show the actual command being executed
	if e.verbose {
		_, _ = fmt.Fprintf(e.output, "Command: %s\n", strings.Join(networkCmd, " "))
	}

	// For now, we'll simulate the network command execution
	// In a real implementation, you would use exec.Command to run the network command
	// cmd := exec.Command(networkCmd[0], networkCmd[1:]...)
	// return cmd.Run()

	return nil
}
