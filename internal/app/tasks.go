package app

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
)

const TasksSchema="iw.tasks/v1"
var issueRefRE=regexp.MustCompile(`#([0-9]+)`) 

type TaskSummary struct{
	Issue int `json:"issue"`
	Title string `json:"title"`
	IssueState string `json:"issue_state"`
	Lifecycle domain.Lifecycle `json:"lifecycle"`
	Phase string `json:"phase,omitempty"`
	Protocol string `json:"protocol"`
	Goal string `json:"goal,omitempty"`
	Dependencies []int `json:"dependencies"`
	DependenciesSatisfied bool `json:"dependencies_satisfied"`
	Ready bool `json:"ready"`
	Health []string `json:"health"`
}
type TasksReport struct{Schema string `json:"schema"`;Repository string `json:"repository"`;Tasks []TaskSummary `json:"tasks"`}
type NextReport struct{Schema string `json:"schema"`;Repository string `json:"repository"`;Task *TaskSummary `json:"task,omitempty"`;Reason string `json:"reason"`}

type TaskService struct{Workflow *WorkflowService}

func(s *TaskService)Tasks(ctx context.Context,o WorkflowOptions)(result.CommandResult,int){
	rep,code:=s.read(ctx,o);if code!=result.ExitOK{return result.Failure("tasks","GITHUB_UNAVAILABLE","cannot build task read model",nil,nil),code}
	r:=result.Success("tasks",rep);r.Repository=rep.Repository;return r,result.ExitOK
}
func(s *TaskService)Next(ctx context.Context,o WorkflowOptions)(result.CommandResult,int){
	rep,code:=s.read(ctx,o);if code!=result.ExitOK{return result.Failure("next","GITHUB_UNAVAILABLE","cannot build task read model",nil,nil),code}
	var selected *TaskSummary;reason:="no actionable task"
	for i:=range rep.Tasks{t:=&rep.Tasks[i];if t.Lifecycle==domain.LifecycleInProgress||t.Lifecycle==domain.LifecyclePaused||t.Lifecycle==domain.LifecycleWaiting||t.Lifecycle==domain.LifecycleBlocked||t.Lifecycle==domain.LifecycleDeferred{c:=*t;selected=&c;reason="existing non-terminal task";break}}
	if selected==nil{for i:=range rep.Tasks{if rep.Tasks[i].Ready{c:=rep.Tasks[i];selected=&c;reason="READY with satisfied dependencies";break}}}
	n:=NextReport{Schema:"iw.next/v1",Repository:rep.Repository,Task:selected,Reason:reason};r:=result.Success("next",n);r.Repository=rep.Repository;if selected!=nil{r.Lifecycle=string(selected.Lifecycle);r.NextActions=[]string{fmt.Sprintf("iw context %d",selected.Issue)}};return r,result.ExitOK
}
func(s *TaskService)read(ctx context.Context,o WorkflowOptions)(TasksReport,int){
	if s.Workflow==nil{return TasksReport{},result.ExitEnvironment}
	root,err:=s.Workflow.Git.RepositoryRoot(ctx,o.CWD);if err!=nil{return TasksReport{},result.ExitEnvironment}
	repo,err:=s.Workflow.resolveRepository(ctx,root,o);if err!=nil{return TasksReport{},result.ExitEnvironment}
	if err:=s.Workflow.GitHub.Authenticated(ctx,repo.Host);err!=nil{return TasksReport{},result.ExitGitHub}
	lister,ok:=s.Workflow.GitHub.(ports.TaskListGitHubPort);if !ok{return TasksReport{},result.ExitEnvironment}
	issues,err:=lister.TaskIssues(ctx,repo);if err!=nil{return TasksReport{},result.ExitGitHub}
	out:=TasksReport{Schema:TasksSchema,Repository:repo.FullName(),Tasks:[]TaskSummary{}}
	for _,iss:=range issues{
		t:=TaskSummary{Issue:iss.Number,Title:iss.Title,IssueState:iss.State,Dependencies:[]int{},Health:[]string{}}
		contract,e:=domain.ParseTaskContract(iss.Body);if e!=nil{t.Lifecycle=domain.LifecycleNeedsAnalysis;t.Health=append(t.Health,"TASK_CONTRACT_INVALID");out.Tasks=append(out.Tasks,t);continue};t.Goal=contract.Goal;t.Dependencies=parseIssueDependencies(contract.DependenciesReferences)
		comments,e:=s.Workflow.GitHub.IssueComments(ctx,repo,iss.Number);if e!=nil{t.Lifecycle=domain.LifecycleNeedsAnalysis;t.Health=append(t.Health,"COMMENTS_UNAVAILABLE");out.Tasks=append(out.Tasks,t);continue}
		events:=[]domain.WorkflowEvent{};bad:=false;for _,c:=range comments{ev,pe:=domain.ParseWorkflowEventComment(c);if pe!=nil{bad=true;break};if ev!=nil{events=append(events,*ev)}};if bad{t.Lifecycle=domain.LifecycleNeedsAnalysis;t.Health=append(t.Health,"PROTOCOL_ERROR");out.Tasks=append(out.Tasks,t);continue}
		agg,e:=domain.ReplayWorkflow(repo.FullName(),iss.Number,events);if e!=nil{t.Lifecycle=domain.LifecycleNeedsAnalysis;t.Health=append(t.Health,domain.WorkflowErrorCode(e));out.Tasks=append(out.Tasks,t);continue};t.Lifecycle,t.Phase,t.Protocol=agg.Lifecycle,agg.Phase,agg.Protocol;out.Tasks=append(out.Tasks,t)
	}
	done:=map[int]bool{};for _,t:=range out.Tasks{if t.Lifecycle==domain.LifecycleCompleted{done[t.Issue]=true}}
	for i:=range out.Tasks{satisfied:=true;for _,d:=range out.Tasks[i].Dependencies{if !done[d]{satisfied=false;break}};out.Tasks[i].DependenciesSatisfied=satisfied;out.Tasks[i].Ready=out.Tasks[i].Lifecycle==domain.LifecycleReady&&satisfied&&len(out.Tasks[i].Health)==0}
	sort.Slice(out.Tasks,func(i,j int)bool{return out.Tasks[i].Issue<out.Tasks[j].Issue});return out,result.ExitOK
}
func parseIssueDependencies(s string)[]int{seen:=map[int]bool{};out:=[]int{};for _,m:=range issueRefRE.FindAllStringSubmatch(s,-1){n,_:=strconv.Atoi(m[1]);if n>0&&!seen[n]{seen[n]=true;out=append(out,n)}};sort.Ints(out);return out}
