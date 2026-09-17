package cli

import (
	"context"

	"github.com/rontian/issue-workflow/internal/app"
	"github.com/rontian/issue-workflow/internal/result"
)

func (c *CLI) runMutation(ctx context.Context, command string, o options, wo app.WorkflowOptions) int {
	vals, err := parseValidationRecords(o.Validation)
	if err != nil { return c.render(result.Failure(command, "INVALID_INVOCATION", err.Error(), nil, nil), result.ExitInvalid, o.JSON) }
	mo := app.MutationOptions{WorkflowOptions: wo, DryRun: o.DryRun, OperationID: o.OperationID, RunID: o.RunID, Summary: o.Summary, Next: o.Next, Reason: o.Reason, Warnings: o.Warnings, Validation: vals, Findings: o.Findings, ScopeAction: o.ScopeAction, ProposalEventID: o.ProposalEventID, Pushed: o.Pushed, PR: o.PR}
	var r result.CommandResult
	var code int
	switch command {
	case "resume":
		if o.DryRun { r, code = c.Workflow.ResumeDryRun(ctx, wo) } else { r, code = c.Workflow.Resume(ctx, mo) }
	case "start": r, code = c.Workflow.Start(ctx, mo)
	case "checkpoint": r, code = c.Workflow.Checkpoint(ctx, mo)
	case "handoff": r, code = c.Workflow.Handoff(ctx, mo)
	case "block": r, code = c.Workflow.Block(ctx, mo)
	case "scope": r, code = c.Workflow.Scope(ctx, mo)
	case "review": r, code = c.Workflow.Review(ctx, mo)
	case "fix": r, code = c.Workflow.Fix(ctx, mo)
	case "final": r, code = c.Workflow.Final(ctx, mo)
	default:
		return c.render(result.Failure(command, "INVALID_INVOCATION", "unknown command", nil, nil), result.ExitInvalid, o.JSON)
	}
	return c.render(r, code, o.JSON)
}
