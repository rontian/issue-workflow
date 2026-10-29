package app

import (
	"context"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

// These tests assert a security property missing from an exit-code-only
// acceptance: a rejected operation must not append any canonical event.
func TestUnknownEventSchemaFailsClosedWithoutNewEvent(t *testing.T) {
	s, gh, _ := stateService(t, false)
	gh.comments = append(gh.comments, domain.GitHubComment{
		ID: 1, Body: "<!-- iw:workflow-event:v99\n{}\n-->",
	})
	r, code := s.Start(context.Background(), mo("poisoned-start"))
	if code != result.ExitProtocol || r.OK || r.Error == nil || r.Error.Code != "UNSUPPORTED_SCHEMA" {
		t.Fatalf("unknown schema mutation: code=%d result=%#v", code, r)
	}
	if gh.appendCalls != 0 || len(gh.comments) != 1 {
		t.Fatalf("unknown schema mutated events: appends=%d comments=%d", gh.appendCalls, len(gh.comments))
	}
	r, code = s.Status(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7})
	if code != result.ExitProtocol || r.OK || r.Error == nil || r.Error.Code != "UNSUPPORTED_SCHEMA" {
		t.Fatalf("unknown schema read: code=%d result=%#v", code, r)
	}
}

func TestConflictingOperationFingerprintNeverAppends(t *testing.T) {
	s, gh, _ := stateService(t, false)
	first := mo("same-id-different-payload")
	if r, code := s.Start(context.Background(), first); code != result.ExitOK || !r.OK {
		t.Fatalf("start: code=%d result=%#v", code, r)
	}
	conflict := mo(first.OperationID)
	conflict.Summary = "a different logical START payload"
	r, code := s.Start(context.Background(), conflict)
	if code != result.ExitConflict || r.OK || r.Error == nil || r.Error.Code != "CONFLICT" {
		t.Fatalf("operation conflict: code=%d result=%#v", code, r)
	}
	if gh.appendCalls != 1 || len(gh.comments) != 1 {
		t.Fatalf("conflicting operation appended an event: appends=%d comments=%d", gh.appendCalls, len(gh.comments))
	}
}

func TestStaleReviewAndValidationHeadsDoNotAppendCompletion(t *testing.T) {
	s, gh, git := stateService(t, true)
	if r, code := s.Start(context.Background(), mo("negative-head-start")); code != result.ExitOK || !r.OK {
		t.Fatalf("start: %d %#v", code, r)
	}
	cp := mo("negative-head-checkpoint")
	cp.Summary, cp.Next = "initial evidence", "review"
	cp.Validation = []domain.ValidationRecord{{ID: "v1", Status: "pass"}}
	if r, code := s.Checkpoint(context.Background(), cp); code != result.ExitOK || !r.OK {
		t.Fatalf("checkpoint: %d %#v", code, r)
	}
	rv := mo("negative-head-review")
	rv.Summary = "passed old HEAD"
	if r, code := s.Review(context.Background(), rv); code != result.ExitOK || !r.OK {
		t.Fatalf("review: %d %#v", code, r)
	}
	git.snap.Head = "def"
	finish := mo("negative-head-complete")
	finish.Summary = "should remain incomplete"
	r, code := s.Complete(context.Background(), finish)
	// A passed review bound to the old HEAD is checked before the general
	// Git-drift guard. Both prevent a write, but REVIEW_REQUIRED is the
	// canonical higher-priority error in this fixture.
	if code != result.ExitValidation || r.Error == nil || r.Error.Code != "REVIEW_REQUIRED" || gh.appendCalls != 3 {
		t.Fatalf("stale review before checkpoint: %d error=%v appends=%d", code, r.Error, gh.appendCalls)
	}
	cp2 := mo("negative-head-cp2")
	cp2.Summary, cp2.Next = "new HEAD without validation", "review again"
	if r, code := s.Checkpoint(context.Background(), cp2); code != result.ExitOK || !r.OK {
		t.Fatalf("checkpoint new HEAD: %d %#v", code, r)
	}
	r, code = s.Complete(context.Background(), finish)
	if code != result.ExitValidation || r.Error == nil || r.Error.Code != "REVIEW_REQUIRED" || gh.appendCalls != 4 {
		t.Fatalf("stale review: %d %#v appends=%d", code, r, gh.appendCalls)
	}
	rv2 := mo("negative-head-review2")
	rv2.Summary = "passed new HEAD"
	if r, code := s.Review(context.Background(), rv2); code != result.ExitOK || !r.OK {
		t.Fatalf("new review: %d %#v", code, r)
	}
	r, code = s.Complete(context.Background(), finish)
	if code != result.ExitValidation || r.Error == nil || r.Error.Code != "VALIDATION_FAILED" || gh.appendCalls != 5 {
		t.Fatalf("stale validation: %d %#v appends=%d", code, r, gh.appendCalls)
	}
}


func TestGitHeadDriftWithoutExistingReviewFailsClosed(t *testing.T) {
	s, gh, git := stateService(t, false)
	if r, code := s.Start(context.Background(), mo("git-drift-start")); code != result.ExitOK || !r.OK {
		t.Fatalf("start: %d %#v", code, r)
	}
	cp := mo("git-drift-checkpoint")
	cp.Summary, cp.Next = "record initial head", "complete after checking git"
	if r, code := s.Checkpoint(context.Background(), cp); code != result.ExitOK || !r.OK {
		t.Fatalf("checkpoint: %d %#v", code, r)
	}
	git.snap.Head = "different-current-head"
	finish := mo("git-drift-complete")
	finish.Summary = "completion must be blocked"
	r, code := s.Complete(context.Background(), finish)
	if code != result.ExitConflict || r.Error == nil || r.Error.Code != "STALE_GIT_STATE" {
		t.Fatalf("unreviewed git drift: code=%d error=%v", code, r.Error)
	}
	if gh.appendCalls != 2 || len(gh.comments) != 2 {
		t.Fatalf("rejected complete must not append: appends=%d comments=%d", gh.appendCalls, len(gh.comments))
	}
}
