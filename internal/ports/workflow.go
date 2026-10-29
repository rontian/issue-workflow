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

type TaskListGitHubPort interface {
	TaskIssues(context.Context, domain.RepositoryIdentity) ([]domain.GitHubIssueDetails, error)
}

// GitHubIssueStatePort handles the native Issue open/closed fact separately
// from the canonical workflow Event Log. Agents must not bypass iw with gh.
type GitHubIssueStatePort interface {
    CloseIssue(context.Context,domain.RepositoryIdentity,int,string) error
    ReopenIssue(context.Context,domain.RepositoryIdentity,int) error
}
