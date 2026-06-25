package platform

import (
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/phillarmonic/drun/v2/internal/ast"
)

const (
	Linux   = "linux"
	Mac     = "mac"
	Windows = "windows"
)

var supportedPlatforms = map[string]string{
	"linux":   Linux,
	"mac":     Mac,
	"darwin":  Mac,
	"windows": Windows,
}

func Normalize(name string) (string, error) {
	normalized, ok := supportedPlatforms[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return "", fmt.Errorf("unsupported platform %q (supported: linux, mac, windows)", name)
	}
	return normalized, nil
}

func Current() string {
	switch runtime.GOOS {
	case "darwin":
		return Mac
	case "linux":
		return Linux
	case "windows":
		return Windows
	default:
		return runtime.GOOS
	}
}

func FormatList(platforms []string) string {
	if len(platforms) == 0 {
		return ""
	}

	values := append([]string(nil), platforms...)
	sort.Strings(values)
	return strings.Join(values, ", ")
}

type Metadata struct {
	Platforms []string
}

func ValidateAnnotations(kind, name string, annotations []ast.Annotation) (Metadata, error) {
	var meta Metadata
	seenNames := make(map[string]struct{}, len(annotations))

	for _, annotation := range annotations {
		if _, exists := seenNames[annotation.Name]; exists {
			return Metadata{}, fmt.Errorf("%s %q has duplicate @%s annotation", kind, name, annotation.Name)
		}
		seenNames[annotation.Name] = struct{}{}

		switch annotation.Name {
		case "platform":
			if len(annotation.Args) == 0 {
				return Metadata{}, fmt.Errorf("%s %q: @platform requires at least one platform", kind, name)
			}

			seenPlatforms := make(map[string]struct{}, len(annotation.Args))
			for _, arg := range annotation.Args {
				normalized, err := Normalize(arg)
				if err != nil {
					return Metadata{}, fmt.Errorf("%s %q: %w", kind, name, err)
				}
				if _, exists := seenPlatforms[normalized]; exists {
					return Metadata{}, fmt.Errorf("%s %q: duplicate platform %q in @platform", kind, name, normalized)
				}
				seenPlatforms[normalized] = struct{}{}
				meta.Platforms = append(meta.Platforms, normalized)
			}
		default:
			return Metadata{}, fmt.Errorf("%s %q has unsupported annotation @%s", kind, name, annotation.Name)
		}
	}

	sort.Strings(meta.Platforms)
	return meta, nil
}

func MatchesCurrent(platforms []string) bool {
	if len(platforms) == 0 {
		return true
	}

	current := Current()
	for _, candidate := range platforms {
		if candidate == current {
			return true
		}
	}
	return false
}
