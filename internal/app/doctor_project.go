package app

import (
	"context"
	"fmt"

	"github.com/rontian/issue-workflow/internal/result"
)

func (s *ProjectService) AugmentDoctor(ctx context.Context,cwd string,base result.CommandResult,baseCode int)(result.CommandResult,int){
	if baseCode!=result.ExitOK||!base.OK{return base,baseCode}
	rep,ok:=base.Data.(DoctorReport);if !ok{return base,baseCode}
	pr,err:=s.Read(ctx,cwd)
	if err!=nil{rep.Checks=append(rep.Checks,Check{ID:"project.config",Category:"project",Status:CheckFail,Required:true,Message:err.Error()});sortChecks(rep.Checks);rep.Summary=summarize(rep.Checks);base.OK=false;base.Error=&result.Error{Code:"PROJECT_CONFIG_INVALID",Message:err.Error()};base.Data=rep;return base,result.ExitProtocol}
	if !pr.Configured{rep.Checks=append(rep.Checks,Check{ID:"project.config",Category:"project",Status:CheckSkip,Required:false,Message:"iw.json not configured (optional)"})}else{rep.Checks=append(rep.Checks,Check{ID:"project.config",Category:"project",Status:CheckPass,Required:true,Message:"iw.json valid"})}
	failureCode:=""
	for i,d:=range pr.Docs{st:=CheckPass;msg:="declared project document exists: "+d.Path;next:="";if !d.Exists{st=CheckFail;msg="declared project document missing: "+d.Path;next="restore or add the declared repository document";if failureCode==""{failureCode="ENVIRONMENT_INVALID"}};rep.Checks=append(rep.Checks,Check{ID:fmt.Sprintf("project.doc.%03d",i+1),Category:"project",Status:st,Required:true,Message:msg,Next:next})}
	for _,e:=range pr.Environment{st:=CheckPass;msg:="project executable available: "+e.Name;next:="";if !e.Available{st=CheckFail;msg="project executable missing: "+e.Name;next="install the project-declared executable in the local environment";failureCode="DEPENDENCY_MISSING"};rep.Checks=append(rep.Checks,Check{ID:"project.executable."+e.Name,Category:"project",Status:st,Required:true,Message:msg,Next:next})}
	sortChecks(rep.Checks);rep.Summary=summarize(rep.Checks);base.Data=rep
	if failureCode!=""{base.OK=false;base.Error=&result.Error{Code:failureCode,Message:"project environment contract is not satisfied"};return base,result.ExitCodeFor(failureCode)}
	return base,result.ExitOK
}
