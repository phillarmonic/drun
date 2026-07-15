package scm

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type cliCall struct {
	executable string
	arguments  []string
}
type queuedCLIRunner struct {
	calls   []cliCall
	outputs [][]byte
}

func (r *queuedCLIRunner) Run(_ context.Context, executable string, arguments, _ []string) ([]byte, error) {
	r.calls = append(r.calls, cliCall{executable: executable, arguments: append([]string(nil), arguments...)})
	if len(r.outputs) == 0 {
		return nil, fmt.Errorf("unexpected CLI call")
	}
	output := r.outputs[0]
	r.outputs = r.outputs[1:]
	return output, nil
}

func TestGitHubCLIUsesPaginationCustomHostAndObjectDates(t *testing.T) {
	runner := &queuedCLIRunner{outputs: [][]byte{
		[]byte(`[[{"ref":"refs/tags/v1.2.3","object":{"type":"commit","sha":"abc"}},{"ref":"refs/tags/v1.3.0","object":{"type":"tag","sha":"def"}}]]`),
		[]byte(`{"committer":{"date":"2026-01-02T03:04:05Z"}}`),
		[]byte(`{"tagger":{"date":"2026-02-03T04:05:06Z"}}`),
	}}
	adapter := &ProviderCLIAdapter{provider: "github", runner: runner}
	session, err := adapter.Open(context.Background(), &GitSource{Alias: "app"}, &GitAccessProfile{Method: "cli", Repository: "team/app", Host: "git.example.test"}, false)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := session.Tags(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[0].Name != "v1.2.3" || refs[0].Date.IsZero() || refs[1].Date.IsZero() {
		t.Fatalf("refs = %#v", refs)
	}
	first := strings.Join(runner.calls[0].arguments, " ")
	if runner.calls[0].executable != "gh" || !strings.Contains(first, "--paginate --slurp") || !strings.Contains(first, "--hostname git.example.test") {
		t.Fatalf("GitHub call = %#v", runner.calls[0])
	}
}

func TestGitLabCLIParsesPaginatedResponsesAndCustomHost(t *testing.T) {
	runner := &queuedCLIRunner{outputs: [][]byte{[]byte(
		`[{"name":"v1.2.3","target":"abc","commit":{"committed_date":"2026-01-02T03:04:05Z"}}]` +
			`[{"name":"v1.3.0","target":"def","commit":{"committed_date":"2026-02-03T04:05:06Z"}}]`,
	)}}
	adapter := &ProviderCLIAdapter{provider: "gitlab", runner: runner}
	session, err := adapter.Open(context.Background(), &GitSource{Alias: "app"}, &GitAccessProfile{Method: "cli", Repository: "team/app", Host: "gitlab.example.test"}, false)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := session.Tags(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 || refs[1].Name != "v1.3.0" || refs[1].Date.IsZero() {
		t.Fatalf("refs = %#v", refs)
	}
	call := strings.Join(runner.calls[0].arguments, " ")
	if runner.calls[0].executable != "glab" || !strings.Contains(call, "--paginate") || !strings.Contains(call, "--hostname gitlab.example.test") || !strings.Contains(call, "team%2Fapp") {
		t.Fatalf("GitLab call = %#v", runner.calls[0])
	}
}
