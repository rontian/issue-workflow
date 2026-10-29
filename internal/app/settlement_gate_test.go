package app

import (
    "context"
    "strings"
    "testing"

    "github.com/rontian/issue-workflow/internal/domain"
)

func TestDeferWithExplicitFutureTxxSettlement(t *testing.T){
    s,_,_:=stateService(t,false)
    // Two tasks: current T01 can finish, remaining T02 must be explicitly deferred.
    gh:=s.GitHub.(*stateGH)
    gh.issue.Body=strings.Replace(gh.issue.Body,"<!-- iw:task-contract:v1","## Implementation Tasks\n\n- [ ] T01: First\n- [ ] T02: Later\n\n<!-- iw:task-contract:v1",1)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    t1:=mo("t1");t1.TaskID="T01";t1.Summary="done first"
    if r,c:=s.TaskComplete(context.Background(),t1);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    d:=mo("defer");d.Reason="next milestone";d.Next="resume when ready"
    if r,c:=s.Defer(context.Background(),d);c!=5||r.Error==nil||r.Error.Code!="TRANSITION_BLOCKED"{t.Fatalf("unsound defer: %d %#v",c,r)}
    st:=mo("settle");st.SettlementItem=domain.SettlementItem{ID:"task:T02",Kind:"implementation",FactStatus:"pending",Disposition:"deferred",Target:"defer",Reason:"future milestone explicitly approved"}
    if r,c:=s.Settle(context.Background(),st);c!=0||!r.OK{t.Fatalf("settle: %d %#v",c,r)}
    if r,c:=s.Defer(context.Background(),d);c!=0||!r.OK||r.State!="DEFERRED"{t.Fatalf("defer: %d %#v",c,r)}
    if r,c:=s.Resume(context.Background(),mo("resume"));c!=0||!r.OK{t.Fatalf("resume: %d %#v",c,r)}
    sr,c:=s.Status(context.Background(),WorkflowOptions{CWD:"/repo",Issue:7})
    if c!=0||!sr.OK{t.Fatalf("%d %#v",c,sr)}
    rep:=sr.Data.(WorkflowReport)
    if rep.TaskProgress.Current!="T02"||len(rep.Settlement.Items)!=0{t.Fatalf("bad deferred resume: %#v %#v",rep.TaskProgress,rep.Settlement)}
}
func TestAutomatedValidationCannotBeAcceptedAsManualRisk(t *testing.T){
    s,_,_:=stateService(t,true)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    item:=mo("settle");item.OwnerConfirmed=true
    item.SettlementItem=domain.SettlementItem{ID:"validation:v1",Kind:"manual_validation",FactStatus:"manual_required",Disposition:"accepted_risk",Target:"complete",Reason:"I confirm the limitations of the manual validation."}
    if r,c:=s.Settle(context.Background(),item);c!=2||r.Error==nil||r.Error.Code!="INVALID_INVOCATION"{t.Fatalf("Agent reclassified automated validation: %d %#v",c,r)}
}
