package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// UserConfig represents user-level drun settings stored in the home directory.
type UserConfig struct {
	ExtraTaskFileSearchPaths []string `yaml:"extraTaskFileSearchPaths"`
}

func getUserConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".drun", "config.yml"), nil
}

func loadUserConfig() (*UserConfig, error) {
	configPath, err := getUserConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &UserConfig{}, nil
	}

	// #nosec G304 -- user config is intentionally loaded from the user's home config path.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config: %w", err)
	}

	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse user config: %w", err)
	}

	config.ExtraTaskFileSearchPaths = normalizeSearchPaths(config.ExtraTaskFileSearchPaths)
	return &config, nil
}

func normalizeSearchPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(paths))
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}
