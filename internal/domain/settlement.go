package domain

import (
    "fmt"
    "regexp"
    "strings"
)

// Settlement is orthogonal to Issue lifecycle. FactStatus is never rewritten
// as pass to represent owner acceptance of residual manual risk.
type SettlementItem struct {
    ID string `json:"id"`
    Kind string `json:"kind"`
    FactStatus string `json:"fact_status"`
    Disposition string `json:"disposition"`
    Target string `json:"target"`
    Reason string `json:"reason,omitempty"`
    Head string `json:"head,omitempty"`
    DecidedBy string `json:"decided_by,omitempty"`
    EventID string `json:"event_id,omitempty"`
}

type SettlementProjection struct{
    Schema string `json:"schema"`
    Items []SettlementItem `json:"items"`
}

var settlementIDRE=regexp.MustCompile(`^[a-z][a-z0-9._:-]*$`)
var settlementKinds=map[string]bool{
    "implementation":true,"automated_validation":true,"manual_validation":true,
    "real_resource_validation":true,"review":true,"git":true,
    "contract":true,"external_wait":true,"acceptance":true,
}
var settlementFacts=map[string]bool{
    "pass":true,"fail":true,"pending":true,"not_run":true,
    "manual_required":true,"findings":true,"not_applicable":true,"done":true,
}
var settlementDispositions=map[string]bool{
    "none":true,"accepted_risk":true,"deferred":true,
    "kept_required":true,"not_applicable":true,"resolved":true,
}

func ValidateSettlementItem(item SettlementItem, ownerConfirmed bool)error{
    if !settlementIDRE.MatchString(item.ID){return fmt.Errorf("invalid settlement item ID %q",item.ID)}
    if !settlementKinds[item.Kind]{return fmt.Errorf("invalid settlement kind %q",item.Kind)}
    if !settlementFacts[item.FactStatus]{return fmt.Errorf("invalid settlement fact status %q",item.FactStatus)}
    if !settlementDispositions[item.Disposition]{return fmt.Errorf("invalid settlement disposition %q",item.Disposition)}
    if item.Target!="complete" && item.Target!="defer"{return fmt.Errorf("settlement target must be complete or defer")}
    if (item.FactStatus=="fail"||item.FactStatus=="findings") && (item.Disposition=="accepted_risk"||item.Disposition=="deferred"){return fmt.Errorf("confirmed failure/finding cannot be waived or deferred")}
    if item.Disposition=="accepted_risk" {
        if item.Kind!="manual_validation"&&item.Kind!="real_resource_validation"&&item.Kind!="acceptance" {
            return fmt.Errorf("accepted_risk is forbidden for %s",item.Kind)
        }
        if !ownerConfirmed || item.DecidedBy!="owner" {
            return fmt.Errorf("accepted_risk requires explicit owner confirmation")
        }
        if len(strings.TrimSpace(item.Reason))<20 {
            return fmt.Errorf("accepted_risk requires a substantive reason of at least 20 characters")
        }
        if item.Head=="" {return fmt.Errorf("accepted_risk requires exact current HEAD")}
        if item.FactStatus=="pass" || item.FactStatus=="done" {
            return fmt.Errorf("accepted risk is not a substitute for factual completion")
        }
    }
    if item.Disposition=="deferred"{
        if item.Target!="defer" {return fmt.Errorf("deferred item must target defer")}
        if item.Kind!="implementation"&&item.Kind!="external_wait"&&item.Kind!="manual_validation"&&item.Kind!="real_resource_validation"&&item.Kind!="acceptance"{
            return fmt.Errorf("cannot defer %s gate",item.Kind)
        }
        if strings.TrimSpace(item.Reason)=="" {return fmt.Errorf("deferred item requires an explicit reason")}
    }
    if item.Disposition=="resolved" && item.FactStatus!="pass" && item.FactStatus!="done"{
        return fmt.Errorf("resolved must have pass/done factual evidence")
    }
    if item.Disposition=="not_applicable" && !ownerConfirmed {return fmt.Errorf("not_applicable requires explicit owner confirmation")}
    if item.Disposition=="not_applicable" && item.Kind!="manual_validation"&&item.Kind!="real_resource_validation"&&item.Kind!="acceptance"{return fmt.Errorf("hard-required item kind cannot be declared not_applicable by settlement")}
    if item.Disposition=="not_applicable" && item.FactStatus!="not_applicable"{
        return fmt.Errorf("not_applicable disposition requires not_applicable fact")
    }
    if item.Disposition=="kept_required" && strings.TrimSpace(item.Reason)==""{
        return fmt.Errorf("kept_required requires reason")
    }
    return nil
}

func SettlementFromData(m map[string]any)(SettlementItem,error){
    fields:=[]string{"item_id","kind","fact_status","disposition","target"}
    values:=map[string]string{}
    for _,field:=range fields{
        v,ok:=m[field].(string)
        if !ok || strings.TrimSpace(v)==""{return SettlementItem{},fmt.Errorf("settlement missing %s",field)}
        values[field]=v
    }
    item:=SettlementItem{
        ID:values["item_id"],Kind:values["kind"],FactStatus:values["fact_status"],
        Disposition:values["disposition"],Target:values["target"],
        Reason:stringData(m,"reason"),Head:stringData(m,"head"),
        DecidedBy:stringData(m,"decided_by"),
    }
    confirmed,boolOK:=m["owner_confirmed"].(bool)
    if !boolOK && m["owner_confirmed"]!=nil{return SettlementItem{},fmt.Errorf("invalid owner_confirmed")}
    if err:=ValidateSettlementItem(item,confirmed);err!=nil{return SettlementItem{},err}
    return item,nil
}

func ReplaySettlement(events []WorkflowEvent)(SettlementProjection,error){
    items:=map[string]SettlementItem{}
    order:=[]string{}
    reset:=func(){items=map[string]SettlementItem{};order=[]string{}}
    for _,ev:=range events{
        if ev.Schema!=WorkflowEventSchemaV2 {continue}
        if ev.EventType==EventReopen||(ev.EventType==EventScope&&scopeAction(ev)=="accept")||(ev.EventType==EventResume&&ev.StateBefore==StateDeferred){reset()}
        if ev.EventType!=EventSettle{continue}
        item,err:=SettlementFromData(ev.Data);if err!=nil{return SettlementProjection{},NewWorkflowError("PROTOCOL_ERROR",err.Error())}
        item.EventID=ev.EventID
        if _,ok:=items[item.ID];!ok{order=append(order,item.ID)}
        if prev,ok:=items[item.ID];ok&&prev.Kind!=item.Kind {
            return SettlementProjection{},NewWorkflowError("PROTOCOL_ERROR","settlement kind mismatch for "+item.ID)
        }
        items[item.ID]=item
    }
    p:=SettlementProjection{Schema:"iw.settlement/v1",Items:[]SettlementItem{}}
    for _,id:=range order{p.Items=append(p.Items,items[id])}
    return p,nil
}

func SettlementFind(p SettlementProjection,id string)(SettlementItem,bool){
    for _,item:=range p.Items {if item.ID==id{return item,true}}
    return SettlementItem{},false
}
func SettlementDispositionValidAtHead(item SettlementItem,head string)bool{
    if item.Disposition=="accepted_risk" {return item.Head!=""&&item.Head==head}
    return true
}
