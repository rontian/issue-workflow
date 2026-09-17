package app

import (
	"fmt"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
)

func buildEventData(t domain.WorkflowEventType, o MutationOptions, rep WorkflowReport) (map[string]any, error) {
	m := map[string]any{}
	if strings.TrimSpace(o.Summary) != "" { m["summary"] = strings.TrimSpace(o.Summary) }
	validation := append([]domain.ValidationRecord(nil), o.Validation...)
	for i := range validation {
		if validation[i].ObservedHead == "" { validation[i].ObservedHead = rep.CurrentGit.Head }
		if validation[i].Source == "" { validation[i].Source = "agent" }
	}
	switch t {
	case domain.EventMigrate:
		m["from_schema"] = domain.WorkflowEventSchemaV1
		m["from_state"] = string(rep.Aggregate.State)
		if p := domain.ProjectPhase(rep.Aggregate.State); p != "" { m["phase"] = p }
	case domain.EventStart, domain.EventResume:
		if _, ok := m["summary"]; !ok { m["summary"] = defaultSummary(t) }
	case domain.EventPause:
		if strings.TrimSpace(o.Reason) == "" { return nil, fmt.Errorf("pause requires --reason") }
		m["reason"] = strings.TrimSpace(o.Reason)
	case domain.EventWait:
		if strings.TrimSpace(o.Reason) == "" || strings.TrimSpace(o.Next) == "" { return nil, fmt.Errorf("wait requires --reason and --next") }
		m["reason"], m["next"] = strings.TrimSpace(o.Reason), strings.TrimSpace(o.Next)
	case domain.EventRecheck:
		m["resolved"] = o.Resolved
		if strings.TrimSpace(o.Reason) != "" { m["reason"] = strings.TrimSpace(o.Reason) }
	case domain.EventDefer:
		if strings.TrimSpace(o.Reason) == "" || strings.TrimSpace(o.Next) == "" { return nil, fmt.Errorf("defer requires --reason and --next") }
		m["reason"], m["next"] = strings.TrimSpace(o.Reason), strings.TrimSpace(o.Next)
	case domain.EventCheckpoint:
		if strings.TrimSpace(o.Summary) == "" || strings.TrimSpace(o.Next) == "" { return nil, fmt.Errorf("checkpoint requires --summary and --next") }
		m["next"] = strings.TrimSpace(o.Next)
		m["validation"] = validationAny(validation)
	case domain.EventHandoff:
		if strings.TrimSpace(o.Summary) == "" || strings.TrimSpace(o.Next) == "" { return nil, fmt.Errorf("handoff requires --summary and --next") }
		m["next"], m["warnings"] = strings.TrimSpace(o.Next), stringAny(o.Warnings)
		if o.Portable {
			remote := strings.TrimSpace(o.Remote); if remote == "" { remote = "origin" }
			if rep.CurrentGit.Detached || rep.CurrentGit.Branch == "" { return nil, fmt.Errorf("portable handoff requires a named branch") }
			m["portable"], m["remote"], m["branch"], m["head"] = true, remote, rep.CurrentGit.Branch, rep.CurrentGit.Head
		}
	case domain.EventBlocked:
		if strings.TrimSpace(o.Reason) == "" || strings.TrimSpace(o.Next) == "" { return nil, fmt.Errorf("block requires --reason and --next") }
		m["reason"], m["next"] = strings.TrimSpace(o.Reason), strings.TrimSpace(o.Next)
	case domain.EventScope:
		a := strings.ToLower(strings.TrimSpace(o.ScopeAction)); m["action"], m["reason"] = a, strings.TrimSpace(o.Reason)
		switch a {
		case "propose":
			if strings.TrimSpace(o.Summary) == "" || strings.TrimSpace(o.Reason) == "" { return nil, fmt.Errorf("scope propose requires --summary and --reason") }
			m["summary"] = strings.TrimSpace(o.Summary)
		case "accept":
			if strings.TrimSpace(o.Reason) == "" { return nil, fmt.Errorf("scope accept requires --reason") }
			if rep.Aggregate.AcceptedContractDigest == "" { return nil, fmt.Errorf("scope accept requires an existing workflow") }
			m["previous_contract_digest"], m["new_contract_digest"] = rep.Aggregate.AcceptedContractDigest, rep.Contract.Digest
			if o.ProposalEventID != "" { m["proposal_event_id"] = o.ProposalEventID }
		case "reject":
			if strings.TrimSpace(o.ProposalEventID) == "" || strings.TrimSpace(o.Reason) == "" { return nil, fmt.Errorf("scope reject requires --proposal-event and --reason") }
			m["proposal_event_id"] = o.ProposalEventID
		default: return nil, fmt.Errorf("scope action must be propose|accept|reject")
		}
	case domain.EventReview:
		if _, ok := m["summary"]; !ok { m["summary"] = defaultSummary(t) }
		m["findings"] = stringAny(o.Findings)
	case domain.EventFix:
		if strings.TrimSpace(o.Summary) == "" { return nil, fmt.Errorf("fix requires --summary") }
	case domain.EventComplete:
		if strings.TrimSpace(o.Summary) == "" { return nil, fmt.Errorf("complete requires --summary") }
		m["validation"] = validationAny(validation)
		delivery := map[string]any{"commit": rep.CurrentGit.Head, "pushed": o.Pushed, "pr": nil}; if o.PR > 0 { delivery["pr"] = o.PR }; m["delivery"] = delivery
	case domain.EventCancel, domain.EventReopen:
		if strings.TrimSpace(o.Reason) == "" { return nil, fmt.Errorf("%s requires --reason", strings.ToLower(string(t))) }
		m["reason"] = strings.TrimSpace(o.Reason)
	}
	return m, nil
}

