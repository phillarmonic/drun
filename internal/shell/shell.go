package shell

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/phillarmonic/drun/internal/model"
)

// Shell represents a shell configuration
type Shell struct {
	Cmd  string
	Args []string
	OS   string
}

// Selector handles shell selection based on OS and configuration
type Selector struct {
	shellConfigs map[string]model.ShellConfig
}

// NewSelector creates a new shell selector
func NewSelector(shellConfigs map[string]model.ShellConfig) *Selector {
	return &Selector{
		shellConfigs: shellConfigs,
	}
}

// Select selects the appropriate shell based on the shell preference and current OS
func (s *Selector) Select(shellPref string, targetOS string) (*Shell, error) {
	if targetOS == "" {
		targetOS = runtime.GOOS
	}

	var shellConfig model.ShellConfig
	var found bool

	if shellPref == "auto" || shellPref == "" {
		// Use OS-specific default
		shellConfig, found = s.shellConfigs[targetOS]
		if !found {
			return nil, fmt.Errorf("no shell configuration found for OS: %s", targetOS)
		}
	} else {
		// Use specific shell configuration
		shellConfig, found = s.shellConfigs[shellPref]
		if !found {
			return nil, fmt.Errorf("shell configuration '%s' not found", shellPref)
		}
	}

	return &Shell{
		Cmd:  shellConfig.Cmd,
		Args: shellConfig.Args,
		OS:   targetOS,
	}, nil
}

// BuildCommand builds the command arguments for executing a script
func (sh *Shell) BuildCommand(script string) []string {
	args := make([]string, len(sh.Args))
	copy(args, sh.Args)

	if sh.OS == "windows" {
		// For PowerShell, we need to handle the script differently
		// Convert some common shell idioms
		script = sh.convertShellIdioms(script)
	}

	args = append(args, script)
	return args
}

// convertShellIdioms converts common POSIX shell idioms to PowerShell equivalents
func (sh *Shell) convertShellIdioms(script string) string {
	if sh.OS != "windows" {
		return script
	}

	// Basic conversions for PowerShell compatibility
	// This is a simplified version - a full implementation would be more comprehensive

	lines := strings.Split(script, "\n")
	var converted []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Convert && to PowerShell equivalent
		if strings.Contains(line, " && ") {
			parts := strings.Split(line, " && ")
			line = strings.Join(parts, "; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; ")
		}

		// Convert || to PowerShell equivalent
		if strings.Contains(line, " || ") {
			parts := strings.Split(line, " || ")
			line = strings.Join(parts, "; if ($LASTEXITCODE -eq 0) { } else { ")
			line += " }"
		}

		converted = append(converted, line)
	}

	return strings.Join(converted, "\n")
}

// Quote quotes a string for the shell
func (sh *Shell) Quote(s string) string {
	if sh.OS == "windows" {
		// PowerShell quoting
		if strings.Contains(s, " ") || strings.Contains(s, "'") || strings.Contains(s, "\"") {
			// Use single quotes and escape single quotes
			return "'" + strings.ReplaceAll(s, "'", "''") + "'"
		}
		return s
	}

	// POSIX shell quoting
	if strings.Contains(s, "'") {
		// If string contains single quotes, use double quotes
		return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\""
	}

	// Use single quotes for simplicity
	return "'" + s + "'"
}

// IsWindows returns true if this is a Windows shell
func (sh *Shell) IsWindows() bool {
	return sh.OS == "windows"
}

// IsPOSIX returns true if this is a POSIX-compatible shell
func (sh *Shell) IsPOSIX() bool {
	return !sh.IsWindows()
}
