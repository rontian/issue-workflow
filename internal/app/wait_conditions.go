package app

import (
    "fmt"

    "github.com/rontian/issue-workflow/internal/domain"
)

// prepareWaitConditions is a hard boundary for all named waiting conditions.
// A simple --resolved flag may only be used with legacy single-reason waits.
func prepareWaitConditions(t domain.WorkflowEventType,rep WorkflowReport,o *MutationOptions)error {
    if t!=domain.EventWait && t!=domain.EventRecheck {
        if len(o.Conditions)>0{return fmt.Errorf("--condition is only allowed for wait/recheck")}
        return nil
    }
    if err:=domain.ValidateWaitConditions(o.Conditions);err!=nil{return err}
    if t==domain.EventWait {
        for _,item:=range o.Conditions {
            if item.Status!="waiting" {return fmt.Errorf("wait must create unresolved external condition %q",item.ID)}
        }
        return nil
    }
    previous:=rep.Aggregate.WaitingConditions
    if len(previous)==0 {
        if len(o.Conditions)>0{return fmt.Errorf("legacy wait has no named conditions; recheck without --condition")}
        return nil
    }
    if !domain.SameWaitConditionIDs(previous,o.Conditions) {
        return fmt.Errorf("recheck must report every recorded condition ID exactly once; cannot skip unresolved items")
    }
    fullyResolved:=domain.WaitConditionsResolved(o.Conditions)
    if o.Resolved && !fullyResolved {
        return fmt.Errorf("--resolved conflicts with unresolved named condition evidence")
    }
    o.Resolved=fullyResolved
    return nil
}
