package app

import (
	"context"
	"fmt"

	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
)

const RestorePlanSchema = "iw.restore-plan/v1"

type RestoreOptions struct { WorkflowOptions; Apply bool }
type RestoreReport struct {
	Schema string `json:"schema"`
	Apply bool `json:"apply"`
	Remote string `json:"remote"`
	Branch string `json:"branch"`
	Head string `json:"head"`
	CurrentBranch string `json:"current_branch"`
	CurrentHead string `json:"current_head"`
	RemoteVerified bool `json:"remote_verified"`
	Applied bool `json:"applied"`
	Context any `json:"context,omitempty"`
}

type RestoreService struct { Workflow *WorkflowService; Git ports.PortableGitPort; Context *ContextService }

func (s *RestoreService) Run(ctx context.Context,o RestoreOptions)(result.CommandResult,int){
	if s.Workflow==nil||s.Git==nil{return result.Failure("restore","ENVIRONMENT_INVALID","portable restore service unavailable",nil,nil),result.ExitEnvironment}
	wr,code:=s.Workflow.Status(ctx,o.WorkflowOptions);if !wr.OK{wr.Command="restore";wr.CanonicalCommand="restore";return wr,code}
	rep:=wr.Data.(WorkflowReport);p:=rep.Aggregate.PortableHandoff;if p==nil{return result.Failure("restore","TRANSITION_BLOCKED","no portable handoff exists for this task",nil,rep),result.ExitWorkflow}
	if rep.CurrentGit.Dirty{return result.Failure("restore","STALE_GIT_STATE","restore requires a clean worktree",nil,rep),result.ExitConflict}
	rr:=RestoreReport{Schema:RestorePlanSchema,Apply:o.Apply,Remote:p.Remote,Branch:p.Branch,Head:p.Head,CurrentBranch:rep.CurrentGit.Branch,CurrentHead:rep.CurrentGit.Head}
	remoteHead,err:=s.Git.RemoteBranchHead(ctx,rep.CurrentGit.RepositoryRoot,p.Remote,p.Branch);if err!=nil{return result.Failure("restore","REMOTE_NOT_FOUND",fmt.Sprintf("portable remote branch unavailable: %v",err),nil,rr),result.ExitEnvironment}
	if remoteHead!=p.Head{return result.Failure("restore","RESTORE_CONFLICT","portable remote branch no longer points at recorded HEAD",map[string]any{"expected":p.Head,"actual":remoteHead},rr),result.ExitConflict}
	rr.RemoteVerified=true
	if !o.Apply{r:=result.Success("restore",rr);r.Repository=wr.Repository;r.Issue=wr.Issue;r.State=wr.State;r.Lifecycle=wr.Lifecycle;r.NextActions=[]string{"re-run with --apply to fetch and restore the recorded branch/HEAD"};return r,result.ExitOK}
	if err:=s.Git.FetchRemote(ctx,rep.CurrentGit.RepositoryRoot,p.Remote);err!=nil{return result.Failure("restore","ENVIRONMENT_INVALID","git fetch failed",map[string]any{"error":err.Error()},rr),result.ExitEnvironment}
	if !s.Git.CommitExists(ctx,rep.CurrentGit.RepositoryRoot,p.Head){return result.Failure("restore","RESTORE_CONFLICT","recorded portable HEAD is not available after fetch",nil,rr),result.ExitConflict}
	if err:=s.Git.RestoreBranch(ctx,rep.CurrentGit.RepositoryRoot,p.Remote,p.Branch,p.Head);err!=nil{return result.Failure("restore","RESTORE_CONFLICT",err.Error(),nil,rr),result.ExitConflict}
	rr.Applied=true
	if s.Context!=nil{if cr,cc:=s.Context.Run(ctx,o.WorkflowOptions);cc==0&&cr.OK{rr.Context=cr.Data}}
	r:=result.Success("restore",rr);r.Repository=wr.Repository;r.Issue=wr.Issue;r.State=wr.State;r.Lifecycle=wr.Lifecycle;r.NextActions=[]string{"use iw context <issue> and continue from recovery.next_actions"};return r,result.ExitOK
}
