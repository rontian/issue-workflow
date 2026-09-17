package ghcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/rontian/issue-workflow/internal/domain"
	"github.com/rontian/issue-workflow/internal/ports"
)

type Client struct {
	runner ports.Runner
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func New(r ports.Runner, stdin io.Reader, stdout, stderr io.Writer) *Client {
	return &Client{runner: r, stdin: stdin, stdout: stdout, stderr: stderr}
}
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) { return c.runWithEnv(ctx, nil, args...) }
func (c *Client) runWithEnv(ctx context.Context, env []string, args ...string) ([]byte, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 20*time.Second); defer cancel()
	res, err := c.runner.Run(cmdCtx, ports.Command{Name: "gh", Args: args, Env: env})
	if err != nil { return nil, err }
	return res.Stdout, nil
}
func repoEnv(repo domain.RepositoryIdentity) []string { if repo.Host == "" { return nil }; return []string{"GH_HOST=" + repo.Host} }
func (c *Client) Version(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "--version"); if err != nil { return "", err }
	return strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0], nil
}
func (c *Client) Authenticated(ctx context.Context, host string) error {
	args := []string{"auth", "status"}; if host != "" { args = append(args, "--hostname", host) }
	_, err := c.run(ctx, args...); return err
}
func target(repo domain.RepositoryIdentity) string { return repo.Owner + "/" + repo.Repo }
func (c *Client) Repository(ctx context.Context, repo domain.RepositoryIdentity) (domain.GitHubRepository, error) {
	out, err := c.runWithEnv(ctx, repoEnv(repo), "repo", "view", target(repo), "--json", "nameWithOwner,url,hasIssuesEnabled")
	if err != nil { return domain.GitHubRepository{}, err }
	var p struct { NameWithOwner string `json:"nameWithOwner"`; URL string `json:"url"`; HasIssuesEnabled bool `json:"hasIssuesEnabled"` }
	if err := json.Unmarshal(out, &p); err != nil { return domain.GitHubRepository{}, fmt.Errorf("decode gh repo view: %w", err) }
	return domain.GitHubRepository{NameWithOwner: p.NameWithOwner, URL: p.URL, HasIssuesEnabled: p.HasIssuesEnabled}, nil
}
func (c *Client) Issue(ctx context.Context, repo domain.RepositoryIdentity, number int) (domain.GitHubIssue, error) {
	out, err := c.runWithEnv(ctx, repoEnv(repo), "issue", "view", strconv.Itoa(number), "--repo", target(repo), "--json", "number,state,title")
	if err != nil { return domain.GitHubIssue{}, err }
	var payload domain.GitHubIssue
	if err := json.Unmarshal(out, &payload); err != nil { return domain.GitHubIssue{}, fmt.Errorf("decode gh issue view: %w", err) }
	return payload, nil
}
func (c *Client) IssueDetails(ctx context.Context, repo domain.RepositoryIdentity, number int) (domain.GitHubIssueDetails, error) {
	out, err := c.runWithEnv(ctx, repoEnv(repo), "issue", "view", strconv.Itoa(number), "--repo", target(repo), "--json", "number,state,title,body,url")
	if err != nil { return domain.GitHubIssueDetails{}, err }
	var p domain.GitHubIssueDetails
	if err := json.Unmarshal(out, &p); err != nil { return domain.GitHubIssueDetails{}, fmt.Errorf("decode gh issue view: %w", err) }
	return p, nil
}
func (c *Client) IssueComments(ctx context.Context, repo domain.RepositoryIdentity, number int) ([]domain.GitHubComment, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/issues/%d/comments?per_page=100", repo.Owner, repo.Repo, number)
	out, err := c.runWithEnv(ctx, repoEnv(repo), "api", "--paginate", "--slurp", endpoint, "-H", "Accept: application/vnd.github+json")
	if err != nil { return nil, err }
	comments, err := ParseCommentPages(out)
	if err != nil { return nil, fmt.Errorf("decode gh issue comments: %w", err) }
	return comments, nil
}
func ParseCommentPages(out []byte) ([]domain.GitHubComment, error) {
	var pages [][]domain.GitHubComment
	if err := json.Unmarshal(out, &pages); err != nil { return nil, err }
	var all []domain.GitHubComment
	for _, p := range pages { all = append(all, p...) }
	return all, nil
}
func (c *Client) CreateIssue(ctx context.Context, repo domain.RepositoryIdentity, title, body string) (domain.GitHubIssueDetails, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second); defer cancel()
	res, err := c.runner.Run(cmdCtx, ports.Command{Name: "gh", Args: []string{"issue", "create", "--repo", target(repo), "--title", title, "--body-file", "-"}, Env: repoEnv(repo), Stdin: []byte(body)})
	if err != nil { return domain.GitHubIssueDetails{}, err }
	u := strings.TrimSpace(string(res.Stdout)); parts := strings.Split(strings.TrimSuffix(u, "/"), "/")
	if len(parts) == 0 { return domain.GitHubIssueDetails{}, fmt.Errorf("gh issue create returned no issue URL") }
	n, err := strconv.Atoi(parts[len(parts)-1]); if err != nil { return domain.GitHubIssueDetails{}, fmt.Errorf("parse created issue number: %w", err) }
	return c.IssueDetails(ctx, repo, n)
}
func (c *Client) AppendIssueComment(ctx context.Context, repo domain.RepositoryIdentity, number int, body string) error {
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second); defer cancel()
	_, err := c.runner.Run(cmdCtx, ports.Command{Name: "gh", Args: []string{"issue", "comment", strconv.Itoa(number), "--repo", target(repo), "--body-file", "-"}, Env: repoEnv(repo), Stdin: []byte(body)})
	return err
}
func (c *Client) Login(ctx context.Context, host string) error {
	args := []string{"auth", "login"}; if host != "" { args = append(args, "--hostname", host) }
	return c.runner.RunInteractive(ctx, ports.Command{Name: "gh", Args: args}, c.stdin, c.stdout, c.stderr)
}
