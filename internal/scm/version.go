package scm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var stableVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type Version struct {
	Major uint64
	Minor uint64
	Patch uint64
	Raw   string
}

func ParseVersion(value string) (Version, error) {
	match := stableVersionPattern.FindStringSubmatch(value)
	if match == nil {
		return Version{}, fmt.Errorf("%q is not a stable MAJOR.MINOR.PATCH version", value)
	}
	parts := [3]uint64{}
	for i := range parts {
		parsed, err := strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version component in %q: %w", value, err)
		}
		parts[i] = parsed
	}
	return Version{Major: parts[0], Minor: parts[1], Patch: parts[2], Raw: value}, nil
}

func (v Version) Compare(other Version) int {
	left := [...]uint64{v.Major, v.Minor, v.Patch}
	right := [...]uint64{other.Major, other.Minor, other.Patch}
	for i := range left {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

type GitVersionConstraint struct {
	Clauses []VersionConstraintClause
}

type VersionConstraintClause struct {
	Operator string
	Version  Version
}

var constraintClausePattern = regexp.MustCompile(`(>=|<=|>|<|=)\s*([0-9]+\.[0-9]+\.[0-9]+)`)

func ParseVersionConstraint(value string) (GitVersionConstraint, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return GitVersionConstraint{}, nil
	}
	matches := constraintClausePattern.FindAllStringSubmatchIndex(value, -1)
	constraint := GitVersionConstraint{}
	position := 0
	for _, match := range matches {
		if strings.TrimSpace(value[position:match[0]]) != "" {
			return GitVersionConstraint{}, fmt.Errorf("invalid version constraint near %q", value[position:])
		}
		version, err := ParseVersion(value[match[4]:match[5]])
		if err != nil {
			return GitVersionConstraint{}, err
		}
		constraint.Clauses = append(constraint.Clauses, VersionConstraintClause{
			Operator: value[match[2]:match[3]], Version: version,
		})
		position = match[1]
	}
	if len(matches) == 0 || strings.TrimSpace(value[position:]) != "" {
		return GitVersionConstraint{}, fmt.Errorf("invalid version constraint %q", value)
	}
	return constraint, nil
}

func SeriesConstraint(series string) (GitVersionConstraint, error) {
	parts := strings.Split(strings.TrimSpace(series), ".")
	if len(parts) < 1 || len(parts) > 2 || parts[0] == "" {
		return GitVersionConstraint{}, fmt.Errorf("series must be a major or major.minor value, got %q", series)
	}
	major, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return GitVersionConstraint{}, fmt.Errorf("invalid series %q", series)
	}
	minor := uint64(0)
	if len(parts) == 2 {
		minor, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return GitVersionConstraint{}, fmt.Errorf("invalid series %q", series)
		}
	}
	lower := Version{Major: major, Minor: minor, Raw: fmt.Sprintf("%d.%d.0", major, minor)}
	upper := Version{Major: major + 1, Raw: fmt.Sprintf("%d.0.0", major+1)}
	if len(parts) == 2 {
		upper = Version{Major: major, Minor: minor + 1, Raw: fmt.Sprintf("%d.%d.0", major, minor+1)}
	}
	return GitVersionConstraint{Clauses: []VersionConstraintClause{
		{Operator: ">=", Version: lower}, {Operator: "<", Version: upper},
	}}, nil
}

func (c GitVersionConstraint) Matches(version Version) bool {
	for _, clause := range c.Clauses {
		comparison := version.Compare(clause.Version)
		switch clause.Operator {
		case ">":
			if comparison <= 0 {
				return false
			}
		case ">=":
			if comparison < 0 {
				return false
			}
		case "<":
			if comparison >= 0 {
				return false
			}
		case "<=":
			if comparison > 0 {
				return false
			}
		case "=":
			if comparison != 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
