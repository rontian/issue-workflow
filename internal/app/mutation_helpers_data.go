package app

import "github.com/rontian/issue-workflow/internal/domain"

func validationAny(v []domain.ValidationRecord) []any {
	out := make([]any, 0, len(v))
	for _, x := range v {
		out = append(out, map[string]any{"id": x.ID, "status": x.Status, "summary": x.Summary})
	}
	return out
}

func stringAny(v []string) []any {
	out := make([]any, 0, len(v))
	for _, x := range v {
		out = append(out, x)
	}
	return out
}

func findOperation(es []domain.WorkflowEvent, op string) *domain.WorkflowEvent {
	for i := range es {
		if es[i].OperationID == op {
			return &es[i]
		}
	}
	return nil
}

func domainString(m map[string]any, k string) string {
	v, _ := m[k].(string)
	return v
}

func parseEventsBestEffort(cs []domain.GitHubComment) ([]domain.WorkflowEvent, error) {
	out := []domain.WorkflowEvent{}
	for _, c := range cs {
		e, err := domain.ParseWorkflowEventComment(c)
		if err != nil {
			return nil, err
		}
		if e != nil {
			out = append(out, *e)
		}
	}
	return out, nil
}
