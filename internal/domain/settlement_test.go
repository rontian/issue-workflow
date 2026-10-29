package domain

import "testing"

func TestSettlementRiskNeverLiesAboutHardFacts(t *testing.T){
    base:=SettlementItem{ID:"validation:v1",Kind:"automated_validation",FactStatus:"fail",Disposition:"accepted_risk",Target:"complete",Head:"abc",DecidedBy:"owner",Reason:"I accept that this automated test failed."}
    if err:=ValidateSettlementItem(base,true);err==nil{t.Fatal("hard automated validation was waived")}
    base.Kind="manual_validation"
    if err:=ValidateSettlementItem(base,true);err==nil{t.Fatal("confirmed manual failure cannot be waived")}
    base.FactStatus="manual_required"
    if err:=ValidateSettlementItem(base,false);err==nil{t.Fatal("Agent claimed owner decision")}
    if err:=ValidateSettlementItem(base,true);err!=nil{t.Fatalf("explicit manual risk should be eligible: %v",err)}
    if !SettlementDispositionValidAtHead(base,"abc") || SettlementDispositionValidAtHead(base,"def"){t.Fatal("HEAD-bound risk expiration failed")}
    base.Reason="short"
    if err:=ValidateSettlementItem(base,true);err==nil{t.Fatal("no substantive owner reason")}
}

func TestSettlementCannotDeferGitReviewOrContract(t *testing.T){
    for _,kind:=range []string{"automated_validation","review","git","contract"}{
        item:=SettlementItem{ID:"validation:v1",Kind:kind,FactStatus:"pending",Disposition:"deferred",Target:"defer",Reason:"future phase"}
        if err:=ValidateSettlementItem(item,false);err==nil{t.Fatalf("forbidden defer kind %q",kind)}
    }
    task:=SettlementItem{ID:"task:T02",Kind:"implementation",FactStatus:"pending",Disposition:"deferred",Target:"defer",Reason:"future milestone"}
    if err:=ValidateSettlementItem(task,false);err!=nil{t.Fatalf("legal future task: %v",err)}
}
func TestSettlementClearsOnDeferredResumeButPreservesEventHistory(t *testing.T){
    settle:=WorkflowEvent{Schema:WorkflowEventSchemaV2,EventType:EventSettle,EventID:"e1",Data:map[string]any{"item_id":"task:T02","kind":"implementation","fact_status":"pending","disposition":"deferred","target":"defer","reason":"later milestone","head":"abc","decided_by":"workflow","owner_confirmed":false}}
    resume:=WorkflowEvent{Schema:WorkflowEventSchemaV2,EventType:EventResume,StateBefore:StateDeferred,Data:map[string]any{}}
    projection,err:=ReplaySettlement([]WorkflowEvent{settle})
    if err!=nil||len(projection.Items)!=1{t.Fatalf("%#v %v",projection,err)}
    projection,err=ReplaySettlement([]WorkflowEvent{settle,resume})
    if err!=nil||len(projection.Items)!=0{t.Fatalf("previous segment leaked: %#v %v",projection,err)}
}

func TestSettlementIDsAreContractReferences(t *testing.T) {
    for _,id:=range []string{"task:T01","task:T123","validation:v1","acceptance:a12","external_wait:manual-check","implementation:item_12"} {
        item:=SettlementItem{ID:id,Kind:"implementation",FactStatus:"pending",Disposition:"deferred",Target:"defer",Reason:"continue in next milestone"}
        if err:=ValidateSettlementItem(item,false);err!=nil{t.Fatalf("canonical ID %q rejected: %v",id,err)}
    }
    for _,id:=range []string{"task:t02","task:T1","validation:V1","acceptance:A1","task:T02/../x"} {
        item:=SettlementItem{ID:id,Kind:"implementation",FactStatus:"pending",Disposition:"deferred",Target:"defer",Reason:"continue in next milestone"}
        if err:=ValidateSettlementItem(item,false);err==nil{t.Fatalf("noncanonical ID %q accepted",id)}
    }
}
