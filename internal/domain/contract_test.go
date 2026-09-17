package domain

import "testing"

func validContractBody(newline string) string {
	return "## Goal" + newline + newline + "G" + newline + newline + "## Scope" + newline + newline + "- S" + newline + newline + "## Out of Scope" + newline + newline + "- O" + newline + newline + "## Acceptance Criteria" + newline + newline + "- [ ] A" + newline + newline + "<!-- iw:task-contract:v1" + newline + "{\"schema\":\"iw.task-contract/v1\",\"contract_id\":\"c1\"}" + newline + "-->" + newline
}
func TestParseTaskContractAndDigestNormalization(t *testing.T) {
	lf := validContractBody("\n")
	crlf := validContractBody("\r\n")
	a, err := ParseTaskContract(lf)
	if err != nil { t.Fatal(err) }
	b, err := ParseTaskContract(crlf)
	if err != nil { t.Fatal(err) }
	if a.Digest != b.Digest { t.Fatalf("digest mismatch: %s %s", a.Digest, b.Digest) }
	if a.Goal != "G" || a.Scope != "- S" || a.ContractID != "c1" { t.Fatalf("unexpected contract: %#v", a) }
}
func TestParseTaskContractRejectsMissingRequiredSection(t *testing.T) {
	body := "## Goal\nG\n## Scope\nS\n## Acceptance Criteria\nA\n<!-- iw:task-contract:v1\n{\"schema\":\"iw.task-contract/v1\",\"contract_id\":\"c1\"}\n-->"
	_, err := ParseTaskContract(body)
	if WorkflowErrorCode(err) != "TASK_CONTRACT_INVALID" { t.Fatalf("got %v", err) }
}
func TestParseTaskContractRejectsUnknownSchema(t *testing.T) {
	body := "## Goal\nG\n## Scope\nS\n## Out of Scope\nO\n## Acceptance Criteria\nA\n<!-- iw:task-contract:v2\n{\"schema\":\"iw.task-contract/v2\",\"contract_id\":\"c1\"}\n-->"
	_, err := ParseTaskContract(body)
	if WorkflowErrorCode(err) != "UNSUPPORTED_SCHEMA" { t.Fatalf("got %v", err) }
}
