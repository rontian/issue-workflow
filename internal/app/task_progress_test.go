package app

import (
    "context"
    "strings"
    "testing"
)

func TestTaskCompleteCannotSkipTxxAndGatesCompletion(t *testing.T){
    s,gh,_:=stateService(t,false)
    gh.issue.Body=strings.Replace(gh.issue.Body,"<!-- iw:task-contract:v1","## Implementation Tasks\n\n- [ ] T01: First\n- [ ] T02: Second\n\n<!-- iw:task-contract:v1",1)
    if r,c:=s.Start(context.Background(),mo("task-start"));c!=0||!r.OK{t.Fatalf("start: %d %#v",c,r)}
    done:=mo("task-complete");done.Summary="done"
    if r,c:=s.Complete(context.Background(),done);c!=5||r.Error==nil||r.Error.Code!="TRANSITION_BLOCKED"{t.Fatalf("completed before Txx: %d %#v",c,r)}
    skip:=mo("task-skip");skip.Summary="skip";skip.TaskID="T02"
    if r,c:=s.TaskComplete(context.Background(),skip);c!=5||r.Error==nil||r.Error.Code!="TRANSITION_BLOCKED"{t.Fatalf("skipped T01: %d %#v",c,r)}
    one:=mo("task-one");one.TaskID="T01";one.Summary="first done"
    if r,c:=s.TaskComplete(context.Background(),one);c!=0||!r.OK{t.Fatalf("T01: %d %#v",c,r)}
    status,c:=s.Status(context.Background(),WorkflowOptions{CWD:"/repo",Issue:7})
    if c!=0||!status.OK||status.Data.(WorkflowReport).TaskProgress.Current!="T02"{t.Fatalf("progress: %d %#v",c,status)}
    if r,c:=s.Complete(context.Background(),done);c!=5||r.Error==nil||r.Error.Code!="TRANSITION_BLOCKED"{t.Fatalf("completed before T02: %d %#v",c,r)}
    two:=mo("task-two");two.TaskID="T02";two.Summary="second done"
    if r,c:=s.TaskComplete(context.Background(),two);c!=0||!r.OK{t.Fatalf("T02: %d %#v",c,r)}
    if r,c:=s.Complete(context.Background(),done);c!=0||!r.OK||r.State!="COMPLETED"{t.Fatalf("complete: %d %#v",c,r)}
}

func TestDeferCannotPretendUnfinishedTaskSettled(t *testing.T){
    s,gh,_:=stateService(t,false)
    gh.issue.Body=strings.Replace(gh.issue.Body,"<!-- iw:task-contract:v1","## Implementation Tasks\n\n- [ ] T01: First\n\n<!-- iw:task-contract:v1",1)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    d:=mo("defer");d.Reason="later";d.Next="come back"
    if r,c:=s.Defer(context.Background(),d);c!=5||r.Error==nil||r.Error.Code!="TRANSITION_BLOCKED"{t.Fatalf("unsound defer: %d %#v",c,r)}
    if gh.appendCalls!=1{t.Fatalf("defer event written: %d",gh.appendCalls)}
}
