package app

import (
    "context"
    "strings"

    "github.com/rontian/issue-workflow/internal/domain"
    "github.com/rontian/issue-workflow/internal/ports"
    "github.com/rontian/issue-workflow/internal/result"
)

const NativeIssueTransitionSchema="iw.github-issue-transition/v1"

type NativeIssueReport struct{
    Schema string `json:"schema"`
    Repository string `json:"repository"`
    Issue int `json:"issue"`
    Lifecycle domain.Lifecycle `json:"lifecycle"`
    DesiredState string `json:"desired_state"`
    PreviousState string `json:"previous_state"`
    DryRun bool `json:"dry_run"`
    Applied bool `json:"applied"`
    Idempotent bool `json:"idempotent"`
}

// NativeIssueState is an explicit, independent GitHub Issue state operation.
// A completed/cancelled workflow event does NOT secretly close the Issue.
func(s *WorkflowService)NativeIssueState(ctx context.Context,o WorkflowOptions,close,dry bool)(result.CommandResult,int){
    command:="issue reopen";desired:="OPEN"
    if close{command="issue close";desired="CLOSED"}
    rr,code:=s.Status(ctx,o);if !rr.OK{rr.Command=command;rr.CanonicalCommand=command;return rr,code}
    rep:=rr.Data.(WorkflowReport)
    if rep.ContractChanged||len(rep.Aggregate.UnresolvedScopeProposals)>0 {
        return mutationFailFrom(command,"CONTRACT_CHANGED","unresolved contract or scope change prevents native Issue transition",rep,rr)
    }
    if rep.GitDrift.Changed||rep.CurrentGit.Dirty{
        return mutationFailFrom(command,"STALE_GIT_STATE","native Issue transition requires clean current Git state",rep,rr)
    }
    if close {
        if rep.Lifecycle!=domain.LifecycleCompleted&&rep.Lifecycle!=domain.LifecycleCancelled {
            return mutationFailFrom(command,"TRANSITION_BLOCKED","GitHub Issue close requires COMPLETED or CANCELLED workflow",rep,rr)
        }
    }else{
        if rep.Lifecycle==domain.LifecycleCompleted||rep.Lifecycle==domain.LifecycleCancelled{
            return mutationFailFrom(command,"TRANSITION_BLOCKED","first run iw reopen to establish an active workflow segment",rep,rr)
        }
    }
    report:=NativeIssueReport{Schema:NativeIssueTransitionSchema,Repository:rep.Repository.FullName(),Issue:o.Issue,Lifecycle:rep.Lifecycle,DesiredState:desired,PreviousState:strings.ToUpper(rep.Issue.State),DryRun:dry}
    if strings.EqualFold(rep.Issue.State,desired){
        report.Idempotent=true
        r:=result.Success(command,report);copyWorkflowContext(&r,rr);return r,result.ExitOK
    }
    if dry{
        r:=result.Success(command,report);copyWorkflowContext(&r,rr);return r,result.ExitOK
    }
    p,ok:=s.GitHub.(ports.GitHubIssueStatePort)
    if !ok{return mutationFailFrom(command,"ENVIRONMENT_INVALID","GitHub adapter cannot change native Issue state",rep,rr)}
    var err error
    if close{
        reason:="completed";if rep.Lifecycle==domain.LifecycleCancelled{reason="not planned"}
        err=p.CloseIssue(ctx,rep.Repository,o.Issue,reason)
    }else{err=p.ReopenIssue(ctx,rep.Repository,o.Issue)}
    // Independently re-read after success *or* unknown outcome.
    state,e:=s.GitHub.IssueDetails(ctx,rep.Repository,o.Issue)
    if e==nil&&strings.EqualFold(state.State,desired){
        report.Applied=true
        if err!=nil{report.Idempotent=true}
        r:=result.Success(command,report);copyWorkflowContext(&r,rr)
        if err!=nil{r.Warnings=append(r.Warnings,"UNKNOWN_OUTCOME_RECOVERED")}
        return r,result.ExitOK
    }
    return workflowFail(command,"GITHUB_UNAVAILABLE","native GitHub Issue state not confirmed; retry this idempotent action after inspecting Issue state",map[string]any{"desired_state":desired,"issue":o.Issue},report)
}
