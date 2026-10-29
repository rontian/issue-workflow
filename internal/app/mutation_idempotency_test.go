package app

import (
	"context"
	"strings"
	"testing"
)

func TestBlockRetryWithSameOperationIDIsIdempotent(t *testing.T) {
	s, g, _ := stateService(t, false)
	if r, c := s.Start(context.Background(), mo("block-idem-start")); c != 0 || !r.OK { t.Fatal(c, r) }
	b := mo("block-idem"); b.Reason = "need approval"; b.Next = "ask owner"
	if r, c := s.Block(context.Background(), b); c != 0 || !r.OK || r.State != "BLOCKED" { t.Fatalf("first block %d %#v", c, r) }
	if r, c := s.Block(context.Background(), b); c != 0 || !r.OK || r.State != "BLOCKED" { t.Fatalf("retry block %d %#v", c, r) }
	if g.appendCalls != 2 { t.Fatalf("appendCalls=%d", g.appendCalls) }
}

func TestScopeAcceptRetryWithSameOperationIDIsIdempotent(t *testing.T) {
	s, gh, _ := stateService(t, false)
	if r, c := s.Start(context.Background(), mo("accept-idem-start")); c != 0 || !r.OK { t.Fatal(c, r) }
	gh.issue.Body = strings.Replace(gh.issue.Body, "## Scope\n\n- scope", "## Scope\n\n- scope\n- approved-extra", 1)
	ac := mo("accept-idem"); ac.ScopeAction = "accept"; ac.Reason = "approved"
	if r, c := s.Scope(context.Background(), ac); c != 0 || !r.OK { t.Fatalf("first accept %d %#v", c, r) }
	if r, c := s.Scope(context.Background(), ac); c != 0 || !r.OK { t.Fatalf("retry accept %d %#v", c, r) }
	if gh.appendCalls != 2 { t.Fatalf("appendCalls=%d", gh.appendCalls) }
}

func TestAcknowledgedCommentWithFailedReadbackCanBeSafelyRetried(t *testing.T) {
    s, gh, _ := stateService(t, false)
    gh.failReadAfterAppend = true
    op := mo("acknowledged-but-readback-failed")
    first, code := s.Start(context.Background(), op)
    if code != 4 || first.OK || first.Error == nil || first.Error.Code != "GITHUB_UNAVAILABLE" {
        t.Fatalf("readback failure must be retryable GitHub error: %d %#v", code, first)
    }
    if first.Error.Details["write_acknowledged"] != true || first.Error.Details["outcome_unknown"] != true {
        t.Fatalf("must report persisted write uncertainty: %#v", first.Error.Details)
    }
    if gh.appendCalls != 1 || len(gh.comments) != 1 {
        t.Fatalf("first attempt must have written exactly once: %d %d", gh.appendCalls, len(gh.comments))
    }
    again, code := s.Start(context.Background(), op)
    if code != 0 || !again.OK || again.State != "IN_PROGRESS" {
        t.Fatalf("retry must recover by operation id: %d %#v", code, again)
    }
    if gh.appendCalls != 1 {
        t.Fatalf("retry appended a duplicate workflow event: %d", gh.appendCalls)
    }
}
