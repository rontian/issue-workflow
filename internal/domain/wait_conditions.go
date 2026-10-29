package domain

import (
    "fmt"
    "regexp"
    "strings"
)

// WaitCondition represents one named, externally checkable condition. It is
// canonical workflow evidence, unlike host-specific Goal/Todo runtime state.
type WaitCondition struct {
    ID string `json:"id"`
    Status string `json:"status"` // waiting | resolved
    Summary string `json:"summary,omitempty"`
}

var waitConditionIDRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

// ParseWaitConditionSpec accepts --condition ID=waiting[:note] or
// ID=resolved:actual-evidence. A resolved condition must carry evidence.
func ParseWaitConditionSpec(spec string) (WaitCondition,error) {
    parts:=strings.SplitN(strings.TrimSpace(spec),"=",2)
    if len(parts)!=2 {return WaitCondition{},fmt.Errorf("condition must be ID=waiting[:note] or ID=resolved:evidence")}
    state:=strings.SplitN(strings.TrimSpace(parts[1]),":",2)
    c:=WaitCondition{ID:strings.TrimSpace(parts[0]),Status:strings.ToLower(strings.TrimSpace(state[0]))}
    if len(state)==2 {c.Summary=strings.TrimSpace(state[1])}
    if err:=ValidateWaitConditions([]WaitCondition{c});err!=nil{return WaitCondition{},err}
    return c,nil
}

func ValidateWaitConditions(items []WaitCondition) error {
    ids:=map[string]bool{}
    for _,c:=range items{
        if !waitConditionIDRE.MatchString(c.ID) {
            return fmt.Errorf("invalid wait condition id %q",c.ID)
        }
        if ids[c.ID] {
            return fmt.Errorf("duplicate wait condition id %q",c.ID)
        }
        ids[c.ID]=true
        if c.Status!="waiting" && c.Status!="resolved" {
            return fmt.Errorf("invalid wait condition status %q",c.Status)
        }
        if c.Status=="resolved" && strings.TrimSpace(c.Summary)=="" {
            return fmt.Errorf("resolved wait condition %q requires evidence summary",c.ID)
        }
    }
    return nil
}

// WaitConditionsFromData reads event JSON without trusting unknown shapes.
func WaitConditionsFromData(data map[string]any) ([]WaitCondition,error) {
    raw,ok:=data["conditions"];if !ok{return nil,nil}
    arr,ok:=raw.([]any);if !ok{return nil,fmt.Errorf("data.conditions must be an array")}
    out:=make([]WaitCondition,0,len(arr))
    for _,item:=range arr{
        m,ok:=item.(map[string]any);if !ok{return nil,fmt.Errorf("data.conditions entry must be an object")}
        id,iok:=m["id"].(string);st,sok:=m["status"].(string)
        if !iok||!sok{return nil,fmt.Errorf("data.conditions entry requires id and status")}
        summary:=""
        if v,exists:=m["summary"];exists{var valid bool;summary,valid=v.(string);if !valid{return nil,fmt.Errorf("data.conditions summary must be string")}}
        out=append(out,WaitCondition{ID:id,Status:st,Summary:summary})
    }
    if err:=ValidateWaitConditions(out);err!=nil{return nil,err}
    return out,nil
}

func WaitConditionsResolved(items []WaitCondition) bool {
    if len(items)==0{return false}
    for _,x:=range items {if x.Status!="resolved"{return false}}
    return true
}

func SameWaitConditionIDs(before,after []WaitCondition) bool {
    if len(before)!=len(after){return false}
    ids:=make(map[string]bool,len(before));for _,c:=range before{ids[c.ID]=true}
    for _,c:=range after{if !ids[c.ID]{return false}}
    return true
}
