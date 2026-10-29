package app

import (
    "context"
    "strings"
    "testing"

    "github.com/rontian/issue-workflow/internal/domain"
)

func TestManualValidationAcceptedRiskIsNonPassAndCurrentHeadBound(t *testing.T){
    s,gh,_:=stateService(t,false)
    gh.issue.Body=strings.Replace(gh.issue.Body,"<!-- iw:task-contract:v1","## Validation\n\n- manual: owner must inspect on a real device\n\n<!-- iw:task-contract:v1",1)
    if r,c:=s.Start(context.Background(),mo("start"));c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    cp:=mo("manual");cp.Summary="manual work not executed";cp.Next="obtain owner decision";cp.Validation=[]domain.ValidationRecord{{ID:"v1",Status:"manual_required",Summary:"real device not available"}}
    if r,c:=s.Checkpoint(context.Background(),cp);c!=0||!r.OK{t.Fatalf("%d %#v",c,r)}
    done:=mo("complete");done.Summary="manual risk accepted"
    if r,c:=s.Complete(context.Background(),done);c!=6||r.Error==nil||r.Error.Code!="VALIDATION_FAILED"{t.Fatalf("missing owner decision: %d %#v",c,r)}
    item:=mo("owner-confirm");item.OwnerConfirmed=true
    item.SettlementItem=domain.SettlementItem{ID:"validation:v1",Kind:"manual_validation",FactStatus:"manual_required",Disposition:"accepted_risk",Target:"complete",Reason:"Owner accepts missing real-device validation for this synthetic test."}
    if r,c:=s.Settle(context.Background(),item);c!=0||!r.OK{t.Fatalf("owner settle: %d %#v",c,r)}
    status,c:=s.Status(context.Background(),WorkflowOptions{CWD:"/repo",Issue:7})
    if c!=0||!status.OK{t.Fatalf("%d %#v",c,status)}
    rep:=status.Data.(WorkflowReport)
    if record:=domain.ValidationRecordsFromEvents(rep.Aggregate.Events)["v1"];record.Status!="manual_required"{t.Fatalf("risk must never become PASS: %#v",record)}
    if r,c:=s.Complete(context.Background(),done);c!=0||!r.OK||r.State!="COMPLETED"{t.Fatalf("settled completion: %d %#v",c,r)}
}
