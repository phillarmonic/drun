package make2drun

import (
	"fmt"
	"regexp"
	"strings"
)

// GenerateDrun converts a Makefile to drun v2 syntax
func GenerateDrun(makefile *Makefile) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Auto-generated from Makefile\n")
	sb.WriteString("# Created by make2drun converter\n\n")
	sb.WriteString("version: 2.0\n\n")

	// Note: Makefile variables will be converted to drun variables within tasks
	if len(makefile.Variables) > 0 {
		sb.WriteString("# Variables from Makefile (will be set in tasks):\n")
		for key, value := range makefile.Variables {
			drunVarName := strings.ToLower(key)
			drunValue := convertMakeVariableValue(value)
			sb.WriteString(fmt.Sprintf("# - $%s = \"%s\"\n", drunVarName, drunValue))
		}
		sb.WriteString("\n")
	}

	// Generate tasks from targets
	for _, target := range makefile.Targets {
		// Check if task commands use any variables
		includeVars := taskUsesVariables(target, makefile)
		generateTaskWithVars(&sb, target, makefile, includeVars)
	}

	return sb.String()
}

// taskUsesVariables checks if a task's commands reference any Makefile variables
func taskUsesVariables(target *MakefileTarget, makefile *Makefile) bool {
	if len(makefile.Variables) == 0 {
		return false
	}

	// Check each command for variable references
	for _, cmd := range target.Commands {
		for varName := range makefile.Variables {
			// Check for $(VAR) or ${VAR} patterns
			if strings.Contains(cmd, "$("+varName+")") || strings.Contains(cmd, "${"+varName+"}") {
				return true
			}
		}
	}
	return false
}

func generateTaskWithVars(sb *strings.Builder, target *MakefileTarget, makefile *Makefile, includeVars bool) {
	taskName := target.Name
	description := target.Description
	if description == "" {
		description = fmt.Sprintf("Run %s target", target.Name)
	}

	// Task header
	fmt.Fprintf(sb, "task \"%s\" means \"%s\":\n", taskName, description)

	// Dependencies
	if len(target.Dependencies) > 0 {
		// Quote each dependency in case it contains special characters
		quotedDeps := make([]string, len(target.Dependencies))
		for i, dep := range target.Dependencies {
			quotedDeps[i] = fmt.Sprintf("\"%s\"", dep)
		}
		fmt.Fprintf(sb, "\tdepends on %s\n", strings.Join(quotedDeps, ", "))
		sb.WriteString("\n")
	}

	// Include variables in first task if requested
	if includeVars && len(makefile.Variables) > 0 {
		sb.WriteString("\t# Set variables from Makefile\n")
		for key, value := range makefile.Variables {
			drunVarName := strings.ToLower(key)
			drunValue := convertMakeVariableValue(value)
			fmt.Fprintf(sb, "\tset $%s to \"%s\"\n", drunVarName, drunValue)
		}
		sb.WriteString("\n")
	}

	// Info message
	fmt.Fprintf(sb, "\tinfo \"Running %s\"\n", taskName)

	// Commands
	if len(target.Commands) > 0 {
		sb.WriteString("\n")
		for _, cmd := range target.Commands {
			generateCommand(sb, cmd, makefile)
		}
	}

	// Success message
	fmt.Fprintf(sb, "\n\tsuccess \"%s completed successfully!\"\n", taskName)
	sb.WriteString("\n")
}

