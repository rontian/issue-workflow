package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
)

type fakeWorkflowGit struct { snapshot domain.GitSnapshot; repo domain.RepositoryIdentity }
func (f fakeWorkflowGit) Version(context.Context) (string, error) { return "git", nil }
func (f fakeWorkflowGit) RepositoryRoot(context.Context, string) (string, error) { return "/repo", nil }
func (f fakeWorkflowGit) ResolveRemote(context.Context, string, string) (domain.RepositoryIdentity, string, error) { return f.repo, "origin", nil }
func (f fakeWorkflowGit) Snapshot(context.Context, string) (domain.GitSnapshot, error) { return f.snapshot, nil }

type fakeWorkflowGH struct { issue domain.GitHubIssueDetails; comments []domain.GitHubComment; repo domain.GitHubRepository }
func (f fakeWorkflowGH) Authenticated(context.Context, string) error { return nil }
func (f fakeWorkflowGH) Repository(context.Context, domain.RepositoryIdentity) (domain.GitHubRepository, error) { return f.repo, nil }
func (f fakeWorkflowGH) IssueDetails(context.Context, domain.RepositoryIdentity, int) (domain.GitHubIssueDetails, error) { return f.issue, nil }
func (f fakeWorkflowGH) IssueComments(context.Context, domain.RepositoryIdentity, int) ([]domain.GitHubComment, error) { return f.comments, nil }
func (f fakeWorkflowGH) CreateIssue(context.Context, domain.RepositoryIdentity, string, string) (domain.GitHubIssueDetails, error) { return f.issue, nil }
func (f fakeWorkflowGH) AppendIssueComment(context.Context, domain.RepositoryIdentity, int, string) error { return nil }

func contractBody() string { return "## Goal\n\nG\n\n## Scope\n\nS\n\n## Out of Scope\n\nO\n\n## Acceptance Criteria\n\n- [ ] A\n\n<!-- iw:task-contract:v1\n{\"schema\":\"iw.task-contract/v1\",\"contract_id\":\"c1\"}\n-->\n" }
func workflowComment(t *testing.T, digest string) domain.GitHubComment {
	t.Helper()
	branch := "main"
	e := domain.WorkflowEvent{Schema: domain.WorkflowEventSchema, EventID: "e1", ParentEventID: nil, OperationID: "o1", RunID: "r1", EventType: domain.EventStart, OccurredAt: "2026-09-15T04:00:00Z", Repository: "o/r", Issue: 7, ContractDigest: digest, StateBefore: domain.StateReady, StateAfter: domain.StateInProgress, Git: &domain.EventGitSnapshot{Branch: &branch, Head: "abc", ChangedPaths: []string{}}, Data: map[string]any{}}
	raw, _ := json.Marshal(e)
	return domain.GitHubComment{ID: 1, CreatedAt: e.OccurredAt, Body: "[START]\n\n<!-- iw:workflow-event:v1\n" + string(raw) + "\n-->"}
}
func newService(body string, comments []domain.GitHubComment) *WorkflowService {
	repo := domain.RepositoryIdentity{Host: "github.com", Owner: "o", Repo: "r"}
	return &WorkflowService{Git: fakeWorkflowGit{repo: repo, snapshot: domain.GitSnapshot{RepositoryRoot: "/repo", Branch: "main", Head: "abc", ChangedPaths: []string{}}}, GitHub: fakeWorkflowGH{issue: domain.GitHubIssueDetails{Number: 7, State: "OPEN", Title: "T", Body: body}, comments: comments, repo: domain.GitHubRepository{NameWithOwner: "o/r", HasIssuesEnabled: true}}}
}
func TestWorkflowStatusSuccess(t *testing.T) {
	body := contractBody(); s := newService(body, []domain.GitHubComment{workflowComment(t, domain.ContractDigest(body))})
	r, code := s.Status(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7})
	if code != 0 || !r.OK { t.Fatalf("%d %#v", code, r) }
	report := r.Data.(WorkflowReport)
	if report.Aggregate.State != domain.StateInProgress || report.ContractChanged { t.Fatalf("%#v", report) }
	if len(r.Warnings) != 0 { t.Fatalf("%v", r.Warnings) }
}
func TestWorkflowStatusReportsContractChangedWithoutFailing(t *testing.T) {
	old := contractBody(); changed := strings.Replace(old, "\nG\n", "\nG changed\n", 1)
	s := newService(changed, []domain.GitHubComment{workflowComment(t, domain.ContractDigest(old))})
	r, code := s.Status(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7})
	if code != 0 || !r.OK { t.Fatalf("%d %#v", code, r) }
	report := r.Data.(WorkflowReport)
	if !report.ContractChanged { t.Fatal("expected contract changed") }
	if len(r.Warnings) == 0 || r.Warnings[0] != "CONTRACT_CHANGED" { t.Fatalf("%v", r.Warnings) }
}
func TestResumeDryRunRejectsReady(t *testing.T) {
	s := newService(contractBody(), nil)
	r, code := s.ResumeDryRun(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7})
	if code != 5 || r.Error == nil || r.Error.Code != "TRANSITION_BLOCKED" { t.Fatalf("%d %#v", code, r) }
}
func TestExplicitRepoMismatchFailsClosed(t *testing.T) {
	s := newService(contractBody(), nil)
	r, code := s.Status(context.Background(), WorkflowOptions{CWD: "/repo", Repo: "x/y", Issue: 7})
	if code != 3 || r.Error == nil || r.Error.Code != "REPOSITORY_IDENTITY_MISMATCH" { t.Fatalf("%d %#v", code, r) }
}