func plannedState(t domain.WorkflowEventType, before domain.WorkflowState, o MutationOptions) (domain.WorkflowState, error) {
	switch t {
	case domain.EventMigrate: return domain.LifecycleState(domain.ProjectLifecycle(before)), nil
	case domain.EventStart:
		if before != domain.StateReady { return "", fmt.Errorf("START requires READY") }; return domain.StateInProgress, nil
	case domain.EventResume:
		if before == domain.StatePaused || before == domain.StateDeferred || before == domain.StateBlocked { return domain.StateInProgress, nil }
		if before == domain.StateInProgress { return before, nil }
		return "", fmt.Errorf("RESUME not allowed from %s", before)
	case domain.EventPause:
		if before != domain.StateInProgress { return "", fmt.Errorf("PAUSE requires IN_PROGRESS") }; return domain.StatePaused, nil
	case domain.EventWait:
		if before != domain.StateInProgress { return "", fmt.Errorf("WAIT requires IN_PROGRESS") }; return domain.StateWaiting, nil
	case domain.EventRecheck:
		if before != domain.StateWaiting { return "", fmt.Errorf("RECHECK requires WAITING") }; if o.Resolved { return domain.StateInProgress, nil }; return before, nil
	case domain.EventDefer:
		if before != domain.StateInProgress { return "", fmt.Errorf("DEFER requires IN_PROGRESS") }; return domain.StateDeferred, nil
	case domain.EventCheckpoint, domain.EventHandoff, domain.EventScope:
		if before == domain.StateReady || before == domain.StateCompleted || before == domain.StateCancelled { return "", fmt.Errorf("%s not allowed from %s", t, before) }; return before, nil
	case domain.EventBlocked:
		if before != domain.StateInProgress && before != domain.StateWaiting && before != domain.StatePaused { return "", fmt.Errorf("BLOCKED not allowed from %s", before) }; return domain.StateBlocked, nil
	case domain.EventReview, domain.EventFix:
		if before != domain.StateInProgress { return "", fmt.Errorf("%s requires IN_PROGRESS", t) }; return before, nil
	case domain.EventComplete:
		if before != domain.StateInProgress { return "", fmt.Errorf("COMPLETE requires IN_PROGRESS") }; return domain.StateCompleted, nil
	case domain.EventCancel:
		if before == domain.StateCompleted || before == domain.StateCancelled { return "", fmt.Errorf("CANCEL requires non-terminal state") }; return domain.StateCancelled, nil
	case domain.EventReopen:
		if before != domain.StateCompleted && before != domain.StateCancelled { return "", fmt.Errorf("REOPEN requires COMPLETED/CANCELLED") }; return domain.StateReady, nil
	}
	return "", fmt.Errorf("unsupported event")
}

func completionGate(rep WorkflowReport, o MutationOptions) error {
	if rep.ContractChanged { return domain.NewWorkflowError("CONTRACT_CHANGED", "contract changed") }
	if len(rep.Aggregate.UnresolvedScopeProposals) > 0 { return domain.NewWorkflowError("SCOPE_UNRESOLVED", "unresolved scope proposals") }
	if rep.Aggregate.ReviewStatus == "findings" || rep.Aggregate.ReviewStatus == "pending" { return domain.NewWorkflowError("REVIEW_REQUIRED", "review has unresolved findings or pending re-review") }
	if rep.CurrentGit.Dirty { return domain.NewWorkflowError("STALE_GIT_STATE", "complete requires a clean worktree") }
	if rep.GitDrift.Changed { return domain.NewWorkflowError("STALE_GIT_STATE", "complete requires current Git to match latest persisted snapshot; checkpoint/review current state first") }
	req := domain.ValidationRequirements(rep.Contract)
	more := append([]domain.ValidationRecord(nil), o.Validation...)
	for i := range more { if more[i].ObservedHead == "" { more[i].ObservedHead = rep.CurrentGit.Head } }
	got := domain.MergeValidationRecords(domain.ValidationRecordsFromEvents(rep.Aggregate.Events), more)
	if miss := domain.MissingValidationsAtHead(req, got, rep.CurrentGit.Head); len(miss) > 0 { return domain.NewWorkflowError("VALIDATION_FAILED", "required validations not passing at current HEAD: "+strings.Join(miss, ",")) }
	return nil
}
