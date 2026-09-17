package app

import (
	"context"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
)

func TestWriteHappyPathToDone(t *testing.T) {
	s, g, _ := stateService(t, true)
	o := mo("op-start")
	if r, c := s.Start(context.Background(), o); c != 0 || !r.OK { t.Fatalf("start %d %#v", c, r) }
	cp := mo("op-cp"); cp.Summary = "implemented"; cp.Next = "review"; cp.Validation = []domain.ValidationRecord{{ID: "v1", Status: "pass"}}
	if r, c := s.Checkpoint(context.Background(), cp); c != 0 || !r.OK { t.Fatalf("cp %d %#v", c, r) }
	rv := mo("op-review"); rv.Summary = "review clean"
	if r, c := s.Review(context.Background(), rv); c != 0 || !r.OK { t.Fatalf("review %d %#v", c, r) }
	fn := mo("op-final"); fn.Summary = "done"
	if r, c := s.Final(context.Background(), fn); c != 0 || !r.OK || r.State != "DONE" { t.Fatalf("final %d %#v", c, r) }
	if g.appendCalls != 4 { t.Fatalf("appendCalls=%d", g.appendCalls) }
}

func TestDryRunDoesNotAppend(t *testing.T) {
	s, g, _ := stateService(t, false)
	o := mo("op-start"); o.DryRun = true
	r, c := s.Start(context.Background(), o)
	if c != 0 || !r.OK { t.Fatalf("%d %#v", c, r) }
	if g.appendCalls != 0 || len(g.comments) != 0 { t.Fatalf("side effect: %d %d", g.appendCalls, len(g.comments)) }
}

func TestOperationIdempotent(t *testing.T) {
	s, g, _ := stateService(t, false)
	o := mo("same-op")
	if r, c := s.Start(context.Background(), o); c != 0 || !r.OK { t.Fatalf("first %d %#v", c, r) }
	if r, c := s.Start(context.Background(), o); c != 0 || !r.OK { t.Fatalf("retry %d %#v", c, r) }
	if g.appendCalls != 1 { t.Fatalf("appendCalls=%d", g.appendCalls) }
}

func TestUnknownOutcomeRecoveredByOperationID(t *testing.T) {
	s, g, _ := stateService(t, false)
	g.failAfterAppend = true
	o := mo("recover-op")
	r, c := s.Start(context.Background(), o)
	if c != 0 || !r.OK || r.State != "IN_PROGRESS" { t.Fatalf("%d %#v", c, r) }
	if g.appendCalls != 1 { t.Fatalf("appendCalls=%d", g.appendCalls) }
}

func TestFinalValidationGapAndGitDriftFailClosed(t *testing.T) {
	s, _, git := stateService(t, true)
	if r, c := s.Start(context.Background(), mo("s")); c != 0 || !r.OK { t.Fatal(c, r) }
	cp := mo("cp"); cp.Summary = "x"; cp.Next = "final"
	if r, c := s.Checkpoint(context.Background(), cp); c != 0 || !r.OK { t.Fatal(c, r) }
	fn := mo("f"); fn.Summary = "done"
	if r, c := s.Final(context.Background(), fn); c != resultExitValidation || r.Error == nil || r.Error.Code != "VALIDATION_FAILED" { t.Fatalf("validation %d %#v", c, r) }
	git.snap.Head = "def"
	if r, c := s.Final(context.Background(), fn); c != 7 || r.Error == nil || r.Error.Code != "STALE_GIT_STATE" { t.Fatalf("drift %d %#v", c, r) }
}

const resultExitValidation = 6
