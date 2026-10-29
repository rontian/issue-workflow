package app

import (
    "context"
    "strings"
    "testing"

    "github.com/rontian/issue-workflow/internal/domain"
)

func TestScopeAcceptInvalidatesPreviouslyPassingValidation(t *testing.T) {
    s,gh,_:=stateService(t,true)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("start: %d %#v",c,r)}
    cp:=mo("cp");cp.Summary="old validation";cp.Next="complete";cp.Validation=[]domain.ValidationRecord{{ID:"v1",Status:"pass"}}
    if r,c:=s.Checkpoint(context.Background(),cp);c!=0||!r.OK{t.Fatalf("cp: %d %#v",c,r)}
    gh.issue.Body=strings.Replace(gh.issue.Body,"go test ./...","go vet ./...",1)
    scope:=mo("scope-accept");scope.ScopeAction="accept";scope.Reason="owner approved a changed validation command"
    if r,c:=s.Scope(context.Background(),scope);c!=0||!r.OK{t.Fatalf("scope: %d %#v",c,r)}
    complete:=mo("complete");complete.Summary="implemented"
    if r,c:=s.Complete(context.Background(),complete);c!=6||r.Error==nil||r.Error.Code!="VALIDATION_FAILED"{t.Fatalf("stale validation: %d %#v",c,r)}
    fresh:=mo("fresh");fresh.Summary="new gate passed";fresh.Next="complete";fresh.Validation=[]domain.ValidationRecord{{ID:"v1",Status:"pass"}}
    if r,c:=s.Checkpoint(context.Background(),fresh);c!=0||!r.OK{t.Fatalf("fresh cp: %d %#v",c,r)}
    if r,c:=s.Complete(context.Background(),complete);c!=0||!r.OK{t.Fatalf("fresh validation: %d %#v",c,r)}
}
