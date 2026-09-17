package domain

import(
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)
type TaskContractInput struct{ContractID string;CreateOperationID string;Goal string;Scope []string;OutOfScope []string;Constraints []string;AcceptanceCriteria []string;Validation []string;Dependencies []string}
func RenderTaskContract(in TaskContractInput)(string,error){
	if strings.TrimSpace(in.ContractID)==""||strings.TrimSpace(in.Goal)==""||len(in.Scope)==0||len(in.OutOfScope)==0||len(in.AcceptanceCriteria)==0{return "",NewWorkflowError("INVALID_INVOCATION","Goal/Scope/Out of Scope/Acceptance Criteria and contract_id are required")}
	metaMap:=map[string]string{"schema":TaskContractSchema,"contract_id":in.ContractID};if strings.TrimSpace(in.CreateOperationID)!=""{metaMap["create_operation_id"]=strings.TrimSpace(in.CreateOperationID)};meta,_:=json.Marshal(metaMap)
	var b strings.Builder;section:=func(name string,lines []string,checkbox bool){fmt.Fprintf(&b,"## %s\n\n",name);for _,x:=range lines{x=strings.TrimSpace(x);if x==""{continue};if checkbox{fmt.Fprintf(&b,"- [ ] %s\n",x)}else{fmt.Fprintf(&b,"- %s\n",x)}};b.WriteString("\n")}
	b.WriteString("## Goal\n\n"+strings.TrimSpace(in.Goal)+"\n\n");section("Scope",in.Scope,false);section("Out of Scope",in.OutOfScope,false);if len(in.Constraints)>0{section("Constraints",in.Constraints,false)};section("Acceptance Criteria",in.AcceptanceCriteria,true);if len(in.Validation)>0{section("Validation",in.Validation,false)};if len(in.Dependencies)>0{section("Dependencies / References",in.Dependencies,false)};fmt.Fprintf(&b,"<!-- iw:task-contract:v1\n%s\n-->\n",meta);return b.String(),nil
}
func RenderWorkflowEvent(e WorkflowEvent)(string,error){raw,err:=json.Marshal(e);if err!=nil{return "",err};marker:="["+string(e.EventType)+"]";human:=eventHumanSummary(e);if human!=""{human="\n\n"+human};version:="v2";if e.Schema==WorkflowEventSchemaV1{version="v1"};return fmt.Sprintf("%s%s\n\n<!-- iw:workflow-event:%s\n%s\n-->\n",marker,human,version,raw),nil}
func eventHumanSummary(e WorkflowEvent)string{for _,k:=range []string{"summary","reason","next"}{if s:=stringData(e.Data,k);s!=""{return s}};return ""}
func OperationFingerprint(t WorkflowEventType,data map[string]any)string{raw,_:=json.Marshal(map[string]any{"event_type":t,"data":data});sum:=sha256.Sum256(raw);return "sha256:"+hex.EncodeToString(sum[:])}
