package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// NetworkManager handles Docker network operations
type NetworkManager struct{}

// NewNetworkManager creates a new network manager
func NewNetworkManager() *NetworkManager {
	return &NetworkManager{}
}

// CheckNetworkExists checks if a Docker network exists
func (nm *NetworkManager) CheckNetworkExists(ctx context.Context, networkName string) (bool, error) {
	// #nosec G204 -- docker network inspection intentionally uses the requested network name.
	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--format", "{{.Name}}", "--filter", fmt.Sprintf("name=^%s$", networkName))
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list networks: %w", err)
	}

	return strings.TrimSpace(string(output)) == networkName, nil
}

// CreateNetwork creates a Docker network
func (nm *NetworkManager) CreateNetwork(ctx context.Context, networkName, driver string, options map[string]string) error {
	args := []string{"network", "create"}

	if driver != "" {
		args = append(args, "--driver", driver)
	}

	// Add options
	for key, value := range options {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, networkName)

	// #nosec G204 -- docker network creation intentionally uses the requested driver and options.
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w\nOutput: %s", networkName, err, string(output))
	}

	return nil
}

// RemoveNetwork removes a Docker network
func (nm *NetworkManager) RemoveNetwork(ctx context.Context, networkName string) error {
	// #nosec G204 -- docker network removal intentionally uses the requested network name.
	cmd := exec.CommandContext(ctx, "docker", "network", "rm", networkName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove network %s: %w\nOutput: %s", networkName, err, string(output))
	}

	return nil
}

// GetNetworkInfo gets information about a Docker network
func (nm *NetworkManager) GetNetworkInfo(ctx context.Context, networkName string) (map[string]string, error) {
	// #nosec G204 -- docker network inspection intentionally uses the requested network name.
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", networkName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect network %s: %w\nOutput: %s", networkName, err, string(output))
	}

	// Parse the JSON output to extract relevant information
	// This is a simplified implementation - in production you'd want proper JSON parsing
	info := make(map[string]string)
	info["raw_output"] = string(output)

	return info, nil
}

// WaitForNetwork waits for a network to be available
func (nm *NetworkManager) WaitForNetwork(ctx context.Context, networkName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			exists, err := nm.CheckNetworkExists(ctx, networkName)
			if err != nil {
				return err
			}
			if exists {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for network %s", networkName)
			}
		}
	}
}
