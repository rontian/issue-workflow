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
	copyKey := func(k string) { if v, ok := data[k]; ok { out[k] = v } }
	switch t {
	case domain.EventMigrate:
		copyKey("from_schema"); copyKey("from_state"); copyKey("phase")
	case domain.EventStart, domain.EventResume:
		copyKey("summary")
	case domain.EventPause, domain.EventCancel, domain.EventReopen:
		copyKey("reason")
	case domain.EventWait, domain.EventDefer, domain.EventBlocked:
		copyKey("reason"); copyKey("next");if t==domain.EventWait{copyKey("conditions")}
	case domain.EventRecheck:
		copyKey("resolved"); copyKey("reason"); copyKey("conditions")
	case domain.EventCheckpoint:
		copyKey("summary"); copyKey("next"); copyKey("validation")
	case domain.EventHandoff:
		copyKey("summary"); copyKey("next"); copyKey("warnings"); copyKey("portable"); copyKey("remote"); copyKey("branch"); copyKey("head")
	case domain.EventScope:
		copyKey("action"); copyKey("summary"); copyKey("reason"); copyKey("proposal_event_id"); copyKey("new_contract_digest")
	case domain.EventReview:
		copyKey("summary"); copyKey("findings")
	case domain.EventFix:
		copyKey("summary")
	case domain.EventTaskComplete:
		copyKey("task_id"); copyKey("summary"); copyKey("validation")
	case domain.EventSettle:
		copyKey("item_id");copyKey("kind");copyKey("fact_status");copyKey("disposition");copyKey("target");copyKey("reason");copyKey("head");copyKey("decided_by");copyKey("owner_confirmed")
	case domain.EventComplete:
		copyKey("summary"); copyKey("validation")
		if d, ok := data["delivery"].(map[string]any); ok { out["delivery"] = map[string]any{"pushed": d["pushed"], "pr": d["pr"]} }
	}
	return out
}
