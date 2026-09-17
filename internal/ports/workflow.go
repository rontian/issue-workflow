package ports

import (
	"context"

	"github.com/rontian/issue-workflow/internal/domain"
)

type WorkflowGitHubPort interface {
	Authenticated(context.Context, string) error
	Repository(context.Context, domain.RepositoryIdentity) (domain.GitHubRepository, error)
	IssueDetails(context.Context, domain.RepositoryIdentity, int) (domain.GitHubIssueDetails, error)
	IssueComments(context.Context, domain.RepositoryIdentity, int) ([]domain.GitHubComment, error)
	CreateIssue(context.Context, domain.RepositoryIdentity, string, string) (domain.GitHubIssueDetails, error)
	AppendIssueComment(context.Context, domain.RepositoryIdentity, int, string) error
}
