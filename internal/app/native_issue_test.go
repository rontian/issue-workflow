package app

import (
    "context"
    "testing"
)

func TestNativeCloseRequiresTerminalWorkflowAndDoesNotMixWithEvent(t *testing.T){
    s,gh,_:=stateService(t,false)
    opts:=WorkflowOptions{CWD:"/repo",Issue:7}
    if r,c:=s.NativeIssueState(context.Background(),opts,true,false);c!=5||r.Error==nil||r.Error.Code!="TRANSITION_BLOCKED"{t.Fatalf("premature close: %d %#v",c,r)}
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    done:=mo("complete");done.Summary="success"
    if r,c:=s.Complete(context.Background(),done);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    if gh.issue.State!="OPEN"{t.Fatal("complete must not close GitHub Issue implicitly")}
    if r,c:=s.NativeIssueState(context.Background(),opts,true,true);c!=0||!r.OK{t.Fatalf("dry close: %d %#v",c,r)}
    if gh.issue.State!="OPEN"{t.Fatal("dry run mutated GH")}
    if r,c:=s.NativeIssueState(context.Background(),opts,true,false);c!=0||!r.OK{t.Fatalf("close: %d %#v",c,r)}
    if gh.issue.State!="CLOSED"{t.Fatal("GH Issue not closed")}
    if r,c:=s.NativeIssueState(context.Background(),opts,true,false);c!=0||!r.OK||!r.Data.(NativeIssueReport).Idempotent{t.Fatalf("idempotent close: %d %#v",c,r)}
}
func TestNativeReopenRequiresWorkflowSegmentFirst(t *testing.T){
    s,gh,_:=stateService(t,false)
    opts:=WorkflowOptions{CWD:"/repo",Issue:7}
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    done:=mo("complete");done.Summary="done"
    if r,c:=s.Complete(context.Background(),done);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    if r,c:=s.NativeIssueState(context.Background(),opts,true,false);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    if r,c:=s.NativeIssueState(context.Background(),opts,false,false);c!=5||r.Error==nil||r.Error.Code!="TRANSITION_BLOCKED"{t.Fatalf("must first iw reopen: %d %#v",c,r)}
    re:=mo("reopen");re.Reason="new segment needs work"
    if r,c:=s.Reopen(context.Background(),re);c!=0||!r.OK||r.State!="READY"{t.Fatalf("%d %#v",c,r)}
    if gh.issue.State!="CLOSED"{t.Fatal("iw reopen must not silently reopen the GH Issue")}
    if r,c:=s.NativeIssueState(context.Background(),opts,false,false);c!=0||!r.OK{t.Fatalf("GH reopen: %d %#v",c,r)}
    if gh.issue.State!="OPEN"{t.Fatal("GH Issue still closed")}
}
