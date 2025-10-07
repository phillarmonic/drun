package make2drun

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// MakefileTarget represents a target in a Makefile
type MakefileTarget struct {
	Name         string
	Dependencies []string
	Commands     []string
	Description  string
	Variables    map[string]string
	IsPhony      bool
}

// Makefile represents a parsed Makefile
type Makefile struct {
	Targets   []*MakefileTarget
	Variables map[string]string
}

// ParseMakefile parses a Makefile and returns a structured representation
func ParseMakefile(filepath string) (*Makefile, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Makefile: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close Makefile: %w", cerr)
		}
	}()

	makefile := &Makefile{
		Targets:   make([]*MakefileTarget, 0),
		Variables: make(map[string]string),
	}

	scanner := bufio.NewScanner(file)
	var currentTarget *MakefileTarget
	var currentDescription string
	var phonyTargets []string
	var continuedLine string
	var inRecipe bool

	targetRegex := regexp.MustCompile(`^([a-zA-Z0-9_\-\.]+)\s*:\s*(.*)$`)
	variableRegex := regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)\s*[:?]?=\s*(.*)$`)
	commentRegex := regexp.MustCompile(`^#\s*(.*)$`)
	phonyRegex := regexp.MustCompile(`^\.PHONY:\s*(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Handle line continuation
		if strings.HasSuffix(trimmedLine, "\\") {
			continuedLine += strings.TrimSuffix(trimmedLine, "\\") + " "
			continue
		}
		if continuedLine != "" {
			line = continuedLine + trimmedLine
			trimmedLine = strings.TrimSpace(line)
			continuedLine = ""
		}

		// Skip empty lines
		if trimmedLine == "" {
			inRecipe = false
			if currentTarget != nil {
				makefile.Targets = append(makefile.Targets, currentTarget)
				currentTarget = nil
			}
			continue
		}

		// Check for .PHONY targets
		if match := phonyRegex.FindStringSubmatch(trimmedLine); match != nil {
			phonyTargets = append(phonyTargets, strings.Fields(match[1])...)
			continue
		}

		// Check for comments (potential descriptions)
		if match := commentRegex.FindStringSubmatch(trimmedLine); match != nil {
			currentDescription = strings.TrimSpace(match[1])
			continue
		}

		// Check for variable definitions
		if match := variableRegex.FindStringSubmatch(trimmedLine); match != nil && !inRecipe {
			makefile.Variables[match[1]] = strings.TrimSpace(match[2])
			currentDescription = ""
			continue
		}

		// Check for target definitions
		if match := targetRegex.FindStringSubmatch(line); match != nil && !strings.HasPrefix(line, "\t") {
			// Save previous target
			if currentTarget != nil {
				makefile.Targets = append(makefile.Targets, currentTarget)
			}

			targetName := strings.TrimSpace(match[1])
			depsStr := strings.TrimSpace(match[2])

			deps := []string{}
			if depsStr != "" {
				for _, dep := range strings.Fields(depsStr) {
					deps = append(deps, strings.TrimSpace(dep))
				}
			}

			currentTarget = &MakefileTarget{
				Name:         targetName,
				Dependencies: deps,
				Commands:     make([]string, 0),
				Description:  currentDescription,
				Variables:    make(map[string]string),
			}
			currentDescription = ""
			inRecipe = true
			continue
		}

		// Check for recipe commands (lines starting with tab)
		if strings.HasPrefix(line, "\t") && currentTarget != nil {
			command := strings.TrimPrefix(line, "\t")
			// Preserve the command exactly as written, just trim leading tab
			currentTarget.Commands = append(currentTarget.Commands, command)
			continue
		}

		inRecipe = false
	}

	// Add the last target if exists
	if currentTarget != nil {
		makefile.Targets = append(makefile.Targets, currentTarget)
	}

	// Mark phony targets
	for _, target := range makefile.Targets {
		for _, phony := range phonyTargets {
			if target.Name == phony {
				target.IsPhony = true
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Makefile: %w", err)
	}

	return makefile, nil
}

// GetTarget retrieves a target by name
func (m *Makefile) GetTarget(name string) *MakefileTarget {
	for _, target := range m.Targets {
		if target.Name == name {
			return target
		}
	}
	return nil
}
