package app

import (
    "context"
    "testing"

    "github.com/rontian/issue-workflow/internal/domain"
)

func TestRecheckEveryNamedExternalWaitCondition(t *testing.T){
    s,gh,_:=stateService(t,false)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("start: %d %#v",c,r)}
    wait:=mo("wait")
    wait.Reason="waiting for CI and owner resources"
    wait.Next="recheck both"
    wait.Conditions=[]domain.WaitCondition{
        {ID:"ci",Status:"waiting",Summary:"pipeline pending"},
        {ID:"owner",Status:"waiting",Summary:"real-device approval pending"},
    }
    if r,c:=s.Wait(context.Background(),wait);c!=0||!r.OK||r.State!="WAITING"{t.Fatalf("wait: %d %#v",c,r)}

    // Reporting only one item is not enough, and --resolved cannot override it.
    incomplete:=mo("incomplete");incomplete.Conditions=[]domain.WaitCondition{{ID:"ci",Status:"resolved",Summary:"CI job passed"}}
    if r,c:=s.Recheck(context.Background(),incomplete);c!=2||r.Error==nil||r.Error.Code!="INVALID_INVOCATION"{t.Fatalf("partial recheck: %d %#v",c,r)}
    bypass:=mo("bypass");bypass.Resolved=true
    if r,c:=s.Recheck(context.Background(),bypass);c!=2||r.Error==nil||r.Error.Code!="INVALID_INVOCATION"{t.Fatalf("blind resolved: %d %#v",c,r)}
    if gh.appendCalls!=2{t.Fatalf("forbidden writes happened: %d",gh.appendCalls)}

    partial:=mo("partial")
    partial.Conditions=[]domain.WaitCondition{{ID:"ci",Status:"resolved",Summary:"CI job passed"},{ID:"owner",Status:"waiting",Summary:"owner has not reviewed"}}
    if r,c:=s.Recheck(context.Background(),partial);c!=0||!r.OK||r.State!="WAITING"{t.Fatalf("partial recheck: %d %#v",c,r)}
    st,_:=s.Status(context.Background(),WorkflowOptions{CWD:"/repo",Issue:7})
    conditions:=st.Data.(WorkflowReport).Aggregate.WaitingConditions
    if len(conditions)!=2 || conditions[0].Status!="resolved" || conditions[1].Status!="waiting" {t.Fatalf("persisted condition facts: %#v",conditions)}

    all:=mo("all")
    all.Conditions=[]domain.WaitCondition{{ID:"owner",Status:"resolved",Summary:"owner approved on target device"},{ID:"ci",Status:"resolved",Summary:"CI job passed"}}
    if r,c:=s.Recheck(context.Background(),all);c!=0||!r.OK||r.State!="IN_PROGRESS"{t.Fatalf("all resolved: %d %#v",c,r)}
    st,_=s.Status(context.Background(),WorkflowOptions{CWD:"/repo",Issue:7})
    conditions=st.Data.(WorkflowReport).Aggregate.WaitingConditions
    if len(conditions)!=2 || !domain.WaitConditionsResolved(conditions){t.Fatalf("resolved facts not replayed: %#v",conditions)}
}

func TestNamedWaitCannotBeginAlreadyResolved(t *testing.T){
    s,gh,_:=stateService(t,false)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    w:=mo("wait");w.Reason="CI";w.Next="recheck"
    w.Conditions=[]domain.WaitCondition{{ID:"ci",Status:"resolved",Summary:"pre-declared success"}}
    if r,c:=s.Wait(context.Background(),w);c!=2||r.Error==nil||r.Error.Code!="INVALID_INVOCATION"{t.Fatalf("%d %#v",c,r)}
    if gh.appendCalls!=1{t.Fatalf("unexpected mutation: %d",gh.appendCalls)}
}
