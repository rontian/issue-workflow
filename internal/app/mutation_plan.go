package app

import (
	"fmt"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
)

func buildEventData(t domain.WorkflowEventType, o MutationOptions, rep WorkflowReport) (map[string]any, error) {
	m := map[string]any{}
	if strings.TrimSpace(o.Summary) != "" {
		m["summary"] = strings.TrimSpace(o.Summary)
	}
	switch t {
	case domain.EventStart, domain.EventResume, domain.EventReview, domain.EventFix:
		if _, ok := m["summary"]; !ok {
			m["summary"] = defaultSummary(t)
		}
		if t == domain.EventReview && len(o.Findings) > 0 {
			m["findings"] = append([]string(nil), o.Findings...)
		}
	case domain.EventCheckpoint:
		if strings.TrimSpace(o.Summary) == "" || strings.TrimSpace(o.Next) == "" {
			return nil, fmt.Errorf("checkpoint requires --summary and --next")
		}
		m["next"] = strings.TrimSpace(o.Next)
		m["validation"] = validationAny(o.Validation)
	case domain.EventHandoff:
		if strings.TrimSpace(o.Summary) == "" || strings.TrimSpace(o.Next) == "" {
			return nil, fmt.Errorf("handoff requires --summary and --next")
		}
		m["next"] = strings.TrimSpace(o.Next)
		m["warnings"] = stringAny(o.Warnings)
	case domain.EventBlocked:
		if strings.TrimSpace(o.Reason) == "" || strings.TrimSpace(o.Next) == "" {
			return nil, fmt.Errorf("block requires --reason and --next")
		}
		m["reason"] = strings.TrimSpace(o.Reason)
		m["next"] = strings.TrimSpace(o.Next)
		m["resume_state"] = string(rep.Aggregate.State)
	case domain.EventScope:
		a := strings.ToLower(strings.TrimSpace(o.ScopeAction))
		m["action"] = a
		m["reason"] = strings.TrimSpace(o.Reason)
		switch a {
		case "propose":
			if strings.TrimSpace(o.Summary) == "" || strings.TrimSpace(o.Reason) == "" {
				return nil, fmt.Errorf("scope propose requires --summary and --reason")
			}
			m["summary"] = strings.TrimSpace(o.Summary)
		case "accept":
			if strings.TrimSpace(o.Reason) == "" {
				return nil, fmt.Errorf("scope accept requires --reason")
			}
			if rep.Aggregate.AcceptedContractDigest == "" {
				return nil, fmt.Errorf("scope accept requires an existing workflow")
			}
			m["previous_contract_digest"] = rep.Aggregate.AcceptedContractDigest
			m["new_contract_digest"] = rep.Contract.Digest
			if o.ProposalEventID != "" {
				m["proposal_event_id"] = o.ProposalEventID
			}
		case "reject":
			if strings.TrimSpace(o.ProposalEventID) == "" || strings.TrimSpace(o.Reason) == "" {
				return nil, fmt.Errorf("scope reject requires --proposal-event and --reason")
			}
			m["proposal_event_id"] = o.ProposalEventID
		default:
			return nil, fmt.Errorf("scope action must be propose|accept|reject")
		}
	case domain.EventFinal:
		if strings.TrimSpace(o.Summary) == "" {
			return nil, fmt.Errorf("final requires --summary")
		}
		m["validation"] = validationAny(o.Validation)
		delivery := map[string]any{"commit": rep.CurrentGit.Head, "pushed": o.Pushed, "pr": nil}
		if o.PR > 0 {
			delivery["pr"] = o.PR
		}
		m["delivery"] = delivery
	}
	return m, nil
}

func plannedState(t domain.WorkflowEventType, before, resume domain.WorkflowState, o MutationOptions) (domain.WorkflowState, error) {
	switch t {
	case domain.EventStart:
		if before != domain.StateReady {
			return "", fmt.Errorf("START requires READY")
		}
		return domain.StateInProgress, nil
	case domain.EventResume:
		if before == domain.StateBlocked {
			if resume == "" {
				return "", fmt.Errorf("BLOCKED resume_state missing")
			}
			return resume, nil
		}
		if before == domain.StateInProgress || before == domain.StateReview || before == domain.StateFix {
			return before, nil
		}
		return "", fmt.Errorf("RESUME not allowed from %s", before)
	case domain.EventCheckpoint, domain.EventHandoff, domain.EventScope:
		if before == domain.StateReady || before == domain.StateDone {
			return "", fmt.Errorf("%s not allowed from %s", t, before)
		}
		return before, nil
	case domain.EventBlocked:
		if before != domain.StateInProgress && before != domain.StateReview && before != domain.StateFix {
			return "", fmt.Errorf("BLOCKED not allowed from %s", before)
		}
		return domain.StateBlocked, nil
	case domain.EventReview:
		if before != domain.StateInProgress && before != domain.StateFix {
			return "", fmt.Errorf("REVIEW requires IN_PROGRESS/FIX")
		}
		return domain.StateReview, nil
	case domain.EventFix:
		if before != domain.StateReview {
			return "", fmt.Errorf("FIX requires REVIEW")
		}
		return domain.StateFix, nil
	case domain.EventFinal:
		if before != domain.StateInProgress && before != domain.StateReview && before != domain.StateFix {
			return "", fmt.Errorf("FINAL requires active state")
		}
		return domain.StateDone, nil
	}
	return "", fmt.Errorf("unsupported event")
}

func finalGate(rep WorkflowReport, o MutationOptions) error {
	if rep.ContractChanged {
		return domain.NewWorkflowError("CONTRACT_CHANGED", "contract changed")
	}
	if len(rep.Aggregate.UnresolvedScopeProposals) > 0 {
		return domain.NewWorkflowError("SCOPE_UNRESOLVED", "unresolved scope proposals")
	}
	if rep.CurrentGit.Dirty {
		return domain.NewWorkflowError("STALE_GIT_STATE", "Final requires a clean worktree")
	}
	if rep.GitDrift.Changed {
		return domain.NewWorkflowError("STALE_GIT_STATE", "Final requires current Git to match latest persisted snapshot; checkpoint/review current state first")
	}
	req := domain.ValidationRequirements(rep.Contract)
	got := domain.MergeValidationRecords(domain.ValidationRecordsFromEvents(rep.Aggregate.Events), o.Validation)
	if miss := domain.MissingValidations(req, got); len(miss) > 0 {
		return domain.NewWorkflowError("VALIDATION_FAILED", "required validations not passing: "+strings.Join(miss, ","))
	}
	return nil
}
