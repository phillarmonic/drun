package scm

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type failFetchRunner struct {
	execCommandRunner
	directory string
}

func (r *failFetchRunner) Run(ctx context.Context, arguments, environment []string) ([]byte, error) {
	if len(arguments) >= 3 && arguments[0] == "init" {
		r.directory = arguments[2]
	}
	if len(arguments) >= 3 && arguments[2] == "fetch" {
		return nil, errors.New("fetch failed")
	}
	return r.execCommandRunner.Run(ctx, arguments, environment)
}

func TestRawGitFilesystemAndRemoteSessions(t *testing.T) {
	worktree := createTagTestRepository(t)
	bare := filepath.Join(t.TempDir(), "repository.git")
	runTestGit(t, "", "clone", "--bare", worktree, bare)

	adapter := NewRawGitAdapter()
	for _, path := range []string{worktree, bare} {
		session, err := adapter.Open(context.Background(), &GitSource{Alias: "local"}, &GitAccessProfile{Method: "filesystem", Path: path}, false)
		if err != nil {
			t.Fatal(err)
		}
		refs, err := session.Tags(context.Background(), true)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 2 || refs[0].Date.IsZero() || refs[1].Date.IsZero() {
			t.Fatalf("filesystem tags for %s = %#v", path, refs)
		}
		dates := map[string]int64{}
		for _, ref := range refs {
			dates[ref.Name] = ref.Date.Unix()
		}
		if dates["v1.2.3"] != 1_767_225_600 || dates["v1.3.0"] != 1_769_904_000 {
			t.Fatalf("filesystem dates for %s = %#v", path, dates)
		}
	}

	remote, err := adapter.Open(context.Background(), &GitSource{Alias: "mirror"}, &GitAccessProfile{Method: "remote", URL: bare}, false)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := remote.Tags(context.Background(), false)
	if err != nil || len(refs) != 2 {
		t.Fatalf("remote refs = %#v, err = %v", refs, err)
	}
	if _, err := remote.Tags(context.Background(), true); err == nil {
		t.Fatal("remote refs unexpectedly provided object metadata")
	}
}

type recordingRunner struct {
	environment []string
}

func (r *recordingRunner) Run(_ context.Context, _ []string, environment []string) ([]byte, error) {
	r.environment = append([]string(nil), environment...)
	return []byte("abc\trefs/tags/v1.2.3\n"), nil
}

func TestRawGitSSHKeyIsPassedByPath(t *testing.T) {
	runner := &recordingRunner{}
	adapter := &RawGitAdapter{runner: runner}
	session, err := adapter.Open(context.Background(), &GitSource{Alias: "app"}, &GitAccessProfile{
		Method: "ssh", URL: "git@example.test:app.git", Key: "/keys/company key",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Tags(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(runner.environment, "\n")
	if !strings.Contains(joined, "GIT_SSH_COMMAND=ssh -i '/keys/company key' -o IdentitiesOnly=yes") {
		t.Fatalf("environment = %q", joined)
	}
}

func TestMetadataFetchRequiresBothGatesAndCleansUp(t *testing.T) {
	worktree := createTagTestRepository(t)
	bare := filepath.Join(t.TempDir(), "repository.git")
	runTestGit(t, "", "clone", "--bare", worktree, bare)
	adapter := NewRawGitAdapter()
	profile := &GitAccessProfile{Method: "remote", URL: bare}

	for _, gates := range []struct {
		metadata   string
		allowFetch bool
	}{
		{metadata: "refs", allowFetch: true},
		{metadata: "fetch", allowFetch: false},
	} {
		session, err := adapter.Open(context.Background(), &GitSource{Alias: "mirror", Metadata: gates.metadata}, profile, gates.allowFetch)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := session.Tags(context.Background(), true); err == nil {
			t.Fatalf("metadata=%q allowFetch=%v unexpectedly retrieved objects", gates.metadata, gates.allowFetch)
		}
	}

	session, err := adapter.Open(context.Background(), &GitSource{Alias: "mirror", Metadata: "fetch"}, profile, true)
	if err != nil {
		t.Fatal(err)
	}
	fetched, ok := session.(*fetchedGitSession)
	if !ok {
		t.Fatalf("session = %T", session)
	}
	directory := fetched.directory
	refs, err := session.Tags(context.Background(), true)
	if err != nil || len(refs) != 2 || refs[0].Date.IsZero() {
		t.Fatalf("fetched refs = %#v, err = %v", refs, err)
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatalf("temporary repository was removed before close: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("temporary repository remains after close: %v", err)
	}
}

func TestMetadataFetchCleansUpAfterFailure(t *testing.T) {
	runner := &failFetchRunner{}
	adapter := &RawGitAdapter{runner: runner}
	_, err := adapter.Open(context.Background(), &GitSource{Alias: "mirror", Metadata: "fetch"}, &GitAccessProfile{Method: "remote", URL: "/missing"}, true)
	if err == nil {
		t.Fatal("expected fetch failure")
	}
	if runner.directory == "" {
		t.Fatal("temporary directory was not initialized")
	}
	if _, statErr := os.Stat(runner.directory); !os.IsNotExist(statErr) {
		t.Fatalf("temporary repository remains after failure: %v", statErr)
	}
}

func createTagTestRepository(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	runTestGit(t, directory, "init")
	runTestGit(t, directory, "config", "user.name", "Drun Test")
	runTestGit(t, directory, "config", "user.email", "drun@example.test")
	if err := os.WriteFile(filepath.Join(directory, "README"), []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, directory, "add", "README")
	runTestGitEnv(t, directory, []string{
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
	}, "commit", "-m", "initial")
	runTestGit(t, directory, "tag", "v1.2.3")
	runTestGitEnv(t, directory, []string{"GIT_COMMITTER_DATE=2026-02-01T00:00:00Z"}, "tag", "-a", "v1.3.0", "-m", "release")
	return directory
}

func runTestGit(t *testing.T, directory string, arguments ...string) {
	runTestGitEnv(t, directory, nil, arguments...)
}

func runTestGitEnv(t *testing.T, directory string, environment []string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", arguments...)
	command.Env = append(os.Environ(), environment...)
	if directory != "" {
		command.Dir = directory
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", arguments, err, output)
	}
}