func generateCommand(sb *strings.Builder, cmd string, makefile *Makefile) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	// Handle special prefixes
	silent := false
	ignoreErrors := false

	if strings.HasPrefix(cmd, "@") {
		silent = true
		cmd = strings.TrimPrefix(cmd, "@")
		cmd = strings.TrimSpace(cmd)
	}

	if strings.HasPrefix(cmd, "-") {
		ignoreErrors = true
		cmd = strings.TrimPrefix(cmd, "-")
		cmd = strings.TrimSpace(cmd)
	}

	// Convert Make variables to drun interpolation
	cmd = convertMakeVariables(cmd)

	// Check if command contains shell features
	needsShell := strings.Contains(cmd, "&&") ||
		strings.Contains(cmd, "||") ||
		strings.Contains(cmd, "|") ||
		strings.Contains(cmd, ">") ||
		strings.Contains(cmd, "<") ||
		strings.Contains(cmd, "$")

	// Special handling for common commands
	if strings.HasPrefix(cmd, "echo ") {
		// Convert echo to info/echo
		message := strings.TrimPrefix(cmd, "echo ")
		message = strings.Trim(message, "\"'")
		if silent {
			fmt.Fprintf(sb, "\techo \"%s\"\n", escapeQuotes(message))
		} else {
			fmt.Fprintf(sb, "\techo \"%s\"\n", escapeQuotes(message))
		}
	} else if strings.HasPrefix(cmd, "mkdir ") {
		// Convert mkdir to create dir
		dirPath := strings.TrimPrefix(cmd, "mkdir ")
		dirPath = strings.TrimSpace(strings.TrimPrefix(dirPath, "-p "))
		dirPath = strings.Trim(dirPath, "\"'")
		fmt.Fprintf(sb, "\tcreate dir \"%s\"\n", dirPath)
	} else if strings.HasPrefix(cmd, "rm ") {
		// Convert rm to delete
		filePath := strings.TrimPrefix(cmd, "rm ")
		filePath = strings.TrimSpace(strings.TrimPrefix(filePath, "-rf "))
		filePath = strings.TrimSpace(strings.TrimPrefix(filePath, "-f "))
		filePath = strings.Trim(filePath, "\"'")
		fmt.Fprintf(sb, "\tdelete \"%s\"\n", filePath)
	} else if needsShell || strings.Contains(cmd, "\n") {
		// Use shell command for complex operations
		if ignoreErrors {
			sb.WriteString("\ttry:\n")
			fmt.Fprintf(sb, "\t\trun \"%s\"\n", escapeQuotes(cmd))
			sb.WriteString("\tignore:\n")
			sb.WriteString("\t\twarn \"Command failed but continuing\"\n")
		} else {
			fmt.Fprintf(sb, "\trun \"%s\"\n", escapeQuotes(cmd))
		}
	} else {
		// Default to run command
		if ignoreErrors {
			sb.WriteString("\ttry:\n")
			fmt.Fprintf(sb, "\t\trun \"%s\"\n", escapeQuotes(cmd))
			sb.WriteString("\tignore:\n")
			sb.WriteString("\t\twarn \"Command failed but continuing\"\n")
		} else {
			fmt.Fprintf(sb, "\trun \"%s\"\n", escapeQuotes(cmd))
		}
	}
}

// convertMakeVariables converts Make variable references to drun interpolation
func convertMakeVariables(s string) string {
	// Convert $(VAR) to {$var}
	re1 := regexp.MustCompile(`\$\(([A-Z_][A-Z0-9_]*)\)`)
	s = re1.ReplaceAllStringFunc(s, func(match string) string {
		varName := re1.FindStringSubmatch(match)[1]
		return fmt.Sprintf("{$%s}", strings.ToLower(varName))
	})

	// Convert ${VAR} to {$var}
	re2 := regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)
	s = re2.ReplaceAllStringFunc(s, func(match string) string {
		varName := re2.FindStringSubmatch(match)[1]
		return fmt.Sprintf("{$%s}", strings.ToLower(varName))
	})

	return s
}

// convertMakeVariableValue converts a Make variable value to drun format
func convertMakeVariableValue(s string) string {
	// Convert Make variable references to drun interpolation
	s = convertMakeVariables(s)

	// Escape quotes
	s = escapeQuotes(s)

	return s
}

// escapeQuotes escapes double quotes in strings
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
