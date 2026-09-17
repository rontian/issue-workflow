package ghcli

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
)

type fakeRunner struct { last ports.Command; out []byte; err error }
func (f *fakeRunner) LookPath(string) (string, error) { return "/bin/gh", nil }
func (f *fakeRunner) Run(_ context.Context, cmd ports.Command) (ports.ProcessResult, error) { f.last = cmd; return ports.ProcessResult{Stdout: f.out}, f.err }
func (f *fakeRunner) RunInteractive(context.Context, ports.Command, io.Reader, io.Writer, io.Writer) error { return f.err }
func TestRepositoryUsesStructuredJSONAndHostEnv(t *testing.T) {
	r := &fakeRunner{out: []byte(`{"nameWithOwner":"owner/repo","url":"https://git.example.com/owner/repo","hasIssuesEnabled":true}`)}
	c := New(r, nil, io.Discard, io.Discard)
	got, err := c.Repository(context.Background(), domain.RepositoryIdentity{Host: "git.example.com", Owner: "owner", Repo: "repo"})
	if err != nil { t.Fatal(err) }
	want := []string{"repo", "view", "owner/repo", "--json", "nameWithOwner,url,hasIssuesEnabled"}
	if !reflect.DeepEqual(r.last.Args, want) { t.Fatalf("args=%v", r.last.Args) }
	if !reflect.DeepEqual(r.last.Env, []string{"GH_HOST=git.example.com"}) { t.Fatalf("env=%v", r.last.Env) }
	if got.NameWithOwner != "owner/repo" || !got.HasIssuesEnabled { t.Fatalf("got=%+v", got) }
}
func TestIssueUsesRepoFlagAndJSON(t *testing.T) {
	r := &fakeRunner{out: []byte(`{"number":12,"state":"OPEN","title":"Task"}`)}
	c := New(r, nil, io.Discard, io.Discard)
	got, err := c.Issue(context.Background(), domain.RepositoryIdentity{Host: "github.com", Owner: "owner", Repo: "repo"}, 12)
	if err != nil { t.Fatal(err) }
	want := []string{"issue", "view", "12", "--repo", "owner/repo", "--json", "number,state,title"}
	if !reflect.DeepEqual(r.last.Args, want) { t.Fatalf("args=%v", r.last.Args) }
	if got.Number != 12 || got.Title != "Task" { t.Fatalf("got=%+v", got) }
}
