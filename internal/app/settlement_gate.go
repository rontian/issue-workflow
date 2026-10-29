package app

import (
    "fmt"
    "strings"

    "github.com/rontian/issue-workflow/internal/domain"
)

// deferSettlementGate prevents a future milestone label from silently erasing
// unfinished planned implementation or active hard-required settlement items.
func deferSettlementGate(rep WorkflowReport)error{
    completed:=map[string]bool{}
    for _,id:=range rep.TaskProgress.Completed{completed[id]=true}
    for _,task:=range rep.TaskProgress.Planned{
        if completed[task.ID]{continue}
        item,ok:=domain.SettlementFind(rep.Settlement,"task:"+task.ID)
        if !ok || item.Kind!="implementation" || item.Target!="defer" || item.Disposition!="deferred"{
            return domain.NewWorkflowError("TRANSITION_BLOCKED",fmt.Sprintf("uncompleted %s requires explicit deferred implementation settlement",task.ID))
        }
    }
    for _,item:=range rep.Settlement.Items {
        if item.Target!="defer"{continue}
        switch item.Disposition{
        case "resolved","not_applicable","deferred":
        case "accepted_risk":
            if !domain.SettlementDispositionValidAtHead(item,rep.CurrentGit.Head) {
                return domain.NewWorkflowError("TRANSITION_BLOCKED","stale HEAD-bound accepted risk for "+item.ID)
            }
        default:
            return domain.NewWorkflowError("TRANSITION_BLOCKED","unsettled current item "+item.ID)
        }
    }
    return nil
}

// Validating a settlement reference against the contract is essential:
// otherwise an Agent could invent 'manual_validation' for an automated gate.
func validateSettlementReference(rep WorkflowReport,item domain.SettlementItem)error{
    if strings.HasPrefix(item.ID,"task:"){
        id:=strings.TrimPrefix(item.ID,"task:")
        found:=false
        for _,task:=range rep.TaskProgress.Planned {if task.ID==id {found=true;break}}
        if !found||item.Kind!="implementation"{return domain.NewWorkflowError("INVALID_INVOCATION","settlement task ID/kind not in Task Contract")}
        return nil
    }
    if strings.HasPrefix(item.ID,"validation:"){
        id:=strings.TrimPrefix(item.ID,"validation:")
        found:=false
        for _,req:=range domain.ValidationRequirements(rep.Contract) {
            if req.ID!=id{continue}
            found=true
            if item.Kind!=req.Kind{return domain.NewWorkflowError("INVALID_INVOCATION","settlement validation kind must match contract-declared requirement")}
            break
        }
        if !found{return domain.NewWorkflowError("INVALID_INVOCATION","unknown validation ID in settlement")}
        return nil
    }
    if strings.HasPrefix(item.ID,"acceptance:"){
        id:=strings.TrimPrefix(item.ID,"acceptance:")
        for _,req:=range domain.AcceptanceRequirements(rep.Contract){
            if req.ID==id {
                if item.Kind!="acceptance"{return domain.NewWorkflowError("INVALID_INVOCATION","acceptance settlement kind mismatch")}
                if item.Disposition=="accepted_risk"&&!req.OwnerDecidable{return domain.NewWorkflowError("INVALID_INVOCATION","acceptance item is not owner-decidable")}
                return nil
            }
        }
        return domain.NewWorkflowError("INVALID_INVOCATION","unknown acceptance ID in settlement")
    }
    return domain.NewWorkflowError("INVALID_INVOCATION","settlement item ID must reference task:, validation:, or acceptance: contract fact")
}

func validationRiskSettled(rep WorkflowReport,req domain.ValidationRequirement,record domain.ValidationRecord)bool {
    if req.Kind!="manual_validation"&&req.Kind!="real_resource_validation"{return false}
    if record.Status!="manual_required" || record.ObservedHead!=rep.CurrentGit.Head{return false}
    item,ok:=domain.SettlementFind(rep.Settlement,"validation:"+req.ID)
    if !ok || item.Target!="complete" || item.Kind!=req.Kind || item.FactStatus!="manual_required" || item.Disposition!="accepted_risk"{return false}
    return domain.SettlementDispositionValidAtHead(item,rep.CurrentGit.Head)
}

func completeSettlementGate(rep WorkflowReport)error{
    for _,item:=range rep.Settlement.Items{
        if item.Target!="complete"{continue}
        switch item.Disposition{
        case "resolved","not_applicable":
        case "accepted_risk":
            if !domain.SettlementDispositionValidAtHead(item,rep.CurrentGit.Head){
                return domain.NewWorkflowError("VALIDATION_FAILED","accepted risk expired after HEAD changed: "+item.ID)
            }
        default:
            return domain.NewWorkflowError("TRANSITION_BLOCKED","unsettled completion item "+item.ID)
        }
    }
    return nil
}
