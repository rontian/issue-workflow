package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
	"github.com/rontian/issue-workflow/internal/result"
)

type WorkflowOptions struct {
	CWD    string
	Repo   string
	Remote string
	Host   string
	Issue  int
}

type RecoveryContext struct {
	State                    domain.WorkflowState     `json:"state"`
	Lifecycle                domain.Lifecycle         `json:"lifecycle"`
	Phase                    string                   `json:"phase,omitempty"`
	LastEventType            domain.WorkflowEventType `json:"last_event_type,omitempty"`
	LastSummary              string                   `json:"last_summary,omitempty"`
	LastNext                 string                   `json:"last_next,omitempty"`
	BlockReason              string                   `json:"block_reason,omitempty"`
	ContractChanged          bool                     `json:"contract_changed"`
	GitDrift                 domain.GitDrift          `json:"git_drift"`
	UnresolvedScopeProposals []string                 `json:"unresolved_scope_proposals"`
	SuggestedActions         []string                 `json:"suggested_actions"`
}

type WorkflowReport struct {
	Repository      domain.RepositoryIdentity `json:"repository_identity"`
	Issue           domain.GitHubIssueDetails `json:"issue"`
	Contract        domain.TaskContract       `json:"contract"`
	Aggregate       domain.WorkflowAggregate  `json:"aggregate"`
	Lifecycle       domain.Lifecycle          `json:"lifecycle"`
	Phase           string                    `json:"phase,omitempty"`
	ProtocolSource  string                    `json:"protocol_source"`
	CurrentGit      domain.GitSnapshot        `json:"current_git"`
	GitDrift        domain.GitDrift           `json:"git_drift"`
	ContractChanged bool                      `json:"contract_changed"`
	Recovery        RecoveryContext           `json:"recovery"`
}

type WorkflowService struct {
	Runner ports.Runner
	Git    ports.GitPort
	GitHub ports.WorkflowGitHubPort
}

func (s *WorkflowService) Status(ctx context.Context, opts WorkflowOptions) (result.CommandResult, int) {
	return s.read(ctx, "status", opts)
}

func (s *WorkflowService) ResumeDryRun(ctx context.Context, opts WorkflowOptions) (result.CommandResult, int) {
	r, code := s.read(ctx, "resume", opts)
	if !r.OK { return r, code }
	report := r.Data.(WorkflowReport)
	if report.Aggregate.State == domain.StateReady || report.Aggregate.State == domain.StateDone {
		rr := result.Failure("resume", "TRANSITION_BLOCKED", "当前 workflow state 不允许 resume dry-run", map[string]any{"state": report.Aggregate.State}, report)
		workflowCopyResultContext(&rr, r)
		return rr, result.ExitWorkflow
	}
	return r, code
}

