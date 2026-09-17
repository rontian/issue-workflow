package domain

import "testing"

func ev(id, parent, op string, typ WorkflowEventType, before, after WorkflowState, data map[string]any) *WorkflowEvent {
	var p *string
	if parent != "" { x := parent; p = &x }
	b := "main"
	return &WorkflowEvent{Schema: WorkflowEventSchema, EventID: id, ParentEventID: p, OperationID: op, RunID: "r", EventType: typ, OccurredAt: "2026-09-15T04:00:00Z", Repository: "o/r", Issue: 7, ContractDigest: digest('a'), StateBefore: before, StateAfter: after, Git: &EventGitSnapshot{Branch: &b, Head: "abc", ChangedPaths: []string{}}, Data: data}
}
func vals(es ...*WorkflowEvent) []WorkflowEvent { out := make([]WorkflowEvent, len(es)); for i, e := range es { out[i] = *e }; return out }
func TestReplayEmpty(t *testing.T) {
	a, err := ReplayWorkflow("o/r", 7, nil)
	if err != nil || a.State != StateReady { t.Fatalf("%#v %v", a, err) }
}
func TestReplayLinearBlockedResume(t *testing.T) {
	a, err := ReplayWorkflow("o/r", 7, vals(ev("e1", "", "o1", EventStart, StateReady, StateInProgress, map[string]any{}), ev("e2", "e1", "o2", EventCheckpoint, StateInProgress, StateInProgress, map[string]any{"summary": "s", "next": "n", "validation": []any{}}), ev("e3", "e2", "o3", EventBlocked, StateInProgress, StateBlocked, map[string]any{"reason": "wait", "next": "n", "resume_state": "IN_PROGRESS"}), ev("e4", "e3", "o4", EventResume, StateBlocked, StateInProgress, map[string]any{})))
	if err != nil { t.Fatal(err) }
	if a.State != StateInProgress || a.BlockReason != "" { t.Fatalf("%#v", a) }
}
func TestReplayDetectsFork(t *testing.T) {
	_, err := ReplayWorkflow("o/r", 7, vals(ev("e1", "", "o1", EventStart, StateReady, StateInProgress, map[string]any{}), ev("e2", "e1", "o2", EventCheckpoint, StateInProgress, StateInProgress, map[string]any{"summary": "s", "next": "n", "validation": []any{}}), ev("e3", "e1", "o3", EventCheckpoint, StateInProgress, StateInProgress, map[string]any{"summary": "s", "next": "n", "validation": []any{}})))
	if WorkflowErrorCode(err) != "EVENT_FORK" { t.Fatalf("%v", err) }
}
func TestReplayDetectsDuplicateOperation(t *testing.T) {
	_, err := ReplayWorkflow("o/r", 7, vals(ev("e1", "", "same", EventStart, StateReady, StateInProgress, map[string]any{}), ev("e2", "e1", "same", EventCheckpoint, StateInProgress, StateInProgress, map[string]any{"summary": "s", "next": "n", "validation": []any{}})))
	if WorkflowErrorCode(err) != "CONFLICT" { t.Fatalf("%v", err) }
}
func TestReplayDetectsInvalidTransition(t *testing.T) {
	_, err := ReplayWorkflow("o/r", 7, vals(ev("e1", "", "o1", EventReview, StateReady, StateReview, map[string]any{})))
	if WorkflowErrorCode(err) != "TRANSITION_BLOCKED" { t.Fatalf("%v", err) }
}
func TestReplayScopeAcceptChangesDigest(t *testing.T) {
	start := ev("e1", "", "o1", EventStart, StateReady, StateInProgress, map[string]any{})
	scope := ev("e2", "e1", "o2", EventScope, StateInProgress, StateInProgress, map[string]any{"action": "accept", "previous_contract_digest": digest('a'), "new_contract_digest": digest('b')})
	scope.ContractDigest = digest('b')
	a, err := ReplayWorkflow("o/r", 7, vals(start, scope))
	if err != nil { t.Fatal(err) }
	if a.AcceptedContractDigest != digest('b') { t.Fatal(a.AcceptedContractDigest) }
}
func TestReplayDetectsBrokenParent(t *testing.T) {
	e1 := ev("e1", "", "o1", EventStart, StateReady, StateInProgress, map[string]any{})
	e2 := ev("e2", "missing", "o2", EventCheckpoint, StateInProgress, StateInProgress, map[string]any{"summary": "s", "next": "n", "validation": []any{}})
	_, err := ReplayWorkflow("o/r", 7, vals(e1, e2))
	if WorkflowErrorCode(err) != "EVENT_CHAIN_BROKEN" { t.Fatalf("%v", err) }
}
func TestCompareGitSnapshot(t *testing.T) {
	b := "main"
	p := &EventGitSnapshot{Branch: &b, Head: "a", ChangedPaths: []string{"a"}}
	c := GitSnapshot{Branch: "dev", Head: "b", Dirty: true, ChangedPaths: []string{"b"}}
	d := CompareGitSnapshot(p, c)
	if !d.Changed || len(d.Fields) < 4 { t.Fatalf("%#v", d) }
}
