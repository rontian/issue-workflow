package domain

import (
    "fmt"
    "regexp"
    "strings"
)

// PlannedTask is a stable within-Issue task identity, unlike Host Todo items.
type PlannedTask struct {
    ID string `json:"id"`
    Title string `json:"title"`
}

type TaskProgress struct {
    Schema string `json:"schema"`
    Planned []PlannedTask `json:"planned"`
    Completed []string `json:"completed"`
    Current string `json:"current,omitempty"`
    Next string `json:"next,omitempty"`
    AllCompleted bool `json:"all_completed"`
}

var plannedTaskLineRE=regexp.MustCompile(`^[-*] \[[ xX]\] (T[0-9]{2,})[ :–—-]+(.+)$`)

// Implementation Tasks is an optional, versioned-in-the-body Task Contract
// section. No matching section means legacy v1 behavior, not a fabricated plan.
func PlannedTasks(c TaskContract)([]PlannedTask,error){
    out:=[]PlannedTask{}
    if strings.TrimSpace(c.ImplementationTasks)==""{return out,nil}
    seen:=map[string]bool{}
    for _,line:=range strings.Split(strings.ReplaceAll(c.ImplementationTasks,"\r\n","\n"),"\n"){
        line=strings.TrimSpace(line)
        if line==""{continue}
        match:=plannedTaskLineRE.FindStringSubmatch(line)
        if len(match)!=3 {return nil,NewWorkflowError("TASK_CONTRACT_INVALID",fmt.Sprintf("Implementation Tasks entry must be '- [ ] T01: Title': %q",line))}
        if seen[match[1]]{return nil,NewWorkflowError("TASK_CONTRACT_INVALID","duplicate stable task ID "+match[1])}
        seen[match[1]]=true
        out=append(out,PlannedTask{ID:match[1],Title:strings.TrimSpace(match[2])})
    }
    return out,nil
}

// ReplayTaskProgress validates TASK_COMPLETE event ordering against the actual
// Issue Body contract. A forged event cannot skip tasks or mark a random ID.
func ReplayTaskProgress(c TaskContract, events []WorkflowEvent)(TaskProgress,error){
    tasks,err:=PlannedTasks(c)
    if err!=nil{return TaskProgress{},err}
    p:=TaskProgress{Schema:"iw.task-progress/v1",Planned:tasks,Completed:[]string{},AllCompleted:len(tasks)==0}
    cursor:=0
    for _,event:=range events{
        if event.Schema!=WorkflowEventSchemaV2{continue}
        if event.EventType==EventReopen || (event.EventType==EventScope && scopeAction(event)=="accept") {
            cursor=0;p.Completed=[]string{}
        }
        if event.EventType!=EventTaskComplete {continue}
        id:=stringData(event.Data,"task_id")
        if cursor>=len(tasks)||tasks[cursor].ID!=id {
            return TaskProgress{},NewWorkflowError("PROTOCOL_ERROR",fmt.Sprintf("TASK_COMPLETE %q is not the current planned task",id))
        }
        cursor++
        p.Completed=append(p.Completed,id)
    }
    if cursor<len(tasks) {p.Current=tasks[cursor].ID}
    if cursor+1<len(tasks){p.Next=tasks[cursor+1].ID}
    p.AllCompleted=cursor==len(tasks)
    return p,nil
}
