package app

import (
	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

func idempotentSuccess(command, op string, rep WorkflowReport, base result.CommandResult) (result.CommandResult, int) {
	r := result.Success(command, rep)
	copyWorkflowContext(&r, base)
	r.Warnings = append(r.Warnings, "IDEMPOTENT_REPLAY")
	r.NextActions = append([]string{"operation already persisted: " + op}, r.NextActions...)
	return r, result.ExitOK
}

func mutationFailFrom(command, code, msg string, rep WorkflowReport, base result.CommandResult) (result.CommandResult, int) {
	r := result.Failure(command, code, msg, nil, rep)
	copyWorkflowContext(&r, base)
	return r, result.ExitCodeFor(code)
}

func copyWorkflowContext(dst *result.CommandResult, src result.CommandResult) {
	dst.Repository = src.Repository
	dst.Issue = src.Issue
	dst.State = src.State
	dst.Warnings = append([]string{}, src.Warnings...)
	dst.Constraints = append([]string{}, src.Constraints...)
	dst.NextActions = append([]string{}, src.NextActions...)
}

func nextAfterMutation(t domain.WorkflowEventType, state string) []string {
	switch t {
	case domain.EventStart, domain.EventResume:
		return []string{"continue authorized work; checkpoint before handoff/final"}
	case domain.EventCheckpoint:
		return []string{"continue work or enter review"}
	case domain.EventHandoff:
		return []string{"next session: iw resume <issue>"}
	case domain.EventBlocked:
		return []string{"resolve blocker then iw resume <issue>"}
	case domain.EventReview:
		return []string{"iw fix <issue> if findings require changes, otherwise iw final <issue>"}
	case domain.EventFix:
		return []string{"apply fixes then iw review <issue>"}
	case domain.EventFinal:
		return []string{"workflow DONE; delivery/Issue closure remains explicit"}
	default:
		return []string{}
	}
}

func defaultSummary(t domain.WorkflowEventType) string {
	return map[domain.WorkflowEventType]string{
		domain.EventStart:  "开始执行任务",
		domain.EventResume: "恢复执行任务",
		domain.EventReview: "进入 Review",
		domain.EventFix:    "进入 Fix",
	}[t]
}
