package app

import (
	"context"
	"sort"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

const ContextSchema = "iw.context/v1"

type ContextValidation struct {
	Requirements []domain.ValidationRequirement `json:"requirements"`
	Evidence     []domain.ValidationRecord      `json:"evidence"`
	Missing      []string                       `json:"missing"`
}

type AgentContext struct {
	Schema      string                    `json:"schema"`
	Project     ProjectReport             `json:"project"`
	Repository domain.RepositoryIdentity  `json:"repository"`
	Issue       domain.GitHubIssueDetails `json:"issue"`
	Task        domain.TaskContract       `json:"task_contract"`
	Lifecycle   domain.Lifecycle          `json:"lifecycle"`
	Phase       string                    `json:"phase,omitempty"`
	Protocol    string                    `json:"protocol_source"`
	Workflow    domain.WorkflowAggregate  `json:"workflow"`
	Git         domain.GitSnapshot        `json:"git"`
	GitDrift    domain.GitDrift           `json:"git_drift"`
	Validation  ContextValidation         `json:"validation"`
	Recovery    RecoveryContext           `json:"recovery"`
	Constraints []string                  `json:"constraints"`
	NextActions []string                  `json:"next_actions"`
}

type ContextService struct { Workflow *WorkflowService; Project *ProjectService }

func (s *ContextService) Run(ctx context.Context, o WorkflowOptions) (result.CommandResult, int) {
	if s.Workflow == nil { return result.Failure("context","ENVIRONMENT_INVALID","workflow service unavailable",nil,nil),result.ExitEnvironment }
	wr,code:=s.Workflow.Status(ctx,o); if !wr.OK { wr.Command="context";wr.CanonicalCommand="context";return wr,code }
	rep:=wr.Data.(WorkflowReport)
	project:=ProjectReport{Schema:ProjectSchema,Config:ProjectConfig{Schema:ProjectSchema},Docs:[]ProjectDocStatus{},Environment:[]ProjectExecutableStatus{}}
	if s.Project!=nil { if pr,err:=s.Project.Read(ctx,o.CWD);err==nil { project=pr } else { wr.Warnings=append(wr.Warnings,"PROJECT_CONFIG_INVALID");wr.Constraints=append(wr.Constraints,err.Error()) } }
	req:=domain.ValidationRequirements(rep.Contract); records:=domain.ValidationRecordsFromEvents(rep.Aggregate.Events); ids:=make([]string,0,len(records));for id:=range records{ids=append(ids,id)};sort.Strings(ids);evidence:=make([]domain.ValidationRecord,0,len(ids));for _,id:=range ids{evidence=append(evidence,records[id])}
	data:=AgentContext{Schema:ContextSchema,Project:project,Repository:rep.Repository,Issue:rep.Issue,Task:rep.Contract,Lifecycle:rep.Lifecycle,Phase:rep.Phase,Protocol:rep.ProtocolSource,Workflow:rep.Aggregate,Git:rep.CurrentGit,GitDrift:rep.GitDrift,Validation:ContextValidation{Requirements:req,Evidence:evidence,Missing:domain.MissingValidationsAtHead(req,records,rep.CurrentGit.Head)},Recovery:rep.Recovery,Constraints:append([]string(nil),wr.Constraints...),NextActions:append([]string(nil),wr.NextActions...)}
	r:=result.Success("context",data);r.Repository,r.Issue,r.State,r.Lifecycle=wr.Repository,wr.Issue,wr.State,string(rep.Lifecycle);r.Warnings=append([]string(nil),wr.Warnings...);r.Constraints=append([]string(nil),wr.Constraints...);r.NextActions=append([]string(nil),wr.NextActions...);return r,result.ExitOK
}
