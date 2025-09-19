package patterns

import (
	"fmt"
	"regexp"
)

// PatternMacro represents a predefined pattern macro
type PatternMacro struct {
	Name        string
	Pattern     string
	Description string
}

// Built-in pattern macros
var builtinMacros = map[string]PatternMacro{
	"semver": {
		Name:        "semver",
		Pattern:     `^v\d+\.\d+\.\d+$`,
		Description: "Basic semantic versioning (e.g., v1.2.3)",
	},
	"semver_extended": {
		Name:        "semver_extended",
		Pattern:     `^v\d+\.\d+\.\d+(-[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?(\+[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?$`,
		Description: "Extended semantic versioning with pre-release and build metadata (e.g., v2.0.1-RC2, v1.0.0-alpha.1+build.123)",
	},
	"uuid": {
		Name:        "uuid",
		Pattern:     `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
		Description: "UUID format (e.g., 550e8400-e29b-41d4-a716-446655440000)",
	},
	"url": {
		Name:        "url",
		Pattern:     `https?://[^\s/$.?#].[^\s]*`,
		Description: "HTTP/HTTPS URL format",
	},
	"ipv4": {
		Name:        "ipv4",
		Pattern:     `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`,
		Description: "IPv4 address format (e.g., 192.168.1.1)",
	},
	"slug": {
		Name:        "slug",
		Pattern:     `^[a-z0-9]+(?:-[a-z0-9]+)*$`,
		Description: "URL slug format (lowercase, hyphens only, e.g., my-project-name)",
	},
	"docker_tag": {
		Name:        "docker_tag",
		Pattern:     `^[a-zA-Z0-9][a-zA-Z0-9._-]*$`,
		Description: "Docker image tag format",
	},
	"git_branch": {
		Name:        "git_branch",
		Pattern:     `^[a-zA-Z0-9][a-zA-Z0-9._/-]*[a-zA-Z0-9]$`,
		Description: "Git branch name format",
	},
}

// GetMacro returns a pattern macro by name
func GetMacro(name string) (PatternMacro, bool) {
	macro, exists := builtinMacros[name]
	return macro, exists
}

// GetAllMacros returns all available pattern macros
func GetAllMacros() map[string]PatternMacro {
	return builtinMacros
}

// ValidatePattern validates a string against a pattern macro
func ValidatePattern(value, macroName string) error {
	macro, exists := GetMacro(macroName)
	if !exists {
		return fmt.Errorf("unknown pattern macro: %s", macroName)
	}

	matched, err := regexp.MatchString(macro.Pattern, value)
	if err != nil {
		return fmt.Errorf("invalid regex pattern in macro '%s': %w", macroName, err)
	}

	if !matched {
		return fmt.Errorf("value '%s' does not match %s pattern (%s)", value, macro.Name, macro.Description)
	}

	return nil
}

// ExpandMacro expands a pattern macro to its regex pattern
func ExpandMacro(macroName string) (string, error) {
	macro, exists := GetMacro(macroName)
	if !exists {
		return "", fmt.Errorf("unknown pattern macro: %s", macroName)
	}

	return macro.Pattern, nil
}
