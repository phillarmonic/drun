package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Domain: Stateless Configuration Management
// This file contains logic for managing stateless drun configurations
// that are stored in the home directory instead of the repository

// StatelessConfig represents the stateless configuration
type StatelessConfig struct {
	// Map of directory paths to their configuration file locations in home directory
	Directories map[string]string `yaml:"directories"`
}

// getStatelessConfigPath returns the path to the stateless configuration file
func getStatelessConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".drun", "stateless.yml"), nil
}

// loadStatelessConfig loads the stateless configuration
func loadStatelessConfig() (*StatelessConfig, error) {
	configPath, err := getStatelessConfigPath()
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &StatelessConfig{
			Directories: make(map[string]string),
		}, nil
	}

	// #nosec G304 -- stateless config is intentionally loaded from the user's home config path.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stateless config: %w", err)
	}

	var config StatelessConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse stateless config: %w", err)
	}

	if config.Directories == nil {
		config.Directories = make(map[string]string)
	}

	return &config, nil
}

// saveStatelessConfig saves the stateless configuration
func saveStatelessConfig(config *StatelessConfig) error {
	configPath, err := getStatelessConfigPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal stateless config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write stateless config: %w", err)
	}

	return nil
}

// getStatelessConfigLocation returns the home directory location for a stateless directory
func getStatelessConfigLocation(directory string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create hash of the directory path
	hash := sha256.Sum256([]byte(absPath))
	hashStr := hex.EncodeToString(hash[:])[:16] // Use first 16 chars

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create location: ~/.drun/stateless/<hash>/spec.drun
	return filepath.Join(homeDir, ".drun", "stateless", hashStr, "spec.drun"), nil
}

// isStatelessDirectory checks if the current directory is marked as stateless
func isStatelessDirectory(directory string) (bool, string, error) {
	absPath, err := filepath.Abs(directory)
	if err != nil {
		return false, "", err
	}

	config, err := loadStatelessConfig()
	if err != nil {
		return false, "", err
	}

	configFile, exists := config.Directories[absPath]
	return exists, configFile, nil
}

// AddStatelessDirectory adds a directory to the stateless configuration
func AddStatelessDirectory(directory string, createTemplate bool) error {
	absPath, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load existing config
	config, err := loadStatelessConfig()
	if err != nil {
		return err
	}

	// Check if already exists
	if _, exists := config.Directories[absPath]; exists {
		return fmt.Errorf("directory '%s' is already marked as stateless", absPath)
	}

	// Get the config location
	configLocation, err := getStatelessConfigLocation(absPath)
	if err != nil {
		return err
	}

	// Add to config
	config.Directories[absPath] = configLocation

	// Save config
	if err := saveStatelessConfig(config); err != nil {
		return err
	}

	fmt.Printf("✅  Marked '%s' as stateless\n", absPath)
	fmt.Printf("📁 Configuration will be stored at: %s\n", configLocation)

	// Create template if requested
	if createTemplate {
		configDir := filepath.Dir(configLocation)
		if err := os.MkdirAll(configDir, 0750); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Check if file already exists
		if _, err := os.Stat(configLocation); err == nil {
			fmt.Printf("⚠️  Configuration file already exists, not overwriting\n")
			return nil
		}

		// Generate starter configuration
		template := generateStarterConfig()
		if err := os.WriteFile(configLocation, []byte(template), 0600); err != nil {
			return fmt.Errorf("failed to create template: %w", err)
		}

		fmt.Printf("📝  Created template configuration file\n")
	} else {
		fmt.Printf("💡 Run 'xdrun --init' in this directory to create a configuration file\n")
	}

	return nil
}

// RemoveStatelessDirectory removes a directory from the stateless configuration
func RemoveStatelessDirectory(directory string, deleteConfig bool) error {
	absPath, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load existing config
	config, err := loadStatelessConfig()
	if err != nil {
		return err
	}

	// Check if exists
	configLocation, exists := config.Directories[absPath]
	if !exists {
		return fmt.Errorf("directory '%s' is not marked as stateless", absPath)
	}

	// Delete config file if requested
	if deleteConfig {
		if err := os.Remove(configLocation); err != nil && !os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: Failed to delete config file: %v\n", err)
		} else {
			fmt.Printf("🗑️  Deleted configuration file: %s\n", configLocation)
		}

		// Try to remove parent directory if empty
		configDir := filepath.Dir(configLocation)
		if err := os.Remove(configDir); err == nil {
			fmt.Printf("🗑️  Removed empty directory: %s\n", configDir)
		}
	}

	// Remove from config
	delete(config.Directories, absPath)

	// Save config
	if err := saveStatelessConfig(config); err != nil {
		return err
	}

	fmt.Printf("✅  Removed stateless marking from '%s'\n", absPath)
	return nil
}

// ListStatelessDirectories lists all stateless directories
func ListStatelessDirectories() error {
	config, err := loadStatelessConfig()
	if err != nil {
		return err
	}

	if len(config.Directories) == 0 {
		fmt.Println("No stateless directories configured")
		return nil
	}

	fmt.Println("Stateless directories:")
	for dir, configFile := range config.Directories {
		// Check if config file exists
		exists := "✓"
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			exists = "✗"
		}
		fmt.Printf("  %s %s\n", exists, dir)
		fmt.Printf("    → %s\n", configFile)
	}

	return nil
}

// ShowStatelessInfo shows information about the current directory's stateless status
func ShowStatelessInfo() error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	isStateless, configLocation, err := isStatelessDirectory(pwd)
	if err != nil {
		return err
	}

	if isStateless {
		fmt.Printf("📁 Current directory is marked as STATELESS\n")
		fmt.Printf("   Config location: %s\n", configLocation)

		if _, err := os.Stat(configLocation); os.IsNotExist(err) {
			fmt.Printf("   Status: ⚠️  Config file does not exist\n")
			fmt.Printf("   Run 'xdrun --init' to create it\n")
		} else {
			fmt.Printf("   Status: ✅  Config file exists\n")
		}
	} else {
		fmt.Printf("📁 Current directory is NOT marked as stateless\n")
		fmt.Printf("   Using local configuration (.drun/spec.drun)\n")
		fmt.Printf("   Run 'xdrun cmd:stateless add' to mark as stateless\n")
	}

	return nil
}

// getStatelessConfigFile returns the config file for a stateless directory, or empty string if not stateless
func getStatelessConfigFile() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	isStateless, configLocation, err := isStatelessDirectory(pwd)
	if err != nil {
		return "", err
	}

	if !isStateless {
		return "", nil
	}

	return configLocation, nil
}
