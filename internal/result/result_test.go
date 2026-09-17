package result

import "testing"

func TestNormalizeMachineLifecycleActions(t *testing.T){
	r:=Success("status",nil);r.State="WAITING";r.Lifecycle="WAITING";n:=7;r.Issue=&n;NormalizeMachine(&r)
	if r.Machine.Schema!=MachineSemanticsSchema||len(r.Machine.NextActions)!=1||r.Machine.NextActions[0].ID!="recheck"||r.Machine.NextActions[0].Command!="recheck"{t.Fatalf("%#v",r.Machine)}
	found:=false;for _,a:=range r.Machine.AllowedActions{if a=="recheck"{found=true}};if !found{t.Fatalf("%v",r.Machine.AllowedActions)}
}
func TestNormalizeMachineErrorIsStructured(t *testing.T){
	r:=Failure("complete","VALIDATION_FAILED","missing validation",nil,nil);r.Lifecycle="IN_PROGRESS";n:=9;r.Issue=&n;NormalizeMachine(&r)
	if len(r.Machine.Constraints)==0||r.Machine.Constraints[0].Code!="VALIDATION_FAILED"{t.Fatalf("%#v",r.Machine)}
	if len(r.Machine.NextActions)==0||r.Machine.NextActions[0].ID!="validate"||r.Machine.NextActions[0].Issue==nil||*r.Machine.NextActions[0].Issue!=9{t.Fatalf("%#v",r.Machine.NextActions)}
}
func TestNormalizeMachineV1WarningSuggestsMigration(t *testing.T){
	r:=Success("status",nil);r.Lifecycle="IN_PROGRESS";r.Warnings=[]string{"WORKFLOW_V1_COMPAT"};NormalizeMachine(&r)
	if len(r.Machine.Warnings)!=1||r.Machine.Warnings[0].Code!="WORKFLOW_V1_COMPAT"{t.Fatalf("%#v",r.Machine)}
	if len(r.Machine.NextActions)==0||r.Machine.NextActions[0].ID!="migrate"{t.Fatalf("%#v",r.Machine.NextActions)}
}