func (s *WorkflowService) read(ctx context.Context, command string, opts WorkflowOptions) (result.CommandResult, int) {
	if opts.Issue <= 0 { return workflowFail(command, "INVALID_INVOCATION", "issue number must be > 0", nil, nil) }
	if s.Runner != nil {
		if _, err := s.Runner.LookPath("git"); err != nil { return workflowFail(command, "DEPENDENCY_MISSING", "git command not found", map[string]any{"dependency": "git"}, nil) }
		if _, err := s.Runner.LookPath("gh"); err != nil { return workflowFail(command, "DEPENDENCY_MISSING", "gh command not found", map[string]any{"dependency": "gh"}, nil) }
	}
	root, err := s.Git.RepositoryRoot(ctx, opts.CWD)
	if err != nil { return workflowFail(command, "NOT_A_GIT_REPOSITORY", "当前目录不是可用 Git repository", nil, nil) }
	repo, err := s.resolveRepository(ctx, root, opts)
	if err != nil { return workflowFail(command, domain.WorkflowErrorCode(err), err.Error(), nil, nil) }
	if err := s.GitHub.Authenticated(ctx, repo.Host); err != nil { return workflowFail(command, "AUTH_REQUIRED", "gh 未认证当前 GitHub host", map[string]any{"host": repo.Host}, nil) }
	remoteRepo, err := s.GitHub.Repository(ctx, repo)
	if err != nil { return workflowFail(command, "REPOSITORY_NOT_FOUND", "GitHub repository 不可访问", map[string]any{"repository": repo.FullName()}, nil) }
	if !strings.EqualFold(remoteRepo.NameWithOwner, repo.FullName()) { return workflowFail(command, "REPOSITORY_IDENTITY_MISMATCH", "gh repository identity 与本地解析不一致", map[string]any{"expected": repo.FullName(), "actual": remoteRepo.NameWithOwner}, nil) }
	issue, err := s.GitHub.IssueDetails(ctx, repo, opts.Issue)
	if err != nil { return workflowFail(command, "ISSUE_NOT_FOUND", "GitHub Issue 不可读取", map[string]any{"issue": opts.Issue}, nil) }
	contract, err := domain.ParseTaskContract(issue.Body)
	if err != nil { return workflowFail(command, workflowErrorCode(err, "TASK_CONTRACT_INVALID"), err.Error(), nil, nil) }
	comments, err := s.GitHub.IssueComments(ctx, repo, opts.Issue)
	if err != nil { return workflowFail(command, "GITHUB_UNAVAILABLE", "无法完整读取 Issue comments", nil, nil) }
	events := make([]domain.WorkflowEvent, 0)
	for _, comment := range comments {
		e, err := domain.ParseWorkflowEventComment(comment)
		if err != nil { return workflowFail(command, workflowErrorCode(err, "PROTOCOL_ERROR"), fmt.Sprintf("comment %d: %v", comment.ID, err), map[string]any{"comment_id": comment.ID}, nil) }
		if e != nil { events = append(events, *e) }
	}
	agg, err := domain.ReplayWorkflow(repo.FullName(), opts.Issue, events)
	if err != nil { return workflowFail(command, workflowErrorCode(err, "PROTOCOL_ERROR"), err.Error(), nil, nil) }
	currentGit, err := s.Git.Snapshot(ctx, root)
	if err != nil { return workflowFail(command, "ENVIRONMENT_INVALID", "无法读取当前 Git snapshot", nil, nil) }
	drift := domain.CompareGitSnapshot(agg.LatestGit, currentGit)
	contractChanged := len(events) > 0 && agg.AcceptedContractDigest != "" && contract.Digest != agg.AcceptedContractDigest
	recovery := buildRecovery(agg, drift, contractChanged)
	lifecycle := domain.ProjectLifecycle(agg.State)
	report := WorkflowReport{Repository: repo, Issue: issue, Contract: contract, Aggregate: agg, Lifecycle: lifecycle, Phase: domain.ProjectPhase(agg.State), ProtocolSource: "iw.workflow-event/v1", CurrentGit: currentGit, GitDrift: drift, ContractChanged: contractChanged, Recovery: recovery}
	r := result.Success(command, report)
	r.Repository = repo.FullName()
	n := opts.Issue
	r.Issue = &n
	r.State = string(agg.State)
	r.Lifecycle = string(lifecycle)
	r.NextActions = append([]string{}, recovery.SuggestedActions...)
	if contractChanged {
		r.Warnings = append(r.Warnings, "CONTRACT_CHANGED")
		r.Constraints = append(r.Constraints, "Issue Body digest 与已接受 contract 不一致；workflow mutation 必须等待 scope reconciliation")
	}
	if drift.Changed { r.Warnings = append(r.Warnings, "STALE_GIT_STATE") }
	if len(agg.UnresolvedScopeProposals) > 0 {
		r.Warnings = append(r.Warnings, "SCOPE_UNRESOLVED")
		r.Constraints = append(r.Constraints, "存在 unresolved scope proposal")
	}
	if agg.State == domain.StateBlocked { r.Constraints = append(r.Constraints, "workflow 当前处于 BLOCKED") }
	return r, result.ExitOK
}

