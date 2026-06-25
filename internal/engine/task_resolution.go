package engine

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
	"github.com/phillarmonic/drun/v2/internal/platform"
)

func enforceDeclarationPlatform(kind, name string, annotations []ast.Annotation) error {
	meta, err := platform.ValidateAnnotations(kind, name, annotations)
	if err != nil {
		return err
	}
	if len(meta.Platforms) == 0 || platform.MatchesCurrent(meta.Platforms) {
		return nil
	}
	return fmt.Errorf("%s %q is only available on [%s]; current platform is %s", kind, name, platform.FormatList(meta.Platforms), platform.Current())
}

func selectTaskVariant(name string, tasks []*ast.TaskStatement) (*ast.TaskStatement, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task %q not found", name)
	}

	current := platform.Current()
	var available []string
	var fallback *ast.TaskStatement
	for _, task := range tasks {
		meta, err := platform.ValidateAnnotations("task", task.Name, task.Annotations)
		if err != nil {
			return nil, err
		}
		if len(meta.Platforms) == 0 {
			if fallback == nil {
				fallback = task
			}
			continue
		}
		available = append(available, platform.FormatList(meta.Platforms))
		for _, allowed := range meta.Platforms {
			if allowed == current {
				return task, nil
			}
		}
	}

	if fallback != nil {
		return fallback, nil
	}

	return nil, fmt.Errorf("task %q has no variant for platform %s; available variants: %s", name, current, strings.Join(uniqueStrings(available), "; "))
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func resolveTaskVariantByName(name string, tasks []*ast.TaskStatement) (*ast.TaskStatement, error) {
	var matches []*ast.TaskStatement
	for _, task := range tasks {
		if task.Name == name {
			matches = append(matches, task)
		}
	}
	return selectTaskVariant(name, matches)
}
