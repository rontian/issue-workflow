package app

import (
	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

func idempotentSuccess(command, op string, rep WorkflowReport, base result.CommandResult) (result.CommandResult, int) {
	r := result.Success(command, rep); copyWorkflowContext(&r, base); r.Warnings = append(r.Warnings, "IDEMPOTENT_REPLAY"); r.NextActions = append([]string{"operation already persisted: " + op}, r.NextActions...); return r, result.ExitOK
}

func mutationFailFrom(command, code, msg string, rep WorkflowReport, base result.CommandResult) (result.CommandResult, int) {
	r := result.Failure(command, code, msg, nil, rep); copyWorkflowContext(&r, base); return r, result.ExitCodeFor(code)
}

func copyWorkflowContext(dst *result.CommandResult, src result.CommandResult) {
	dst.Repository = src.Repository; dst.Issue = src.Issue; dst.State = src.State; dst.Lifecycle = src.Lifecycle
	dst.Warnings = append([]string{}, src.Warnings...); dst.Constraints = append([]string{}, src.Constraints...); dst.NextActions = append([]string{}, src.NextActions...)
}

func nextAfterMutation(t domain.WorkflowEventType, lifecycle string) []string {
	switch t {
	case domain.EventMigrate: return []string{"workflow migrated to v2; continue with the intended lifecycle command"}
	case domain.EventStart, domain.EventResume: return []string{"continue authorized work; checkpoint before handoff or complete"}
	case domain.EventPause: return []string{"resume with iw resume <issue>"}
	case domain.EventWait: return []string{"recheck the external condition with iw recheck <issue>"}
	case domain.EventRecheck: if lifecycle == string(domain.LifecycleWaiting) { return []string{"condition still pending; recheck later"} }; return []string{"continue authorized work"}
	case domain.EventDefer: return []string{"resume the later segment with iw resume <issue> when authorized"}
	case domain.EventCheckpoint: return []string{"continue work, review, or hand off"}
	case domain.EventHandoff: return []string{"next session: iw context <issue> then iw resume <issue> if lifecycle requires it"}
	case domain.EventBlocked: return []string{"resolve blocker then iw resume <issue>"}
	case domain.EventReview: return []string{"iw fix <issue> if findings exist; otherwise complete when gates pass"}
	case domain.EventFix: return []string{"apply fixes then iw review <issue>"}
	case domain.EventComplete: return []string{"workflow COMPLETED; GitHub Issue closure remains explicit"}
	case domain.EventCancel: return []string{"workflow CANCELLED; reopen explicitly if work resumes"}
	case domain.EventReopen: return []string{"workflow returned to READY; use iw start <issue>"}
	default: return []string{}
	}
}

func defaultSummary(t domain.WorkflowEventType) string {
	return map[domain.WorkflowEventType]string{domain.EventStart: "开始执行任务", domain.EventResume: "恢复执行任务", domain.EventReview: "执行 Review", domain.EventFix: "修复 Review finding"}[t]
}