func (s *WorkflowService) resolveRepository(ctx context.Context, root string, opts WorkflowOptions) (domain.RepositoryIdentity, error) {
	remoteIdentity, _, remoteErr := s.Git.ResolveRemote(ctx, root, opts.Remote)
	if opts.Repo == "" {
		if remoteErr != nil { return domain.RepositoryIdentity{}, domain.NewWorkflowError("REMOTE_NOT_FOUND", "无法从 Git remote 推导 repository") }
		if opts.Host != "" { remoteIdentity.Host = opts.Host }
		return remoteIdentity, nil
	}
	parts := strings.Split(opts.Repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" { return domain.RepositoryIdentity{}, domain.NewWorkflowError("INVALID_INVOCATION", "--repo 必须是 OWNER/REPO") }
	host := opts.Host
	if remoteErr == nil {
		if !strings.EqualFold(remoteIdentity.FullName(), opts.Repo) { return domain.RepositoryIdentity{}, domain.NewWorkflowError("REPOSITORY_IDENTITY_MISMATCH", "--repo 与当前 Git remote 不一致") }
		if host == "" { host = remoteIdentity.Host }
	}
	if host == "" { host = "github.com" }
	return domain.RepositoryIdentity{Host: host, Owner: parts[0], Repo: parts[1]}, nil
}

func buildRecovery(agg domain.WorkflowAggregate, drift domain.GitDrift, contractChanged bool) RecoveryContext {
	rc := RecoveryContext{State: agg.State, Lifecycle: domain.ProjectLifecycle(agg.State), Phase: domain.ProjectPhase(agg.State), ContractChanged: contractChanged, GitDrift: drift, UnresolvedScopeProposals: append([]string{}, agg.UnresolvedScopeProposals...), SuggestedActions: []string{}, BlockReason: agg.BlockReason}
	for i := len(agg.Events) - 1; i >= 0; i-- {
		e := agg.Events[i]
		if rc.LastEventType == "" { rc.LastEventType = e.EventType }
		if rc.LastSummary == "" { rc.LastSummary = workflowDataString(e.Data, "summary") }
		if rc.LastNext == "" { rc.LastNext = workflowDataString(e.Data, "next") }
		if rc.LastSummary != "" && rc.LastNext != "" { break }
	}
	if contractChanged { rc.SuggestedActions = append(rc.SuggestedActions, "检查 Issue Body 变更并执行显式 scope reconciliation") }
	if len(agg.UnresolvedScopeProposals) > 0 { rc.SuggestedActions = append(rc.SuggestedActions, "处理 unresolved scope proposal") }
	if drift.Changed { rc.SuggestedActions = append(rc.SuggestedActions, "检查当前 Git 状态与 latest persisted snapshot 的差异") }
	if agg.State == domain.StateDone {
		rc.LastNext = ""
		rc.SuggestedActions = append(rc.SuggestedActions, "任务已 COMPLETED，无恢复动作")
	} else if rc.LastNext != "" {
		rc.SuggestedActions = append(rc.SuggestedActions, rc.LastNext)
	} else {
		switch agg.State {
		case domain.StateReady: rc.SuggestedActions = append(rc.SuggestedActions, "使用 iw start 开始任务")
		case domain.StateInProgress: rc.SuggestedActions = append(rc.SuggestedActions, "继续实现当前 Scope")
		case domain.StateReview: rc.SuggestedActions = append(rc.SuggestedActions, "继续只读 Review")
		case domain.StateFix: rc.SuggestedActions = append(rc.SuggestedActions, "继续修复已授权 finding")
		case domain.StateBlocked: rc.SuggestedActions = append(rc.SuggestedActions, "解决 blocker 后再执行 resume")
		}
	}
	rc.SuggestedActions = workflowUniqueStrings(rc.SuggestedActions)
	return rc
}

func workflowDataString(m map[string]any, k string) string { v, _ := m[k].(string); return strings.TrimSpace(v) }
func workflowUniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in { if s != "" && !seen[s] { seen[s] = true; out = append(out, s) } }
	return out
}
func workflowErrorCode(err error, fallback string) string { if c := domain.WorkflowErrorCode(err); c != "" { return c }; return fallback }
func workflowFail(command, code, msg string, details map[string]any, data any) (result.CommandResult, int) { r := result.Failure(command, code, msg, details, data); return r, result.ExitCodeFor(code) }
func workflowCopyResultContext(dst *result.CommandResult, src result.CommandResult) {
	dst.Repository = src.Repository
	dst.Issue = src.Issue
	dst.State = src.State
	dst.Lifecycle = src.Lifecycle
	dst.Warnings = append([]string{}, src.Warnings...)
	dst.Constraints = append([]string{}, src.Constraints...)
	dst.NextActions = append([]string{}, src.NextActions...)
}
