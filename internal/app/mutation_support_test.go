package app

import (
	"context"
	"errors"
	"testing"

	"github.com/rontian/issue-workflow/internal/domain"
)

type stateGH struct{repo domain.GitHubRepository;issue domain.GitHubIssueDetails;comments []domain.GitHubComment;appendCalls int;failAfterAppend bool}
func(g *stateGH)Authenticated(context.Context,string)error{return nil};func(g *stateGH)Repository(context.Context,domain.RepositoryIdentity)(domain.GitHubRepository,error){return g.repo,nil};func(g *stateGH)IssueDetails(context.Context,domain.RepositoryIdentity,int)(domain.GitHubIssueDetails,error){return g.issue,nil};func(g *stateGH)IssueComments(context.Context,domain.RepositoryIdentity,int)([]domain.GitHubComment,error){return append([]domain.GitHubComment(nil),g.comments...),nil};func(g *stateGH)CreateIssue(context.Context,domain.RepositoryIdentity,string,string)(domain.GitHubIssueDetails,error){return g.issue,nil};func(g *stateGH)AppendIssueComment(_ context.Context,_ domain.RepositoryIdentity,_ int,body string)error{g.appendCalls++;g.comments=append(g.comments,domain.GitHubComment{ID:int64(len(g.comments)+1),Body:body,CreatedAt:"2026-09-15T08:00:00Z"});if g.failAfterAppend{g.failAfterAppend=false;return errors.New("connection reset")};return nil}

type mutableGit struct{repo domain.RepositoryIdentity;snap domain.GitSnapshot;remoteHead string}
func(g *mutableGit)Version(context.Context)(string,error){return "git",nil};func(g *mutableGit)RepositoryRoot(context.Context,string)(string,error){if g.snap.RepositoryRoot!=""{return g.snap.RepositoryRoot,nil};return "/repo",nil};func(g *mutableGit)ResolveRemote(context.Context,string,string)(domain.RepositoryIdentity,string,error){return g.repo,"origin",nil};func(g *mutableGit)Snapshot(context.Context,string)(domain.GitSnapshot,error){return g.snap,nil};func(g *mutableGit)RemoteBranchHead(context.Context,string,string,string)(string,error){if g.remoteHead!=""{return g.remoteHead,nil};return g.snap.Head,nil};func(g *mutableGit)FetchRemote(context.Context,string,string)error{return nil};func(g *mutableGit)CommitExists(context.Context,string,string)bool{return true};func(g *mutableGit)RestoreBranch(context.Context,string,string,string,string)error{return nil}
func testContract(t *testing.T,withValidation bool)string{t.Helper();v:=[]string{};if withValidation{v=[]string{"go test ./..."}};b,e:=domain.RenderTaskContract(domain.TaskContractInput{ContractID:"c1",Goal:"goal",Scope:[]string{"scope"},OutOfScope:[]string{"none"},AcceptanceCriteria:[]string{"done"},Validation:v});if e!=nil{t.Fatal(e)};return b}
func stateService(t *testing.T,withValidation bool)(*WorkflowService,*stateGH,*mutableGit){t.Helper();repo:=domain.RepositoryIdentity{Host:"github.com",Owner:"o",Repo:"r"};g:=&stateGH{repo:domain.GitHubRepository{NameWithOwner:"o/r",HasIssuesEnabled:true},issue:domain.GitHubIssueDetails{Number:7,State:"OPEN",Title:"Task",Body:testContract(t,withValidation)}};git:=&mutableGit{repo:repo,snap:domain.GitSnapshot{RepositoryRoot:"/repo",Branch:"main",Head:"abc",Dirty:false,ChangedPaths:[]string{}},remoteHead:"abc"};return &WorkflowService{Git:git,GitHub:g},g,git}
func mo(op string)MutationOptions{return MutationOptions{WorkflowOptions:WorkflowOptions{CWD:"/repo",Issue:7},OperationID:op,RunID:"run-1"}}
