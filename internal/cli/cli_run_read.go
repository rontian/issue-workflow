package cli

import(
	"context"
	"github.com/rontian/issue-workflow/internal/app"
	"github.com/rontian/issue-workflow/internal/result"
)

func(c *CLI)runEnvironmentOrRead(ctx context.Context,command string,o options,wo app.WorkflowOptions,cwd string)(bool,int){
	switch command{
	case "version":return true,c.render(app.VersionResult(),result.ExitOK,o.JSON)
	case "capabilities":return true,c.render(app.CapabilitiesResult(),result.ExitOK,o.JSON)
	case "project status":if c.Project==nil{return true,c.render(result.Failure(command,"ENVIRONMENT_INVALID","project service unavailable",nil,nil),result.ExitEnvironment,o.JSON)};r,code:=c.Project.Status(ctx,cwd);return true,c.render(r,code,o.JSON)
	case "tasks":if c.Tasks==nil{return true,c.render(result.Failure(command,"ENVIRONMENT_INVALID","task read model unavailable",nil,nil),result.ExitEnvironment,o.JSON)};r,code:=c.Tasks.Tasks(ctx,wo);return true,c.render(r,code,o.JSON)
	case "next":if c.Tasks==nil{return true,c.render(result.Failure(command,"ENVIRONMENT_INVALID","task read model unavailable",nil,nil),result.ExitEnvironment,o.JSON)};r,code:=c.Tasks.Next(ctx,wo);return true,c.render(r,code,o.JSON)
	case "context":if c.Context==nil{return true,c.render(result.Failure(command,"ENVIRONMENT_INVALID","context service unavailable",nil,nil),result.ExitEnvironment,o.JSON)};r,code:=c.Context.Run(ctx,wo);return true,c.render(r,code,o.JSON)
	case "restore":if c.Restore==nil{return true,c.render(result.Failure(command,"ENVIRONMENT_INVALID","restore service unavailable",nil,nil),result.ExitEnvironment,o.JSON)};r,code:=c.Restore.Run(ctx,app.RestoreOptions{WorkflowOptions:wo,Apply:o.Apply});return true,c.render(r,code,o.JSON)
	case "doctor":
		if o.Global&&(o.Repo!=""||o.Remote!=""||o.Issue>0){return true,c.render(result.Failure("doctor","INVALID_INVOCATION","--global 不能与 --repo/--remote/--issue 同时使用",nil,nil),result.ExitInvalid,o.JSON)}
		r,code:=c.Doctor.Run(ctx,app.DoctorOptions{CWD:cwd,Repo:o.Repo,Remote:o.Remote,Host:o.Host,Global:o.Global,Issue:o.Issue});if o.Issue>0&&c.Workflow!=nil{r,code=c.Workflow.AugmentDoctorIssue(ctx,wo,r,code)};return true,c.render(r,code,o.JSON)
	case "init":
		if o.Global&&(o.Repo!=""||o.Remote!=""||o.Issue>0){return true,c.render(result.Failure("init","INVALID_INVOCATION","--global 不能与 --repo/--remote/--issue 同时使用",nil,nil),result.ExitInvalid,o.JSON)}
		r,code:=c.Init.Run(ctx,app.InitOptions{Doctor:app.DoctorOptions{CWD:cwd,Repo:o.Repo,Remote:o.Remote,Host:o.Host,Global:o.Global,Issue:o.Issue},DryRun:o.DryRun,Interactive:o.Interactive,InteractiveTerminal:c.IsTerminal()});return true,c.render(r,code,o.JSON)
	case "status":r,code:=c.Workflow.Status(ctx,wo);return true,c.render(r,code,o.JSON)
	case "new":r,code:=c.Workflow.NewIssue(ctx,app.NewIssueOptions{WorkflowOptions:wo,DryRun:o.DryRun,Title:o.Title,Goal:o.Goal,Scope:o.ScopeItems,OutOfScope:o.OutOfScope,Constraints:o.Constraints,Acceptance:o.Acceptance,Validation:o.Validation,Dependencies:o.Dependencies,ContractID:o.ContractID});return true,c.render(r,code,o.JSON)
	default:return false,0
	}
}
