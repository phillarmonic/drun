package scm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type cliRunner interface {
	Run(context.Context, string, []string, []string) ([]byte, error)
}

type execCLIRunner struct{}

func (execCLIRunner) Run(ctx context.Context, executable string, arguments, environment []string) ([]byte, error) {
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Env = append(os.Environ(), environment...)
	return command.Output()
}

type ProviderCLIAdapter struct {
	provider string
	runner   cliRunner
}

func NewProviderCLIAdapter(provider string) *ProviderCLIAdapter {
	return &ProviderCLIAdapter{provider: provider, runner: execCLIRunner{}}
}

func (a *ProviderCLIAdapter) Capabilities() GitProviderCapabilities {
	return GitProviderCapabilities{RemoteRefs: true, ObjectMetadata: true, ProviderAPI: true}
}

func (a *ProviderCLIAdapter) Open(_ context.Context, source *GitSource, profile *GitAccessProfile, _ bool) (GitRepositorySession, error) {
	if profile.Repository == "" {
		return nil, fmt.Errorf("Git source %q CLI access requires repository", source.Alias)
	}
	host := profile.Host
	if host == "" {
		if a.provider == "github" {
			host = "github.com"
		} else {
			host = "gitlab.com"
		}
	}
	return &providerCLISession{provider: a.provider, runner: a.runner, repository: profile.Repository, host: host}, nil
}

type providerCLISession struct {
	provider   string
	runner     cliRunner
	repository string
	host       string
}

func (s *providerCLISession) Tags(ctx context.Context, withMetadata bool) ([]GitRef, error) {
	if s.provider == "github" {
		return s.githubTags(ctx, withMetadata)
	}
	return s.gitlabTags(ctx, withMetadata)
}

func (s *providerCLISession) githubTags(ctx context.Context, withMetadata bool) ([]GitRef, error) {
	endpoint := "repos/" + s.repository + "/git/matching-refs/tags/"
	output, err := s.runner.Run(ctx, "gh", []string{"api", "--paginate", "--slurp", "--hostname", s.host, endpoint}, nil)
	if err != nil {
		return nil, fmt.Errorf("listing GitHub tags with provider CLI failed: %w", err)
	}
	var pages [][]struct {
		Ref    string `json:"ref"`
		Object struct {
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(output, &pages); err != nil {
		return nil, fmt.Errorf("decoding GitHub tag response: %w", err)
	}
	refs := make([]GitRef, 0)
	for _, page := range pages {
		for _, item := range page {
			ref := GitRef{Name: strings.TrimPrefix(item.Ref, "refs/tags/"), TargetHash: item.Object.SHA}
			if withMetadata {
				ref.Date, err = s.githubObjectDate(ctx, item.Object.Type, item.Object.SHA)
				if err != nil {
					return nil, err
				}
			}
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func (s *providerCLISession) githubObjectDate(ctx context.Context, objectType, sha string) (time.Time, error) {
	endpoint := "repos/" + s.repository + "/git/commits/" + sha
	field := "committer"
	if objectType == "tag" {
		endpoint = "repos/" + s.repository + "/git/tags/" + sha
		field = "tagger"
	}
	output, err := s.runner.Run(ctx, "gh", []string{"api", "--hostname", s.host, endpoint}, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("reading GitHub %s metadata failed: %w", objectType, err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(output, &object); err != nil {
		return time.Time{}, err
	}
	var identity struct {
		Date time.Time `json:"date"`
	}
	if err := json.Unmarshal(object[field], &identity); err != nil {
		return time.Time{}, err
	}
	return identity.Date, nil
}

func (s *providerCLISession) gitlabTags(ctx context.Context, withMetadata bool) ([]GitRef, error) {
	endpoint := "projects/" + url.PathEscape(s.repository) + "/repository/tags?per_page=100"
	output, err := s.runner.Run(ctx, "glab", []string{"api", "--paginate", "--hostname", s.host, endpoint}, nil)
	if err != nil {
		return nil, fmt.Errorf("listing GitLab tags with provider CLI failed: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	refs := make([]GitRef, 0)
	for {
		var page []struct {
			Name      string    `json:"name"`
			Target    string    `json:"target"`
			CreatedAt time.Time `json:"created_at"`
			Commit    struct {
				CommittedDate time.Time `json:"committed_date"`
			} `json:"commit"`
		}
		if err := decoder.Decode(&page); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decoding GitLab tag response: %w", err)
		}
		for _, item := range page {
			ref := GitRef{Name: item.Name, TargetHash: item.Target}
			if withMetadata {
				ref.Date = item.CreatedAt
				if ref.Date.IsZero() {
					ref.Date = item.Commit.CommittedDate
				}
			}
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func (s *providerCLISession) Close() error { return nil }
