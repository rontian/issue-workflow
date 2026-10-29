package app

import (
    "context"
    "fmt"
    "strings"

    "github.com/rontian/issue-workflow/internal/domain"
)

// deferGate verifies only facts the Core can prove. Future-scope and per-item
// settlement decisions are separate protocol work: a --reason is not evidence
// that unfinished implementation or manual acceptance has been settled.
func deferGate(ctx context.Context, s *WorkflowService, rep WorkflowReport, o MutationOptions) error {
    if rep.ContractChanged {
        return domain.NewWorkflowError("CONTRACT_CHANGED", "contract changed")
    }
    if len(rep.Aggregate.UnresolvedScopeProposals)>0 {
        return domain.NewWorkflowError("SCOPE_UNRESOLVED", "defer cannot release unresolved scope proposals")
    }
    if rep.CurrentGit.Dirty || rep.GitDrift.Changed || rep.CurrentGit.Detached || strings.TrimSpace(rep.CurrentGit.Branch)=="" {
        return domain.NewWorkflowError("STALE_GIT_STATE", "defer requires clean, attached, checkpointed Git HEAD")
    }
    requireReview,err:=projectRequireReview(rep.CurrentGit.RepositoryRoot)
    if err!=nil {
        return domain.NewWorkflowError("PROJECT_CONFIG_INVALID",err.Error())
    }
    review:=rep.Aggregate.ReviewStatus
    if review=="findings" || review=="pending" ||
        (review=="passed" && rep.Aggregate.ReviewHead!=rep.CurrentGit.Head) ||
        (requireReview && (review!="passed" || rep.Aggregate.ReviewHead!=rep.CurrentGit.Head)) {
        return domain.NewWorkflowError("REVIEW_REQUIRED", "defer requires no unresolved/stale review and a current-HEAD review when policy requires it")
    }
    requirements:=domain.ValidationRequirements(rep.Contract)
    records:=domain.ValidationRecordsFromEvents(rep.Aggregate.Events)
    missing:=[]string{}
    for _,requirement:=range requirements {
        rec:=records[requirement.ID]
        if len(domain.MissingValidationsAtHead([]domain.ValidationRequirement{requirement},records,rep.CurrentGit.Head))==0{continue}
        if requirement.Kind=="manual_validation"||requirement.Kind=="real_resource_validation"{
            item,ok:=domain.SettlementFind(rep.Settlement,"validation:"+requirement.ID)
            if ok && item.Kind==requirement.Kind && item.Target=="defer" && (item.Disposition=="deferred" ||
                (item.Disposition=="accepted_risk"&&item.FactStatus=="manual_required"&&rec.Status=="manual_required"&&rec.ObservedHead==rep.CurrentGit.Head&&domain.SettlementDispositionValidAtHead(item,rep.CurrentGit.Head))){
                continue
            }
        }
        missing=append(missing,requirement.ID)
    }
    if len(missing)>0{return domain.NewWorkflowError("VALIDATION_FAILED","defer requires passing automated validation and settled manual requirements: "+strings.Join(missing,","))}
    // Unlike a human --pushed claim, this checks exact published Git state.
    _,remote,err:=s.Git.ResolveRemote(ctx,rep.CurrentGit.RepositoryRoot,o.Remote)
    if err!=nil {
        return domain.NewWorkflowError("REMOTE_NOT_FOUND",fmt.Sprintf("cannot resolve Git remote for defer: %v",err))
    }
    checker,ok:=s.Git.(interface{RemoteBranchHead(context.Context,string,string,string)(string,error)})
    if !ok {
        return domain.NewWorkflowError("ENVIRONMENT_INVALID","Git adapter cannot verify published HEAD")
    }
    remoteHead,err:=checker.RemoteBranchHead(ctx,rep.CurrentGit.RepositoryRoot,remote,rep.CurrentGit.Branch)
    if err!=nil || remoteHead!=rep.CurrentGit.Head {
        return domain.NewWorkflowError("STALE_GIT_STATE","defer requires exact current HEAD published to the configured remote branch")
    }
    return nil
}
