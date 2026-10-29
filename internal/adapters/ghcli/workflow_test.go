package ghcli

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
)

type fakeWorkflowRunner struct {
	last ports.Command
	out  []byte
}

func (f *fakeWorkflowRunner) LookPath(string) (string, error) { return "/bin/gh", nil }
func (f *fakeWorkflowRunner) Run(_ context.Context, c ports.Command) (ports.ProcessResult, error) {
	f.last = c
	return ports.ProcessResult{Stdout: f.out}, nil
}
func (f *fakeWorkflowRunner) RunInteractive(context.Context, ports.Command, io.Reader, io.Writer, io.Writer) error { return nil }

func TestParseCommentPagesFlattensPages(t *testing.T) {
	raw := []byte(`[[{"id":1,"body":"a","created_at":"2026-09-15T04:00:00Z"}],[{"id":2,"body":"b","created_at":"2026-09-15T04:00:01Z"}]]`)
	got, err := ParseCommentPages(raw)
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 { t.Fatalf("%#v", got) }
}

func TestIssueCommentsUsesPaginateSlurp(t *testing.T) {
	r := &fakeWorkflowRunner{out: []byte(`[[]]`)}
	c := New(r, nil, nil, nil)
	repo := domain.RepositoryIdentity{Host: "ghe.example", Owner: "o", Repo: "r"}
	_, err := c.IssueComments(context.Background(), repo, 7)
	if err != nil { t.Fatal(err) }
	want := []string{"api", "--paginate", "--slurp", "repos/o/r/issues/7/comments?per_page=100", "-H", "Accept: application/vnd.github+json"}
	if !reflect.DeepEqual(r.last.Args, want) { t.Fatalf("args=%v", r.last.Args) }
	if !reflect.DeepEqual(r.last.Env, []string{"GH_HOST=ghe.example"}) { t.Fatalf("env=%v", r.last.Env) }
}

func TestIssueDetailsRequestsBody(t *testing.T) {
	r := &fakeWorkflowRunner{out: []byte(`{"number":7,"state":"OPEN","title":"T","body":"B","url":"U"}`)}
	c := New(r, nil, nil, nil)
	d, err := c.IssueDetails(context.Background(), domain.RepositoryIdentity{Owner: "o", Repo: "r"}, 7)
	if err != nil { t.Fatal(err) }
	if d.Body != "B" || d.URL != "U" { t.Fatalf("%#v", d) }
}

func TestAppendIssueCommentUsesBodyFileStdin(t *testing.T) {
	r := &fakeWorkflowRunner{out: []byte("https://github.com/o/r/issues/7#issuecomment-1\n")}
	c := New(r, nil, io.Discard, io.Discard)
	err := c.AppendIssueComment(context.Background(), domain.RepositoryIdentity{Host: "github.com", Owner: "o", Repo: "r"}, 7, "body with $()")
	if err != nil { t.Fatal(err) }
	want := []string{"issue", "comment", "7", "--repo", "o/r", "--body-file", "-"}
	if !reflect.DeepEqual(r.last.Args, want) { t.Fatalf("args=%v", r.last.Args) }
	if string(r.last.Stdin) != "body with $()" { t.Fatalf("stdin=%q", r.last.Stdin) }
}

func TestCreateIssueUsesBodyFileStdin(t *testing.T) {
	r := &workflowSequenceRunner{outs: [][]byte{[]byte("https://github.com/o/r/issues/12\n"), []byte(`{"number":12,"state":"OPEN","title":"T","body":"B","url":"https://github.com/o/r/issues/12"}`)}}
	c := New(r, nil, io.Discard, io.Discard)
	got, err := c.CreateIssue(context.Background(), domain.RepositoryIdentity{Host: "github.com", Owner: "o", Repo: "r"}, "T", "B")
	if err != nil { t.Fatal(err) }
	if got.Number != 12 { t.Fatalf("%+v", got) }
	if len(r.calls) != 2 { t.Fatalf("calls=%d", len(r.calls)) }
	if !reflect.DeepEqual(r.calls[0].Args, []string{"issue", "create", "--repo", "o/r", "--title", "T", "--body-file", "-"}) { t.Fatalf("args=%v", r.calls[0].Args) }
	if string(r.calls[0].Stdin) != "B" { t.Fatalf("stdin=%q", r.calls[0].Stdin) }
}