type failGH struct{ fakeWorkflowGH }
func (f failGH) IssueComments(context.Context, domain.RepositoryIdentity, int) ([]domain.GitHubComment, error) { return nil, errors.New("boom") }

func TestDoneRecoveryDoesNotReuseOldNext(t *testing.T) {
	body := contractBody(); digest := domain.ContractDigest(body); branch := "main"
	start := domain.WorkflowEvent{Schema: domain.WorkflowEventSchema, EventID: "e1", OperationID: "o1", RunID: "r1", EventType: domain.EventStart, OccurredAt: "2026-09-15T04:00:00Z", Repository: "o/r", Issue: 7, ContractDigest: digest, StateBefore: domain.StateReady, StateAfter: domain.StateInProgress, Git: &domain.EventGitSnapshot{Branch: &branch, Head: "abc", ChangedPaths: []string{}}, Data: map[string]any{}}
	p1 := "e1"
	cp := domain.WorkflowEvent{Schema: domain.WorkflowEventSchema, EventID: "e2", ParentEventID: &p1, OperationID: "o2", RunID: "r1", EventType: domain.EventCheckpoint, OccurredAt: "2026-09-15T04:01:00Z", Repository: "o/r", Issue: 7, ContractDigest: digest, StateBefore: domain.StateInProgress, StateAfter: domain.StateInProgress, Git: &domain.EventGitSnapshot{Branch: &branch, Head: "abc", ChangedPaths: []string{}}, Data: map[string]any{"summary": "x", "next": "review", "validation": []any{}}}
	p2 := "e2"
	final := domain.WorkflowEvent{Schema: domain.WorkflowEventSchema, EventID: "e3", ParentEventID: &p2, OperationID: "o3", RunID: "r1", EventType: domain.EventFinal, OccurredAt: "2026-09-15T04:02:00Z", Repository: "o/r", Issue: 7, ContractDigest: digest, StateBefore: domain.StateInProgress, StateAfter: domain.StateDone, Git: &domain.EventGitSnapshot{Branch: &branch, Head: "abc", ChangedPaths: []string{}}, Data: map[string]any{"summary": "done", "validation": []any{}, "delivery": map[string]any{"commit": "abc", "pushed": false, "pr": nil}}}
	comments := []domain.GitHubComment{}
	for i, e := range []domain.WorkflowEvent{start, cp, final} { raw, _ := domain.RenderWorkflowEvent(e); comments = append(comments, domain.GitHubComment{ID: int64(i + 1), Body: raw, CreatedAt: e.OccurredAt}) }
	s := newService(body, comments)
	r, c := s.Status(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7})
	if c != 0 || !r.OK { t.Fatalf("%d %#v", c, r) }
	recovery := r.Data.(WorkflowReport).Recovery
	got := recovery.SuggestedActions
	if len(got) != 1 || got[0] != "任务已 DONE，无恢复动作" { t.Fatalf("%v", got) }
	if recovery.LastNext != "" { t.Fatalf("DONE recovery must not expose stale next: %q", recovery.LastNext) }
}
