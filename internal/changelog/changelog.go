// Package changelog implements Keep a Changelog (https://keepachangelog.com)
// promotion: moving the Unreleased section into a dated release section.
package changelog

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	unreleasedHeadingRe = regexp.MustCompile(`^## \[Unreleased\]\s*$`)
	releaseHeadingRe    = regexp.MustCompile(`^## \[([^]]+)\]`)
	subsectionHeadingRe = regexp.MustCompile(`^### \S`)
	linkDefinitionRe    = regexp.MustCompile(`^\[[^]]+\]:\s*\S+`)
	unreleasedLinkRe    = regexp.MustCompile(`^\[Unreleased\]:\s*(\S+)/compare/(v?)([^/.]+(?:\.[^/.]+)*)\.\.\.HEAD\s*$`)
	versionRe           = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

// NormalizeVersion strips an optional leading "v" and validates that the
// result is a plain semantic version (X.Y.Z).
func NormalizeVersion(raw string) (string, error) {
	version := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	if !versionRe.MatchString(version) {
		return "", fmt.Errorf("version %q is not a semantic version (expected X.Y.Z)", raw)
	}
	return version, nil
}

// ParseDate validates a YYYY-MM-DD release date override.
func ParseDate(raw string) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(raw), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("date %q is not a valid calendar date (expected YYYY-MM-DD)", raw)
	}
	return date, nil
}

// Promote moves the Unreleased entries of a Keep a Changelog document into a
// dated release section for version, leaving an emptied Unreleased section
// behind. When the document carries an "[Unreleased]: .../compare/<prev>...HEAD"
// link definition, the comparison links are updated as well.
//
// Promotion is idempotent so release preparation can be re-run before the
// release is actually published: when the release section already exists, new
// Unreleased entries are merged into it (its date and comparison links stay
// untouched), and an empty Unreleased section is a no-op.
func Promote(content, rawVersion string, date time.Time) (string, error) {
	version, err := NormalizeVersion(rawVersion)
	if err != nil {
		return "", err
	}

	lines := strings.Split(content, "\n")

	unreleasedAt := -1
	for i, line := range lines {
		if unreleasedHeadingRe.MatchString(line) {
			unreleasedAt = i
			break
		}
	}
	if unreleasedAt == -1 {
		return "", fmt.Errorf("no '## [Unreleased]' section found")
	}

	// The Unreleased body runs until the next section heading or the link
	// definition block, whichever comes first.
	bodyEnd := len(lines)
	for i := unreleasedAt + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") || linkDefinitionRe.MatchString(lines[i]) {
			bodyEnd = i
			break
		}
	}
	body := trimBlankLines(lines[unreleasedAt+1 : bodyEnd])
	rest := trimLeadingBlankLines(lines[bodyEnd:])

	// The emptied Unreleased section keeps the subsection skeleton it had.
	skeleton := []string{"## [Unreleased]", ""}
	for _, line := range body {
		if subsectionHeadingRe.MatchString(line) {
			skeleton = append(skeleton, line, "")
		}
	}

	// A release section for this version may already exist from a previous
	// prepare run; merge into it instead of failing.
	releaseAt := findReleaseSection(rest, version)
	if releaseAt != -1 {
		entries := parseSectionBody(body)
		if entries.isEmpty() {
			return content, nil
		}
		rest = mergeReleaseSection(rest, releaseAt, entries)
		out := make([]string, 0, len(lines)+len(skeleton))
		out = append(out, lines[:unreleasedAt]...)
		out = append(out, skeleton...)
		out = append(out, rest...)
		return strings.Join(out, "\n"), nil
	}

	release := []string{fmt.Sprintf("## [%s] - %s", version, date.Format("2006-01-02")), ""}
	release = append(release, body...)
	release = append(release, "")

	rest = updateCompareLinks(rest, version)

	out := make([]string, 0, len(lines)+len(skeleton)+len(release))
	out = append(out, lines[:unreleasedAt]...)
	out = append(out, skeleton...)
	out = append(out, release...)
	out = append(out, rest...)
	return strings.Join(out, "\n"), nil
}

