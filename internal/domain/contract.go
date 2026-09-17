package domain

import(
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)
const contractMarker="<!-- iw:task-contract:"
func ContractDigest(body string)string{normalized:=strings.ReplaceAll(body,"\r\n","\n");sum:=sha256.Sum256([]byte(normalized));return "sha256:"+hex.EncodeToString(sum[:])}
func ParseTaskContract(body string)(TaskContract,error){
	if strings.Count(body,contractMarker)!=1{return TaskContract{},NewWorkflowError("TASK_CONTRACT_INVALID","task contract metadata block must appear exactly once")};start:=strings.Index(body,contractMarker);lineEnd:=strings.Index(body[start:],"\n");if lineEnd<0{return TaskContract{},NewWorkflowError("TASK_CONTRACT_INVALID","task contract metadata header is malformed")};lineEnd+=start;header:=strings.TrimSpace(body[start:lineEnd]);if header!="<!-- iw:task-contract:v1"{return TaskContract{},NewWorkflowError("UNSUPPORTED_SCHEMA",fmt.Sprintf("unsupported task contract marker %q",header))};endRel:=strings.Index(body[lineEnd+1:],"-->");if endRel<0{return TaskContract{},NewWorkflowError("TASK_CONTRACT_INVALID","task contract metadata block is not closed")};end:=lineEnd+1+endRel;raw:=strings.TrimSpace(body[lineEnd+1:end])
	var meta struct{Schema string `json:"schema"`;ContractID string `json:"contract_id"`;CreateOperationID string `json:"create_operation_id,omitempty"`};if err:=json.Unmarshal([]byte(raw),&meta);err!=nil{return TaskContract{},NewWorkflowError("TASK_CONTRACT_INVALID","task contract metadata JSON is invalid")};if meta.Schema!=TaskContractSchema{return TaskContract{},NewWorkflowError("UNSUPPORTED_SCHEMA",fmt.Sprintf("unsupported task contract schema %q",meta.Schema))};if strings.TrimSpace(meta.ContractID)==""{return TaskContract{},NewWorkflowError("TASK_CONTRACT_INVALID","contract_id is required")}
	sections:=parseH2Sections(body[:start]);required:=[]string{"Goal","Scope","Out of Scope","Acceptance Criteria"};for _,name:=range required{if strings.TrimSpace(sections[name])==""{return TaskContract{},NewWorkflowError("TASK_CONTRACT_INVALID","missing or empty section: "+name)}}
	return TaskContract{Schema:meta.Schema,ContractID:meta.ContractID,CreateOperationID:meta.CreateOperationID,Digest:ContractDigest(body),Goal:sections["Goal"],Scope:sections["Scope"],OutOfScope:sections["Out of Scope"],Constraints:sections["Constraints"],AcceptanceCriteria:sections["Acceptance Criteria"],Validation:sections["Validation"],DependenciesReferences:sections["Dependencies / References"]},nil
}
func parseH2Sections(body string)map[string]string{out:=map[string]string{};lines:=strings.Split(strings.ReplaceAll(body,"\r\n","\n"),"\n");current:="";var buf []string;flush:=func(){if current!=""{out[current]=strings.TrimSpace(strings.Join(buf,"\n"))}};for _,line:=range lines{if strings.HasPrefix(line,"## "){flush();current=strings.TrimSpace(strings.TrimPrefix(line,"## "));buf=nil;continue};if current!=""{buf=append(buf,line)}};flush();return out}
