package cli

import "testing"

func TestParseStatusAllowsFlagsAfterIssue(t *testing.T) {
	cmd, o, _, err := parse([]string{"status", "123", "--json", "--repo", "o/r"})
	if err != nil { t.Fatal(err) }
	if cmd != "status" || o.Issue != 123 || !o.JSON || o.Repo != "o/r" { t.Fatalf("%s %#v", cmd, o) }
}
func TestParseResumeRequiresIssue(t *testing.T) {
	_, _, _, err := parse([]string{"resume", "--dry-run"})
	if err == nil { t.Fatal("expected error") }
}
func TestParseStatusRejectsMismatchedIssue(t *testing.T) {
	_, _, _, err := parse([]string{"status", "123", "--issue", "124"})
	if err == nil { t.Fatal("expected mismatch") }
}
func TestParseScopeActionAndIssue(t *testing.T) {
	cmd, o, _, err := parse([]string{"scope", "accept", "12", "--reason", "approved", "--proposal-event", "e1", "--json"})
	if err != nil { t.Fatal(err) }
	if cmd != "scope" || o.ScopeAction != "accept" || o.Issue != 12 || o.Reason != "approved" || o.ProposalEventID != "e1" || !o.JSON { t.Fatalf("%s %#v", cmd, o) }
}
func TestParseNewRepeatableFlags(t *testing.T) {
	cmd, o, _, err := parse([]string{"new", "--title", "T", "--goal", "G", "--scope", "a", "--scope", "b", "--out-of-scope", "x", "--acceptance", "done", "--validation", "go test ./..."})
	if err != nil { t.Fatal(err) }
	if cmd != "new" || len(o.ScopeItems) != 2 || len(o.Validation) != 1 || o.Title != "T" { t.Fatalf("%s %#v", cmd, o) }
}
func TestParseMutationValidationRecordFlag(t *testing.T) {
	cmd, o, _, err := parse([]string{"checkpoint", "9", "--summary", "s", "--next", "n", "--validation", "v1=pass:ok"})
	if err != nil { t.Fatal(err) }
	if cmd != "checkpoint" || o.Issue != 9 || len(o.Validation) != 1 { t.Fatalf("%s %#v", cmd, o) }
}
