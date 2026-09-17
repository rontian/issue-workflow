package app

import "github.com/rontian/issue-workflow/internal/domain"

func gitEventSnapshot(g domain.GitSnapshot) *domain.EventGitSnapshot {
	var b *string
	if !g.Detached {
		x := g.Branch
		b = &x
	}
	return &domain.EventGitSnapshot{Branch: b, Head: g.Head, Detached: g.Detached, Dirty: g.Dirty, ChangedPaths: append([]string{}, g.ChangedPaths...)}
}

func logicalOperationData(t domain.WorkflowEventType, data map[string]any) map[string]any {
	out := map[string]any{}
	copyKey := func(k string) {
		if v, ok := data[k]; ok {
			out[k] = v
		}
	}
	switch t {
	case domain.EventStart, domain.EventResume, domain.EventReview, domain.EventFix:
		copyKey("summary")
		copyKey("findings")
	case domain.EventCheckpoint:
		copyKey("summary")
		copyKey("next")
		copyKey("validation")
	case domain.EventHandoff:
		copyKey("summary")
		copyKey("next")
		copyKey("warnings")
	case domain.EventBlocked:
		copyKey("reason")
		copyKey("next")
	case domain.EventScope:
		copyKey("action")
		copyKey("summary")
		copyKey("reason")
		copyKey("proposal_event_id")
		copyKey("new_contract_digest")
	case domain.EventFinal:
		copyKey("summary")
		copyKey("validation")
		if d, ok := data["delivery"].(map[string]any); ok {
			out["delivery"] = map[string]any{"pushed": d["pushed"], "pr": d["pr"]}
		}
	}
	return out
}
