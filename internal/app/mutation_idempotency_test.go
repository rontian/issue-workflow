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
