package domain

import "testing"

func TestCurrentHeadRequiresExplicitHeadBoundValidationEvidence(t *testing.T){
    req:=[]ValidationRequirement{{ID:"v1",Text:"go test ./..."}}
    tests:=[]struct{head string;status string;missing bool}{
        {"","pass",true},
        {"old","pass",true},
        {"current","pass",false},
        {"current","fail",true},
        {"current","manual_required",true},
        {"current","not_applicable",true},
    }
    for _,tt:=range tests {
        evidence:=map[string]ValidationRecord{"v1":{ID:"v1",Status:tt.status,ObservedHead:tt.head}}
        missing:=MissingValidationsAtHead(req,evidence,"current")
        if (len(missing)>0)!=tt.missing {t.Fatalf("head=%q status=%q missing=%v",tt.head,tt.status,missing)}
    }
}
