package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
)

func (s *WorkflowService) NewIssue(ctx context.Context,o NewIssueOptions)(result.CommandResult,int){
	if strings.TrimSpace(o.Title)==""{return workflowFail("new","INVALID_INVOCATION","--title is required",nil,nil)}
	op:=strings.TrimSpace(o.OperationID);if op==""{var err error;op,err=domain.NewID();if err!=nil{return workflowFail("new","INTERNAL_ERROR","无法生成 operation_id",nil,nil)}}
	id:=strings.TrimSpace(o.ContractID);if id==""{id=op}
	body,err:=domain.RenderTaskContract(domain.TaskContractInput{ContractID:id,CreateOperationID:op,Goal:o.Goal,Scope:o.Scope,OutOfScope:o.OutOfScope,Constraints:o.Constraints,AcceptanceCriteria:o.Acceptance,Validation:o.Validation,Dependencies:o.Dependencies});if err!=nil{return workflowFail("new",workflowErrorCode(err,"INVALID_INVOCATION"),err.Error(),nil,nil)}
	repo,err:=s.repositoryForCommand(ctx,o.WorkflowOptions);if err!=nil{return workflowFail("new",workflowErrorCode(err,"ENVIRONMENT_INVALID"),err.Error(),nil,nil)}
	preview:=NewIssuePreview{Title:o.Title,Body:body,ContractID:id,OperationID:op,DryRun:o.DryRun}
	if o.DryRun{r:=result.Success("new",preview);r.Repository=repo.FullName();r.Lifecycle=string(domain.LifecycleReady);return r,result.ExitOK}
	lister,ok:=s.GitHub.(ports.TaskListGitHubPort);if !ok{return workflowFail("new","ENVIRONMENT_INVALID","GitHub adapter cannot perform idempotent task creation",nil,preview)}
	find:=func()(*domain.GitHubIssueDetails,bool,error){issues,e:=lister.TaskIssues(ctx,repo);if e!=nil{return nil,false,e};for _,iss:=range issues{c,pe:=domain.ParseTaskContract(iss.Body);if pe!=nil||c.CreateOperationID!=op{continue};if iss.Title!=o.Title||iss.Body!=body{return &iss,true,domain.NewWorkflowError("CONFLICT","create operation_id already exists with different task contract")};copy:=iss;return &copy,false,nil};return nil,false,nil}
	if existing,conflict,e:=find();e!=nil{return workflowFail("new",workflowErrorCode(e,"GITHUB_UNAVAILABLE"),e.Error(),map[string]any{"operation_id":op},preview)}else if conflict{return workflowFail("new","CONFLICT","create operation_id already exists with different task contract",map[string]any{"operation_id":op,"issue":existing.Number},preview)}else if existing!=nil{return createdIssueResult(repo,*existing,preview,"IDEMPOTENT_REPLAY")}
	issue,err:=s.GitHub.CreateIssue(ctx,repo,o.Title,body);if err!=nil{
		existing,conflict,e2:=find();if e2==nil&&existing!=nil&&!conflict{return createdIssueResult(repo,*existing,preview,"UNKNOWN_OUTCOME_RECOVERED")};if conflict{return workflowFail("new","CONFLICT","create operation_id was persisted with different task contract",map[string]any{"operation_id":op},preview)}
		r:=result.Failure("new","GITHUB_UNAVAILABLE","Issue create outcome unknown; retry the same logical create with the same --operation-id",map[string]any{"contract_id":id,"operation_id":op,"outcome_unknown":true},preview);r.Repository=repo.FullName();return r,result.ExitGitHub
	}
	return createdIssueResult(repo,issue,preview,"")
}

func createdIssueResult(repo domain.RepositoryIdentity,issue domain.GitHubIssueDetails,preview NewIssuePreview,warning string)(result.CommandResult,int){preview.Issue=&issue;r:=result.Success("new",preview);r.Repository=repo.FullName();n:=issue.Number;r.Issue=&n;r.State=string(domain.StateReady);r.Lifecycle=string(domain.LifecycleReady);r.NextActions=[]string{fmt.Sprintf("iw start %d",issue.Number)};if warning!=""{r.Warnings=append(r.Warnings,warning)};return r,result.ExitOK}