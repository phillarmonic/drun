package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Domain: Folder trust management
// Maintains a list of directories trusted for security-sensitive operations such as
// open url. The list is stored in the user's home directory at ~/.drun/trusted.yml.

// TrustConfig represents the persisted set of trusted directories.
type TrustConfig struct {
	TrustedDirs []string `yaml:"trustedDirs"`
}

func getTrustConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".drun", "trusted.yml"), nil
}

func loadTrustConfig() (*TrustConfig, error) {
	configPath, err := getTrustConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &TrustConfig{}, nil
	}

	// #nosec G304 -- trust config is intentionally loaded from the user's home config path.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read trust config: %w", err)
	}

	var config TrustConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse trust config: %w", err)
	}

	return &config, nil
}

func saveTrustConfig(config *TrustConfig) error {
	configPath, err := getTrustConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal trust config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write trust config: %w", err)
	}

	return nil
}

// IsDirTrusted checks whether the given absolute directory path (or any of its
// parents) is in the trusted list.
func IsDirTrusted(dir string) (bool, error) {
	config, err := loadTrustConfig()
	if err != nil {
		return false, err
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("failed to resolve directory: %w", err)
	}

	trusted := make(map[string]struct{}, len(config.TrustedDirs))
	for _, d := range config.TrustedDirs {
		trusted[d] = struct{}{}
	}

	current := absDir
	for {
		if _, ok := trusted[current]; ok {
			return true, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return false, nil
}

// TrustDir adds the given directory to the trusted list.
func TrustDir(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	config, err := loadTrustConfig()
	if err != nil {
		return err
	}

	for _, d := range config.TrustedDirs {
		if d == absDir {
			return nil // already trusted
		}
	}

	config.TrustedDirs = append(config.TrustedDirs, absDir)
	sort.Strings(config.TrustedDirs)
	return saveTrustConfig(config)
}

// UntrustDir removes the given directory from the trusted list.
func UntrustDir(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	config, err := loadTrustConfig()
	if err != nil {
		return err
	}

	filtered := config.TrustedDirs[:0]
	found := false
	for _, d := range config.TrustedDirs {
		if d == absDir {
			found = true
			continue
		}
		filtered = append(filtered, d)
	}

	if !found {
		return nil
	}

	config.TrustedDirs = filtered
	return saveTrustConfig(config)
}
