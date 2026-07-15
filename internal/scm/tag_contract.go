package scm

import (
	"fmt"
	"regexp"
	"strings"
)

type TagFormatMacro struct {
	Name    string
	Pattern string
}

type TagFormatMacroRegistry struct {
	macros map[string]TagFormatMacro
}

func NewTagFormatMacroRegistry() *TagFormatMacroRegistry {
	return &TagFormatMacroRegistry{macros: make(map[string]TagFormatMacro)}
}

func DefaultTagFormatMacroRegistry() *TagFormatMacroRegistry {
	registry := NewTagFormatMacroRegistry()
	registry.Register(TagFormatMacro{Name: "version", Pattern: `[0-9]+\.[0-9]+\.[0-9]+`})
	return registry
}

func (r *TagFormatMacroRegistry) Register(macro TagFormatMacro) {
	r.macros[macro.Name] = macro
}

type GitVersionTagContract struct {
	Preset   string
	Formats  []string
	Pattern  string
	matchers []compiledTagMatcher
}

type compiledTagMatcher struct {
	pattern        *regexp.Regexp
	versionCapture int
}

func NewGitVersionTagContract(preset string, formats []string, pattern string, macros *TagFormatMacroRegistry) (*GitVersionTagContract, error) {
	forms := 0
	if preset != "" {
		forms++
	}
	if len(formats) > 0 {
		forms++
	}
	if pattern != "" {
		forms++
	}
	if forms > 1 {
		return nil, fmt.Errorf("version tags must use exactly one of a preset, format(s), or pattern")
	}
	if macros == nil {
		macros = DefaultTagFormatMacroRegistry()
	}
	contract := &GitVersionTagContract{Preset: preset, Formats: append([]string(nil), formats...), Pattern: pattern}
	switch {
	case pattern != "":
		matcher, err := compileRawTagPattern(pattern)
		if err != nil {
			return nil, err
		}
		contract.matchers = []compiledTagMatcher{matcher}
	case len(formats) > 0:
		for _, format := range formats {
			matcher, err := macros.Compile(format)
			if err != nil {
				return nil, err
			}
			contract.matchers = append(contract.matchers, matcher)
		}
	default:
		presetFormats, err := presetTagFormats(preset)
		if err != nil {
			return nil, err
		}
		contract.Formats = presetFormats
		for _, format := range presetFormats {
			matcher, err := macros.Compile(format)
			if err != nil {
				return nil, err
			}
			contract.matchers = append(contract.matchers, matcher)
		}
	}
	return contract, nil
}

func presetTagFormats(preset string) ([]string, error) {
	switch preset {
	case "", "semver_optional_v":
		return []string{"{version}", "v{version}"}, nil
	case "semver":
		return []string{"v{version}"}, nil
	default:
		return nil, fmt.Errorf("unknown version tag preset %q", preset)
	}
}

func (r *TagFormatMacroRegistry) Compile(format string) (compiledTagMatcher, error) {
	var pattern strings.Builder
	pattern.WriteString("^")
	macroCount := 0
	for i := 0; i < len(format); {
		switch {
		case strings.HasPrefix(format[i:], "{{"):
			pattern.WriteString(regexp.QuoteMeta("{"))
			i += 2
		case strings.HasPrefix(format[i:], "}}"):
			pattern.WriteString(regexp.QuoteMeta("}"))
			i += 2
		case format[i] == '{':
			end := strings.IndexByte(format[i+1:], '}')
			if end < 0 {
				return compiledTagMatcher{}, fmt.Errorf("unclosed tag format macro in %q", format)
			}
			end += i + 1
			name := format[i+1 : end]
			macro, ok := r.macros[name]
			if !ok {
				return compiledTagMatcher{}, fmt.Errorf("unknown tag format macro {%s}", name)
			}
			if name != "version" {
				return compiledTagMatcher{}, fmt.Errorf("tag format %q does not define {version}", format)
			}
			macroCount++
			pattern.WriteString("(?P<version>")
			pattern.WriteString(macro.Pattern)
			pattern.WriteString(")")
			i = end + 1
		case format[i] == '}':
			return compiledTagMatcher{}, fmt.Errorf("unescaped closing brace in tag format %q; use }} for a literal brace", format)
		default:
			start := i
			for i < len(format) && format[i] != '{' && format[i] != '}' {
				i++
			}
			pattern.WriteString(regexp.QuoteMeta(format[start:i]))
		}
	}
	if macroCount != 1 {
		return compiledTagMatcher{}, fmt.Errorf("tag format %q must contain exactly one {version} macro", format)
	}
	pattern.WriteString("$")
	return compileRawTagPattern(pattern.String())
}

func compileRawTagPattern(pattern string) (compiledTagMatcher, error) {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return compiledTagMatcher{}, fmt.Errorf("invalid version tag pattern: %w", err)
	}
	index := -1
	count := 0
	for i, name := range compiled.SubexpNames() {
		if name == "version" {
			index = i
			count++
		}
	}
	if count != 1 {
		message := "version tag pattern must contain exactly one named capture called version"
		if format, ok := simpleTagFormatSuggestion(pattern); ok {
			message += fmt.Sprintf("; for this convention, prefer version tags: %q", format)
		}
		return compiledTagMatcher{}, fmt.Errorf("%s", message)
	}
	return compiledTagMatcher{pattern: compiled, versionCapture: index}, nil
}

func simpleTagFormatSuggestion(pattern string) (string, bool) {
	if !strings.HasPrefix(pattern, "^") || !strings.HasSuffix(pattern, "$") {
		return "", false
	}
	body := strings.TrimSuffix(strings.TrimPrefix(pattern, "^"), "$")
	versionPattern := `[0-9]+\.[0-9]+\.[0-9]+`
	for _, capture := range []string{"(" + versionPattern + ")", "(?P<version>" + versionPattern + ")"} {
		position := strings.Index(body, capture)
		if position < 0 {
			continue
		}
		prefix, ok := literalRegexText(body[:position])
		if !ok {
			return "", false
		}
		suffix, ok := literalRegexText(body[position+len(capture):])
		if !ok {
			return "", false
		}
		return strings.ReplaceAll(strings.ReplaceAll(prefix, "{", "{{"), "}", "}}") + "{version}" + strings.ReplaceAll(strings.ReplaceAll(suffix, "{", "{{"), "}", "}}"), true
	}
	return "", false
}

func literalRegexText(value string) (string, bool) {
	var result strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' {
			i++
			if i >= len(value) {
				return "", false
			}
			result.WriteByte(value[i])
			continue
		}
		if strings.ContainsRune(".*+?()[]|^$", rune(value[i])) {
			return "", false
		}
		result.WriteByte(value[i])
	}
	return result.String(), true
}

func (c *GitVersionTagContract) Extract(tag string) (Version, bool, error) {
	var selected *Version
	for _, matcher := range c.matchers {
		match := matcher.pattern.FindStringSubmatch(tag)
		if match == nil {
			continue
		}
		version, err := ParseVersion(match[matcher.versionCapture])
		if err != nil {
			continue
		}
		if selected != nil && selected.Raw != version.Raw {
			return Version{}, false, fmt.Errorf("version tag formats extract conflicting versions %q and %q from tag %q", selected.Raw, version.Raw, tag)
		}
		selected = &version
	}
	if selected == nil {
		return Version{}, false, nil
	}
	return *selected, true, nil
}
