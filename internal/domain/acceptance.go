package domain

import "strings"

type AcceptanceRequirement struct {
    ID string `json:"id"`
    Text string `json:"text"`
    OwnerDecidable bool `json:"owner_decidable"`
}

// Owner risk is only possible when an acceptance entry explicitly declares
// "owner:" rather than relying on the agent to classify hard requirements.
func AcceptanceRequirements(c TaskContract)[]AcceptanceRequirement{
    out:=[]AcceptanceRequirement{}
    for _,line:=range strings.Split(strings.ReplaceAll(c.AcceptanceCriteria,"\r\n","\n"),"\n"){
        line=strings.TrimSpace(line)
        if line==""{continue}
        line=strings.TrimSpace(strings.TrimPrefix(line,"- [ ]"))
        line=strings.TrimSpace(strings.TrimPrefix(line,"- [x]"))
        line=strings.TrimSpace(strings.TrimPrefix(line,"-"))
        if line==""{continue}
        n:=len(out)+1
        id:="a"+itoa(n)
        low:=strings.ToLower(line)
        out=append(out,AcceptanceRequirement{ID:id,Text:line,OwnerDecidable:strings.HasPrefix(low,"owner:")})
    }
    return out
}
func itoa(n int)string{
    // Requirement indexes are deterministic and only used as legacy v1
    // contract-local identifiers, never as event IDs.
    if n<10{return string(rune('0'+n))}
    digits:=""
    for n>0{digits=string(rune('0'+n%10))+digits;n/=10}
    return digits
}
