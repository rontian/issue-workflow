package domain

import (
    "strings"
    "testing"
)

func TestTaskProgressOrderedAndContractBound(t *testing.T){
    body,err:=RenderTaskContract(TaskContractInput{ContractID:"c1",Goal:"goal",Scope:[]string{"implement"},OutOfScope:[]string{"unrelated"},AcceptanceCriteria:[]string{"done"},Tasks:[]string{"T01: First","T02: Second"}})
    if err!=nil{t.Fatal(err)}
    contract,err:=ParseTaskContract(body);if err!=nil{t.Fatal(err)}
    progress,err:=ReplayTaskProgress(contract,nil)
    if err!=nil||progress.Current!="T01"||progress.Next!="T02"||progress.AllCompleted {t.Fatalf("%#v %v",progress,err)}
    skipped:=WorkflowEvent{Schema:WorkflowEventSchemaV2,EventType:EventTaskComplete,Data:map[string]any{"task_id":"T02","summary":"second"}}
    if _,err:=ReplayTaskProgress(contract,[]WorkflowEvent{skipped});err==nil{t.Fatal("skipped task accepted")}
    first:=WorkflowEvent{Schema:WorkflowEventSchemaV2,EventType:EventTaskComplete,Data:map[string]any{"task_id":"T01","summary":"first"}}
    progress,err=ReplayTaskProgress(contract,[]WorkflowEvent{first})
    if err!=nil||progress.Current!="T02"||progress.Next!=""||progress.AllCompleted {t.Fatalf("%#v %v",progress,err)}
    progress,err=ReplayTaskProgress(contract,[]WorkflowEvent{first,skipped})
    if err!=nil||!progress.AllCompleted||len(progress.Completed)!=2 {t.Fatalf("%#v %v",progress,err)}
    // New accepted scope is a new task contract; old evidence cannot carry over.
    scope:=WorkflowEvent{Schema:WorkflowEventSchemaV2,EventType:EventScope,Data:map[string]any{"action":"accept"}}
    progress,err=ReplayTaskProgress(contract,[]WorkflowEvent{first,scope})
    if err!=nil||progress.Current!="T01"||len(progress.Completed)!=0 {t.Fatalf("%#v %v",progress,err)}
}
func TestTaskPlanRejectsDuplicateAndMalformedIDs(t *testing.T){
    c:=TaskContract{ImplementationTasks:"- [ ] T01: First\n- [ ] T01: Duplicate"}
    if _,err:=PlannedTasks(c);err==nil{t.Fatal("duplicate task accepted")}
    c.ImplementationTasks="- [ ] just prose"
    if _,err:=PlannedTasks(c);err==nil{t.Fatal("malformed planned task accepted")}
    c.ImplementationTasks="- [ ] T01: First\n- [ ] T02: Second"
    if tasks,err:=PlannedTasks(c);err!=nil||len(tasks)!=2||!strings.HasPrefix(tasks[1].ID,"T"){t.Fatalf("%#v %v",tasks,err)}
}
