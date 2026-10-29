package domain

import "testing"

func TestWaitConditionSpecValidation(t *testing.T){
    if _,err:=ParseWaitConditionSpec("ci=resolved");err==nil{t.Fatal("resolved requires actual evidence")}
    if _,err:=ParseWaitConditionSpec("ci=done");err==nil{t.Fatal("unknown status accepted")}
    if _,err:=ParseWaitConditionSpec("../unsafe=waiting");err==nil{t.Fatal("invalid id accepted")}
    if c,err:=ParseWaitConditionSpec("ci=resolved:CI build passed");err!=nil||c.ID!="ci"||c.Status!="resolved"{t.Fatalf("%#v %v",c,err)}
    if err:=ValidateWaitConditions([]WaitCondition{{ID:"ci",Status:"waiting"},{ID:"ci",Status:"waiting"}});err==nil{t.Fatal("duplicate ID accepted")}
}
func TestForgedRecheckEventCannotSkipNamedConditions(t *testing.T){
    aggregate:=WorkflowAggregate{WaitingConditions:[]WaitCondition{{ID:"ci",Status:"waiting"},{ID:"owner",Status:"waiting"}}}
    event:=WorkflowEvent{Schema:WorkflowEventSchemaV2,EventType:EventRecheck,Data:map[string]any{
        "resolved":true,"conditions":[]any{map[string]any{"id":"ci","status":"resolved","summary":"pass"}},
    }}
    if err:=validateWaitEventAgainstAggregate(aggregate,event);err==nil{t.Fatal("replay accepted skipped condition")}
    event.Data["conditions"]=[]any{map[string]any{"id":"ci","status":"resolved","summary":"pass"},map[string]any{"id":"owner","status":"waiting"}}
    if err:=validateWaitEventAgainstAggregate(aggregate,event);err==nil{t.Fatal("replay accepted resolved=true with pending condition")}
    event.Data["resolved"]=false
    if err:=validateWaitEventAgainstAggregate(aggregate,event);err!=nil{t.Fatalf("valid partial recheck rejected: %v",err)}
}
