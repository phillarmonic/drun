package scm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type commandRunner interface {
	Run(context.Context, []string, []string) ([]byte, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, args, environment []string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Env = append(os.Environ(), environment...)
	return command.Output()
}

// RawGitAdapter provides Git-native ref access for filesystem and remote profiles.
type RawGitAdapter struct {
	runner commandRunner
}

func NewRawGitAdapter() *RawGitAdapter { return &RawGitAdapter{runner: execCommandRunner{}} }

func (a *RawGitAdapter) Capabilities() GitProviderCapabilities {
	return GitProviderCapabilities{RemoteRefs: true, ObjectMetadata: true}
}

func (a *RawGitAdapter) Open(ctx context.Context, source *GitSource, profile *GitAccessProfile, allowFetch bool) (GitRepositorySession, error) {
	if profile.Method == "filesystem" {
		return &rawGitSession{runner: a.runner, path: profile.Path, local: true}, nil
	}
	locator := profile.URL
	if locator == "" {
		return nil, fmt.Errorf("Git source %q %s access has no remote locator", source.Alias, profile.Method)
	}
	environment := []string{"GIT_TERMINAL_PROMPT=0"}
	if profile.Method == "ssh" && profile.Key != "" {
		environment = append(environment, "GIT_SSH_COMMAND=ssh -i "+shellQuote(profile.Key)+" -o IdentitiesOnly=yes")
	}
	if source.Metadata == "fetch" && allowFetch {
		return a.openFetchedSession(ctx, source, locator, environment)
	}
	return &rawGitSession{runner: a.runner, locator: locator, environment: environment}, nil
}

func (a *RawGitAdapter) openFetchedSession(ctx context.Context, source *GitSource, locator string, environment []string) (GitRepositorySession, error) {
	directory, err := os.MkdirTemp("", "drun-git-metadata-*")
	if err != nil {
		return nil, fmt.Errorf("creating isolated metadata storage: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	if _, err := a.runner.Run(ctx, []string{"init", "--bare", directory}, environment); err != nil {
		cleanup()
		return nil, fmt.Errorf("initializing isolated metadata storage: %w", err)
	}
	refspec := "+refs/tags/*:refs/tags/*"
	if _, err := a.runner.Run(ctx, []string{"-C", directory, "fetch", "--force", "--tags", "--no-write-fetch-head", locator, refspec}, environment); err != nil {
		cleanup()
		return nil, fmt.Errorf("fetching metadata for Git source %q failed: %w", source.Alias, err)
	}
	return &fetchedGitSession{
		rawGitSession: rawGitSession{runner: a.runner, path: directory, local: true, environment: environment},
		directory:     directory,
	}, nil
}

func DefaultGitAdapters() map[string]GitProviderAdapter {
	raw := NewRawGitAdapter()
	adapters := make(map[string]GitProviderAdapter)
	for _, provider := range []string{"github", "gitlab", "generic"} {
		for _, method := range []string{"https", "ssh"} {
			adapters[AdapterKey(provider, method)] = raw
		}
	}
	adapters[AdapterKey("generic", "remote")] = raw
	adapters[AdapterKey("generic", "filesystem")] = raw
	adapters[AdapterKey("github", "cli")] = NewProviderCLIAdapter("github")
	adapters[AdapterKey("gitlab", "cli")] = NewProviderCLIAdapter("gitlab")
	return adapters
}

type rawGitSession struct {
	runner      commandRunner
	locator     string
	path        string
	local       bool
	environment []string
}

func (s *rawGitSession) Tags(ctx context.Context, withMetadata bool) ([]GitRef, error) {
	if s.local {
		return s.localTags(ctx)
	}
	if withMetadata {
		return nil, fmt.Errorf("object metadata is unavailable from remote refs; declare metadata: fetch and use allow fetch")
	}
	output, err := s.runner.Run(ctx, []string{"ls-remote", "--tags", "--refs", s.locator}, s.environment)
	if err != nil {
		return nil, fmt.Errorf("listing remote Git tags failed: %w", err)
	}
	return parseRemoteRefs(output), nil
}

func (s *rawGitSession) localTags(ctx context.Context) ([]GitRef, error) {
	format := "%(refname:strip=2)%00%(objectname)%00%(creatordate:unix)"
	output, err := s.runner.Run(ctx, []string{"-C", s.path, "for-each-ref", "--format=" + format, "refs/tags"}, s.environment)
	if err != nil {
		return nil, fmt.Errorf("inspecting local Git tags failed: %w", err)
	}
	refs := make([]GitRef, 0)
	for _, line := range bytes.Split(bytes.TrimSpace(output), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		fields := bytes.Split(line, []byte{0})
		if len(fields) != 3 {
			continue
		}
		seconds, _ := strconv.ParseInt(string(fields[2]), 10, 64)
		refs = append(refs, GitRef{Name: string(fields[0]), TargetHash: string(fields[1]), Date: time.Unix(seconds, 0)})
	}
	return refs, nil
}

func (s *rawGitSession) Close() error { return nil }

type fetchedGitSession struct {
	rawGitSession
	directory string
}

func (s *fetchedGitSession) Close() error {
	if s.directory == "" {
		return nil
	}
	err := os.RemoveAll(s.directory)
	s.directory = ""
	return err
}

func parseRemoteRefs(output []byte) []GitRef {
	refs := make([]GitRef, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !strings.HasPrefix(fields[1], "refs/tags/") {
			continue
		}
		refs = append(refs, GitRef{Name: strings.TrimPrefix(fields[1], "refs/tags/"), TargetHash: fields[0]})
	}
	return refs
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
