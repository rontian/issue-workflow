package domain

import (
	"fmt"
	"sort"
	"strings"
)

func ReplayWorkflow(repo string, issue int, events []WorkflowEvent) (WorkflowAggregate, error) {
	agg := WorkflowAggregate{State: StateReady, Events: append([]WorkflowEvent{}, events...), UnresolvedScopeProposals: []string{}}
	eventIDs := map[string]struct{}{}
	ops := map[string]string{}
	children := map[string]int{}
	parentKey := func(p *string) string { if p == nil { return "<root>" }; return *p }
	for _, e := range events {
		if _, ok := eventIDs[e.EventID]; ok { return agg, NewWorkflowError("PROTOCOL_ERROR", "duplicate event_id: "+e.EventID) }
		eventIDs[e.EventID] = struct{}{}
		if prev, ok := ops[e.OperationID]; ok { return agg, NewWorkflowError("CONFLICT", fmt.Sprintf("duplicate operation_id %s on %s and %s", e.OperationID, prev, e.EventID)) }
		ops[e.OperationID] = e.EventID
		k := parentKey(e.ParentEventID); children[k]++
		if children[k] > 1 { return agg, NewWorkflowError("EVENT_FORK", "multiple events share parent "+k) }
	}
	proposals := map[string]struct{}{}
	var lastID *string
	accepted := ""
	resumeState := WorkflowState("")
	for i := range events {
		e := events[i]
		if e.Repository != repo || e.Issue != issue { return agg, NewWorkflowError("PROTOCOL_ERROR", "workflow event repository/issue identity mismatch") }
		if lastID == nil {
			if e.ParentEventID != nil { return agg, NewWorkflowError("EVENT_CHAIN_BROKEN", "first workflow event must have null parent_event_id") }
		} else if e.ParentEventID == nil || *e.ParentEventID != *lastID {
			return agg, NewWorkflowError("EVENT_CHAIN_BROKEN", "parent_event_id does not match previous event")
		}
		if e.StateBefore != agg.State { return agg, NewWorkflowError("EVENT_CHAIN_BROKEN", fmt.Sprintf("state_before %s does not match current state %s", e.StateBefore, agg.State)) }
		if accepted == "" {
			if e.EventType != EventStart { return agg, NewWorkflowError("TRANSITION_BLOCKED", "first workflow event must be START") }
			accepted = e.ContractDigest
		}
		if e.EventType != EventScope || scopeAction(e) != "accept" {
			if e.ContractDigest != accepted { return agg, NewWorkflowError("PROTOCOL_ERROR", "event contract_digest does not match accepted contract") }
		}
		if err := validateEventPayload(e); err != nil { return agg, err }
		next, err := applyTransition(agg.State, resumeState, e)
		if err != nil { return agg, err }
		if e.EventType == EventBlocked {
			resumeState = e.StateBefore
			agg.BlockReason = stringData(e.Data, "reason")
			agg.ResumeState = resumeState
		}
		if e.EventType == EventResume && e.StateBefore == StateBlocked {
			agg.BlockReason = ""
			agg.ResumeState = ""
			resumeState = ""
		}
		if e.EventType == EventScope {
			switch scopeAction(e) {
			case "propose":
				proposals[e.EventID] = struct{}{}
			case "reject":
				id := stringData(e.Data, "proposal_event_id")
				if id == "" { return agg, NewWorkflowError("PROTOCOL_ERROR", "SCOPE reject requires proposal_event_id") }
				if _, ok := proposals[id]; !ok { return agg, NewWorkflowError("PROTOCOL_ERROR", "SCOPE reject references unknown proposal") }
				delete(proposals, id)
			case "accept":
				prev := stringData(e.Data, "previous_contract_digest")
				nextDigest := stringData(e.Data, "new_contract_digest")
				if prev == "" || nextDigest == "" || prev != accepted || nextDigest != e.ContractDigest { return agg, NewWorkflowError("PROTOCOL_ERROR", "SCOPE accept digest fields are invalid") }
				accepted = nextDigest
				if id := stringData(e.Data, "proposal_event_id"); id != "" {
					if _, ok := proposals[id]; !ok { return agg, NewWorkflowError("PROTOCOL_ERROR", "SCOPE accept references unknown proposal") }
					delete(proposals, id)
				}
			default:
				return agg, NewWorkflowError("PROTOCOL_ERROR", "SCOPE action is invalid")
			}
		}
		agg.State = next
		if e.Git != nil { g := *e.Git; agg.LatestGit = &g }
		copyE := e; agg.LastEvent = &copyE
		id := e.EventID; lastID = &id
	}
	agg.AcceptedContractDigest = accepted
	for id := range proposals { agg.UnresolvedScopeProposals = append(agg.UnresolvedScopeProposals, id) }
	sort.Strings(agg.UnresolvedScopeProposals)
	return agg, nil
}

