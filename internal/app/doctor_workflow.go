package app

import (
	"context"
	"fmt"

	"github.com/rontian/issue-workflow/internal/result"
)

func (s *WorkflowService) AugmentDoctorIssue(ctx context.Context, opts WorkflowOptions, base result.CommandResult, baseExit int) (result.CommandResult, int) {
	report, ok := base.Data.(DoctorReport)
	if !ok {
		return base, baseExit
	}
	if !base.OK {
		report.Checks = append(report.Checks, Check{ID: "workflow.protocol", Category: "workflow", Status: CheckSkip, Required: true, Message: "环境检查未通过，跳过 workflow protocol 重建"})
		report.Summary = summarize(report.Checks)
		base.Data = report
		return base, baseExit
	}
	wr, code := s.Status(ctx, opts)
	if !wr.OK {
		ec := "PROTOCOL_ERROR"
		if wr.Error != nil {
			ec = wr.Error.Code
		}
		protocolFailure := ec == "TASK_CONTRACT_INVALID" || ec == "UNSUPPORTED_SCHEMA" || ec == "PROTOCOL_ERROR" || ec == "EVENT_CHAIN_BROKEN" || ec == "EVENT_FORK" || ec == "CONFLICT" || ec == "TRANSITION_BLOCKED"
		status := CheckSkip
		required := false
		if protocolFailure {
			status = CheckFail
			required = true
		}
		report.Checks = append(report.Checks, Check{ID: "workflow.protocol", Category: "workflow", Status: status, Required: required, Message: fmt.Sprintf("workflow read-model: %s", ec)})
		report.Summary = summarize(report.Checks)
		if protocolFailure {
			rr := result.Failure("doctor", ec, wr.Error.Message, wr.Error.Details, report)
			rr.Repository = base.Repository
			rr.Issue = base.Issue
			return rr, code
		}
		base.Data = report
		return base, baseExit
	}
	wf := wr.Data.(WorkflowReport)
	report.Checks = append(report.Checks, Check{ID: "workflow.protocol", Category: "workflow", Status: CheckPass, Required: true, Message: fmt.Sprintf("Task Contract 与 event chain 可重建，state=%s", wf.Aggregate.State)})
	if wf.ContractChanged {
		report.Checks = append(report.Checks, Check{ID: "workflow.contract", Category: "workflow", Status: CheckWarn, Required: false, Message: "Issue Body digest 与 latest accepted contract 不一致"})
	}
	if wf.GitDrift.Changed {
		report.Checks = append(report.Checks, Check{ID: "workflow.git-drift", Category: "workflow", Status: CheckWarn, Required: false, Message: "当前 Git 与 latest persisted snapshot 存在差异"})
	}
	report.Summary = summarize(report.Checks)
	base.Data = report
	return base, baseExit
}
