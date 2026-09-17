package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

func (s *WorkflowService) Migrate(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "migrate", domain.EventMigrate, o) }
func (s *WorkflowService) Start(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "start", domain.EventStart, o) }
func (s *WorkflowService) Resume(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "resume", domain.EventResume, o) }
func (s *WorkflowService) Pause(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "pause", domain.EventPause, o) }
func (s *WorkflowService) Wait(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "wait", domain.EventWait, o) }
func (s *WorkflowService) Recheck(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "recheck", domain.EventRecheck, o) }
func (s *WorkflowService) Defer(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "defer", domain.EventDefer, o) }
func (s *WorkflowService) Checkpoint(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "checkpoint", domain.EventCheckpoint, o) }
func (s *WorkflowService) Handoff(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "handoff", domain.EventHandoff, o) }
func (s *WorkflowService) Block(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "block", domain.EventBlocked, o) }
func (s *WorkflowService) Review(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "review", domain.EventReview, o) }
func (s *WorkflowService) Fix(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "fix", domain.EventFix, o) }
func (s *WorkflowService) Complete(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "complete", domain.EventComplete, o) }
func (s *WorkflowService) Final(ctx context.Context, o MutationOptions) (result.CommandResult, int) {
	r, code := s.mutate(ctx, "final", domain.EventComplete, o)
	r.CanonicalCommand = "complete"
	if r.OK { r.Warnings = append(r.Warnings, "COMPAT_ALIAS_FINAL_USE_COMPLETE") }
	return r, code
}
func (s *WorkflowService) Cancel(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "cancel", domain.EventCancel, o) }
func (s *WorkflowService) Reopen(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "reopen", domain.EventReopen, o) }
func (s *WorkflowService) Scope(ctx context.Context, o MutationOptions) (result.CommandResult, int) { return s.mutate(ctx, "scope", domain.EventScope, o) }