func applyTransition(state, blockedResume WorkflowState, e WorkflowEvent) (WorkflowState, error) {
	neutral := func() (WorkflowState, error) {
		if e.StateAfter != state { return "", NewWorkflowError("TRANSITION_BLOCKED", string(e.EventType)+" must be state-neutral") }
		return state, nil
	}
	switch e.EventType {
	case EventStart:
		if state != StateReady || e.StateAfter != StateInProgress { return "", NewWorkflowError("TRANSITION_BLOCKED", "START requires READY -> IN_PROGRESS") }
		return StateInProgress, nil
	case EventCheckpoint, EventHandoff, EventScope:
		if state == StateReady || state == StateDone { return "", NewWorkflowError("TRANSITION_BLOCKED", string(e.EventType)+" is not allowed in "+string(state)) }
		return neutral()
	case EventBlocked:
		if state != StateInProgress && state != StateReview && state != StateFix { return "", NewWorkflowError("TRANSITION_BLOCKED", "BLOCKED is not allowed from "+string(state)) }
		if e.StateAfter != StateBlocked { return "", NewWorkflowError("TRANSITION_BLOCKED", "BLOCKED must transition to BLOCKED") }
		if stringData(e.Data, "reason") == "" || stringData(e.Data, "next") == "" || WorkflowState(stringData(e.Data, "resume_state")) != state { return "", NewWorkflowError("PROTOCOL_ERROR", "BLOCKED payload is invalid") }
		return StateBlocked, nil
	case EventResume:
		if state == StateBlocked {
			if blockedResume == "" || e.StateAfter != blockedResume { return "", NewWorkflowError("TRANSITION_BLOCKED", "RESUME must restore BLOCKED resume_state") }
			return blockedResume, nil
		}
		if state != StateInProgress && state != StateReview && state != StateFix { return "", NewWorkflowError("TRANSITION_BLOCKED", "RESUME is not allowed from "+string(state)) }
		return neutral()
	case EventReview:
		if (state != StateInProgress && state != StateFix) || e.StateAfter != StateReview { return "", NewWorkflowError("TRANSITION_BLOCKED", "REVIEW requires IN_PROGRESS/FIX -> REVIEW") }
		return StateReview, nil
	case EventFix:
		if state != StateReview || e.StateAfter != StateFix { return "", NewWorkflowError("TRANSITION_BLOCKED", "FIX requires REVIEW -> FIX") }
		return StateFix, nil
	case EventFinal:
		if (state != StateInProgress && state != StateReview && state != StateFix) || e.StateAfter != StateDone { return "", NewWorkflowError("TRANSITION_BLOCKED", "FINAL requires active state -> DONE") }
		return StateDone, nil
	}
	return "", NewWorkflowError("PROTOCOL_ERROR", "unsupported workflow event")
}

func scopeAction(e WorkflowEvent) string { return strings.ToLower(stringData(e.Data, "action")) }
func stringData(m map[string]any, k string) string { v, ok := m[k]; if !ok { return "" }; s, _ := v.(string); return strings.TrimSpace(s) }

func validateEventPayload(e WorkflowEvent) error {
	requireString := func(k string) error {
		if stringData(e.Data, k) == "" { return NewWorkflowError("PROTOCOL_ERROR", string(e.EventType)+" requires data."+k) }
		return nil
	}
	requireArray := func(k string) error {
		v, ok := e.Data[k]
		if !ok { return NewWorkflowError("PROTOCOL_ERROR", string(e.EventType)+" requires data."+k) }
		if _, ok := v.([]any); !ok { return NewWorkflowError("PROTOCOL_ERROR", string(e.EventType)+" data."+k+" must be an array") }
		return nil
	}
	switch e.EventType {
	case EventCheckpoint:
		if err := requireString("summary"); err != nil { return err }
		if err := requireString("next"); err != nil { return err }
		return requireArray("validation")
	case EventHandoff:
		if err := requireString("summary"); err != nil { return err }
		if err := requireString("next"); err != nil { return err }
		return requireArray("warnings")
	case EventFinal:
		if err := requireString("summary"); err != nil { return err }
		if err := requireArray("validation"); err != nil { return err }
		if v, ok := e.Data["delivery"]; !ok { return NewWorkflowError("PROTOCOL_ERROR", "FINAL requires data.delivery") } else if _, ok := v.(map[string]any); !ok { return NewWorkflowError("PROTOCOL_ERROR", "FINAL data.delivery must be an object") }
	case EventScope:
		if scopeAction(e) == "propose" {
			if err := requireString("summary"); err != nil { return err }
			return requireString("reason")
		}
	}
	return nil
}
