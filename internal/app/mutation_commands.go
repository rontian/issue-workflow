package app

import (
	"context"
	"strings"
	"time"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

func (s *WorkflowService) Start(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "start", domain.EventStart, o) }
func (s *WorkflowService) Resume(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "resume", domain.EventResume, o) }
func (s *WorkflowService) Checkpoint(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "checkpoint", domain.EventCheckpoint, o) }
func (s *WorkflowService) Handoff(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "handoff", domain.EventHandoff, o) }
func (s *WorkflowService) Block(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "block", domain.EventBlocked, o) }
func (s *WorkflowService) Review(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "review", domain.EventReview, o) }
func (s *WorkflowService) Fix(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "fix", domain.EventFix, o) }
func (s *WorkflowService) Final(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "final", domain.EventFinal, o) }
func (s *WorkflowService) Scope(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "scope", domain.EventScope, o) }

func (s *WorkflowService) mutate(ctx context.Context, command string, t domain.WorkflowEventType, o MutationOptions) (result.CommandResult, int) {
	rr, code := s.read(ctx, command, o.WorkflowOptions)
	if !rr.OK { return rr, code }
	rep := rr.Data.(WorkflowReport)
	if rep.Issue.State != "OPEN" { return mutationFailFrom(command, "TRANSITION_BLOCKED", "Issue is not open", rep, rr) }
	if t != domain.EventScope || strings.ToLower(o.ScopeAction) != "accept" {
		if rep.ContractChanged { return mutationFailFrom(command, "CONTRACT_CHANGED", "Issue Body changed; scope accept or restore contract before mutation", rep, rr) }
	}
	data, err := buildEventData(t, o, rep)
	if err != nil { return mutationFailFrom(command, "INVALID_INVOCATION", err.Error(), rep, rr) }
	fingerprint := domain.OperationFingerprint(t, logicalOperationData(t, data))
	data["_operation_fingerprint"] = fingerprint
	op := strings.TrimSpace(o.OperationID)
	if op == "" { op, err = domain.NewID(); if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", "cannot generate operation id", rep, rr) } }
	if existing := findOperation(rep.Aggregate.Events, op); existing != nil {
		if existing.EventType != t || domainString(existing.Data, "_operation_fingerprint") != fingerprint { return mutationFailFrom(command, "CONFLICT", "operation_id already exists with different operation", rep, rr) }
		return idempotentSuccess(command, op, rep, rr)
	}
	run := strings.TrimSpace(o.RunID)
	if run == "" { run, err = domain.NewID(); if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", "cannot generate run id", rep, rr) } }
	eid, err := domain.NewID()
	if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", "cannot generate event id", rep, rr) }
	before := rep.Aggregate.State
	after, err := plannedState(t, before, rep.Aggregate.ResumeState, o)
	if err != nil { return mutationFailFrom(command, "TRANSITION_BLOCKED", err.Error(), rep, rr) }
	digest := rep.Aggregate.AcceptedContractDigest
	if digest == "" { digest = rep.Contract.Digest }
	if t == domain.EventScope && strings.ToLower(o.ScopeAction) == "accept" { digest = rep.Contract.Digest }
	var parent *string
	if rep.Aggregate.LastEvent != nil { x := rep.Aggregate.LastEvent.EventID; parent = &x }
	ev := domain.WorkflowEvent{Schema: domain.WorkflowEventSchema, EventID: eid, ParentEventID: parent, OperationID: op, RunID: run, EventType: t, OccurredAt: time.Now().UTC().Format(time.RFC3339), Repository: rep.Repository.FullName(), Issue: rep.Issue.Number, ContractDigest: digest, StateBefore: before, StateAfter: after, Data: data}
	if t != domain.EventScope { ev.Git = gitEventSnapshot(rep.CurrentGit) }
	trial := append(append([]domain.WorkflowEvent(nil), rep.Aggregate.Events...), ev)
	if _, err := domain.ReplayWorkflow(rep.Repository.FullName(), rep.Issue.Number, trial); err != nil { return mutationFailFrom(command, workflowErrorCode(err, "PROTOCOL_ERROR"), err.Error(), rep, rr) }
	if t == domain.EventFinal { if err := finalGate(rep, o); err != nil { return mutationFailFrom(command, workflowErrorCode(err, "VALIDATION_FAILED"), err.Error(), rep, rr) } }
	comment, err := domain.RenderWorkflowEvent(ev)
	if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", err.Error(), rep, rr) }
	preview := MutationPreview{Event: ev, Comment: comment, OperationID: op, DryRun: o.DryRun}
	if o.DryRun {
		r := result.Success(command, preview); copyWorkflowContext(&r, rr); r.State = string(after); return r, result.ExitOK
	}
	if err := s.GitHub.AppendIssueComment(ctx, rep.Repository, rep.Issue.Number, comment); err != nil {
		comments, e2 := s.GitHub.IssueComments(ctx, rep.Repository, rep.Issue.Number)
		if e2 == nil {
			events, _ := parseEventsBestEffort(comments)
			if ex := findOperation(events, op); ex != nil && ex.EventType == t && domainString(ex.Data, "_operation_fingerprint") == fingerprint {
				sr, sc := s.Status(ctx, o.WorkflowOptions)
				if sr.OK { sr.Command = command; sr.Warnings = append(sr.Warnings, "UNKNOWN_OUTCOME_RECOVERED"); sr.NextActions = nextAfterMutation(t, sr.State) }
				return sr, sc
			}
		}
		r := result.Failure(command, "GITHUB_UNAVAILABLE", "workflow write outcome unknown; retry with the same --operation-id", map[string]any{"operation_id": op, "outcome_unknown": true}, preview)
		copyWorkflowContext(&r, rr)
		return r, result.ExitGitHub
	}
	sr, sc := s.Status(ctx, o.WorkflowOptions)
	if !sr.OK { return sr, sc }
	sr.Command = command
	sr.NextActions = nextAfterMutation(t, sr.State)
	return sr, result.ExitOK
}