func (s *WorkflowService) mutate(ctx context.Context, command string, t domain.WorkflowEventType, o MutationOptions) (result.CommandResult, int) {
	rr, code := s.read(ctx, command, o.WorkflowOptions)
	if !rr.OK { return rr, code }
	rep := rr.Data.(WorkflowReport)
	if rep.Issue.State != "OPEN" && t != domain.EventReopen && t != domain.EventMigrate { return mutationFailFrom(command, "GITHUB_ISSUE_CLOSED", "GitHub Issue is closed; reopen it explicitly before active workflow mutation", rep, rr) }
	if rep.Aggregate.Protocol == "v1" && t != domain.EventMigrate { return mutationFailFrom(command, "MIGRATION_REQUIRED", "v1 workflow must be explicitly migrated before v2 mutation; run iw migrate <issue>", rep, rr) }
	if t == domain.EventMigrate && rep.Aggregate.Protocol != "v1" { return mutationFailFrom(command, "TRANSITION_BLOCKED", "migrate requires existing v1 workflow history", rep, rr) }
	if t != domain.EventMigrate && (t != domain.EventScope || strings.ToLower(o.ScopeAction) != "accept") && rep.ContractChanged { return mutationFailFrom(command, "CONTRACT_CHANGED", "Issue Body changed; scope accept or restore contract before mutation", rep, rr) }
	if t == domain.EventHandoff && o.Portable {
		if rep.CurrentGit.Dirty { return mutationFailFrom(command, "STALE_GIT_STATE", "portable handoff requires clean worktree", rep, rr) }
		if rep.CurrentGit.Detached || rep.CurrentGit.Branch == "" { return mutationFailFrom(command, "STALE_GIT_STATE", "portable handoff requires named branch", rep, rr) }
		_, remote, err := s.Git.ResolveRemote(ctx, rep.CurrentGit.RepositoryRoot, o.Remote)
		if err != nil { return mutationFailFrom(command, "REMOTE_NOT_FOUND", "portable handoff cannot resolve Git remote", rep, rr) }
		o.Remote = remote
		pg, ok := s.Git.(interface{ RemoteBranchHead(context.Context, string, string, string) (string, error) })
		if !ok { return mutationFailFrom(command, "ENVIRONMENT_INVALID", "Git adapter cannot verify portable handoff", rep, rr) }
		remoteHead, err := pg.RemoteBranchHead(ctx, rep.CurrentGit.RepositoryRoot, remote, rep.CurrentGit.Branch)
		if err != nil || remoteHead != rep.CurrentGit.Head { return mutationFailFrom(command, "STALE_GIT_STATE", "portable handoff requires exact HEAD to be published on remote branch", rep, rr) }
	}
	data, err := buildEventData(t, o, rep)
	if err != nil { return mutationFailFrom(command, "INVALID_INVOCATION", err.Error(), rep, rr) }
	fingerprint := domain.OperationFingerprint(t, logicalOperationData(t, data)); data["_operation_fingerprint"] = fingerprint
	op := strings.TrimSpace(o.OperationID)
	if op == "" { op, err = domain.NewID(); if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", "cannot generate operation id", rep, rr) } }
	if existing := findOperation(rep.Aggregate.Events, op); existing != nil {
		if existing.EventType != t || domainString(existing.Data, "_operation_fingerprint") != fingerprint { return mutationFailFrom(command, "CONFLICT", "operation_id already exists with different operation", rep, rr) }
		return idempotentSuccess(command, op, rep, rr)
	}
	run := strings.TrimSpace(o.RunID); if run == "" { run, err = domain.NewID(); if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", "cannot generate run id", rep, rr) } }
	eid, err := domain.NewID(); if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", "cannot generate event id", rep, rr) }
	before := rep.Aggregate.State
	if t == domain.EventMigrate { before = domain.LifecycleState(domain.ProjectLifecycle(rep.Aggregate.State)) }
	after, err := plannedState(t, rep.Aggregate.State, o); if err != nil { return mutationFailFrom(command, "TRANSITION_BLOCKED", err.Error(), rep, rr) }
	digest := rep.Aggregate.AcceptedContractDigest; if digest == "" { digest = rep.Contract.Digest }
	if t == domain.EventScope && strings.ToLower(o.ScopeAction) == "accept" { digest = rep.Contract.Digest }
	var parent *string; if rep.Aggregate.LastEvent != nil { x := rep.Aggregate.LastEvent.EventID; parent = &x }
	ev := domain.WorkflowEvent{Schema: domain.WorkflowEventSchemaV2, EventID: eid, ParentEventID: parent, OperationID: op, RunID: run, EventType: t, OccurredAt: time.Now().UTC().Format(time.RFC3339), Repository: rep.Repository.FullName(), Issue: rep.Issue.Number, ContractDigest: digest, StateBefore: before, StateAfter: after, Data: data}
	if t != domain.EventScope { ev.Git = gitEventSnapshot(rep.CurrentGit) }
	trial := append(append([]domain.WorkflowEvent(nil), rep.Aggregate.Events...), ev)
	if _, err := domain.ReplayWorkflow(rep.Repository.FullName(), rep.Issue.Number, trial); err != nil { return mutationFailFrom(command, workflowErrorCode(err, "PROTOCOL_ERROR"), err.Error(), rep, rr) }
	if t == domain.EventComplete { if err := completionGate(rep, o); err != nil { return mutationFailFrom(command, workflowErrorCode(err, "VALIDATION_FAILED"), err.Error(), rep, rr) } }
	comment, err := domain.RenderWorkflowEvent(ev); if err != nil { return mutationFailFrom(command, "INTERNAL_ERROR", err.Error(), rep, rr) }
	preview := MutationPreview{Event: ev, Comment: comment, OperationID: op, DryRun: o.DryRun}
	if o.DryRun {
		r := result.Success(command, preview); copyWorkflowContext(&r, rr); r.State, r.Lifecycle = string(after), string(domain.ProjectLifecycle(after)); return r, result.ExitOK
	}
	if err := s.GitHub.AppendIssueComment(ctx, rep.Repository, rep.Issue.Number, comment); err != nil {
		comments, e2 := s.GitHub.IssueComments(ctx, rep.Repository, rep.Issue.Number)
		if e2 == nil {
			events, _ := parseEventsBestEffort(comments)
			if ex := findOperation(events, op); ex != nil && ex.EventType == t && domainString(ex.Data, "_operation_fingerprint") == fingerprint {
				sr, sc := s.Status(ctx, o.WorkflowOptions); if sr.OK { sr.Command = command; sr.CanonicalCommand = command; sr.Warnings = append(sr.Warnings, "UNKNOWN_OUTCOME_RECOVERED"); sr.NextActions = nextAfterMutation(t, sr.Lifecycle) }; return sr, sc
			}
		}
		r := result.Failure(command, "GITHUB_UNAVAILABLE", "workflow write outcome unknown; retry with the same --operation-id", map[string]any{"operation_id": op, "outcome_unknown": true}, preview); copyWorkflowContext(&r, rr); return r, result.ExitGitHub
	}
	sr, sc := s.Status(ctx, o.WorkflowOptions); if !sr.OK { return sr, sc }
	sr.Command, sr.CanonicalCommand = command, command
	sr.NextActions = nextAfterMutation(t, sr.Lifecycle)
	return sr, result.ExitOK
}

func portableError(remote, branch, head string) error { return fmt.Errorf("remote %s branch %s does not expose HEAD %s", remote, branch, head) }
