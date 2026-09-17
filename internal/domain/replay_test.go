package domain

import "testing"

func ev(id, parent, op string, typ WorkflowEventType, before, after WorkflowState, data map[string]any) *WorkflowEvent {
	var p *string
	if parent != "" { x := parent; p = &x }
	b := "main"
	return &WorkflowEvent{Schema: WorkflowEventSchemaV1, EventID: id, ParentEventID: p, OperationID: op, RunID: "r", EventType: typ, OccurredAt: "2026-09-15T04:00:00Z", Repository: "o/r", Issue: 7, ContractDigest: digest('a'), StateBefore: before, StateAfter: after, Git: &EventGitSnapshot{Branch: &b, Head: "abc", ChangedPaths: []string{}}, Data: data}
}
func ev2(id, parent, op string, typ WorkflowEventType, before, after WorkflowState, data map[string]any) *WorkflowEvent {
	e := ev(id, parent, op, typ, before, after, data); e.Schema = WorkflowEventSchemaV2; return e
}
func vals(es ...*WorkflowEvent) []WorkflowEvent { out := make([]WorkflowEvent, len(es)); for i, e := range es { out[i] = *e }; return out }
func TestReplayEmpty(t *testing.T) {
	a, err := ReplayWorkflow("o/r", 7, nil)
	if err != nil || a.State != StateReady || a.Lifecycle != LifecycleReady { t.Fatalf("%#v %v", a, err) }
}
func TestReplayLinearBlockedResume(t *testing.T) {
	a, err := ReplayWorkflow("o/r", 7, vals(ev("e1", "", "o1", EventStart, StateReady, StateInProgress, map[string]any{}), ev("e2", "e1", "o2", EventCheckpoint, StateInProgress, StateInProgress, map[string]any{"summary": "s", "next": "n", "validation": []any{}}), ev("e3", "e2", "o3", EventBlocked, StateInProgress, StateBlocked, map[string]any{"reason": "wait", "next": "n", "resume_state": "IN_PROGRESS"}), ev("e4", "e3", "o4", EventResume, StateBlocked, StateInProgress, map[string]any{})))
	if err != nil { t.Fatal(err) }
	if a.State != StateInProgress || a.BlockReason != "" || a.Protocol != "v1" { t.Fatalf("%#v", a) }
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
func TestReplayRequiresExplicitV1ToV2Migration(t *testing.T) {
	start := ev("e1", "", "o1", EventStart, StateReady, StateInProgress, map[string]any{})
	cp := ev2("e2", "e1", "o2", EventCheckpoint, StateInProgress, StateInProgress, map[string]any{"summary":"s","next":"n","validation":[]any{}})
	_, err := ReplayWorkflow("o/r", 7, vals(start, cp))
	if WorkflowErrorCode(err) != "MIGRATION_REQUIRED" { t.Fatalf("%v", err) }
}
func TestReplayV2MigrationLifecycle(t *testing.T) {
	start := ev("e1", "", "o1", EventStart, StateReady, StateInProgress, map[string]any{})
	migrate := ev2("e2", "e1", "o2", EventMigrate, StateInProgress, StateInProgress, map[string]any{"from_schema":WorkflowEventSchemaV1,"from_state":"IN_PROGRESS","phase":"IMPLEMENTATION"})
	pause := ev2("e3", "e2", "o3", EventPause, StateInProgress, StatePaused, map[string]any{"reason":"session boundary"})
	resume := ev2("e4", "e3", "o4", EventResume, StatePaused, StateInProgress, map[string]any{"summary":"resume"})
	wait := ev2("e5", "e4", "o5", EventWait, StateInProgress, StateWaiting, map[string]any{"reason":"external","next":"recheck"})
	recheck1 := ev2("e6", "e5", "o6", EventRecheck, StateWaiting, StateWaiting, map[string]any{"resolved":false})
	recheck2 := ev2("e7", "e6", "o7", EventRecheck, StateWaiting, StateInProgress, map[string]any{"resolved":true})
	review := ev2("e8", "e7", "o8", EventReview, StateInProgress, StateInProgress, map[string]any{"summary":"clean","findings":[]any{}})
	complete := ev2("e9", "e8", "o9", EventComplete, StateInProgress, StateCompleted, map[string]any{"summary":"done","validation":[]any{},"delivery":map[string]any{"commit":"abc","pushed":false,"pr":nil}})
	a, err := ReplayWorkflow("o/r", 7, vals(start,migrate,pause,resume,wait,recheck1,recheck2,review,complete))
	if err != nil { t.Fatal(err) }
	if a.Protocol != "v2" || a.State != StateCompleted || a.Lifecycle != LifecycleCompleted || a.ReviewStatus != "passed" { t.Fatalf("%#v", a) }
}
func TestCompareGitSnapshot(t *testing.T) {
	b := "main"; p := &EventGitSnapshot{Branch: &b, Head: "a", ChangedPaths: []string{"a"}}; c := GitSnapshot{Branch: "dev", Head: "b", Dirty: true, ChangedPaths: []string{"b"}}
	d := CompareGitSnapshot(p, c); if !d.Changed || len(d.Fields) < 4 { t.Fatalf("%#v", d) }
}
