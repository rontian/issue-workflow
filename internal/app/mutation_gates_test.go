package app

import (
	"context"
	"strings"
	"testing"
)

func TestScopeProposalBlocksCompleteUntilRejected(t *testing.T) {
	s, _, _ := stateService(t, false)
	if r, c := s.Start(context.Background(), mo("s-scope")); c != 0 || !r.OK { t.Fatal(c, r) }
	p := mo("scope-p"); p.ScopeAction, p.Summary, p.Reason = "propose", "add migration", "required by implementation"
	r, c := s.Scope(context.Background(), p); if c != 0 || !r.OK { t.Fatalf("propose %d %#v", c, r) }
	rep := r.Data.(WorkflowReport); if len(rep.Aggregate.UnresolvedScopeProposals) != 1 { t.Fatalf("%+v", rep.Aggregate.UnresolvedScopeProposals) }; proposal := rep.Aggregate.UnresolvedScopeProposals[0]
	fn := mo("scope-final"); fn.Summary = "done"
	if r, c := s.Complete(context.Background(), fn); c != 5 || r.Error == nil || r.Error.Code != "SCOPE_UNRESOLVED" { t.Fatalf("complete %d %#v", c, r) }
	rej := mo("scope-r"); rej.ScopeAction, rej.ProposalEventID, rej.Reason = "reject", proposal, "not needed"
	if r, c := s.Scope(context.Background(), rej); c != 0 || !r.OK { t.Fatalf("reject %d %#v", c, r) }
	if r, c := s.Complete(context.Background(), fn); c != 0 || !r.OK || r.State != "COMPLETED" { t.Fatalf("complete after reject %d %#v", c, r) }
}

func TestContractDriftBlocksMutationUntilScopeAccept(t *testing.T) {
	s, gh, _ := stateService(t, false)
	if r, c := s.Start(context.Background(), mo("drift-start")); c != 0 || !r.OK { t.Fatal(c, r) }
	gh.issue.Body = strings.Replace(gh.issue.Body, "## Scope\n\n- scope", "## Scope\n\n- scope\n- migration", 1)
	cp := mo("drift-cp"); cp.Summary = "x"; cp.Next = "y"
	if r, c := s.Checkpoint(context.Background(), cp); c != 7 || r.Error == nil || r.Error.Code != "CONTRACT_CHANGED" { t.Fatalf("checkpoint %d %#v", c, r) }
	ac := mo("drift-accept"); ac.ScopeAction = "accept"; ac.Reason = "approved"
	if r, c := s.Scope(context.Background(), ac); c != 0 || !r.OK { t.Fatalf("accept %d %#v", c, r) }
	if r, c := s.Status(context.Background(), WorkflowOptions{CWD: "/repo", Issue: 7}); c != 0 || !r.OK || r.Data.(WorkflowReport).ContractChanged { t.Fatalf("status %d %#v", c, r) }
}

func TestBlockedMustResumeBeforeComplete(t *testing.T) {
	s, _, _ := stateService(t, false)
	if r, c := s.Start(context.Background(), mo("b-start")); c != 0 || !r.OK { t.Fatal(c, r) }
	b := mo("b-block"); b.Reason = "need decision"; b.Next = "get approval"
	if r, c := s.Block(context.Background(), b); c != 0 || !r.OK || r.State != "BLOCKED" { t.Fatalf("block %d %#v", c, r) }
	fn := mo("b-final"); fn.Summary = "done"
	if r, c := s.Complete(context.Background(), fn); c != 5 || r.Error == nil || r.Error.Code != "TRANSITION_BLOCKED" { t.Fatalf("complete %d %#v", c, r) }
	if r, c := s.Resume(context.Background(), mo("b-resume")); c != 0 || !r.OK || r.State != "IN_PROGRESS" { t.Fatalf("resume %d %#v", c, r) }
}

func TestReviewFindingsBlockCompletionUntilFixAndReview(t *testing.T) {
	s,_,_:=stateService(t,false);if r,c:=s.Start(context.Background(),mo("r-start"));c!=0||!r.OK{t.Fatal(c,r)}
	rv:=mo("r-review1");rv.Summary="found issue";rv.Findings=[]string{"F01"};if r,c:=s.Review(context.Background(),rv);c!=0||!r.OK{t.Fatal(c,r)}
	fn:=mo("r-complete");fn.Summary="done";if r,c:=s.Complete(context.Background(),fn);c!=6||r.Error==nil||r.Error.Code!="REVIEW_REQUIRED"{t.Fatalf("complete %d %#v",c,r)}
	fx:=mo("r-fix");fx.Summary="fixed F01";if r,c:=s.Fix(context.Background(),fx);c!=0||!r.OK{t.Fatal(c,r)}
	rv2:=mo("r-review2");rv2.Summary="clean";if r,c:=s.Review(context.Background(),rv2);c!=0||!r.OK{t.Fatal(c,r)}
	if r,c:=s.Complete(context.Background(),fn);c!=0||r.State!="COMPLETED"{t.Fatalf("complete %d %#v",c,r)}
}
