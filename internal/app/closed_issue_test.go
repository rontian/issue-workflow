package app

import (
	"context"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

func TestClosedGitHubIssueBlocksActiveMutation(t *testing.T) {
	s, g, _ := stateService(t, false)
	g.issue.State = "CLOSED"
	r, code := s.Start(context.Background(), mo("closed-start"))
	if code != result.ExitWorkflow || r.Error == nil || r.Error.Code != "GITHUB_ISSUE_CLOSED" {
		t.Fatalf("%d %#v", code, r)
	}
	if g.appendCalls != 0 {
		t.Fatalf("closed issue mutation appended %d comments", g.appendCalls)
	}
}

func TestWorkflowReopenAllowedWhileGitHubIssueRemainsClosed(t *testing.T) {
	s, g, _ := stateService(t, false)
	if r, code := s.Start(context.Background(), mo("closed-reopen-start")); code != result.ExitOK || !r.OK {
		t.Fatalf("start %d %#v", code, r)
	}
	cancel := mo("closed-reopen-cancel")
	cancel.Reason = "stop"
	if r, code := s.Cancel(context.Background(), cancel); code != result.ExitOK || !r.OK {
		t.Fatalf("cancel %d %#v", code, r)
	}
	g.issue.State = "CLOSED"
	reopen := mo("closed-reopen")
	reopen.Reason = "resume task"
	r, code := s.Reopen(context.Background(), reopen)
	if code != result.ExitOK || !r.OK || r.State != string(domain.StateReady) {
		t.Fatalf("reopen %d %#v", code, r)
	}
	if g.issue.State != "CLOSED" {
		t.Fatalf("Core must not reopen GitHub Issue implicitly: %s", g.issue.State)
	}
	found := false
	for _, warning := range r.Warnings {
		if warning == "GITHUB_ISSUE_CLOSED" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected GITHUB_ISSUE_CLOSED warning: %v", r.Warnings)
	}
	if len(r.Constraints) == 0 {
		t.Fatalf("expected closed issue constraint")
	}
}

func TestMigrateAllowedWhileGitHubIssueClosed(t *testing.T) {
	s, g, _ := stateService(t, false)
	g.comments = []domain.GitHubComment{workflowComment(t, domain.ContractDigest(g.issue.Body))}
	g.issue.State = "CLOSED"
	r, code := s.Migrate(context.Background(), mo("closed-migrate"))
	if code != result.ExitOK || !r.OK {
		t.Fatalf("migrate %d %#v", code, r)
	}
	if g.appendCalls != 1 {
		t.Fatalf("appendCalls=%d", g.appendCalls)
	}
}
