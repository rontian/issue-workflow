package domain

import (
    "fmt"
    "strings"
)

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
        id:=fmt.Sprintf("a%d",len(out)+1)
        low:=strings.ToLower(line)
        out=append(out,AcceptanceRequirement{ID:id,Text:line,OwnerDecidable:strings.HasPrefix(low,"owner:")})
    }
    return out
}
