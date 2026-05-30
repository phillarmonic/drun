package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/phillarmonic/drun/internal/ast"
)

// Domain: Task Name Resolution
// This file contains logic for matching partial task names with disambiguation

// ResolvePartialTaskName resolves a partial task name to a full task name
// Returns the full task name if exactly one match is found
// Returns an error if multiple matches are found (ambiguous) or no matches
func ResolvePartialTaskName(partialName string, program *ast.Program) (string, error) {
	// If the partial name exactly matches a task, return it
	for _, task := range program.Tasks {
		if task.Name == partialName {
			return task.Name, nil
		}
	}

	// Find all tasks that start with the partial name
	var matches []string
	for _, task := range program.Tasks {
		if strings.HasPrefix(task.Name, partialName) {
			matches = append(matches, task.Name)
		}
	}

	// No matches found
	if len(matches) == 0 {
		// Try to suggest similar task names
		suggestions := findSimilarTaskNames(partialName, program)
		if len(suggestions) > 0 {
			return "", fmt.Errorf("task '%s' not found\n\nDid you mean one of these?\n%s",
				partialName, formatTaskSuggestions(suggestions))
		}
		return "", fmt.Errorf("task '%s' not found", partialName)
	}

	// Exactly one match - return it
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches - ambiguous
	sort.Strings(matches)
	return "", fmt.Errorf("ambiguous task name '%s' - matches multiple tasks:\n%s\n\nPlease use more characters to disambiguate",
		partialName, formatTaskMatches(matches, partialName))
}

// findSimilarTaskNames finds task names that are similar to the given name
func findSimilarTaskNames(name string, program *ast.Program) []string {
	var similar []string
	nameLower := strings.ToLower(name)

	for _, task := range program.Tasks {
		taskLower := strings.ToLower(task.Name)

		// Check if the task name contains the partial name
		if strings.Contains(taskLower, nameLower) {
			similar = append(similar, task.Name)
			continue
		}

		// Check if they have similar prefixes (first 2-3 characters)
		if len(name) >= 2 && len(task.Name) >= 2 {
			if nameLower[:2] == taskLower[:2] {
				similar = append(similar, task.Name)
				continue
			}
		}

		// Check Levenshtein distance for short names
		if len(name) >= 3 && levenshteinDistance(nameLower, taskLower) <= 2 {
			similar = append(similar, task.Name)
		}
	}

	sort.Strings(similar)
	return similar
}

// formatTaskMatches formats task matches for display
func formatTaskMatches(matches []string, partialName string) string {
	var result strings.Builder
	for _, match := range matches {
		fmt.Fprintf(&result, "  - %s (use: xdrun %s)\n",
			match, getDisambiguatingPrefix(match, matches))
	}
	return result.String()
}

// formatTaskSuggestions formats task suggestions for display
func formatTaskSuggestions(suggestions []string) string {
	var result strings.Builder
	for _, suggestion := range suggestions {
		fmt.Fprintf(&result, "  - %s\n", suggestion)
	}
	return result.String()
}

// getDisambiguatingPrefix returns the shortest prefix that uniquely identifies a task
func getDisambiguatingPrefix(taskName string, allMatches []string) string {
	// Start with the full name
	prefix := taskName

	// Try progressively shorter prefixes
	for i := 1; i <= len(taskName); i++ {
		testPrefix := taskName[:i]
		matchCount := 0

		for _, match := range allMatches {
			if strings.HasPrefix(match, testPrefix) {
				matchCount++
			}
		}

		if matchCount == 1 {
			return testPrefix
		}
	}

	return prefix
}

// levenshteinDistance calculates the Levenshtein distance between two strings
// This is used for fuzzy matching when suggesting similar task names
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
