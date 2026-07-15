package scm

import (
	"context"
	"fmt"
	"sort"
)

type GitTagQuery struct {
	Result         string
	TagPreset      string
	TagFormat      string
	TagPattern     string
	Series         string
	VersionMatcher string
	OrderBy        string
}

type GitTagQueryResult struct {
	Tag     string
	Version Version
	Date    int64
}

func (q GitTagQuery) Execute(ctx context.Context, session GitRepositorySession, source *GitSource) (GitTagQueryResult, error) {
	contract, err := q.contract(source)
	if err != nil {
		return GitTagQueryResult{}, err
	}
	constraint, err := q.constraint()
	if err != nil {
		return GitTagQueryResult{}, err
	}
	orderBy := q.OrderBy
	if orderBy == "" {
		orderBy = "version"
	}
	refs, err := session.Tags(ctx, orderBy == "date")
	if err != nil {
		return GitTagQueryResult{}, err
	}
	candidates := make([]GitTagQueryResult, 0, len(refs))
	for _, ref := range refs {
		version, matched, extractErr := contract.Extract(ref.Name)
		if extractErr != nil {
			return GitTagQueryResult{}, extractErr
		}
		if !matched || !constraint.Matches(version) {
			continue
		}
		if orderBy == "date" && ref.Date.IsZero() {
			return GitTagQueryResult{}, fmt.Errorf("tag %q has no date metadata", ref.Name)
		}
		candidates = append(candidates, GitTagQueryResult{Tag: ref.Name, Version: version, Date: ref.Date.UnixNano()})
	}
	if len(candidates) == 0 {
		return GitTagQueryResult{}, fmt.Errorf("no stable version tags matched the query")
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if orderBy == "date" && left.Date != right.Date {
			return left.Date < right.Date
		}
		if comparison := left.Version.Compare(right.Version); comparison != 0 {
			return comparison < 0
		}
		return left.Tag < right.Tag
	})
	return candidates[len(candidates)-1], nil
}

func (q GitTagQuery) contract(source *GitSource) (*GitVersionTagContract, error) {
	if q.TagPreset != "" || q.TagFormat != "" || q.TagPattern != "" {
		formats := []string(nil)
		if q.TagFormat != "" {
			formats = []string{q.TagFormat}
		}
		return NewGitVersionTagContract(q.TagPreset, formats, q.TagPattern, nil)
	}
	if source != nil && source.VersionTags != nil {
		return source.VersionTags, nil
	}
	return NewGitVersionTagContract("semver_optional_v", nil, "", nil)
}

func (q GitTagQuery) constraint() (GitVersionConstraint, error) {
	if q.Series != "" {
		return SeriesConstraint(q.Series)
	}
	return ParseVersionConstraint(q.VersionMatcher)
}

func (r GitTagQueryResult) Value(result string) string {
	if result == "tag" {
		return r.Tag
	}
	return r.Version.Raw
}
