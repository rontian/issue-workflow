package ports

import (
	"context"

	"github.com/rontian/issue-workflow/internal/domain"
)

type GitPort interface {
	Version(context.Context) (string, error)
	RepositoryRoot(context.Context, string) (string, error)
	ResolveRemote(context.Context, string, string) (domain.RepositoryIdentity, string, error)
	Snapshot(context.Context, string) (domain.GitSnapshot, error)
}

type PortableGitPort interface {
	GitPort
	RemoteBranchHead(context.Context, string, string, string) (string, error)
	FetchRemote(context.Context, string, string) error
	CommitExists(context.Context, string, string) bool
	RestoreBranch(context.Context, string, string, string, string) error
}

type GitHubPort interface {
	Version(context.Context) (string, error)
	Authenticated(context.Context, string) error
	Repository(context.Context, domain.RepositoryIdentity) (domain.GitHubRepository, error)
	Issue(context.Context, domain.RepositoryIdentity, int) (domain.GitHubIssue, error)
	Login(context.Context, string) error
}