type workflowSequenceRunner struct {
	calls []ports.Command
	outs  [][]byte
}
func (r *workflowSequenceRunner) LookPath(string) (string, error) { return "/bin/gh", nil }
func (r *workflowSequenceRunner) Run(_ context.Context, c ports.Command) (ports.ProcessResult, error) {
	r.calls = append(r.calls, c)
	out := []byte{}
	if len(r.outs) > 0 { out = r.outs[0]; r.outs = r.outs[1:] }
	return ports.ProcessResult{Stdout: out}, nil
}
func (r *workflowSequenceRunner) RunInteractive(context.Context, ports.Command, io.Reader, io.Writer, io.Writer) error { return nil }

type flakyCommentRunner struct {
	calls int
	failUntil int
	out []byte
}
func (r *flakyCommentRunner) LookPath(string) (string, error) { return "/bin/gh", nil }
func (r *flakyCommentRunner) Run(_ context.Context, _ ports.Command) (ports.ProcessResult, error) {
	r.calls++
	if r.calls <= r.failUntil { return ports.ProcessResult{}, errors.New("temporary transport failure") }
	return ports.ProcessResult{Stdout: r.out}, nil
}
func (r *flakyCommentRunner) RunInteractive(context.Context, ports.Command, io.Reader, io.Writer, io.Writer) error { return nil }

func TestIssueCommentsRetriesOnlyTransportReadFailure(t *testing.T) {
	runner := &flakyCommentRunner{failUntil: 1, out: []byte(`[[{"id":42,"body":"ok"}]]`)}
	c := New(runner, nil, nil, nil)
	comments, err := c.IssueComments(context.Background(), domain.RepositoryIdentity{Host: "github.com", Owner: "o", Repo: "r"}, 7)
	if err != nil { t.Fatal(err) }
	if runner.calls != 2 || len(comments) != 1 || comments[0].ID != 42 {
		t.Fatalf("read-only retry: calls=%d comments=%#v", runner.calls, comments)
	}
}

func TestIssueCommentsPermanentFailureRemainsClosed(t *testing.T) {
	runner := &flakyCommentRunner{failUntil: 9}
	c := New(runner, nil, nil, nil)
	comments, err := c.IssueComments(context.Background(), domain.RepositoryIdentity{Owner: "o", Repo: "r"}, 7)
	if err == nil || comments != nil || runner.calls != 2 {
		t.Fatalf("must fail closed after bounded retry: calls=%d comments=%#v err=%v", runner.calls, comments, err)
	}
}

func TestIssueCommentsMalformedPageDoesNotRetry(t *testing.T) {
	runner := &flakyCommentRunner{out: []byte(`{"not":"pages"}`)}
	c := New(runner, nil, nil, nil)
	_, err := c.IssueComments(context.Background(), domain.RepositoryIdentity{Owner: "o", Repo: "r"}, 7)
	if err == nil || runner.calls != 1 {
		t.Fatalf("invalid page structure must fail without retry: calls=%d err=%v", runner.calls, err)
	}
}

func TestAppendIssueCommentFailureIsNotRetried(t *testing.T) {
	runner := &flakyCommentRunner{failUntil: 9}
	c := New(runner, nil, nil, nil)
	err := c.AppendIssueComment(context.Background(), domain.RepositoryIdentity{Owner: "o", Repo: "r"}, 7, "potentially written body")
	if err == nil || runner.calls != 1 {
		t.Fatalf("write outcome unknown: never retry append: calls=%d err=%v", runner.calls, err)
	}
}

func TestIssueCommentsCancelledContextDoesNotRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &flakyCommentRunner{failUntil: 9}
	c := New(runner, nil, nil, nil)
	_, err := c.IssueComments(ctx, domain.RepositoryIdentity{Owner: "o", Repo: "r"}, 7)
	if !errors.Is(err, context.Canceled) || runner.calls != 0 {
		t.Fatalf("cancelled read must not call gh: calls=%d err=%v", runner.calls, err)
	}
}
