package app

import (
	"context"
	"strings"

	"github.com/rontian/issue-workflow/internal/domain"
)

func (s *WorkflowService) repositoryForCommand(ctx context.Context, o WorkflowOptions) (domain.RepositoryIdentity, error) {
	if s.Runner != nil {
		if _, err := s.Runner.LookPath("git"); err != nil {
			return domain.RepositoryIdentity{}, domain.NewWorkflowError("DEPENDENCY_MISSING", "git command not found")
		}
		if _, err := s.Runner.LookPath("gh"); err != nil {
			return domain.RepositoryIdentity{}, domain.NewWorkflowError("DEPENDENCY_MISSING", "gh command not found")
		}
	}
	root, err := s.Git.RepositoryRoot(ctx, o.CWD)
	if err != nil {
		return domain.RepositoryIdentity{}, domain.NewWorkflowError("NOT_A_GIT_REPOSITORY", "当前目录不是 Git repository")
	}
	repo, err := s.resolveRepository(ctx, root, o)
	if err != nil {
		return domain.RepositoryIdentity{}, err
	}
	if err := s.GitHub.Authenticated(ctx, repo.Host); err != nil {
		return domain.RepositoryIdentity{}, domain.NewWorkflowError("AUTH_REQUIRED", "gh 未认证当前 GitHub host")
	}
	remote, err := s.GitHub.Repository(ctx, repo)
	if err != nil {
		return domain.RepositoryIdentity{}, domain.NewWorkflowError("REPOSITORY_NOT_FOUND", "GitHub repository 不可访问")
	}
	if !strings.EqualFold(remote.NameWithOwner, repo.FullName()) {
		return domain.RepositoryIdentity{}, domain.NewWorkflowError("REPOSITORY_IDENTITY_MISMATCH", "gh repository identity 与 Git remote 不一致")
	}
	if !remote.HasIssuesEnabled {
		return domain.RepositoryIdentity{}, domain.NewWorkflowError("ENVIRONMENT_INVALID", "repository Issues 未启用")
	}
	return repo, nil
}