// findReleaseSection locates a "## [<version>] ..." heading in lines,
// tolerating a leading "v" in the heading. Returns -1 when absent.
func findReleaseSection(lines []string, version string) int {
	for i, line := range lines {
		if match := releaseHeadingRe.FindStringSubmatch(line); match != nil {
			if strings.TrimPrefix(match[1], "v") == version {
				return i
			}
		}
	}
	return -1
}

// subsection is one "### ..." block of a release section.
type subsection struct {
	heading string
	lines   []string
}

// sectionBody is the parsed content of a release section: free-form preamble
// lines followed by subsections.
type sectionBody struct {
	preamble []string
	subs     []subsection
}

func (s sectionBody) isEmpty() bool {
	if len(s.preamble) > 0 {
		return false
	}
	for _, sub := range s.subs {
		if len(sub.lines) > 0 {
			return false
		}
	}
	return true
}

// parseSectionBody splits trimmed section body lines into preamble and
// subsections. Blank runs separate blocks; empty subsections are preserved so
// the skeleton shape survives a merge.
func parseSectionBody(body []string) sectionBody {
	var result sectionBody
	current := -1
	for _, line := range body {
		if subsectionHeadingRe.MatchString(line) {
			result.subs = append(result.subs, subsection{heading: line})
			current = len(result.subs) - 1
			continue
		}
		if current == -1 {
			result.preamble = append(result.preamble, line)
		} else {
			result.subs[current].lines = append(result.subs[current].lines, line)
		}
	}
	result.preamble = trimBlankLines(result.preamble)
	for i := range result.subs {
		result.subs[i].lines = trimBlankLines(result.subs[i].lines)
	}
	return result
}

// renderSectionBody renders a merged section body in canonical form: preamble,
// then each non-empty subsection, blocks separated by single blank lines.
func renderSectionBody(body sectionBody) []string {
	var out []string
	if len(body.preamble) > 0 {
		out = append(out, body.preamble...)
	}
	for _, sub := range body.subs {
		if len(sub.lines) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, sub.heading, "")
		out = append(out, sub.lines...)
	}
	return out
}

// mergeReleaseSection folds new Unreleased entries into the existing release
// section starting at releaseAt within lines. Matching subsections are
// appended to; new subsections and preamble lines are added in order. The
// section heading (including its date) is preserved verbatim.
func mergeReleaseSection(lines []string, releaseAt int, entries sectionBody) []string {
	bodyEnd := len(lines)
	for i := releaseAt + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") || linkDefinitionRe.MatchString(lines[i]) {
			bodyEnd = i
			break
		}
	}

	target := parseSectionBody(trimBlankLines(lines[releaseAt+1 : bodyEnd]))
	target.preamble = append(target.preamble, entries.preamble...)
	for _, incoming := range entries.subs {
		if len(incoming.lines) == 0 {
			continue
		}
		merged := false
		for i := range target.subs {
			if target.subs[i].heading == incoming.heading {
				target.subs[i].lines = append(target.subs[i].lines, incoming.lines...)
				merged = true
				break
			}
		}
		if !merged {
			target.subs = append(target.subs, incoming)
		}
	}

	merged := renderSectionBody(target)

	out := make([]string, 0, len(lines)+len(entries.preamble)+8)
	out = append(out, lines[:releaseAt+1]...)
	if len(merged) > 0 {
		out = append(out, "")
		out = append(out, merged...)
	}
	out = append(out, "")
	out = append(out, lines[bodyEnd:]...)
	return out
}

// updateCompareLinks rewrites the Unreleased comparison link to start at the
// new release and inserts the new release's own comparison link after it.
// Documents without an Unreleased link are returned unchanged.
func updateCompareLinks(lines []string, version string) []string {
	for i, line := range lines {
		match := unreleasedLinkRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		base, prefix, previous := match[1], match[2], match[3]
		updated := make([]string, 0, len(lines)+1)
		updated = append(updated, lines[:i]...)
		updated = append(updated, fmt.Sprintf("[Unreleased]: %s/compare/%s%s...HEAD", base, prefix, version))
		updated = append(updated, fmt.Sprintf("[%s]: %s/compare/%s%s...%s%s", version, base, prefix, previous, prefix, version))
		updated = append(updated, lines[i+1:]...)
		return updated
	}
	return lines
}

func trimBlankLines(lines []string) []string {
	return trimTrailingBlankLines(trimLeadingBlankLines(lines))
}

func trimLeadingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	return lines
}

func trimTrailingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
