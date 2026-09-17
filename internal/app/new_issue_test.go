package app

import(
	"context"
	"errors"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
)

type newIssueGH struct{repo domain.GitHubRepository;issues []domain.GitHubIssueDetails;createCalls int;failAfterCreate bool}
func(g *newIssueGH)Authenticated(context.Context,string)error{return nil}
func(g *newIssueGH)Repository(context.Context,domain.RepositoryIdentity)(domain.GitHubRepository,error){return g.repo,nil}
func(g *newIssueGH)IssueDetails(context.Context,domain.RepositoryIdentity,int)(domain.GitHubIssueDetails,error){return domain.GitHubIssueDetails{},nil}
func(g *newIssueGH)IssueComments(context.Context,domain.RepositoryIdentity,int)([]domain.GitHubComment,error){return nil,nil}
func(g *newIssueGH)AppendIssueComment(context.Context,domain.RepositoryIdentity,int,string)error{return nil}
func(g *newIssueGH)TaskIssues(context.Context,domain.RepositoryIdentity)([]domain.GitHubIssueDetails,error){return append([]domain.GitHubIssueDetails(nil),g.issues...),nil}
func(g *newIssueGH)CreateIssue(_ context.Context,_ domain.RepositoryIdentity,title,body string)(domain.GitHubIssueDetails,error){g.createCalls++;iss:=domain.GitHubIssueDetails{Number:100+g.createCalls,State:"OPEN",Title:title,Body:body,URL:"https://example/issues/x"};g.issues=append(g.issues,iss);if g.failAfterCreate{g.failAfterCreate=false;return domain.GitHubIssueDetails{},errors.New("connection reset")};return iss,nil}

func newIssueService()(*WorkflowService,*newIssueGH){repo:=domain.RepositoryIdentity{Host:"github.com",Owner:"o",Repo:"r"};git:=&mutableGit{repo:repo,snap:domain.GitSnapshot{RepositoryRoot:"/repo",Branch:"main",Head:"abc",ChangedPaths:[]string{}},remoteHead:"abc"};gh:=&newIssueGH{repo:domain.GitHubRepository{NameWithOwner:"o/r",HasIssuesEnabled:true}};return &WorkflowService{Git:git,GitHub:gh},gh}
func newOptions(op,title string)NewIssueOptions{return NewIssueOptions{WorkflowOptions:WorkflowOptions{CWD:"/repo"},OperationID:op,Title:title,Goal:"goal",Scope:[]string{"scope"},OutOfScope:[]string{"none"},Acceptance:[]string{"done"}}}

func TestNewIssueOperationIDIdempotent(t *testing.T){s,g:=newIssueService();r,c:=s.NewIssue(context.Background(),newOptions("create-1","Task"));if c!=0||!r.OK{t.Fatalf("first %d %#v",c,r)};first:=*r.Issue;r2,c2:=s.NewIssue(context.Background(),newOptions("create-1","Task"));if c2!=0||!r2.OK||r2.Issue==nil||*r2.Issue!=first{t.Fatalf("retry %d %#v",c2,r2)};if g.createCalls!=1{t.Fatalf("createCalls=%d",g.createCalls)};p:=r2.Data.(NewIssuePreview);contract,err:=domain.ParseTaskContract(p.Body);if err!=nil||contract.CreateOperationID!="create-1"||contract.ContractID!="create-1"{t.Fatalf("contract=%#v err=%v",contract,err)}}
func TestNewIssueUnknownOutcomeRecovered(t *testing.T){s,g:=newIssueService();g.failAfterCreate=true;r,c:=s.NewIssue(context.Background(),newOptions("create-recover","Task"));if c!=0||!r.OK||r.Issue==nil{t.Fatalf("%d %#v",c,r)};if g.createCalls!=1{t.Fatalf("createCalls=%d",g.createCalls)};found:=false;for _,w:=range r.Warnings{if w=="UNKNOWN_OUTCOME_RECOVERED"{found=true}};if !found{t.Fatalf("warnings=%v",r.Warnings)}}
func TestNewIssueOperationConflict(t *testing.T){s,_:=newIssueService();if r,c:=s.NewIssue(context.Background(),newOptions("create-conflict","Task A"));c!=0||!r.OK{t.Fatal(c,r)};r,c:=s.NewIssue(context.Background(),newOptions("create-conflict","Task B"));if c!=7||r.Error==nil||r.Error.Code!="CONFLICT"{t.Fatalf("%d %#v",c,r)}}
