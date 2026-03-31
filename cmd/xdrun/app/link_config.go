package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Domain: Directory link management
// Maintains a mapping between directories and drun task files stored in the user's home directory.

// LinkConfig represents the persisted set of directory links.
type LinkConfig struct {
	Links map[string]string `yaml:"links"`
}

func getLinkConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".drun", "links.yml"), nil
}

func loadLinkConfig() (*LinkConfig, error) {
	configPath, err := getLinkConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &LinkConfig{
			Links: make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read link config: %w", err)
	}

	var config LinkConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse link config: %w", err)
	}

	if config.Links == nil {
		config.Links = make(map[string]string)
	}

	return &config, nil
}

func saveLinkConfig(config *LinkConfig) error {
	configPath, err := getLinkConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create link config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal link config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write link config: %w", err)
	}

	return nil
}

func clearLinkConfig() error {
	configPath, err := getLinkConfigPath()
	if err != nil {
		return err
	}

	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove link config: %w", err)
	}

	return nil
}

// LinkDirectories registers the provided directories to the given task file.
// When taskFile is empty, the current discovery routines are used to determine the task file.
func LinkDirectories(directories []string, taskFile string) error {
	parsedDirs := parseDirectoryArgs(directories)
	if len(parsedDirs) == 0 {
		return fmt.Errorf("no directories provided")
	}

	var targetFile string
	var err error
	if strings.TrimSpace(taskFile) == "" {
		targetFile, err = FindConfigFile("")
		if err != nil {
			return fmt.Errorf("failed to detect task file: %w", err)
		}
	} else {
		targetFile = strings.TrimSpace(taskFile)
		if !filepath.IsAbs(targetFile) {
			targetFile, err = filepath.Abs(targetFile)
			if err != nil {
				return fmt.Errorf("failed to resolve task file path: %w", err)
			}
		}
	}

	if !filepath.IsAbs(targetFile) {
		targetFile, err = filepath.Abs(targetFile)
		if err != nil {
			return fmt.Errorf("failed to resolve task file path: %w", err)
		}
	}

	if _, err := os.Stat(targetFile); err != nil {
		return fmt.Errorf("linked task file '%s' not found: %w", targetFile, err)
	}

	config, err := loadLinkConfig()
	if err != nil {
		return err
	}

	if config.Links == nil {
		config.Links = make(map[string]string)
	}

	for _, dir := range parsedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to resolve directory '%s': %w", dir, err)
		}

		info, err := os.Stat(absDir)
		if err != nil {
			return fmt.Errorf("directory '%s' not found: %w", absDir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("'%s' is not a directory", absDir)
		}

		config.Links[absDir] = targetFile
		fmt.Printf("🔗 Linked %s ➜ %s\n", absDir, targetFile)
	}

	if err := saveLinkConfig(config); err != nil {
		return err
	}

	return nil
}

// UnlinkDirectories removes the provided directories from the link configuration.
func UnlinkDirectories(directories []string) error {
	parsedDirs := parseDirectoryArgs(directories)
	if len(parsedDirs) == 0 {
		return fmt.Errorf("no directories provided")
	}

	config, err := loadLinkConfig()
	if err != nil {
		return err
	}

	changed := false
	for _, dir := range parsedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to resolve directory '%s': %w", dir, err)
		}

		if _, exists := config.Links[absDir]; exists {
			delete(config.Links, absDir)
			fmt.Printf("❌  Unlinked %s\n", absDir)
			changed = true
		} else {
			fmt.Printf("ℹ️  Directory %s was not linked\n", absDir)
		}
	}

	if !changed {
		return nil
	}

	if len(config.Links) == 0 {
		return clearLinkConfig()
	}

	return saveLinkConfig(config)
}

// UnlinkAllDirectories removes all directory links.
func UnlinkAllDirectories() error {
	if err := clearLinkConfig(); err != nil {
		return err
	}
	fmt.Println("🧹 Removed all directory links")
	return nil
}

// parseDirectoryArgs expands comma-separated directory lists.
func parseDirectoryArgs(args []string) []string {
	seen := make(map[string]struct{})
	var result []string

	for _, arg := range args {
		for _, part := range strings.Split(arg, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}

	return result
}

// getLinkedConfigFile returns the linked task file for the provided directory or its parents.
func getLinkedConfigFile(directory string) (string, error) {
	config, err := loadLinkConfig()
	if err != nil {
		return "", err
	}

	if len(config.Links) == 0 {
		return "", nil
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory: %w", err)
	}

	current := absDir
	for {
		if linkedFile, exists := config.Links[current]; exists {
			if _, err := os.Stat(linkedFile); err != nil {
				return "", fmt.Errorf("linked task file '%s' for directory '%s' not found: %w", linkedFile, current, err)
			}
			return linkedFile, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", nil
}
