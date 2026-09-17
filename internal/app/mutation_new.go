package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/result"
)

func (s *WorkflowService) NewIssue(ctx context.Context, o NewIssueOptions) (result.CommandResult, int) {
	if strings.TrimSpace(o.Title) == "" {
		return workflowFail("new", "INVALID_INVOCATION", "--title is required", nil, nil)
	}
	id := strings.TrimSpace(o.ContractID)
	if id == "" {
		var err error
		id, err = domain.NewID()
		if err != nil {
			return workflowFail("new", "INTERNAL_ERROR", "无法生成 contract_id", nil, nil)
		}
	}
	body, err := domain.RenderTaskContract(domain.TaskContractInput{ContractID: id, Goal: o.Goal, Scope: o.Scope, OutOfScope: o.OutOfScope, Constraints: o.Constraints, AcceptanceCriteria: o.Acceptance, Validation: o.Validation, Dependencies: o.Dependencies})
	if err != nil {
		return workflowFail("new", workflowErrorCode(err, "INVALID_INVOCATION"), err.Error(), nil, nil)
	}
	repo, err := s.repositoryForCommand(ctx, o.WorkflowOptions)
	if err != nil {
		return workflowFail("new", workflowErrorCode(err, "ENVIRONMENT_INVALID"), err.Error(), nil, nil)
	}
	preview := NewIssuePreview{Title: o.Title, Body: body, ContractID: id, DryRun: o.DryRun}
	if o.DryRun {
		r := result.Success("new", preview)
		r.Repository = repo.FullName()
		return r, result.ExitOK
	}
	issue, err := s.GitHub.CreateIssue(ctx, repo, o.Title, body)
	if err != nil {
		r := result.Failure("new", "GITHUB_UNAVAILABLE", "Issue create outcome unknown; inspect GitHub before retrying", map[string]any{"contract_id": id, "outcome_unknown": true}, preview)
		r.Repository = repo.FullName()
		return r, result.ExitGitHub
	}
	preview.Issue = &issue
	r := result.Success("new", preview)
	r.Repository = repo.FullName()
	n := issue.Number
	r.Issue = &n
	r.State = string(domain.StateReady)
	r.NextActions = []string{fmt.Sprintf("iw start %d", issue.Number)}
	return r, result.ExitOK
}
