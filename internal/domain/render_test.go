package domain

import (
	"strings"
	"testing"
)

func TestRenderTaskContractRoundTrip(t *testing.T) {
	body, e := RenderTaskContract(TaskContractInput{ContractID: "c1", Goal: "g", Scope: []string{"s"}, OutOfScope: []string{"o"}, AcceptanceCriteria: []string{"a"}, Validation: []string{"go test ./..."}})
	if e != nil { t.Fatal(e) }
	c, e := ParseTaskContract(body)
	if e != nil { t.Fatal(e) }
	if c.ContractID != "c1" || !strings.Contains(c.AcceptanceCriteria, "[ ] a") { t.Fatalf("%+v", c) }
}
func TestValidationRequirements(t *testing.T) {
	c := TaskContract{Validation: "- `go test ./...`\n- go vet ./..."}
	r := ValidationRequirements(c)
	if len(r) != 2 || r[0].ID != "v1" || r[0].Text != "go test ./..." { t.Fatalf("%+v", r) }
}
