package domain

import "testing"

func TestTaskContractCreateOperationRoundTrip(t *testing.T){body,err:=RenderTaskContract(TaskContractInput{ContractID:"c1",CreateOperationID:"op1",Goal:"g",Scope:[]string{"s"},OutOfScope:[]string{"o"},AcceptanceCriteria:[]string{"a"}});if err!=nil{t.Fatal(err)};c,err:=ParseTaskContract(body);if err!=nil{t.Fatal(err)};if c.ContractID!="c1"||c.CreateOperationID!="op1"{t.Fatalf("%#v",c)}}
